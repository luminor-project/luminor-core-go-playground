package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

type mockRepository struct {
	rentals map[string]domain.Rental
}

func newMockRepo() *mockRepository {
	return &mockRepository{rentals: make(map[string]domain.Rental)}
}

func (m *mockRepository) Create(_ context.Context, rental domain.Rental) error {
	m.rentals[rental.ID] = rental
	return nil
}

func (m *mockRepository) FindByID(_ context.Context, id string) (domain.Rental, error) {
	r, ok := m.rentals[id]
	if !ok {
		return domain.Rental{}, domain.ErrRentalNotFound
	}
	return r, nil
}

func (m *mockRepository) FindBySubjectID(_ context.Context, subjectID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, r := range m.rentals {
		if r.SubjectID == subjectID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRepository) FindByTenantPartyID(_ context.Context, tenantPartyID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, r := range m.rentals {
		if r.TenantPartyID == tenantPartyID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRepository) FindByOrgID(_ context.Context, orgID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, r := range m.rentals {
		if r.OrgID == orgID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRepository) ExistsBySubjectAndTenant(_ context.Context, subjectID, tenantPartyID string) (bool, error) {
	for _, r := range m.rentals {
		if r.SubjectID == subjectID && r.TenantPartyID == tenantPartyID {
			return true, nil
		}
	}
	return false, nil
}

func TestCreateRental_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewRentalService(repo, testClock)

	rental, err := svc.CreateRental(context.Background(), "subject-1", "party-1", "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rental.ID == "" {
		t.Error("expected non-empty ID")
	}
	if rental.SubjectID != "subject-1" {
		t.Errorf("expected subject 'subject-1', got %q", rental.SubjectID)
	}
	if rental.TenantPartyID != "party-1" {
		t.Errorf("expected tenant 'party-1', got %q", rental.TenantPartyID)
	}
	if rental.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", rental.OrgID)
	}
	if rental.CreatedAt != testClock.Now() {
		t.Errorf("expected created at %v, got %v", testClock.Now(), rental.CreatedAt)
	}
}

func TestCreateRental_DuplicateSubjectTenant(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewRentalService(repo, testClock)

	_, err := svc.CreateRental(context.Background(), "subject-1", "party-1", "org-1", "account-1")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.CreateRental(context.Background(), "subject-1", "party-1", "org-1", "account-1")
	if !errors.Is(err, domain.ErrDuplicateRental) {
		t.Errorf("expected ErrDuplicateRental, got %v", err)
	}
}

func TestFindByTenantPartyID(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewRentalService(repo, testClock)

	_, _ = svc.CreateRental(context.Background(), "subject-1", "party-1", "org-1", "account-1")
	_, _ = svc.CreateRental(context.Background(), "subject-2", "party-1", "org-1", "account-1")
	_, _ = svc.CreateRental(context.Background(), "subject-3", "party-2", "org-1", "account-1")

	rentals, err := svc.FindByTenantPartyID(context.Background(), "party-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 2 {
		t.Errorf("expected 2 rentals for party-1, got %d", len(rentals))
	}
}

func TestFindByOrgID(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewRentalService(repo, testClock)

	_, _ = svc.CreateRental(context.Background(), "s-1", "p-1", "org-1", "a-1")
	_, _ = svc.CreateRental(context.Background(), "s-2", "p-2", "org-2", "a-2")

	rentals, err := svc.FindByOrgID(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 1 {
		t.Errorf("expected 1 rental for org-1, got %d", len(rentals))
	}
}
