package facade

import (
	"context"
	"fmt"
)

// ErrPartyNotFound is returned when a party ID is not recognized.
var ErrPartyNotFound = fmt.Errorf("party not found")

// PartyInfoDTO holds party data for cross-vertical communication.
type PartyInfoDTO struct {
	ID        string
	ActorKind string // "human", "assistant"
	Name      string
}

// PartyFacade provides party lookup operations.
type PartyFacade interface {
	GetPartyInfo(ctx context.Context, partyID string) (PartyInfoDTO, error)
}
