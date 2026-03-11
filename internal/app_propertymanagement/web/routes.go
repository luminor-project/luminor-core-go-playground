package web

import (
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

// RegisterRoutes registers the property management HTTP routes.
// All routes are guarded by RequireAuth + RequirePartyKind("property_manager").
func RegisterRoutes(
	mux *http.ServeMux,
	pm pmUseCases,
	parties partyLister,
	subjects subjectLister,
	rentals rentalLister,
	accounts activeOrgProvider,
) {
	h := NewHandler(pm, parties, subjects, rentals, accounts)

	guard := func(next http.HandlerFunc) http.Handler {
		return auth.RequireAuth(auth.RequirePartyKind("property_manager")(http.HandlerFunc(next)))
	}

	mux.Handle("GET /property-management", guard(h.ShowDashboard))
	mux.Handle("POST /property-management/properties", guard(h.HandleCreateProperty))
	mux.Handle("POST /property-management/tenants", guard(h.HandleCreateTenant))
	mux.Handle("POST /property-management/rentals", guard(h.HandleAssignTenant))
	mux.Handle("POST /property-management/tenants/{tenantId}/invite", guard(h.HandleInviteTenant))
}
