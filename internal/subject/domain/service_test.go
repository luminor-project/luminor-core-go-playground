package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

type mockRepository struct {
	subjects map[string]domain.Subject
}

func newMockRepo() *mockRepository {
	return &mockRepository{subjects: make(map[string]domain.Subject)}
}

func (m *mockRepository) Create(_ context.Context, subject domain.Subject) error {
	m.subjects[subject.ID] = subject
	return nil
}

func (m *mockRepository) FindByID(_ context.Context, id string) (domain.Subject, error) {
	s, ok := m.subjects[id]
	if !ok {
		return domain.Subject{}, domain.ErrSubjectNotFound
	}
	return s, nil
}

func (m *mockRepository) FindByIDs(_ context.Context, ids []string) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, id := range ids {
		if s, ok := m.subjects[id]; ok {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockRepository) FindByOrganizationID(_ context.Context, orgID string) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, s := range m.subjects {
		if s.OwningOrganizationID == orgID {
			result = append(result, s)
		}
	}
	return result, nil
}

func TestCreateSubject_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewSubjectService(repo, testClock)

	subject, err := svc.CreateSubject(context.Background(), "Flussufer Apartments", "Unit 12A", "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subject.ID == "" {
		t.Error("expected non-empty ID")
	}
	if subject.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", subject.Name)
	}
	if subject.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", subject.Detail)
	}
	if subject.OwningOrganizationID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", subject.OwningOrganizationID)
	}
	if subject.CreatedAt != testClock.Now() {
		t.Errorf("expected created at %v, got %v", testClock.Now(), subject.CreatedAt)
	}
}

func TestCreateSubject_EmptyName(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewSubjectService(repo, testClock)

	_, err := svc.CreateSubject(context.Background(), "  ", "detail", "org-1", "account-1")
	if !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestCreateSubject_TrimsName(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewSubjectService(repo, testClock)

	subject, err := svc.CreateSubject(context.Background(), "  Flussufer  ", "  Unit 12A  ", "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subject.Name != "Flussufer" {
		t.Errorf("expected trimmed name, got %q", subject.Name)
	}
	if subject.Detail != "Unit 12A" {
		t.Errorf("expected trimmed detail, got %q", subject.Detail)
	}
}

func TestCreateSubject_PersistsToRepo(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewSubjectService(repo, testClock)

	subject, err := svc.CreateSubject(context.Background(), "Flussufer", "Unit 12A", "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, err := svc.FindByID(context.Background(), subject.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Flussufer" {
		t.Errorf("expected name 'Flussufer', got %q", found.Name)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewSubjectService(repo, testClock)

	_, err := svc.FindByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrSubjectNotFound) {
		t.Errorf("expected ErrSubjectNotFound, got %v", err)
	}
}

func TestFindByOrganizationID(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewSubjectService(repo, testClock)

	_, _ = svc.CreateSubject(context.Background(), "Property A", "Unit 1", "org-1", "account-1")
	_, _ = svc.CreateSubject(context.Background(), "Property B", "Unit 2", "org-1", "account-1")
	_, _ = svc.CreateSubject(context.Background(), "Other", "Unit 3", "org-2", "account-2")

	subjects, err := svc.FindByOrganizationID(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subjects) != 2 {
		t.Errorf("expected 2 subjects for org-1, got %d", len(subjects))
	}
}
