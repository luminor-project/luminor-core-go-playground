package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
)

// PostgresRepository implements domain.Repository for the organization vertical.
type PostgresRepository struct {
	pool *pgxpool.Pool
	db   dbExecutor
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewPostgresRepository creates a new PostgreSQL-backed organization repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, db: pool}
}

func (r *PostgresRepository) withTx(tx pgx.Tx) *PostgresRepository {
	return &PostgresRepository{pool: r.pool, db: tx}
}

func (r *PostgresRepository) ExecuteInTx(ctx context.Context, fn func(repo domain.Repository) error) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return fn(r.withTx(tx))
	})
}

// Organizations

func (r *PostgresRepository) CreateOrganization(ctx context.Context, org domain.Organization) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO organizations (id, owning_users_id, name, created_at) VALUES ($1, $2, $3, $4)`,
		org.ID, org.OwningUsersID, nilIfEmpty(org.Name), org.CreatedAt)
	return err
}

func (r *PostgresRepository) FindOrganizationByID(ctx context.Context, id string) (domain.Organization, error) {
	var org domain.Organization
	var name *string
	err := r.db.QueryRow(ctx,
		`SELECT id, owning_users_id, name, created_at FROM organizations WHERE id = $1`, id).
		Scan(&org.ID, &org.OwningUsersID, &name, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Organization{}, domain.ErrOrganizationNotFound
		}
		return domain.Organization{}, err
	}
	if name != nil {
		org.Name = *name
	}
	return org, nil
}

func (r *PostgresRepository) UpdateOrganization(ctx context.Context, org domain.Organization) error {
	_, err := r.db.Exec(ctx,
		`UPDATE organizations SET name = $2 WHERE id = $1`,
		org.ID, nilIfEmpty(org.Name))
	return err
}

func (r *PostgresRepository) GetAllOrganizationsForUser(ctx context.Context, userID string) ([]domain.Organization, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, owning_users_id, name, created_at FROM organizations WHERE owning_users_id = $1
		 UNION
		 SELECT o.id, o.owning_users_id, o.name, o.created_at FROM organizations o
		 INNER JOIN organization_members om ON o.id = om.organization_id
		 WHERE om.account_core_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []domain.Organization
	for rows.Next() {
		var org domain.Organization
		var name *string
		if err := rows.Scan(&org.ID, &org.OwningUsersID, &name, &org.CreatedAt); err != nil {
			return nil, err
		}
		if name != nil {
			org.Name = *name
		}
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

func (r *PostgresRepository) UserIsOwnerOfOrganization(ctx context.Context, userID, orgID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE id = $1 AND owning_users_id = $2)`,
		orgID, userID).Scan(&exists)
	return exists, err
}

// Members

func (r *PostgresRepository) AddMember(ctx context.Context, member domain.OrganizationMember) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO organization_members (account_core_id, organization_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		member.AccountCoreID, member.OrganizationID)
	return err
}

func (r *PostgresRepository) IsMember(ctx context.Context, accountCoreID, orgID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM organization_members WHERE account_core_id = $1 AND organization_id = $2
			UNION ALL
			SELECT 1 FROM organizations WHERE owning_users_id = $1 AND id = $2
		)`, accountCoreID, orgID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetMemberIDs(ctx context.Context, orgID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT account_core_id FROM organization_members WHERE organization_id = $1`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) GetOwnerID(ctx context.Context, orgID string) (string, error) {
	var ownerID string
	err := r.db.QueryRow(ctx,
		`SELECT owning_users_id FROM organizations WHERE id = $1`, orgID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrOrganizationNotFound
		}
		return "", err
	}
	return ownerID, nil
}

// Groups

func (r *PostgresRepository) CreateGroup(ctx context.Context, group domain.Group) error {
	rights := make([]string, len(group.AccessRights))
	for i, ar := range group.AccessRights {
		rights[i] = ar.String()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO organization_groups (id, organization_id, name, access_rights, is_default_for_new_members, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		group.ID, group.OrganizationID, group.Name, rights, group.IsDefaultForNewMembers, group.CreatedAt)
	return err
}

func (r *PostgresRepository) FindGroupByID(ctx context.Context, id string) (domain.Group, error) {
	var group domain.Group
	var rights []string
	err := r.db.QueryRow(ctx,
		`SELECT id, organization_id, name, access_rights, is_default_for_new_members, created_at
		 FROM organization_groups WHERE id = $1`, id).
		Scan(&group.ID, &group.OrganizationID, &group.Name, &rights, &group.IsDefaultForNewMembers, &group.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Group{}, domain.ErrGroupNotFound
		}
		return domain.Group{}, err
	}
	group.AccessRights = parseAccessRights(rights)
	return group, nil
}

