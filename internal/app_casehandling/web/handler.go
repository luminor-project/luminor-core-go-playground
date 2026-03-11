package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/web/templates"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/auth"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/render"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

type dashboardReader interface {
	FindAll(ctx context.Context) ([]infra.CaseDashboardRow, error)
	FindByID(ctx context.Context, id string) (infra.CaseDashboardRow, error)
}

type workitemCommands interface {
	ConfirmOutboundMessage(ctx context.Context, workItemID string, dto workitemfacade.ConfirmOutboundMessageDTO) error
	AddNote(ctx context.Context, workItemID string, dto workitemfacade.AddNoteDTO) (string, error)
	EditNote(ctx context.Context, workItemID string, dto workitemfacade.EditNoteDTO) error
	DeleteNote(ctx context.Context, workItemID string, dto workitemfacade.DeleteNoteDTO) error
}

type notesReader interface {
	FindNotesByEntryIndex(ctx context.Context, workItemID string, entryIndex int) ([]infra.TimelineNote, error)
}

// Handler handles HTTP requests for the case handling UI.
type Handler struct {
	dashboard dashboardReader
	workitems workitemCommands
	notes     notesReader
}

// NewHandler creates a new case handling HTTP handler.
func NewHandler(dashboard dashboardReader, workitems workitemCommands, notes notesReader) *Handler {
	return &Handler{
		dashboard: dashboard,
		workitems: workitems,
		notes:     notes,
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

	user := auth.MustUserFromContext(r.Context())
	operatorPartyID := user.ActivePartyID

	if err := h.workitems.ConfirmOutboundMessage(r.Context(), id, workitemfacade.ConfirmOutboundMessageDTO{
		ConfirmedByPartyID: operatorPartyID,
		Body:               body,
	}); err != nil {
		slog.Error("confirm and send failed", "id", id, "error", err)
		http.Error(w, "failed to confirm", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, i18n.LocalizedPath(r.Context(), fmt.Sprintf("/cases/%s", id)), http.StatusSeeOther)
}

// ShowNotesPartial renders the notes pane for a specific timeline entry.
func (h *Handler) ShowNotesPartial(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entryIndexStr := r.PathValue("entryIndex")
	entryIndex, err := strconv.Atoi(entryIndexStr)
	if err != nil {
		http.Error(w, "invalid entry index", http.StatusBadRequest)
		return
	}

	notes, err := h.notes.FindNotesByEntryIndex(r.Context(), id, entryIndex)
	if err != nil {
		slog.Error("failed to load notes", "id", id, "entry_index", entryIndex, "error", err)
		http.Error(w, "failed to load notes", http.StatusInternalServerError)
		return
	}

	render.Page(w, r, templates.NotesPane(id, entryIndex, notes))
}

// HandleAddNote adds a note to a timeline entry.
func (h *Handler) HandleAddNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entryIndexStr := r.PathValue("entryIndex")
	entryIndex, err := strconv.Atoi(entryIndexStr)
	if err != nil {
		http.Error(w, "invalid entry index", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	body := r.FormValue("body")
	if body == "" {
		http.Error(w, "note body required", http.StatusBadRequest)
		return
	}

	user := auth.MustUserFromContext(r.Context())
	authorID := user.ActivePartyID

	_, err = h.workitems.AddNote(r.Context(), id, workitemfacade.AddNoteDTO{
		EntryIndex: entryIndex,
		AuthorID:   authorID,
		Body:       body,
	})
	if err != nil {
		slog.Error("add note failed", "id", id, "entry_index", entryIndex, "error", err)
		http.Error(w, "failed to add note", http.StatusInternalServerError)
		return
	}

	// Return updated notes pane
	notes, err := h.notes.FindNotesByEntryIndex(r.Context(), id, entryIndex)
	if err != nil {
		slog.Error("failed to reload notes", "id", id, "entry_index", entryIndex, "error", err)
		http.Error(w, "failed to reload notes", http.StatusInternalServerError)
		return
	}

	render.Page(w, r, templates.NotesPane(id, entryIndex, notes))
}

// HandleEditNote edits an existing note.
func (h *Handler) HandleEditNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	noteID := r.PathValue("noteId")
	entryIndexStr := r.PathValue("entryIndex")
	entryIndex, err := strconv.Atoi(entryIndexStr)
	if err != nil {
		http.Error(w, "invalid entry index", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	body := r.FormValue("body")
	if body == "" {
		http.Error(w, "note body required", http.StatusBadRequest)
		return
	}

	if err := h.workitems.EditNote(r.Context(), id, workitemfacade.EditNoteDTO{
		NoteID: noteID,
		Body:   body,
	}); err != nil {
		slog.Error("edit note failed", "id", id, "note_id", noteID, "error", err)
		http.Error(w, "failed to edit note", http.StatusInternalServerError)
		return
	}

	// Return updated notes pane
	notes, err := h.notes.FindNotesByEntryIndex(r.Context(), id, entryIndex)
	if err != nil {
		slog.Error("failed to reload notes", "id", id, "entry_index", entryIndex, "error", err)
		http.Error(w, "failed to reload notes", http.StatusInternalServerError)
		return
	}

	render.Page(w, r, templates.NotesPane(id, entryIndex, notes))
}

// HandleDeleteNote soft-deletes a note.
func (h *Handler) HandleDeleteNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	noteID := r.PathValue("noteId")
	entryIndexStr := r.PathValue("entryIndex")
	entryIndex, err := strconv.Atoi(entryIndexStr)
	if err != nil {
		http.Error(w, "invalid entry index", http.StatusBadRequest)
		return
	}

	if err := h.workitems.DeleteNote(r.Context(), id, workitemfacade.DeleteNoteDTO{
		NoteID: noteID,
	}); err != nil {
		slog.Error("delete note failed", "id", id, "note_id", noteID, "error", err)
		http.Error(w, "failed to delete note", http.StatusInternalServerError)
		return
	}

	// Return updated notes pane
	notes, err := h.notes.FindNotesByEntryIndex(r.Context(), id, entryIndex)
	if err != nil {
		slog.Error("failed to reload notes", "id", id, "entry_index", entryIndex, "error", err)
		http.Error(w, "failed to reload notes", http.StatusInternalServerError)
		return
	}

	render.Page(w, r, templates.NotesPane(id, entryIndex, notes))
}
