package web

import (
	"net/http"
)

// RegisterRoutes registers the case handling HTTP routes.
func RegisterRoutes(mux *http.ServeMux, dashboard dashboardReader, cases caseUseCases) {
	h := NewHandler(dashboard, cases)

	mux.HandleFunc("GET /cases", h.ShowCaseWorkbench)
	mux.HandleFunc("GET /cases/{id}", h.ShowCaseWorkbench)
	mux.HandleFunc("GET /cases/{id}/partial", h.ShowCaseDetailPartial)
	mux.HandleFunc("POST /cases/{id}/confirm", h.HandleConfirm)
}
