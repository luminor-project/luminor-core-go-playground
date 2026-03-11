package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPartyNotFound    = errors.New("party not found")
	ErrInvalidPartyKind = errors.New("invalid party kind")
	ErrEmptyName        = errors.New("party name must not be empty")
)

// ActorKind distinguishes human actors from virtual assistants (PIL-3).
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

// ValidPartyKinds returns all recognized party kinds.
func ValidPartyKinds() []PartyKind {
	return []PartyKind{PartyKindTenant, PartyKindPropertyManager, PartyKindAssistant}
}

// IsValidPartyKind checks whether the given kind is recognized.
func IsValidPartyKind(k PartyKind) bool {
	for _, v := range ValidPartyKinds() {
		if v == k {
			return true
		}
	}
	return false
}

// Party is a business/legal entity participating in workflows.
type Party struct {
	ID                   string
	ActorKind            ActorKind
	PartyKind            PartyKind
	Name                 string
	OwningOrganizationID string
	CreatedByAccountID   string
	CreatedAt            time.Time
}

// NewParty creates a new party with a generated UUID and validated fields.
func NewParty(name string, actorKind ActorKind, partyKind PartyKind, orgID, createdByAccountID string, now time.Time) (Party, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Party{}, ErrEmptyName
	}
	if !IsValidPartyKind(partyKind) {
		return Party{}, ErrInvalidPartyKind
	}

	return Party{
		ID:                   uuid.New().String(),
		ActorKind:            actorKind,
		PartyKind:            partyKind,
		Name:                 trimmed,
		OwningOrganizationID: orgID,
		CreatedByAccountID:   createdByAccountID,
		CreatedAt:            now,
	}, nil
}
