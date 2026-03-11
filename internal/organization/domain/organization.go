package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Organization represents a team/workspace.
type Organization struct {
	ID            string
	OwningUsersID string
	Name          string
	CreatedAt     time.Time
}

// NewOrganization creates a new organization.
func NewOrganization(owningUsersID, name string, now time.Time) Organization {
	return Organization{
		ID:            uuid.New().String(),
		OwningUsersID: owningUsersID,
		Name:          truncate(strings.TrimSpace(name), 256),
		CreatedAt:     now,
	}
}

// Group represents a permission group within an organization.
type Group struct {
	ID                     string
	OrganizationID         string
	Name                   string
	AccessRights           []AccessRight
	IsDefaultForNewMembers bool
	CreatedAt              time.Time
}

// NewGroup creates a new group.
func NewGroup(orgID, name string, accessRights []AccessRight, isDefault bool, now time.Time) Group {
	return Group{
		ID:                     uuid.New().String(),
		OrganizationID:         orgID,
		Name:                   name,
		AccessRights:           accessRights,
		IsDefaultForNewMembers: isDefault,
		CreatedAt:              now,
	}
}

// IsAdministratorsGroup returns true if this is the administrators group.
func (g Group) IsAdministratorsGroup() bool {
	return g.Name == "Administrators"
}

// IsTeamMembersGroup returns true if this is the team members group.
func (g Group) IsTeamMembersGroup() bool {
	return g.Name == "Team Members"
}

// HasAccessRight checks if the group has a specific access right.
func (g Group) HasAccessRight(right AccessRight) bool {
	for _, r := range g.AccessRights {
		if r == right || r == AccessRightFullAccess {
			return true
		}
	}
	return false
}

// Invitation represents an invitation to join an organization.
type Invitation struct {
	ID             string
	OrganizationID string
	Email          string
	CreatedAt      time.Time
}

// NewInvitation creates a new invitation.
func NewInvitation(orgID, email string, now time.Time) Invitation {
	return Invitation{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Email:          strings.ToLower(strings.TrimSpace(email)),
		CreatedAt:      now,
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
