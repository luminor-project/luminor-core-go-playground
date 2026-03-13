package facade

import (
	"errors"
	"time"
)

var (
	ErrAlreadyLinked       = errors.New("account already linked to this party")
	ErrPendingLinkNotFound = errors.New("pending party link not found")
)

// AccountInfoDTO holds account data for cross-vertical communication.
type AccountInfoDTO struct {
	ID                            string
	Email                         string
	Roles                         []string
	CreatedAt                     time.Time
	CurrentlyActiveOrganizationID string
	CurrentlyActivePartyID        string
}

// PartyMembershipDTO holds a party membership for cross-vertical communication.
type PartyMembershipDTO struct {
	AccountID string
	PartyID   string
	OrgID     string
	CreatedAt time.Time
}

// DisplayName returns a display-friendly name.
func (d AccountInfoDTO) DisplayName() string {
	return d.Email
}

// RegistrationDTO holds data for account registration.
type RegistrationDTO struct {
	Email           string
	PlainPassword   string
	MustSetPassword bool
}

// PasswordResetRequestDTO initiates the reset flow.
type PasswordResetRequestDTO struct {
	Email string
}

// PasswordResetCompletionDTO finalizes the reset.
// Note: AccountID is no longer required - it's extracted from the validated token.
type PasswordResetCompletionDTO struct {
	RawToken    string
	NewPassword string
}
