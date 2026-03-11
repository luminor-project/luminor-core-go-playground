package subscriber_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/party/subscriber"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type upsertCall struct {
	ID                 string
	ActorKind          string
	PartyKind          string
	Name               string
	OrgID              string
	CreatedByAccountID string
	CreatedAt          time.Time
}

type fakeWriter struct {
	calls []upsertCall
	err   error
}

func (w *fakeWriter) UpsertProjection(_ context.Context, id, actorKind, partyKind, name, orgID, createdByAccountID string, createdAt time.Time) error {
	w.calls = append(w.calls, upsertCall{
		ID:                 id,
		ActorKind:          actorKind,
		PartyKind:          partyKind,
		Name:               name,
		OrgID:              orgID,
		CreatedByAccountID: createdByAccountID,
		CreatedAt:          createdAt,
	})
	return w.err
}

func TestProjection_PartyRegistered(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	writer := &fakeWriter{}
	subscriber.RegisterProjectionSubscribers(bus, writer)

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	err := eventbus.Publish(context.Background(), bus, partyfacade.PartyRegisteredEvent{
		PartyID:            "party-1",
		ActorKind:          partyfacade.ActorKindHuman,
		PartyKind:          partyfacade.PartyKindTenant,
		Name:               "Anna Schmidt",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
		RegisteredAt:       now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.calls) != 1 {
		t.Fatalf("expected 1 upsert call, got %d", len(writer.calls))
	}

	c := writer.calls[0]
	if c.ID != "party-1" {
		t.Errorf("expected ID 'party-1', got %q", c.ID)
	}
	if c.ActorKind != "human" {
		t.Errorf("expected actor kind 'human', got %q", c.ActorKind)
	}
	if c.PartyKind != "tenant" {
		t.Errorf("expected party kind 'tenant', got %q", c.PartyKind)
	}
	if c.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", c.Name)
	}
	if c.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", c.OrgID)
	}
	if c.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", c.CreatedByAccountID)
	}
	if c.CreatedAt != now {
		t.Errorf("expected created at %v, got %v", now, c.CreatedAt)
	}
}

func TestProjection_PartyRegistered_WriterError(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	writer := &fakeWriter{err: fmt.Errorf("db is down")}
	subscriber.RegisterProjectionSubscribers(bus, writer)

	err := eventbus.Publish(context.Background(), bus, partyfacade.PartyRegisteredEvent{
		PartyID:      "party-1",
		ActorKind:    partyfacade.ActorKindHuman,
		PartyKind:    partyfacade.PartyKindTenant,
		Name:         "Anna Schmidt",
		OrgID:        "org-1",
		RegisteredAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error when writer fails")
	}
}
