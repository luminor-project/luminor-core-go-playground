package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
)

// PostgresRepository implements domain.Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed subject repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) UpsertProjection(ctx context.Context, id, subjectKind, name, detail, orgID, createdByAccountID string, createdAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO subjects (id, subject_kind, name, detail, owning_organization_id, created_by_account_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET subject_kind = $2, name = $3, detail = $4, owning_organization_id = $5, created_by_account_id = $6, created_at = $7`,
		id, subjectKind, name, detail, orgID, createdByAccountID, createdAt)
	if err != nil {
		return fmt.Errorf("upsert subject projection: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (domain.Subject, error) {
	var s domain.Subject
	err := r.pool.QueryRow(ctx,
		`SELECT id, subject_kind, name, detail, owning_organization_id, created_by_account_id, created_at
		 FROM subjects WHERE id = $1`, id).
		Scan(&s.ID, &s.SubjectKind, &s.Name, &s.Detail, &s.OwningOrganizationID, &s.CreatedByAccountID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subject{}, domain.ErrSubjectNotFound
		}
		return domain.Subject{}, fmt.Errorf("find subject by id: %w", err)
	}
	return s, nil
}

func (r *PostgresRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.Subject, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, subject_kind, name, detail, owning_organization_id, created_by_account_id, created_at
		 FROM subjects WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("query subjects by ids: %w", err)
	}
	defer rows.Close()
	return scanSubjects(rows)
}

func (r *PostgresRepository) FindByOrganizationID(ctx context.Context, orgID string) ([]domain.Subject, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, subject_kind, name, detail, owning_organization_id, created_by_account_id, created_at
		 FROM subjects WHERE owning_organization_id = $1`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query subjects by org: %w", err)
	}
	defer rows.Close()
	return scanSubjects(rows)
}

func (r *PostgresRepository) FindByOrgAndKind(ctx context.Context, orgID string, kind domain.SubjectKind) ([]domain.Subject, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, subject_kind, name, detail, owning_organization_id, created_by_account_id, created_at
		 FROM subjects WHERE owning_organization_id = $1 AND subject_kind = $2`, orgID, string(kind))
	if err != nil {
		return nil, fmt.Errorf("query subjects by org and kind: %w", err)
	}
	defer rows.Close()
	return scanSubjects(rows)
}

func scanSubjects(rows pgx.Rows) ([]domain.Subject, error) {
	var result []domain.Subject
	for rows.Next() {
		var s domain.Subject
		if err := rows.Scan(&s.ID, &s.SubjectKind, &s.Name, &s.Detail, &s.OwningOrganizationID, &s.CreatedByAccountID, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
