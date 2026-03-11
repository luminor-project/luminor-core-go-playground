package facade

// CreatePropertyDTO holds data for creating a property (subject).
type CreatePropertyDTO struct {
	Name               string
	Detail             string
	OrgID              string
	CreatedByAccountID string
}

// CreateTenantDTO holds data for creating a tenant party.
type CreateTenantDTO struct {
	Name               string
	OrgID              string
	CreatedByAccountID string
}

// AssignTenantDTO holds data for assigning a tenant to a property (creating a rental).
type AssignTenantDTO struct {
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
}

// InviteTenantDTO holds data for inviting a tenant to create an account.
type InviteTenantDTO struct {
	TenantPartyID  string
	Email          string
	OrgID          string
	ActorAccountID string
}
