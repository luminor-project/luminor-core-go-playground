package facade

import "time"

// PartyRegisteredEvent is published when a new party is registered.
type PartyRegisteredEvent struct {
	PartyID            string
	ActorKind          ActorKind
	PartyKind          PartyKind
	Name               string
	OrgID              string
	CreatedByAccountID string
	RegisteredAt       time.Time
}
