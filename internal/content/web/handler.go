package web

import (
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/content/config"
	"github.com/luminor-project/luminor-core-go-playground/internal/content/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
)

// GreetingsProvider provides random greetings for display.
type GreetingsProvider interface {
	GetGreeting() config.Greeting
}

// Handler handles content-related HTTP requests.
type Handler struct {
	greetings GreetingsProvider
}

// NewHandler creates a new content handler with the provided dependencies.
func NewHandler(greetings GreetingsProvider) *Handler {
	return &Handler{
		greetings: greetings,
	}
}

// ShowHomepage renders the homepage.
func (h *Handler) ShowHomepage(w http.ResponseWriter, r *http.Request) {
	greeting := h.greetings.GetGreeting()
	render.Page(w, r, templates.Homepage(greeting.Text))
}

// ShowAbout renders the about page.
func (h *Handler) ShowAbout(w http.ResponseWriter, r *http.Request) {
	render.Page(w, r, templates.About())
}

// ShowLivingStyleguide renders the living styleguide page.
func (h *Handler) ShowLivingStyleguide(w http.ResponseWriter, r *http.Request) {
	render.Page(w, r, templates.LivingStyleguide())
}

// ShowStyleguideWorkbench renders the workbench pattern reference page.
func (h *Handler) ShowStyleguideWorkbench(w http.ResponseWriter, r *http.Request) {
	render.Page(w, r, templates.StyleguideWorkbench())
}
