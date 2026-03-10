package facade

import "context"

var demoParties = map[string]PartyInfoDTO{
	"party-anna-schmidt": {
		ID:        "party-anna-schmidt",
		ActorKind: ActorKindHuman,
		Name:      "Anna Schmidt",
	},
	"party-sarah": {
		ID:        "party-sarah",
		ActorKind: ActorKindHuman,
		Name:      "Sarah",
	},
	"party-ki-assistent": {
		ID:        "party-ki-assistent",
		ActorKind: ActorKindAssistant,
		Name:      "KI-Assistent",
	},
}

type demoPartyFacade struct{}

// NewDemoPartyFacade creates a facade with hardcoded demo party data.
func NewDemoPartyFacade() *demoPartyFacade {
	return &demoPartyFacade{}
}

func (f *demoPartyFacade) GetPartyInfo(_ context.Context, partyID string) (PartyInfoDTO, error) {
	p, ok := demoParties[partyID]
	if !ok {
		return PartyInfoDTO{}, ErrPartyNotFound
	}
	return p, nil
}
