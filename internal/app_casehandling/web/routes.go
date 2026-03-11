package web

import (
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
)

// RegisterRoutes registers the case handling HTTP routes.
// All routes are guarded by RequireAuth + RequirePartyKind("property_manager").
func RegisterRoutes(mux *http.ServeMux, dashboard dashboardReader, workitems workitemCommands, notes notesReader) {
	h := NewHandler(dashboard, workitems, notes)

	guard := func(next http.HandlerFunc) http.Handler {
		return auth.RequireAuth(auth.RequirePartyKind("property_manager")(http.HandlerFunc(next)))
	}

	mux.Handle("GET /cases", guard(h.ShowCaseWorkbench))
	mux.Handle("GET /cases/{id}", guard(h.ShowCaseWorkbench))
	mux.Handle("GET /cases/{id}/partial", guard(h.ShowCaseDetailPartial))
	mux.Handle("POST /cases/{id}/confirm", guard(h.HandleConfirm))

	// Note routes
	mux.Handle("GET /cases/{id}/entries/{entryIndex}/notes", guard(h.ShowNotesPartial))
	mux.Handle("POST /cases/{id}/entries/{entryIndex}/notes", guard(h.HandleAddNote))
	mux.Handle("PUT /cases/{id}/entries/{entryIndex}/notes/{noteId}", guard(h.HandleEditNote))
	mux.Handle("DELETE /cases/{id}/entries/{entryIndex}/notes/{noteId}", guard(h.HandleDeleteNote))
}
