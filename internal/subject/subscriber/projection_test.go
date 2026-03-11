package subscriber_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/subscriber"
)

type upsertCall struct {
	ID                 string
	SubjectKind        string
	Name               string
	Detail             string
	OrgID              string
	CreatedByAccountID string
	CreatedAt          time.Time
}

type fakeWriter struct {
	calls []upsertCall
	err   error
}

func (w *fakeWriter) UpsertProjection(_ context.Context, id, subjectKind, name, detail, orgID, createdByAccountID string, createdAt time.Time) error {
	w.calls = append(w.calls, upsertCall{
		ID:                 id,
		SubjectKind:        subjectKind,
		Name:               name,
		Detail:             detail,
		OrgID:              orgID,
		CreatedByAccountID: createdByAccountID,
		CreatedAt:          createdAt,
	})
	return w.err
}

func TestProjection_SubjectRegistered(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	writer := &fakeWriter{}
	subscriber.RegisterProjectionSubscribers(bus, writer)

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	err := eventbus.Publish(context.Background(), bus, subjectfacade.SubjectRegisteredEvent{
		SubjectID:          "subject-1",
		SubjectKind:        subjectfacade.SubjectKindDwelling,
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
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
	if c.ID != "subject-1" {
		t.Errorf("expected ID 'subject-1', got %q", c.ID)
	}
	if c.SubjectKind != "dwelling" {
		t.Errorf("expected kind 'dwelling', got %q", c.SubjectKind)
	}
	if c.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", c.Name)
	}
	if c.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", c.Detail)
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

func TestProjection_SubjectRegistered_WriterError(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	writer := &fakeWriter{err: fmt.Errorf("db is down")}
	subscriber.RegisterProjectionSubscribers(bus, writer)

	err := eventbus.Publish(context.Background(), bus, subjectfacade.SubjectRegisteredEvent{
		SubjectID:    "subject-1",
		SubjectKind:  subjectfacade.SubjectKindDwelling,
		Name:         "Flussufer Apartments",
		Detail:       "Unit 12A",
		OrgID:        "org-1",
		RegisteredAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error when writer fails")
	}
}
