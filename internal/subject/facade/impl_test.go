package facade_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
)

type fakeService struct {
	subjects map[string]domain.Subject
}

func newFakeService() *fakeService {
	return &fakeService{subjects: make(map[string]domain.Subject)}
}

func (f *fakeService) CreateSubject(_ context.Context, name, detail, orgID, createdByAccountID string) (domain.Subject, error) {
	s := domain.Subject{
		ID:                   "generated-id",
		Name:                 name,
		Detail:               detail,
		OwningOrganizationID: orgID,
		CreatedByAccountID:   createdByAccountID,
		CreatedAt:            time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}
	f.subjects[s.ID] = s
	return s, nil
}

func (f *fakeService) FindByID(_ context.Context, id string) (domain.Subject, error) {
	s, ok := f.subjects[id]
	if !ok {
		return domain.Subject{}, domain.ErrSubjectNotFound
	}
	return s, nil
}

func (f *fakeService) FindByIDs(_ context.Context, ids []string) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, id := range ids {
		if s, ok := f.subjects[id]; ok {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeService) FindByOrganizationID(_ context.Context, orgID string) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, s := range f.subjects {
		if s.OwningOrganizationID == orgID {
			result = append(result, s)
		}
	}
	return result, nil
}

func TestGetSubjectInfo_MapsCorrectly(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.subjects["s-1"] = domain.Subject{
		ID:                   "s-1",
		Name:                 "Flussufer Apartments",
		Detail:               "Unit 12A",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(svc)

	dto, err := fac.GetSubjectInfo(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.ID != "s-1" {
		t.Errorf("expected ID 's-1', got %q", dto.ID)
	}
	if dto.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", dto.Name)
	}
	if dto.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", dto.Detail)
	}
}

func TestGetSubjectInfo_NotFound(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	fac := facade.New(svc)

	_, err := fac.GetSubjectInfo(context.Background(), "nonexistent")
	if !errors.Is(err, facade.ErrSubjectNotFound) {
		t.Errorf("expected ErrSubjectNotFound, got %v", err)
	}
}

func TestCreateSubject_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	fac := facade.New(svc)

	id, err := fac.CreateSubject(context.Background(), facade.CreateSubjectDTO{
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
		OwningOrgID:        "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestListSubjectsByOrg_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.subjects["s-1"] = domain.Subject{
		ID:                   "s-1",
		Name:                 "Property A",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(svc)

	subjects, err := fac.ListSubjectsByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subjects) != 1 {
		t.Errorf("expected 1 subject, got %d", len(subjects))
	}
}
