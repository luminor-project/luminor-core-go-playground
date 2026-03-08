package facade

import "time"

// OrganizationDTO holds organization data for cross-vertical communication.
type OrganizationDTO struct {
	ID       string
	Name     string
	IsOwned  bool
	IsActive bool
}

// MemberDTO holds member data for the organization dashboard.
type MemberDTO struct {
	ID            string
	DisplayName   string
	Email         string
	IsOwner       bool
	IsCurrentUser bool
	GroupIDs      []string
}

// GroupDTO holds group data for the organization dashboard.
type GroupDTO struct {
	ID        string
	Name      string
	IsDefault bool
}

// InvitationDTO holds invitation data for display.
type InvitationDTO struct {
	ID        string
	Email     string
	CreatedAt time.Time
}
