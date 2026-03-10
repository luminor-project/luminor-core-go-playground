package web

import (
	"net/http"
)

// RegisterRoutes registers the case handling HTTP routes.
func RegisterRoutes(mux *http.ServeMux, dashboard dashboardReader, cases caseUseCases, notes notesReader) {
	h := NewHandler(dashboard, cases, notes)

	mux.HandleFunc("GET /cases", h.ShowCaseWorkbench)
	mux.HandleFunc("GET /cases/{id}", h.ShowCaseWorkbench)
	mux.HandleFunc("GET /cases/{id}/partial", h.ShowCaseDetailPartial)
	mux.HandleFunc("POST /cases/{id}/confirm", h.HandleConfirm)

	// Note routes
	mux.HandleFunc("GET /cases/{id}/entries/{entryIndex}/notes", h.ShowNotesPartial)
	mux.HandleFunc("POST /cases/{id}/entries/{entryIndex}/notes", h.HandleAddNote)
	mux.HandleFunc("PUT /cases/{id}/entries/{entryIndex}/notes/{noteId}", h.HandleEditNote)
	mux.HandleFunc("DELETE /cases/{id}/entries/{entryIndex}/notes/{noteId}", h.HandleDeleteNote)
}
