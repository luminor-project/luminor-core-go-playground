package web

import (
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/content/config"
)

// RegisterRoutes registers content-related routes on the mux.
func RegisterRoutes(mux *http.ServeMux, enableLivingStyleguide bool, greetings *config.GreetingsConfiguration) {
	h := NewHandler(greetings)

	mux.HandleFunc("GET /{$}", h.ShowHomepage)
	mux.HandleFunc("GET /about", h.ShowAbout)
	if enableLivingStyleguide {
		mux.HandleFunc("GET /living-styleguide", h.ShowLivingStyleguide)
		mux.HandleFunc("GET /living-styleguide/workbench", h.ShowStyleguideWorkbench)
	}
}
