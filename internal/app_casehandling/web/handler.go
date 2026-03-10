package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
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

// ShowCaseWorkbench renders the two-pane workbench (list + detail).
// Handles both GET /cases and GET /cases/{id}.
func (h *Handler) ShowCaseWorkbench(w http.ResponseWriter, r *http.Request) {
	cases, err := h.dashboard.FindAll(r.Context())
	if err != nil {
		slog.Error("failed to load cases", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id := r.PathValue("id")

	var selected *infra.CaseDashboardRow
	if id != "" {
		c, err := h.dashboard.FindByID(r.Context(), id)
		if err != nil {
			slog.Error("failed to load case", "id", id, "error", err)
			http.Error(w, "case not found", http.StatusNotFound)
			return
		}
		selected = &c
	} else if len(cases) > 0 {
		selected = &cases[0]
	}

	render.Page(w, r, templates.CaseWorkbench(cases, selected))
}

// ShowCaseDetailPartial renders just the detail pane (htmx partial).
func (h *Handler) ShowCaseDetailPartial(w http.ResponseWriter, r *http.Request) {
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

	render.Page(w, r, templates.CaseDetailPane(c))
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

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), fmt.Sprintf("/cases/%s", id)), http.StatusSeeOther)
}
