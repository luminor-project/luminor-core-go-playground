package facade

// ActiveOrgChangedEvent is dispatched when a user's active organization changes.
type ActiveOrgChangedEvent struct {
	OrganizationID string
	AffectedUserID string
}

// AccountJoinedOrgEvent is dispatched when an account joins an organization
// (either through registration/default org creation, or through invitation acceptance).
type AccountJoinedOrgEvent struct {
	AccountID string
	OrgID     string
	Email     string
}
