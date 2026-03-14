package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyTaken   = errors.New("email already taken")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountNotFound     = errors.New("account not found")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrAlreadyLinked       = errors.New("account already linked to this party")
	ErrPendingLinkNotFound = errors.New("pending party link not found")
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
	CreatePasswordResetToken(ctx context.Context, token PasswordResetToken) error
	FindPasswordResetTokenByHash(ctx context.Context, tokenHash string) (PasswordResetToken, error)
	MarkPasswordResetTokenAsUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
	DeleteExpiredPasswordResetTokens(ctx context.Context, before time.Time) error
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

// CreatePasswordResetToken creates a new password reset token for an account.
// Returns the plaintext token to be sent to the user.
func (s *AccountService) CreatePasswordResetToken(ctx context.Context, accountID string, expiryDuration time.Duration) (string, error) {
	plainToken, hashedToken, err := GenerateResetToken()
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}

	token := PasswordResetToken{
		AccountID: accountID,
		TokenHash: hashedToken,
		ExpiresAt: s.clock.Now().Add(expiryDuration),
		CreatedAt: s.clock.Now(),
	}

	// Delete any existing token for this account first
	if err := s.repo.ExecuteInTx(ctx, func(repo Repository) error {
		return repo.CreatePasswordResetToken(ctx, token)
	}); err != nil {
		return "", fmt.Errorf("create password reset token: %w", err)
	}

	return plainToken, nil
}

// ValidateResetToken validates a password reset token and returns the associated account ID.
func (s *AccountService) ValidateResetToken(ctx context.Context, plainToken string) (string, error) {
	tokenHash := HashToken(plainToken)

	token, err := s.repo.FindPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		return "", ErrInvalidResetToken
	}

	if !token.IsValid(s.clock.Now()) {
		if token.UsedAt != nil {
			return "", ErrResetTokenAlreadyUsed
		}
		return "", ErrResetTokenExpired
	}

	return token.AccountID, nil
}

// ResetPassword resets an account's password using a valid reset token.
func (s *AccountService) ResetPassword(ctx context.Context, plainToken, newPlainPassword string) error {
	if len(newPlainPassword) < 8 {
		return ErrPasswordTooShort
	}

	tokenHash := HashToken(plainToken)

	token, err := s.repo.FindPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		return ErrInvalidResetToken
	}

	now := s.clock.Now()
	if !token.IsValid(now) {
		if token.UsedAt != nil {
			return ErrResetTokenAlreadyUsed
		}
		return ErrResetTokenExpired
	}

	// Update password and mark token as used in a transaction
	if err := s.repo.ExecuteInTx(ctx, func(repo Repository) error {
		account, err := repo.FindByID(ctx, token.AccountID)
		if err != nil {
			return fmt.Errorf("find account: %w", err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(newPlainPassword), s.bcryptCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		account.PasswordHash = string(hash)
		account.MustSetPassword = false

		if err := repo.Update(ctx, account); err != nil {
			return fmt.Errorf("update account: %w", err)
		}

		if err := repo.MarkPasswordResetTokenAsUsed(ctx, tokenHash, now); err != nil {
			return fmt.Errorf("mark token as used: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
