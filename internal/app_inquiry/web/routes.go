package web

import (
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

// RegisterRoutes registers the inquiry HTTP routes.
// All routes are guarded by RequireAuth + RequirePartyKind("tenant").
func RegisterRoutes(
	mux *http.ServeMux,
	inquiry inquiryUseCases,
	rentals rentalLister,
	accounts activeOrgProvider,
) {
	h := NewHandler(inquiry, rentals, accounts)

	guard := func(next http.HandlerFunc) http.Handler {
		return auth.RequireAuth(auth.RequirePartyKind("tenant")(http.HandlerFunc(next)))
	}

	mux.Handle("GET /inquiry", guard(h.ShowInquiryForm))
	mux.Handle("POST /inquiry", guard(h.HandleSubmitInquiry))
	mux.Handle("GET /inquiry/submitted", guard(h.ShowSubmitted))
}
