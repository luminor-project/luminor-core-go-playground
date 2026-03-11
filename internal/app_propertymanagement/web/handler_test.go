package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_propertymanagement/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

type fakePMFacade struct {
	createPropertyCalls int
	createTenantCalls   int
	assignTenantCalls   int
	inviteTenantCalls   int
}

func (f *fakePMFacade) CreateProperty(_ context.Context, _ facade.CreatePropertyDTO) (string, error) {
	f.createPropertyCalls++
	return "subject-1", nil
}

func (f *fakePMFacade) CreateTenant(_ context.Context, _ facade.CreateTenantDTO) (string, error) {
	f.createTenantCalls++
	return "party-1", nil
}

func (f *fakePMFacade) AssignTenantToProperty(_ context.Context, _ facade.AssignTenantDTO) (string, error) {
	f.assignTenantCalls++
	return "rental-1", nil
}

func (f *fakePMFacade) InviteTenant(_ context.Context, _ facade.InviteTenantDTO) error {
	f.inviteTenantCalls++
	return nil
}

type fakePartyLister struct {
	parties []partyfacade.PartyInfoDTO
}

func (f *fakePartyLister) ListPartiesByOrgAndKind(_ context.Context, _ string, _ partyfacade.PartyKind) ([]partyfacade.PartyInfoDTO, error) {
	return f.parties, nil
}

type fakeSubjectLister struct {
	subjects []subjectfacade.SubjectInfoDTO
}

func (f *fakeSubjectLister) ListSubjectsByOrgAndKind(_ context.Context, _ string, _ subjectfacade.SubjectKind) ([]subjectfacade.SubjectInfoDTO, error) {
	return f.subjects, nil
}

type fakeRentalLister struct {
	rentals []rentalfacade.RentalInfoDTO
}

func (f *fakeRentalLister) ListRentalsByOrg(_ context.Context, _ string) ([]rentalfacade.RentalInfoDTO, error) {
	return f.rentals, nil
}

type fakeAccountInfo struct {
	activeOrgID string
}

func (f *fakeAccountInfo) GetActiveOrgID(_ context.Context, _ string) (string, error) {
	return f.activeOrgID, nil
}

func newTestUser() auth.User {
	return auth.User{
		ID:              "user-1",
		Email:           "pm@example.com",
		ActivePartyID:   "pm-party-1",
		ActivePartyKind: "property_manager",
	}
}

func TestHandleCreateProperty_RedirectsOnSuccess(t *testing.T) {
	t.Parallel()

	pmFac := &fakePMFacade{}
	h := NewHandler(pmFac, nil, nil, nil, &fakeAccountInfo{activeOrgID: "org-1"})

	form := url.Values{}
	form.Set("name", "Flussufer Apartments")
	form.Set("detail", "Unit 12A")
	req := httptest.NewRequest(http.MethodPost, "/property-management/properties", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), newTestUser()))
	rr := httptest.NewRecorder()

	h.HandleCreateProperty(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if pmFac.createPropertyCalls != 1 {
		t.Fatalf("expected 1 call, got %d", pmFac.createPropertyCalls)
	}
}

func TestHandleCreateTenant_RedirectsOnSuccess(t *testing.T) {
	t.Parallel()

	pmFac := &fakePMFacade{}
	h := NewHandler(pmFac, nil, nil, nil, &fakeAccountInfo{activeOrgID: "org-1"})

	form := url.Values{}
	form.Set("name", "Anna Schmidt")
	req := httptest.NewRequest(http.MethodPost, "/property-management/tenants", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), newTestUser()))
	rr := httptest.NewRecorder()

	h.HandleCreateTenant(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if pmFac.createTenantCalls != 1 {
		t.Fatalf("expected 1 call, got %d", pmFac.createTenantCalls)
	}
}

func TestHandleAssignTenant_RedirectsOnSuccess(t *testing.T) {
	t.Parallel()

	pmFac := &fakePMFacade{}
	h := NewHandler(pmFac, nil, nil, nil, &fakeAccountInfo{activeOrgID: "org-1"})

	form := url.Values{}
	form.Set("subject_id", "subject-1")
	form.Set("tenant_party_id", "party-1")
	req := httptest.NewRequest(http.MethodPost, "/property-management/rentals", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), newTestUser()))
	rr := httptest.NewRecorder()

	h.HandleAssignTenant(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if pmFac.assignTenantCalls != 1 {
		t.Fatalf("expected 1 call, got %d", pmFac.assignTenantCalls)
	}
}

func TestHandleInviteTenant_RedirectsOnSuccess(t *testing.T) {
	t.Parallel()

	pmFac := &fakePMFacade{}
	h := NewHandler(pmFac, nil, nil, nil, &fakeAccountInfo{activeOrgID: "org-1"})

	form := url.Values{}
	form.Set("email", "anna@example.com")
	req := httptest.NewRequest(http.MethodPost, "/property-management/tenants/{tenantId}/invite", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("tenantId", "party-1")
	req = req.WithContext(auth.WithUser(req.Context(), newTestUser()))
	rr := httptest.NewRecorder()

	h.HandleInviteTenant(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if pmFac.inviteTenantCalls != 1 {
		t.Fatalf("expected 1 call, got %d", pmFac.inviteTenantCalls)
	}
}
