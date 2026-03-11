package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyTaken  = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountNotFound    = errors.New("account not found")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
)

// ValidationError represents a user-facing validation failure with a translation key.
type ValidationError struct{ Key string }

func (e *ValidationError) Error() string { return e.Key }

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Repository defines the persistence interface for accounts.
type Repository interface {
	FindByID(ctx context.Context, id string) (AccountCore, error)
	FindByEmail(ctx context.Context, email string) (AccountCore, error)
	Create(ctx context.Context, account AccountCore) error
	Update(ctx context.Context, account AccountCore) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
	FindByIDs(ctx context.Context, ids []string) ([]AccountCore, error)
	ExecuteInTx(ctx context.Context, fn func(repo Repository) error) error
}

// AccountService handles core account business logic.
type AccountService struct {
	repo       Repository
	clock      Clock
	bcryptCost int
}

// NewAccountService creates a new AccountService.
func NewAccountService(repo Repository, clock Clock) *AccountService {
	return &AccountService{repo: repo, clock: clock, bcryptCost: bcrypt.DefaultCost}
}

// WithBcryptCost returns a copy of the service with the given bcrypt cost.
func (s *AccountService) WithBcryptCost(cost int) *AccountService {
	s.bcryptCost = cost
	return s
}

// Register creates a new account with hashed password.
func (s *AccountService) Register(ctx context.Context, email, plainPassword string) (AccountCore, error) {
	if len(plainPassword) < 8 {
		return AccountCore{}, ErrPasswordTooShort
	}

	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return AccountCore{}, fmt.Errorf("check email exists: %w", err)
	}
	if exists {
		return AccountCore{}, ErrEmailAlreadyTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), s.bcryptCost)
	if err != nil {
		return AccountCore{}, fmt.Errorf("hash password: %w", err)
	}

	account := NewAccountCore(email, string(hash), s.clock.Now())

	if err := s.repo.Create(ctx, account); err != nil {
		return AccountCore{}, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

// Authenticate verifies credentials and returns the account.
func (s *AccountService) Authenticate(ctx context.Context, email, plainPassword string) (AccountCore, error) {
	account, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return AccountCore{}, ErrInvalidCredentials
		}
		return AccountCore{}, fmt.Errorf("find account: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(plainPassword)); err != nil {
		return AccountCore{}, ErrInvalidCredentials
	}

	return account, nil
}

// SetPassword updates the password for an account.
func (s *AccountService) SetPassword(ctx context.Context, accountID, newPlainPassword string) error {
	if len(newPlainPassword) < 8 {
		return ErrPasswordTooShort
	}

	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("find account: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPlainPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	account.PasswordHash = string(hash)
	account.MustSetPassword = false

	return s.repo.Update(ctx, account)
}

// SetActiveOrganization sets the currently active organization for the account.
func (s *AccountService) SetActiveOrganization(ctx context.Context, accountID, orgID string) error {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("find account: %w", err)
	}

	account.CurrentlyActiveOrganizationID = orgID
	return s.repo.Update(ctx, account)
}

// FindByID returns an account by ID.
func (s *AccountService) FindByID(ctx context.Context, id string) (AccountCore, error) {
	return s.repo.FindByID(ctx, id)
}

// FindByEmail returns an account by email.
func (s *AccountService) FindByEmail(ctx context.Context, email string) (AccountCore, error) {
	return s.repo.FindByEmail(ctx, email)
}

// FindByIDs returns accounts by IDs.
func (s *AccountService) FindByIDs(ctx context.Context, ids []string) ([]AccountCore, error) {
	return s.repo.FindByIDs(ctx, ids)
}
