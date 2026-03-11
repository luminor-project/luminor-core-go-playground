package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_inquiry/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

type fakeInquiryFacade struct {
	submitCalls int
}

func (f *fakeInquiryFacade) SubmitInquiry(_ context.Context, _ facade.SubmitInquiryDTO) (string, error) {
	f.submitCalls++
	return "workitem-1", nil
}

type fakeRentalLister struct {
	rentals []rentalfacade.RentalInfoDTO
}

func (f *fakeRentalLister) ListRentalsByTenant(_ context.Context, _ string) ([]rentalfacade.RentalInfoDTO, error) {
	return f.rentals, nil
}

type fakeAccountInfo struct {
	activeOrgID string
}

func (f *fakeAccountInfo) GetActiveOrgID(_ context.Context, _ string) (string, error) {
	return f.activeOrgID, nil
}

func tenantUser() auth.User {
	return auth.User{
		ID:              "user-1",
		Email:           "anna@example.com",
		ActivePartyID:   "tenant-1",
		ActivePartyKind: "tenant",
	}
}

func TestHandleSubmitInquiry_RedirectsToConfirmation(t *testing.T) {
	t.Parallel()

	inquiryFac := &fakeInquiryFacade{}
	h := NewHandler(inquiryFac, nil, &fakeAccountInfo{activeOrgID: "org-1"})

	form := url.Values{}
	form.Set("body", "My heating is broken")
	req := httptest.NewRequest(http.MethodPost, "/inquiry", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(auth.WithUser(req.Context(), tenantUser()))
	rr := httptest.NewRecorder()

	h.HandleSubmitInquiry(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rr.Code)
	}
	if inquiryFac.submitCalls != 1 {
		t.Fatalf("expected 1 call, got %d", inquiryFac.submitCalls)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "/inquiry/submitted") {
		t.Errorf("expected redirect to /inquiry/submitted, got %q", loc)
	}
}
