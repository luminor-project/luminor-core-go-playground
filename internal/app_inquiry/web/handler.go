package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_inquiry/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

type inquiryUseCases interface {
	SubmitInquiry(ctx context.Context, dto facade.SubmitInquiryDTO) (string, error)
}

type rentalLister interface {
	ListRentalsByTenant(ctx context.Context, tenantPartyID string) ([]rentalfacade.RentalInfoDTO, error)
}

type activeOrgProvider interface {
	GetActiveOrgID(ctx context.Context, accountID string) (string, error)
}

// Handler handles inquiry HTTP requests.
type Handler struct {
	inquiry  inquiryUseCases
	rentals  rentalLister
	accounts activeOrgProvider
}

// NewHandler creates a new inquiry handler.
func NewHandler(inquiry inquiryUseCases, rentals rentalLister, accounts activeOrgProvider) *Handler {
	return &Handler{
		inquiry:  inquiry,
		rentals:  rentals,
		accounts: accounts,
	}
}

// ShowInquiryForm renders the inquiry form for tenants.
func (h *Handler) ShowInquiryForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, "<h1>Submit Inquiry</h1><form method=post><textarea name=body></textarea><button type=submit>Submit</button></form>")
}

// HandleSubmitInquiry handles POST /inquiry.
func (h *Handler) HandleSubmitInquiry(w http.ResponseWriter, r *http.Request) {
	user := auth.MustUserFromContext(r.Context())
	orgID, err := h.accounts.GetActiveOrgID(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to get active org", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.invalidForm"), http.StatusBadRequest)
		return
	}

	_, err = h.inquiry.SubmitInquiry(r.Context(), facade.SubmitInquiryDTO{
		TenantPartyID: user.ActivePartyID,
		OrgID:         orgID,
		Body:          r.FormValue("body"),
	})
	if err != nil {
		slog.Error("submit inquiry failed", "error", err)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), "/inquiry/submitted"), http.StatusSeeOther)
}

// ShowSubmitted renders the inquiry submission confirmation page.
func (h *Handler) ShowSubmitted(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, "<h1>Inquiry Submitted</h1><p>Your inquiry has been submitted. A property manager will review it shortly.</p>")
}
