package web

import (
	"net/http"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/content/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
)

// Handler handles content-related HTTP requests.
type Handler struct{}

// NewHandler creates a new content handler.
func NewHandler() *Handler {
	return &Handler{}
}

// greetingForTime returns the appropriate greeting based on the time of day.
// Morning: 6-12, Afternoon: 12-18, Evening: 18-22, Night: 22-6.
func greetingForTime(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour >= 6 && hour < 12:
		return "homepage.greeting.morning"
	case hour >= 12 && hour < 18:
		return "homepage.greeting.afternoon"
	case hour >= 18 && hour < 22:
		return "homepage.greeting.evening"
	default:
		return "homepage.greeting.night"
	}
}

// ShowHomepage renders the homepage.
func (h *Handler) ShowHomepage(w http.ResponseWriter, r *http.Request) {
	greetingKey := greetingForTime(time.Now())
	render.Page(w, r, templates.Homepage(greetingKey))
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
