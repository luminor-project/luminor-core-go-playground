package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

func applyAll(s *domain.Subject, events []domain.DomainEvent) {
	for _, e := range events {
		s.Apply(e.EventType, e.Payload)
	}
}

func TestRegisterSubject_Success(t *testing.T) {
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

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventSubjectRegistered {
		t.Errorf("expected event type %s, got %s", domain.EventSubjectRegistered, events[0].EventType)
	}

	payload := events[0].Payload.(domain.SubjectRegistered)
	if payload.SubjectID != "subject-1" {
		t.Errorf("expected subject ID 'subject-1', got %q", payload.SubjectID)
	}
	if payload.SubjectKind != domain.SubjectKindDwelling {
		t.Errorf("expected kind 'dwelling', got %q", payload.SubjectKind)
	}
	if payload.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", payload.Name)
	}
	if payload.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", payload.Detail)
	}
	if payload.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", payload.OrgID)
	}
	if payload.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", payload.CreatedByAccountID)
	}
	if payload.RegisteredAt != testClock.Now() {
		t.Errorf("expected registered at %v, got %v", testClock.Now(), payload.RegisteredAt)
	}
}

func TestRegisterSubject_InvalidKind(t *testing.T) {
	t.Parallel()
	s := domain.NewSubject(testClock)

	_, err := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:   "subject-1",
		SubjectKind: domain.SubjectKind("invalid"),
		Name:        "Name",
		Detail:      "detail",
		OrgID:       "org-1",
	})
	if !errors.Is(err, domain.ErrInvalidSubjectKind) {
		t.Errorf("expected ErrInvalidSubjectKind, got %v", err)
	}
}

func TestRegisterSubject_EmptyName(t *testing.T) {
	t.Parallel()
	s := domain.NewSubject(testClock)

	_, err := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:   "subject-1",
		SubjectKind: domain.SubjectKindDwelling,
		Name:        "   ",
		Detail:      "detail",
		OrgID:       "org-1",
	})
	if !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestRegisterSubject_AlreadyRegistered(t *testing.T) {
	t.Parallel()
	s := domain.NewSubject(testClock)

	events, _ := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:   "subject-1",
		SubjectKind: domain.SubjectKindDwelling,
		Name:        "Flussufer Apartments",
		Detail:      "Unit 12A",
		OrgID:       "org-1",
	})
	applyAll(s, events)

	_, err := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:   "subject-2",
		SubjectKind: domain.SubjectKindDwelling,
		Name:        "Another Property",
		OrgID:       "org-1",
	})
	if !errors.Is(err, domain.ErrAlreadyRegistered) {
		t.Errorf("expected ErrAlreadyRegistered, got %v", err)
	}
}

func TestRegisterSubject_TrimsNameAndDetail(t *testing.T) {
	t.Parallel()
	s := domain.NewSubject(testClock)

	events, err := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:   "subject-1",
		SubjectKind: domain.SubjectKindDwelling,
		Name:        "  Flussufer  ",
		Detail:      "  Unit 12A  ",
		OrgID:       "org-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := events[0].Payload.(domain.SubjectRegistered)
	if payload.Name != "Flussufer" {
		t.Errorf("expected trimmed name 'Flussufer', got %q", payload.Name)
	}
	if payload.Detail != "Unit 12A" {
		t.Errorf("expected trimmed detail 'Unit 12A', got %q", payload.Detail)
	}
}

func TestApply_SubjectRegistered(t *testing.T) {
	t.Parallel()
	s := domain.NewSubject(testClock)

	events, _ := s.RegisterSubject(domain.RegisterSubjectCmd{
		SubjectID:          "subject-1",
		SubjectKind:        domain.SubjectKindDwelling,
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	applyAll(s, events)

	if s.ID != "subject-1" {
		t.Errorf("expected ID 'subject-1', got %q", s.ID)
	}
	if s.SubjectKind != domain.SubjectKindDwelling {
		t.Errorf("expected kind 'dwelling', got %q", s.SubjectKind)
	}
	if s.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", s.Name)
	}
	if s.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", s.Detail)
	}
	if s.OwningOrganizationID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", s.OwningOrganizationID)
	}
	if s.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", s.CreatedByAccountID)
	}
	if s.CreatedAt != testClock.Now() {
		t.Errorf("expected created at %v, got %v", testClock.Now(), s.CreatedAt)
	}
	if !s.Registered {
		t.Error("expected Registered=true")
	}
	if s.Version != 1 {
		t.Errorf("expected Version 1, got %d", s.Version)
	}
}

func TestValidSubjectKinds(t *testing.T) {
	t.Parallel()
	kinds := domain.ValidSubjectKinds()
	if len(kinds) != 1 {
		t.Errorf("expected 1 valid kind, got %d", len(kinds))
	}
	if kinds[0] != domain.SubjectKindDwelling {
		t.Errorf("expected 'dwelling', got %q", kinds[0])
	}
}

func TestIsValidSubjectKind(t *testing.T) {
	t.Parallel()
	if !domain.IsValidSubjectKind(domain.SubjectKindDwelling) {
		t.Error("expected 'dwelling' to be valid")
	}
	if domain.IsValidSubjectKind(domain.SubjectKind("bogus")) {
		t.Error("expected 'bogus' to be invalid")
	}
}
