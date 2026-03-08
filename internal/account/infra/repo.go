package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
		`SELECT id, email, password_hash, roles, must_set_password, currently_active_organization_id, created_at
		 FROM account_cores WHERE id = $1`, id)

	return scanAccount(row)
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (domain.AccountCore, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, roles, must_set_password, currently_active_organization_id, created_at
		 FROM account_cores WHERE email = $1`, email)

	return scanAccount(row)
}

func (r *PostgresRepository) Create(ctx context.Context, account domain.AccountCore) error {
	rolesJSON, err := json.Marshal(account.RoleStrings())
	if err != nil {
		return fmt.Errorf("marshal roles: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO account_cores (id, email, password_hash, roles, must_set_password, currently_active_organization_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		account.ID, account.Email, account.PasswordHash, rolesJSON,
		account.MustSetPassword, nilIfEmpty(account.CurrentlyActiveOrganizationID), account.CreatedAt)
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
		 must_set_password = $5, currently_active_organization_id = $6
		 WHERE id = $1`,
		account.ID, account.Email, account.PasswordHash, rolesJSON,
		account.MustSetPassword, nilIfEmpty(account.CurrentlyActiveOrganizationID))
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
		`SELECT id, email, password_hash, roles, must_set_password, currently_active_organization_id, created_at
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

	err := s.Scan(&a.ID, &a.Email, &a.PasswordHash, &rolesJSON,
		&a.MustSetPassword, &activeOrgID, &a.CreatedAt)
	if err != nil {
		return domain.AccountCore{}, fmt.Errorf("scan account: %w", err)
	}

	if activeOrgID != nil {
		a.CurrentlyActiveOrganizationID = *activeOrgID
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

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
