package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyTaken         = errors.New("email already taken")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrAccountNotFound           = errors.New("account not found")
	ErrPasswordTooShort          = errors.New("password must be at least 8 characters")
	ErrAlreadyLinked             = errors.New("account already linked to this party")
	ErrPendingLinkNotFound       = errors.New("pending party link not found")
	ErrPasswordResetTokenInvalid = errors.New("invalid or expired reset token")
	ErrPasswordResetExpired      = errors.New("password reset link has expired")
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

	// Party membership methods
	CreatePartyMembership(ctx context.Context, membership PartyMembership) error
	FindPartyMembershipsByAccountAndOrg(ctx context.Context, accountID, orgID string) ([]PartyMembership, error)
	ExistsPartyMembership(ctx context.Context, accountID, partyID string) (bool, error)
	FindAccountIDsByPartyID(ctx context.Context, partyID string) ([]string, error)

	// Pending party link methods
	CreatePendingPartyLink(ctx context.Context, link PendingPartyLink) error
	FindPendingPartyLinkByInvitationID(ctx context.Context, invitationID string) (PendingPartyLink, error)
	DeletePendingPartyLink(ctx context.Context, id string) error

	// Password reset token methods
	SavePasswordResetToken(ctx context.Context, token PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, tokenHash string) (PasswordResetToken, error)
	DeletePasswordResetToken(ctx context.Context, tokenHash string) error
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

// SetActiveParty sets the currently active party for the account.
func (s *AccountService) SetActiveParty(ctx context.Context, accountID, partyID string) error {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("find account: %w", err)
	}

	account.CurrentlyActivePartyID = partyID
	return s.repo.Update(ctx, account)
}

// LinkPartyToAccount creates a membership linking an account to a party.
func (s *AccountService) LinkPartyToAccount(ctx context.Context, accountID, partyID, orgID string) error {
	return s.repo.CreatePartyMembership(ctx, PartyMembership{
		AccountID: accountID,
		PartyID:   partyID,
		OrgID:     orgID,
		CreatedAt: s.clock.Now(),
	})
}

// GetPartyMembershipsForAccount returns all party memberships for an account in an org.
func (s *AccountService) GetPartyMembershipsForAccount(ctx context.Context, accountID, orgID string) ([]PartyMembership, error) {
	return s.repo.FindPartyMembershipsByAccountAndOrg(ctx, accountID, orgID)
}

// GetAccountIDsForParty returns all account IDs linked to a party.
func (s *AccountService) GetAccountIDsForParty(ctx context.Context, partyID string) ([]string, error) {
	return s.repo.FindAccountIDsByPartyID(ctx, partyID)
}

// CreatePendingPartyLink creates a deferred party link for an invitation.
func (s *AccountService) CreatePendingPartyLink(ctx context.Context, invitationID, partyID, orgID string) (PendingPartyLink, error) {
	link := PendingPartyLink{
		ID:           uuid.New().String(),
		InvitationID: invitationID,
		PartyID:      partyID,
		OrgID:        orgID,
		CreatedAt:    s.clock.Now(),
	}
	if err := s.repo.CreatePendingPartyLink(ctx, link); err != nil {
		return PendingPartyLink{}, fmt.Errorf("create pending party link: %w", err)
	}
	return link, nil
}

// ResolvePendingPartyLink resolves a pending link by linking the party to the account.
func (s *AccountService) ResolvePendingPartyLink(ctx context.Context, invitationID, accountID string) error {
	link, err := s.repo.FindPendingPartyLinkByInvitationID(ctx, invitationID)
	if err != nil {
		return err
	}

	if err := s.LinkPartyToAccount(ctx, accountID, link.PartyID, link.OrgID); err != nil {
		return fmt.Errorf("link party to account: %w", err)
	}

	return s.repo.DeletePendingPartyLink(ctx, link.ID)
}

// RequestPasswordReset generates a password reset token for the given email.
// Returns the account ID and token if the email exists, or empty strings if not found.
// The error is always nil to prevent timing attacks (same response time for existing/non-existing emails).
func (s *AccountService) RequestPasswordReset(ctx context.Context, email string) (accountID string, token string, err error) {
	account, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			// Return empty values but no error - prevents timing attacks
			return "", "", nil
		}
		return "", "", fmt.Errorf("find account: %w", err)
	}

	// Generate cryptographically secure random token (32 bytes = 64 hex chars)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	token = hex.EncodeToString(b)

	// Hash the token for storage (SHA-256)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// Token expires in 1 hour
	expiresAt := s.clock.Now().Add(1 * time.Hour)

	resetToken := PasswordResetToken{
		TokenHash: tokenHash,
		AccountID: account.ID,
		Email:     account.Email,
		ExpiresAt: expiresAt,
		CreatedAt: s.clock.Now(),
	}

	if err := s.repo.SavePasswordResetToken(ctx, resetToken); err != nil {
		return "", "", fmt.Errorf("save password reset token: %w", err)
	}

	return account.ID, token, nil
}

// ValidatePasswordResetToken checks if a token is valid and not expired.
// Returns the associated account ID if valid.
func (s *AccountService) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	// Hash the token for lookup
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	resetToken, err := s.repo.FindPasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrPasswordResetTokenInvalid) {
			return "", ErrPasswordResetTokenInvalid
		}
		return "", fmt.Errorf("find password reset token: %w", err)
	}

	// Check if token is expired
	if s.clock.Now().After(resetToken.ExpiresAt) {
		return "", ErrPasswordResetExpired
	}

	return resetToken.AccountID, nil
}

// ResetPassword validates the token and updates the password.
// The token is deleted after successful use (one-time use).
func (s *AccountService) ResetPassword(ctx context.Context, token, newPlainPassword string) error {
	if len(newPlainPassword) < 8 {
		return ErrPasswordTooShort
	}

	// Validate token and get account ID
	accountID, err := s.ValidatePasswordResetToken(ctx, token)
	if err != nil {
		return err
	}

	// Hash the token for deletion
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// Hash the new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPlainPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update account password and delete token in a transaction
	return s.repo.ExecuteInTx(ctx, func(repo Repository) error {
		// Delete the token first (one-time use)
		if err := repo.DeletePasswordResetToken(ctx, tokenHash); err != nil {
			return fmt.Errorf("delete password reset token: %w", err)
		}

		// Update the password
		account, err := repo.FindByID(ctx, accountID)
		if err != nil {
			return fmt.Errorf("find account: %w", err)
		}

		account.PasswordHash = string(passwordHash)
		account.MustSetPassword = false

		if err := repo.Update(ctx, account); err != nil {
			return fmt.Errorf("update account password: %w", err)
		}

		return nil
	})
}
