package web

import "net/http"

// RegisterRoutes registers content-related routes on the mux.
func RegisterRoutes(mux *http.ServeMux, enableLivingStyleguide bool) {
	h := NewHandler()

	mux.HandleFunc("GET /{$}", h.ShowHomepage)
	mux.HandleFunc("GET /about", h.ShowAbout)
	mux.HandleFunc("GET /team", h.ShowTeam)
	if enableLivingStyleguide {
		mux.HandleFunc("GET /living-styleguide", h.ShowLivingStyleguide)
		mux.HandleFunc("GET /living-styleguide/workbench", h.ShowStyleguideWorkbench)
	}
}
