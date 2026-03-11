package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
)

func TestDeserializeEvent_SubjectRegistered(t *testing.T) {
	t.Parallel()

	s := domain.NewSubject(testClock)
	events, err := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:          "subject-1",
		SubjectKind:        domain.SubjectKindDwelling,
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := json.Marshal(events[0].Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := domain.DeserializeEvent(domain.EventSubjectRegistered, raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	e, ok := got.(domain.SubjectRegistered)
	if !ok {
		t.Fatalf("expected SubjectRegistered, got %T", got)
	}
	if e.SubjectID != "subject-1" {
		t.Errorf("expected subject ID 'subject-1', got %q", e.SubjectID)
	}
	if e.SubjectKind != domain.SubjectKindDwelling {
		t.Errorf("expected kind 'dwelling', got %q", e.SubjectKind)
	}
	if e.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", e.Name)
	}
	if e.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", e.Detail)
	}
	if e.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", e.OrgID)
	}
	if e.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", e.CreatedByAccountID)
	}
	if e.RegisteredAt != testClock.Now() {
		t.Errorf("expected registered at %v, got %v", testClock.Now(), e.RegisteredAt)
	}
}

func TestDeserializeEvent_UnknownType(t *testing.T) {
	t.Parallel()
	_, err := domain.DeserializeEvent("subject.Unknown.v1", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestDeserializeEvent_MalformedJSON(t *testing.T) {
	t.Parallel()
	_, err := domain.DeserializeEvent(domain.EventSubjectRegistered, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDeserializeEvent_Roundtrip_ApplyConsistency(t *testing.T) {
	t.Parallel()

	s1 := domain.NewSubject(testClock)
	events, _ := s1.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:          "subject-1",
		SubjectKind:        domain.SubjectKindDwelling,
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	applyAll(s1, events)

	raw, _ := json.Marshal(events[0].Payload)
	deserialized, err := domain.DeserializeEvent(events[0].EventType, raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	s2 := domain.NewSubject(testClock)
	s2.Apply(events[0].EventType, deserialized)

	if s1.ID != s2.ID {
		t.Errorf("ID mismatch: %q vs %q", s1.ID, s2.ID)
	}
	if s1.SubjectKind != s2.SubjectKind {
		t.Errorf("SubjectKind mismatch: %q vs %q", s1.SubjectKind, s2.SubjectKind)
	}
	if s1.Name != s2.Name {
		t.Errorf("Name mismatch: %q vs %q", s1.Name, s2.Name)
	}
	if s1.Detail != s2.Detail {
		t.Errorf("Detail mismatch: %q vs %q", s1.Detail, s2.Detail)
	}
	if s1.OwningOrganizationID != s2.OwningOrganizationID {
		t.Errorf("OwningOrganizationID mismatch: %q vs %q", s1.OwningOrganizationID, s2.OwningOrganizationID)
	}
	if s1.CreatedAt != s2.CreatedAt {
		t.Errorf("CreatedAt mismatch: %v vs %v", s1.CreatedAt, s2.CreatedAt)
	}
	if s1.Version != s2.Version {
		t.Errorf("Version mismatch: %d vs %d", s1.Version, s2.Version)
	}
}
