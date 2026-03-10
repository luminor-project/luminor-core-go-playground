package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
)

type dashboardReader interface {
	FindAll(ctx context.Context) ([]infra.CaseDashboardRow, error)
	FindByID(ctx context.Context, id string) (infra.CaseDashboardRow, error)
}

type caseUseCases interface {
	ConfirmAndSend(ctx context.Context, workItemID, operatorPartyID, body string) error
}

// Handler handles HTTP requests for the case handling UI.
type Handler struct {
	dashboard dashboardReader
	cases     caseUseCases
}

// NewHandler creates a new case handling HTTP handler.
func NewHandler(dashboard dashboardReader, cases caseUseCases) *Handler {
	return &Handler{
		dashboard: dashboard,
		cases:     cases,
	}
}

// ShowCaseList renders the case list page.
func (h *Handler) ShowCaseList(w http.ResponseWriter, r *http.Request) {
	cases, err := h.dashboard.FindAll(r.Context())
	if err != nil {
		slog.Error("failed to load cases", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	render.Page(w, r, templates.CaseList(cases))
}

// ShowCaseDetail renders the case detail page with timeline.
func (h *Handler) ShowCaseDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing case ID", http.StatusBadRequest)
		return
	}

	c, err := h.dashboard.FindByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to load case", "id", id, "error", err)
		http.Error(w, "case not found", http.StatusNotFound)
		return
	}

	render.Page(w, r, templates.CaseDetail(c))
}

// HandleConfirm processes the confirm-and-send action.
func (h *Handler) HandleConfirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing case ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	body := r.FormValue("body")

	// Use a hardcoded operator for V1 demo
	operatorPartyID := "party-sarah"

	if err := h.cases.ConfirmAndSend(r.Context(), id, operatorPartyID, body); err != nil {
		slog.Error("confirm and send failed", "id", id, "error", err)
		http.Error(w, "failed to confirm", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/cases/"+id, http.StatusSeeOther)
}
