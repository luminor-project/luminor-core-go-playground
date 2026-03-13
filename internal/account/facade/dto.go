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

// MagicLinkRequestDTO holds data for requesting a magic link.
type MagicLinkRequestDTO struct {
	Email string
}

// MagicLinkResultDTO holds the result of verifying a magic link.
type MagicLinkResultDTO struct {
	AccountID string
	Email     string
	Roles     []string
}
