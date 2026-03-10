package facade_test

import (
	"context"
	"errors"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
)

func TestDemoPartyFacade_KnownParties(t *testing.T) {
	t.Parallel()
	f := facade.NewDemoPartyFacade()
	ctx := context.Background()

	tests := []struct {
		id        string
		name      string
		actorKind string
	}{
		{"party-anna-schmidt", "Anna Schmidt", "human"},
		{"party-sarah", "Sarah", "human"},
		{"party-ki-assistent", "KI-Assistent", "assistant"},
	}

	for _, tt := range tests {
		info, err := f.GetPartyInfo(ctx, tt.id)
		if err != nil {
			t.Fatalf("GetPartyInfo(%q): unexpected error: %v", tt.id, err)
		}
		if info.Name != tt.name {
			t.Errorf("expected name %q, got %q", tt.name, info.Name)
		}
		if info.ActorKind != tt.actorKind {
			t.Errorf("expected actorKind %q, got %q", tt.actorKind, info.ActorKind)
		}
	}
}

func TestDemoPartyFacade_UnknownParty(t *testing.T) {
	t.Parallel()
	f := facade.NewDemoPartyFacade()
	ctx := context.Background()

	_, err := f.GetPartyInfo(ctx, "party-unknown")
	if !errors.Is(err, facade.ErrPartyNotFound) {
		t.Fatalf("expected ErrPartyNotFound, got: %v", err)
	}
}
