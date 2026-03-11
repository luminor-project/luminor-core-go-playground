package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"golang.org/x/crypto/bcrypt"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

// mockRepository is an in-memory repository for testing.
type mockRepository struct {
	accounts map[string]domain.AccountCore
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		accounts: make(map[string]domain.AccountCore),
	}
}

func (m *mockRepository) FindByID(_ context.Context, id string) (domain.AccountCore, error) {
	a, ok := m.accounts[id]
	if !ok {
		return domain.AccountCore{}, domain.ErrAccountNotFound
	}
	return a, nil
}

func (m *mockRepository) FindByEmail(_ context.Context, email string) (domain.AccountCore, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return a, nil
		}
	}
	return domain.AccountCore{}, domain.ErrAccountNotFound
}

func (m *mockRepository) Create(_ context.Context, account domain.AccountCore) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockRepository) Update(_ context.Context, account domain.AccountCore) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, a := range m.accounts {
		if a.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) ExistsByID(_ context.Context, id string) (bool, error) {
	_, ok := m.accounts[id]
	return ok, nil
}

func (m *mockRepository) FindByIDs(_ context.Context, ids []string) ([]domain.AccountCore, error) {
	var result []domain.AccountCore
	for _, id := range ids {
		if a, ok := m.accounts[id]; ok {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) ExecuteInTx(_ context.Context, fn func(repo domain.Repository) error) error {
	return fn(m)
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", account.Email)
	}
	if account.ID == "" {
		t.Error("expected non-empty ID")
	}
	if !account.HasRole(domain.RoleUser) {
		t.Error("expected user role")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = svc.Register(context.Background(), "test@example.com", "password456")
	if !errors.Is(err, domain.ErrEmailAlreadyTaken) {
		t.Errorf("expected ErrEmailAlreadyTaken, got %v", err)
	}
}

func TestRegister_NormalizesEmail(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, err := svc.Register(context.Background(), "  Test@Example.COM  ", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected normalized email, got %q", account.Email)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	account, err := svc.Authenticate(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	if account.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", account.Email)
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	_, err = svc.Authenticate(context.Background(), "test@example.com", "wrong")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticate_NonexistentUser(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Authenticate(context.Background(), "nonexistent@example.com", "password123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestSetPassword(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "old_password")

	err := svc.SetPassword(context.Background(), account.ID, "new_password")
	if err != nil {
		t.Fatalf("set password failed: %v", err)
	}

	// Old password should no longer work
	_, err = svc.Authenticate(context.Background(), "test@example.com", "old_password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Error("old password should no longer work")
	}

	// New password should work
	_, err = svc.Authenticate(context.Background(), "test@example.com", "new_password")
	if err != nil {
		t.Errorf("new password should work, got: %v", err)
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	_, err := svc.Register(context.Background(), "test@example.com", "short")
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestSetPassword_PasswordTooShort(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.SetPassword(context.Background(), account.ID, "short")
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestSetActiveOrganization(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewAccountService(repo, testClock).WithBcryptCost(bcrypt.MinCost)

	account, _ := svc.Register(context.Background(), "test@example.com", "password123")

	err := svc.SetActiveOrganization(context.Background(), account.ID, "org-123")
	if err != nil {
		t.Fatalf("set active org failed: %v", err)
	}

	updated, _ := svc.FindByID(context.Background(), account.ID)
	if updated.CurrentlyActiveOrganizationID != "org-123" {
		t.Errorf("expected active org 'org-123', got %q", updated.CurrentlyActiveOrganizationID)
	}
}
