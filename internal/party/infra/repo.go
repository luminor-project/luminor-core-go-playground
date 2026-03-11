package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
)

// PostgresRepository implements domain.Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed party repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// UpsertProjection inserts or updates a party in the read model (parties table).
func (r *PostgresRepository) UpsertProjection(ctx context.Context, id, actorKind, partyKind, name, orgID, createdByAccountID string, createdAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO parties (id, actor_kind, party_kind, name, owning_organization_id, created_by_account_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET actor_kind = $2, party_kind = $3, name = $4, owning_organization_id = $5, created_by_account_id = $6, created_at = $7`,
		id, actorKind, partyKind, name, orgID, createdByAccountID, createdAt)
	if err != nil {
		return fmt.Errorf("upsert party projection: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (domain.Party, error) {
	var p domain.Party
	err := r.pool.QueryRow(ctx,
		`SELECT id, actor_kind, party_kind, name, owning_organization_id, created_by_account_id, created_at
		 FROM parties WHERE id = $1`, id).
		Scan(&p.ID, &p.ActorKind, &p.PartyKind, &p.Name, &p.OwningOrganizationID, &p.CreatedByAccountID, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Party{}, domain.ErrPartyNotFound
		}
		return domain.Party{}, fmt.Errorf("find party by id: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.Party, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, actor_kind, party_kind, name, owning_organization_id, created_by_account_id, created_at
		 FROM parties WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("query parties by ids: %w", err)
	}
	defer rows.Close()
	return scanParties(rows)
}

func (r *PostgresRepository) FindByOrganizationID(ctx context.Context, orgID string) ([]domain.Party, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, actor_kind, party_kind, name, owning_organization_id, created_by_account_id, created_at
		 FROM parties WHERE owning_organization_id = $1`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query parties by org: %w", err)
	}
	defer rows.Close()
	return scanParties(rows)
}

func (r *PostgresRepository) FindByOrgAndKind(ctx context.Context, orgID string, kind domain.PartyKind) ([]domain.Party, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, actor_kind, party_kind, name, owning_organization_id, created_by_account_id, created_at
		 FROM parties WHERE owning_organization_id = $1 AND party_kind = $2`, orgID, string(kind))
	if err != nil {
		return nil, fmt.Errorf("query parties by org and kind: %w", err)
	}
	defer rows.Close()
	return scanParties(rows)
}

func scanParties(rows pgx.Rows) ([]domain.Party, error) {
	var result []domain.Party
	for rows.Next() {
		var p domain.Party
		if err := rows.Scan(&p.ID, &p.ActorKind, &p.PartyKind, &p.Name, &p.OwningOrganizationID, &p.CreatedByAccountID, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan party: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
