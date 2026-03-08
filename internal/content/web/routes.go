package web

import "net/http"

// RegisterRoutes registers content-related routes on the mux.
func RegisterRoutes(mux *http.ServeMux, enableLivingStyleguide bool) {
	h := NewHandler()

	mux.HandleFunc("GET /{$}", h.ShowHomepage)
	mux.HandleFunc("GET /about", h.ShowAbout)
	if enableLivingStyleguide {
		mux.HandleFunc("GET /living-styleguide", h.ShowLivingStyleguide)
	}
}
