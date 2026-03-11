package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// AccountCore is the core entity representing a user account.
type AccountCore struct {
	ID                            string
	Email                         string
	PasswordHash                  string
	Roles                         []Role
	MustSetPassword               bool
	CurrentlyActiveOrganizationID string
	CurrentlyActivePartyID        string
	CreatedAt                     time.Time
}

// PartyMembership links an account to a party within an organization.
type PartyMembership struct {
	AccountID string
	PartyID   string
	OrgID     string
	CreatedAt time.Time
}

// PendingPartyLink holds a deferred party-account link for an invitation.
type PendingPartyLink struct {
	ID           string
	InvitationID string
	PartyID      string
	OrgID        string
	CreatedAt    time.Time
}

// NewAccountCore creates a new account with generated UUID and normalized email.
func NewAccountCore(email, passwordHash string, now time.Time) AccountCore {
	return AccountCore{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: passwordHash,
		Roles:        []Role{RoleUser},
		CreatedAt:    now,
	}
}

// HasRole checks if the account has a specific role.
func (a AccountCore) HasRole(role Role) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AddRole adds a role if not already present.
func (a *AccountCore) AddRole(role Role) {
	if !a.HasRole(role) {
		a.Roles = append(a.Roles, role)
	}
}

// RemoveRole removes a role from the account.
func (a *AccountCore) RemoveRole(roleToRemove Role) {
	var remaining []Role
	for _, r := range a.Roles {
		if r != roleToRemove {
			remaining = append(remaining, r)
		}
	}
	a.Roles = remaining
}

// IsAdmin returns true if the account has the admin role.
func (a AccountCore) IsAdmin() bool {
	return a.HasRole(RoleAdmin)
}

// RoleStrings returns roles as string slice.
func (a AccountCore) RoleStrings() []string {
	result := make([]string, len(a.Roles))
	for i, r := range a.Roles {
		result[i] = r.String()
	}
	return result
}

// DisplayName returns a display-friendly name (email for now).
func (a AccountCore) DisplayName() string {
	return a.Email
}
