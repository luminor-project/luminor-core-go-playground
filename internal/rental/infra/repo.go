package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

// PostgresRepository implements domain.Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed rental repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, rental domain.Rental) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rentals (id, subject_id, tenant_party_id, org_id, created_by_account_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		rental.ID, rental.SubjectID, rental.TenantPartyID, rental.OrgID, rental.CreatedByAccountID, rental.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert rental: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (domain.Rental, error) {
	var rental domain.Rental
	err := r.pool.QueryRow(ctx,
		`SELECT id, subject_id, tenant_party_id, org_id, created_by_account_id, created_at
		 FROM rentals WHERE id = $1`, id).
		Scan(&rental.ID, &rental.SubjectID, &rental.TenantPartyID, &rental.OrgID, &rental.CreatedByAccountID, &rental.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Rental{}, domain.ErrRentalNotFound
		}
		return domain.Rental{}, fmt.Errorf("find rental by id: %w", err)
	}
	return rental, nil
}

func (r *PostgresRepository) FindBySubjectID(ctx context.Context, subjectID string) ([]domain.Rental, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, subject_id, tenant_party_id, org_id, created_by_account_id, created_at
		 FROM rentals WHERE subject_id = $1`, subjectID)
	if err != nil {
		return nil, fmt.Errorf("query rentals by subject: %w", err)
	}
	defer rows.Close()
	return scanRentals(rows)
}

func (r *PostgresRepository) FindByTenantPartyID(ctx context.Context, tenantPartyID string) ([]domain.Rental, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, subject_id, tenant_party_id, org_id, created_by_account_id, created_at
		 FROM rentals WHERE tenant_party_id = $1`, tenantPartyID)
	if err != nil {
		return nil, fmt.Errorf("query rentals by tenant: %w", err)
	}
	defer rows.Close()
	return scanRentals(rows)
}

func (r *PostgresRepository) FindByOrgID(ctx context.Context, orgID string) ([]domain.Rental, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, subject_id, tenant_party_id, org_id, created_by_account_id, created_at
		 FROM rentals WHERE org_id = $1`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query rentals by org: %w", err)
	}
	defer rows.Close()
	return scanRentals(rows)
}

func (r *PostgresRepository) ExistsBySubjectAndTenant(ctx context.Context, subjectID, tenantPartyID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM rentals WHERE subject_id = $1 AND tenant_party_id = $2)`,
		subjectID, tenantPartyID).Scan(&exists)
	return exists, err
}

func scanRentals(rows pgx.Rows) ([]domain.Rental, error) {
	var result []domain.Rental
	for rows.Next() {
		var rental domain.Rental
		if err := rows.Scan(&rental.ID, &rental.SubjectID, &rental.TenantPartyID, &rental.OrgID, &rental.CreatedByAccountID, &rental.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rental: %w", err)
		}
		result = append(result, rental)
	}
	return result, rows.Err()
}
