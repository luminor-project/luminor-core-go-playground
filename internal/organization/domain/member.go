package domain

// OrganizationMember tracks a user's membership in an organization.
type OrganizationMember struct {
	AccountCoreID  string
	OrganizationID string
}

// GroupMember tracks a user's membership in a group.
type GroupMember struct {
	AccountCoreID string
	GroupID       string
}