func (r *PostgresRepository) GetGroups(ctx context.Context, orgID string) ([]domain.Group, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, organization_id, name, access_rights, is_default_for_new_members, created_at
		 FROM organization_groups WHERE organization_id = $1 ORDER BY created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		var rights []string
		if err := rows.Scan(&g.ID, &g.OrganizationID, &g.Name, &rights, &g.IsDefaultForNewMembers, &g.CreatedAt); err != nil {
			return nil, err
		}
		g.AccessRights = parseAccessRights(rights)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *PostgresRepository) GetGroupsOfUser(ctx context.Context, userID, orgID string) ([]domain.Group, error) {
	rows, err := r.db.Query(ctx,
		`SELECT g.id, g.organization_id, g.name, g.access_rights, g.is_default_for_new_members, g.created_at
		 FROM organization_groups g
		 INNER JOIN organization_group_members gm ON g.id = gm.group_id
		 WHERE gm.account_core_id = $1 AND g.organization_id = $2`, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		var rights []string
		if err := rows.Scan(&g.ID, &g.OrganizationID, &g.Name, &rights, &g.IsDefaultForNewMembers, &g.CreatedAt); err != nil {
			return nil, err
		}
		g.AccessRights = parseAccessRights(rights)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *PostgresRepository) AddUserToGroup(ctx context.Context, gm domain.GroupMember) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO organization_group_members (account_core_id, group_id, organization_id)
		 SELECT $1, g.id, g.organization_id
		 FROM organization_groups g
		 WHERE g.id = $2
		 ON CONFLICT DO NOTHING`,
		gm.AccountCoreID, gm.GroupID)
	return err
}

func (r *PostgresRepository) RemoveUserFromGroup(ctx context.Context, accountCoreID, groupID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM organization_group_members
		 WHERE account_core_id = $1 AND group_id = $2`,
		accountCoreID, groupID)
	return err
}

func (r *PostgresRepository) IsGroupMember(ctx context.Context, accountCoreID, groupID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM organization_group_members WHERE account_core_id = $1 AND group_id = $2)`,
		accountCoreID, groupID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT account_core_id FROM organization_group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) GetDefaultGroup(ctx context.Context, orgID string) (domain.Group, error) {
	var group domain.Group
	var rights []string
	err := r.db.QueryRow(ctx,
		`SELECT id, organization_id, name, access_rights, is_default_for_new_members, created_at
		 FROM organization_groups WHERE organization_id = $1 AND is_default_for_new_members = true LIMIT 1`, orgID).
		Scan(&group.ID, &group.OrganizationID, &group.Name, &rights, &group.IsDefaultForNewMembers, &group.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Group{}, fmt.Errorf("no default group for org %s", orgID)
		}
		return domain.Group{}, err
	}
	group.AccessRights = parseAccessRights(rights)
	return group, nil
}

// Invitations

func (r *PostgresRepository) CreateInvitation(ctx context.Context, inv domain.Invitation) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO organization_invitations (id, organization_id, email, created_at) VALUES ($1, $2, $3, $4)`,
		inv.ID, inv.OrganizationID, inv.Email, inv.CreatedAt)
	return err
}

func (r *PostgresRepository) FindInvitationByID(ctx context.Context, id string) (domain.Invitation, error) {
	var inv domain.Invitation
	err := r.db.QueryRow(ctx,
		`SELECT id, organization_id, email, created_at FROM organization_invitations WHERE id = $1`, id).
		Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invitation{}, domain.ErrInvitationNotFound
		}
		return domain.Invitation{}, err
	}
	return inv, nil
}

func (r *PostgresRepository) GetPendingInvitations(ctx context.Context, orgID string) ([]domain.Invitation, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, organization_id, email, created_at FROM organization_invitations
		 WHERE organization_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []domain.Invitation
	for rows.Next() {
		var inv domain.Invitation
		if err := rows.Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

func (r *PostgresRepository) InvitationExistsForEmail(ctx context.Context, orgID, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM organization_invitations WHERE organization_id = $1 AND email = $2)`,
		orgID, email).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) DeleteInvitation(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM organization_invitations WHERE id = $1`, id)
	return err
}

// Helpers

func parseAccessRights(rights []string) []domain.AccessRight {
	result := make([]domain.AccessRight, len(rights))
	for i, r := range rights {
		result[i] = domain.AccessRight(r)
	}
	return result
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
