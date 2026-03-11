package domain

import "time"

// Event type constants for the party aggregate.
const (
	EventPartyRegistered = "party.PartyRegistered.v1"
)

// DomainEvent is what command methods return. Domain-local, zero platform imports.
type DomainEvent struct {
	EventType string
	Payload   any
}

// PartyRegistered is emitted when a new party is registered.
type PartyRegistered struct {
	PartyID            string    `json:"party_id"`
	ActorKind          ActorKind `json:"actor_kind"`
	PartyKind          PartyKind `json:"party_kind"`
	Name               string    `json:"name"`
	OrgID              string    `json:"org_id"`
	CreatedByAccountID string    `json:"created_by_account_id"`
	RegisteredAt       time.Time `json:"registered_at"`
}
