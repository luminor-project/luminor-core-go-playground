package web

import (
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/content/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/content/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
)

// Handler handles content-related HTTP requests.
type Handler struct {
	clock clock.Clock
}

// NewHandler creates a new content handler.
func NewHandler() *Handler {
	return &Handler{clock: clock.New()}
}

// ShowHomepage renders the homepage.
func (h *Handler) ShowHomepage(w http.ResponseWriter, r *http.Request) {
	now := h.clock.Now()
	tod := domain.DetermineTimeOfDay(now)
	render.Page(w, r, templates.Homepage(tod))
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
