package facade

import "fmt"

// ErrPartyNotFound is returned when a party ID is not recognized.
var ErrPartyNotFound = fmt.Errorf("party not found")

// ActorKind represents the type of actor (human or AI assistant).
type ActorKind string

const (
	ActorKindHuman     ActorKind = "human"
	ActorKindAssistant ActorKind = "assistant"
)

// PartyKind is the business classification of a party.
type PartyKind string

const (
	PartyKindTenant          PartyKind = "tenant"
	PartyKindPropertyManager PartyKind = "property_manager"
	PartyKindAssistant       PartyKind = "assistant"
)

// PartyInfoDTO holds party data for cross-vertical communication.
type PartyInfoDTO struct {
	ID        string
	ActorKind ActorKind
	PartyKind PartyKind
	Name      string
}

// CreatePartyDTO holds data for creating a new party.
type CreatePartyDTO struct {
	Name               string
	ActorKind          ActorKind
	PartyKind          PartyKind
	OwningOrgID        string
	CreatedByAccountID string
}
