package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrPartyNotFound     = errors.New("party not found")
	ErrInvalidPartyKind  = errors.New("invalid party kind")
	ErrEmptyName         = errors.New("party name must not be empty")
	ErrAlreadyRegistered = errors.New("party already registered")
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
	PartyKindPropertyOwner   PartyKind = "property_owner"
)

// ValidPartyKinds returns all recognized party kinds.
func ValidPartyKinds() []PartyKind {
	return []PartyKind{PartyKindTenant, PartyKindPropertyManager, PartyKindAssistant, PartyKindPropertyOwner}
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

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Party is the event-sourced aggregate for business/legal entities.
type Party struct {
	ID                   string
	ActorKind            ActorKind
	PartyKind            PartyKind
	Name                 string
	OwningOrganizationID string
	CreatedByAccountID   string
	CreatedAt            time.Time
	Registered           bool
	Version              int
	clock                Clock
}

// NewParty creates a Party aggregate with the given clock.
func NewParty(clock Clock) *Party {
	return &Party{clock: clock}
}

// Apply reconstitutes state from a single event payload.
func (p *Party) Apply(eventType string, payload any) {
	switch eventType {
	case EventPartyRegistered:
		e := payload.(PartyRegistered)
		p.ID = e.PartyID
		p.ActorKind = e.ActorKind
		p.PartyKind = e.PartyKind
		p.Name = e.Name
		p.OwningOrganizationID = e.OrgID
		p.CreatedByAccountID = e.CreatedByAccountID
		p.CreatedAt = e.RegisteredAt
		p.Registered = true
	default:
		panic("party.Apply: unknown event type: " + eventType)
	}
	p.Version++
}

// RegisterPartyCmd holds the data needed to register a new party.
type RegisterPartyCmd struct {
	PartyID            string
	Name               string
	ActorKind          ActorKind
	PartyKind          PartyKind
	OrgID              string
	CreatedByAccountID string
}

// RegisterParty registers a new party entity.
func (p *Party) RegisterParty(cmd RegisterPartyCmd) ([]DomainEvent, error) {
	if p.Registered {
		return nil, ErrAlreadyRegistered
	}
	if !IsValidPartyKind(cmd.PartyKind) {
		return nil, ErrInvalidPartyKind
	}
	trimmed := strings.TrimSpace(cmd.Name)
	if trimmed == "" {
		return nil, ErrEmptyName
	}

	return []DomainEvent{
		{EventType: EventPartyRegistered, Payload: PartyRegistered{
			PartyID:            cmd.PartyID,
			ActorKind:          cmd.ActorKind,
			PartyKind:          cmd.PartyKind,
			Name:               trimmed,
			OrgID:              cmd.OrgID,
			CreatedByAccountID: cmd.CreatedByAccountID,
			RegisteredAt:       p.clock.Now(),
		}},
	}, nil
}
