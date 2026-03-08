package facade

// ActiveOrgChangedEvent is dispatched when a user's active organization changes.
type ActiveOrgChangedEvent struct {
	OrganizationID string
	AffectedUserID string
}
