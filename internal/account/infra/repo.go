package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
)

// PostgresRepository implements domain.Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
	db   dbExecutor
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewPostgresRepository creates a new PostgreSQL-backed account repository.
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

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (domain.AccountCore, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, roles, must_set_password, currently_active_organization_id, currently_active_party_id, created_at
		 FROM account_cores WHERE id = $1`, id)

	return scanAccount(row)
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (domain.AccountCore, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, roles, must_set_password, currently_active_organization_id, currently_active_party_id, created_at
		 FROM account_cores WHERE email = $1`, email)

	return scanAccount(row)
}

func (r *PostgresRepository) Create(ctx context.Context, account domain.AccountCore) error {
	rolesJSON, err := json.Marshal(account.RoleStrings())
	if err != nil {
		return fmt.Errorf("marshal roles: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO account_cores (id, email, password_hash, roles, must_set_password, currently_active_organization_id, currently_active_party_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		account.ID, account.Email, account.PasswordHash, rolesJSON,
		account.MustSetPassword, nilIfEmpty(account.CurrentlyActiveOrganizationID), nilIfEmpty(account.CurrentlyActivePartyID), account.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}

	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, account domain.AccountCore) error {
	rolesJSON, err := json.Marshal(account.RoleStrings())
	if err != nil {
		return fmt.Errorf("marshal roles: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`UPDATE account_cores SET email = $2, password_hash = $3, roles = $4,
		 must_set_password = $5, currently_active_organization_id = $6, currently_active_party_id = $7
		 WHERE id = $1`,
		account.ID, account.Email, account.PasswordHash, rolesJSON,
		account.MustSetPassword, nilIfEmpty(account.CurrentlyActiveOrganizationID), nilIfEmpty(account.CurrentlyActivePartyID))
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM account_cores WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM account_cores WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.AccountCore, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, email, password_hash, roles, must_set_password, currently_active_organization_id, currently_active_party_id, created_at
		 FROM account_cores WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []domain.AccountCore
	for rows.Next() {
		a, err := scanAccountFrom(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}

	return accounts, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAccountFrom(s scanner) (domain.AccountCore, error) {
	var a domain.AccountCore
	var rolesJSON []byte
	var activeOrgID *string
	var activePartyID *string

	err := s.Scan(&a.ID, &a.Email, &a.PasswordHash, &rolesJSON,
		&a.MustSetPassword, &activeOrgID, &activePartyID, &a.CreatedAt)
	if err != nil {
		return domain.AccountCore{}, fmt.Errorf("scan account: %w", err)
	}

	if activeOrgID != nil {
		a.CurrentlyActiveOrganizationID = *activeOrgID
	}
	if activePartyID != nil {
		a.CurrentlyActivePartyID = *activePartyID
	}

	var roleStrings []string
	if err := json.Unmarshal(rolesJSON, &roleStrings); err != nil {
		return domain.AccountCore{}, fmt.Errorf("unmarshal roles: %w", err)
	}
	for _, rs := range roleStrings {
		if role, ok := domain.ParseRole(rs); ok {
			a.Roles = append(a.Roles, role)
		}
	}

	return a, nil
}

func scanAccount(row pgx.Row) (domain.AccountCore, error) {
	a, err := scanAccountFrom(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AccountCore{}, domain.ErrAccountNotFound
		}
		return domain.AccountCore{}, err
	}
	return a, nil
}

// Party membership methods.

func (r *PostgresRepository) CreatePartyMembership(ctx context.Context, m domain.PartyMembership) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO account_party_memberships (account_id, party_id, org_id, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (account_id, party_id) DO NOTHING`,
		m.AccountID, m.PartyID, m.OrgID, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert party membership: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindPartyMembershipsByAccountAndOrg(ctx context.Context, accountID, orgID string) ([]domain.PartyMembership, error) {
	rows, err := r.db.Query(ctx,
		`SELECT account_id, party_id, org_id, created_at
		 FROM account_party_memberships
		 WHERE account_id = $1 AND org_id = $2`, accountID, orgID)
	if err != nil {
		return nil, fmt.Errorf("query memberships: %w", err)
	}
	defer rows.Close()

	var result []domain.PartyMembership
	for rows.Next() {
		var m domain.PartyMembership
		if err := rows.Scan(&m.AccountID, &m.PartyID, &m.OrgID, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) ExistsPartyMembership(ctx context.Context, accountID, partyID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM account_party_memberships WHERE account_id = $1 AND party_id = $2)`,
		accountID, partyID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) FindAccountIDsByPartyID(ctx context.Context, partyID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT account_id FROM account_party_memberships WHERE party_id = $1`, partyID)
	if err != nil {
		return nil, fmt.Errorf("query account IDs by party: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan account id: %w", err)
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

// Pending party link methods.

func (r *PostgresRepository) CreatePendingPartyLink(ctx context.Context, link domain.PendingPartyLink) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO account_party_pending_links (id, invitation_id, party_id, org_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		link.ID, link.InvitationID, link.PartyID, link.OrgID, link.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert pending party link: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindPendingPartyLinkByInvitationID(ctx context.Context, invitationID string) (domain.PendingPartyLink, error) {
	var link domain.PendingPartyLink
	err := r.db.QueryRow(ctx,
		`SELECT id, invitation_id, party_id, org_id, created_at
		 FROM account_party_pending_links WHERE invitation_id = $1`, invitationID).
		Scan(&link.ID, &link.InvitationID, &link.PartyID, &link.OrgID, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PendingPartyLink{}, domain.ErrPendingLinkNotFound
		}
		return domain.PendingPartyLink{}, fmt.Errorf("find pending link: %w", err)
	}
	return link, nil
}

func (r *PostgresRepository) DeletePendingPartyLink(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM account_party_pending_links WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pending party link: %w", err)
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Magic link token methods

func (r *PostgresRepository) CreateMagicLinkToken(ctx context.Context, token domain.MagicLinkToken) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO magic_link_tokens (id, account_id, token_hash, expires_at, used_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.AccountID, token.TokenHash, token.ExpiresAt, token.UsedAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert magic link token: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindMagicLinkTokenByHash(ctx context.Context, tokenHash string) (domain.MagicLinkToken, error) {
	var token domain.MagicLinkToken
	var usedAt *time.Time

	err := r.db.QueryRow(ctx,
		`SELECT id, account_id, token_hash, expires_at, used_at, created_at
		 FROM magic_link_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&token.ID, &token.AccountID, &token.TokenHash, &token.ExpiresAt, &usedAt, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MagicLinkToken{}, domain.ErrMagicLinkNotFound
		}
		return domain.MagicLinkToken{}, fmt.Errorf("find magic link token: %w", err)
	}

	token.UsedAt = usedAt
	return token, nil
}

func (r *PostgresRepository) MarkMagicLinkTokenUsed(ctx context.Context, tokenID string, usedAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE magic_link_tokens SET used_at = $1 WHERE id = $2`,
		usedAt, tokenID)
	if err != nil {
		return fmt.Errorf("mark magic link token used: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CountActiveMagicLinkTokens(ctx context.Context, accountID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM magic_link_tokens
		 WHERE account_id = $1 AND used_at IS NULL AND expires_at > now()`,
		accountID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active magic link tokens: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) InvalidateExistingMagicLinkTokens(ctx context.Context, accountID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE magic_link_tokens SET used_at = now()
		 WHERE account_id = $1 AND used_at IS NULL`,
		accountID)
	if err != nil {
		return fmt.Errorf("invalidate existing magic link tokens: %w", err)
	}
	return nil
}
