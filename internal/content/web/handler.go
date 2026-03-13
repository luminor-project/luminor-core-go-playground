package web

import (
	"math/rand"
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

// greetingPools holds multiple greeting options for each time of day.
var greetingPools = map[string][]string{
	"morning": {
		"homepage.greeting.morning.riseAndShine",
		"homepage.greeting.morning.freshStart",
		"homepage.greeting.morning.helloSunshine",
		"homepage.greeting.morning.newDay",
		"homepage.greeting.morning.readyToGo",
	},
	"afternoon": {
		"homepage.greeting.afternoon.coffeeTime",
		"homepage.greeting.afternoon.halfwayThere",
		"homepage.greeting.afternoon.productive",
		"homepage.greeting.afternoon.lunchBreak",
		"homepage.greeting.afternoon.keepingGoing",
	},
	"evening": {
		"homepage.greeting.evening.windingDown",
		"homepage.greeting.evening.almostThere",
		"homepage.greeting.evening.greatWork",
		"homepage.greeting.evening.timeToRelax",
		"homepage.greeting.evening.wellDone",
	},
	"night": {
		"homepage.greeting.night.stillAwake",
		"homepage.greeting.night.nightOwl",
		"homepage.greeting.night.restWell",
		"homepage.greeting.night.sweetDreams",
		"homepage.greeting.night.seeYouTomorrow",
	},
}

// greetingForTime returns a random greeting based on the time of day.
// Morning: 6-12, Afternoon: 12-18, Evening: 18-22, Night: 22-6.
func greetingForTime(t time.Time) string {
	hour := t.Hour()
	var poolKey string
	switch {
	case hour >= 6 && hour < 12:
		poolKey = "morning"
	case hour >= 12 && hour < 18:
		poolKey = "afternoon"
	case hour >= 18 && hour < 22:
		poolKey = "evening"
	default:
		poolKey = "night"
	}

	pool := greetingPools[poolKey]
	return pool[rand.Intn(len(pool))]
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
