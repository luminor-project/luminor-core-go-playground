package facade_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

type fakeService struct {
	rentals map[string]domain.Rental
}

func newFakeService() *fakeService {
	return &fakeService{rentals: make(map[string]domain.Rental)}
}

func (f *fakeService) CreateRental(_ context.Context, subjectID, tenantPartyID, orgID, createdByAccountID string) (domain.Rental, error) {
	for _, r := range f.rentals {
		if r.SubjectID == subjectID && r.TenantPartyID == tenantPartyID {
			return domain.Rental{}, domain.ErrDuplicateRental
		}
	}
	r := domain.Rental{
		ID:                 "generated-id",
		SubjectID:          subjectID,
		TenantPartyID:      tenantPartyID,
		OrgID:              orgID,
		CreatedByAccountID: createdByAccountID,
		CreatedAt:          time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}
	f.rentals[r.ID] = r
	return r, nil
}

func (f *fakeService) FindByTenantPartyID(_ context.Context, tenantPartyID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, r := range f.rentals {
		if r.TenantPartyID == tenantPartyID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (f *fakeService) FindBySubjectID(_ context.Context, subjectID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, r := range f.rentals {
		if r.SubjectID == subjectID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (f *fakeService) FindByOrgID(_ context.Context, orgID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, r := range f.rentals {
		if r.OrgID == orgID {
			result = append(result, r)
		}
	}
	return result, nil
}

func TestCreateRental_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	fac := facade.New(svc)

	id, err := fac.CreateRental(context.Background(), facade.CreateRentalDTO{
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateRental_Duplicate(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	fac := facade.New(svc)

	_, _ = fac.CreateRental(context.Background(), facade.CreateRentalDTO{
		SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1", CreatedByAccountID: "a-1",
	})

	_, err := fac.CreateRental(context.Background(), facade.CreateRentalDTO{
		SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1", CreatedByAccountID: "a-1",
	})
	if !errors.Is(err, facade.ErrDuplicateRental) {
		t.Errorf("expected ErrDuplicateRental, got %v", err)
	}
}

func TestListRentalsByOrg_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.rentals["r-1"] = domain.Rental{ID: "r-1", SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1"}
	fac := facade.New(svc)

	rentals, err := fac.ListRentalsByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 1 {
		t.Errorf("expected 1 rental, got %d", len(rentals))
	}
}

func TestListRentalsByTenant_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.rentals["r-1"] = domain.Rental{ID: "r-1", SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1"}
	fac := facade.New(svc)

	rentals, err := fac.ListRentalsByTenant(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 1 {
		t.Errorf("expected 1 rental, got %d", len(rentals))
	}
}
