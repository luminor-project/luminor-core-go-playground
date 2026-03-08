package testharness

import (
	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
)

// MakeOrganization creates a test organization with sensible defaults.
func MakeOrganization(ownerID, name string) domain.Organization {
	return domain.NewOrganization(ownerID, name)
}

// MakeInvitation creates a test invitation.
func MakeInvitation(orgID, email string) domain.Invitation {
	return domain.NewInvitation(orgID, email)
}
