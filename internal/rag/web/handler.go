package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/luminor-project/luminor-core-go-playground/internal/rag/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/rag/facade"
)

type ragUseCases interface {
	IndexDocument(ctx context.Context, dto facade.IndexDocumentDTO) (facade.DocumentDTO, error)
	Search(ctx context.Context, query string, limit int, threshold float64) ([]facade.SearchResultDTO, error)
	Chat(ctx context.Context, query string, limit int, threshold float64) (facade.ChatResponseDTO, error)
	DeleteDocument(ctx context.Context, documentID string) error
}

type handler struct {
	rag ragUseCases
}

// NewHandler creates a new RAG HTTP handler.
func NewHandler(rag ragUseCases) *handler {
	return &handler{rag: rag}
}

// --- Request/Response types ---

type indexDocumentRequest struct {
	Title      string            `json:"title"`
	SourceType string            `json:"source_type"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
}

type indexDocumentResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	SourceType string `json:"source_type"`
}

type searchRequest struct {
	Query     string  `json:"query"`
	Limit     int     `json:"limit"`
	Threshold float64 `json:"threshold"`
}

type searchResultResponse struct {
	ChunkID    string            `json:"chunk_id"`
	DocumentID string            `json:"document_id"`
	Content    string            `json:"content"`
	Score      float64           `json:"score"`
	Title      string            `json:"title"`
	SourceType string            `json:"source_type"`
	Metadata   map[string]string `json:"metadata"`
}

type searchResponse struct {
	Results []searchResultResponse `json:"results"`
}

type chatRequest struct {
	Query     string  `json:"query"`
	Limit     int     `json:"limit"`
	Threshold float64 `json:"threshold"`
}

type chatResponse struct {
	Answer  string                 `json:"answer"`
	Sources []searchResultResponse `json:"sources"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Handlers ---

func (h *handler) HandleIndexDocument(w http.ResponseWriter, r *http.Request) {
	var req indexDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Title == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "title and content are required"})
		return
	}
	if req.SourceType == "" {
		req.SourceType = "text"
	}

	doc, err := h.rag.IndexDocument(r.Context(), facade.IndexDocumentDTO{
		Title:      req.Title,
		SourceType: req.SourceType,
		Content:    req.Content,
		Metadata:   req.Metadata,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmptyContent) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "document content is empty"})
			return
		}
		slog.Error("index document failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to index document"})
		return
	}

	writeJSON(w, http.StatusCreated, indexDocumentResponse{
		ID:         doc.ID,
		Title:      doc.Title,
		SourceType: doc.SourceType,
	})
}

func (h *handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "query is required"})
		return
	}

	results, err := h.rag.Search(r.Context(), req.Query, req.Limit, req.Threshold)
	if err != nil {
		slog.Error("search failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "search failed"})
		return
	}

	writeJSON(w, http.StatusOK, searchResponse{Results: toSearchResultResponses(results)})
}

func (h *handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "query is required"})
		return
	}

	resp, err := h.rag.Chat(r.Context(), req.Query, req.Limit, req.Threshold)
	if err != nil {
		slog.Error("chat failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "chat failed"})
		return
	}

	writeJSON(w, http.StatusOK, chatResponse{
		Answer:  resp.Answer,
		Sources: toSearchResultResponses(resp.Sources),
	})
}

func (h *handler) HandleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	documentID := r.PathValue("documentId")
	if documentID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "document ID is required"})
		return
	}

	if err := h.rag.DeleteDocument(r.Context(), documentID); err != nil {
		slog.Error("delete document failed", "error", err, "document_id", documentID)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete document"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("json encode failed", "error", err)
	}
}

func toSearchResultResponses(dtos []facade.SearchResultDTO) []searchResultResponse {
	results := make([]searchResultResponse, len(dtos))
	for i, dto := range dtos {
		results[i] = searchResultResponse{
			ChunkID:    dto.ChunkID,
			DocumentID: dto.DocumentID,
			Content:    dto.Content,
			Score:      dto.Score,
			Title:      dto.Title,
			SourceType: dto.SourceType,
			Metadata:   dto.Metadata,
		}
	}
	return results
}
