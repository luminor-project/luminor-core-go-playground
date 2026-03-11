package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/rag/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/rag/facade"
)

type fakeRAG struct {
	indexDocumentFunc  func(ctx context.Context, dto facade.IndexDocumentDTO) (facade.DocumentDTO, error)
	searchFunc         func(ctx context.Context, query string, limit int, threshold float64) ([]facade.SearchResultDTO, error)
	chatFunc           func(ctx context.Context, query string, limit int, threshold float64) (facade.ChatResponseDTO, error)
	deleteDocumentFunc func(ctx context.Context, documentID string) error
}

func (f *fakeRAG) IndexDocument(ctx context.Context, dto facade.IndexDocumentDTO) (facade.DocumentDTO, error) {
	if f.indexDocumentFunc != nil {
		return f.indexDocumentFunc(ctx, dto)
	}
	return facade.DocumentDTO{}, nil
}

func (f *fakeRAG) Search(ctx context.Context, query string, limit int, threshold float64) ([]facade.SearchResultDTO, error) {
	if f.searchFunc != nil {
		return f.searchFunc(ctx, query, limit, threshold)
	}
	return nil, nil
}

func (f *fakeRAG) Chat(ctx context.Context, query string, limit int, threshold float64) (facade.ChatResponseDTO, error) {
	if f.chatFunc != nil {
		return f.chatFunc(ctx, query, limit, threshold)
	}
	return facade.ChatResponseDTO{}, nil
}

func (f *fakeRAG) DeleteDocument(ctx context.Context, documentID string) error {
	if f.deleteDocumentFunc != nil {
		return f.deleteDocumentFunc(ctx, documentID)
	}
	return nil
}

// --- IndexDocument tests ---

func TestHandleIndexDocument_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		indexDocumentFunc: func(_ context.Context, dto facade.IndexDocumentDTO) (facade.DocumentDTO, error) {
			return facade.DocumentDTO{ID: "doc-1", Title: dto.Title, SourceType: dto.SourceType}, nil
		},
	})

	body := `{"title":"Test Doc","content":"Some content","source_type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/documents", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleIndexDocument(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp indexDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "doc-1" {
		t.Errorf("expected ID doc-1, got %q", resp.ID)
	}
}

func TestHandleIndexDocument_MissingTitle(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{})

	body := `{"content":"Some content"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/documents", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleIndexDocument(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleIndexDocument_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{})

	req := httptest.NewRequest(http.MethodPost, "/api/rag/documents", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	h.HandleIndexDocument(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleIndexDocument_EmptyContent(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		indexDocumentFunc: func(_ context.Context, _ facade.IndexDocumentDTO) (facade.DocumentDTO, error) {
			return facade.DocumentDTO{}, domain.ErrEmptyContent
		},
	})

	body := `{"title":"Test","content":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/documents", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleIndexDocument(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleIndexDocument_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		indexDocumentFunc: func(_ context.Context, _ facade.IndexDocumentDTO) (facade.DocumentDTO, error) {
			return facade.DocumentDTO{}, errors.New("db down")
		},
	})

	body := `{"title":"Test","content":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/documents", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleIndexDocument(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Search tests ---

func TestHandleSearch_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		searchFunc: func(_ context.Context, _ string, _ int, _ float64) ([]facade.SearchResultDTO, error) {
			return []facade.SearchResultDTO{{ChunkID: "c-1", Content: "hit"}}, nil
		},
	})

	body := `{"query":"test query","limit":5,"threshold":0.5}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/search", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp searchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].ChunkID != "c-1" {
		t.Errorf("unexpected results: %+v", resp.Results)
	}
}

func TestHandleSearch_EmptyQuery(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{})

	body := `{"query":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/search", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSearch_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		searchFunc: func(_ context.Context, _ string, _ int, _ float64) ([]facade.SearchResultDTO, error) {
			return nil, errors.New("search down")
		},
	})

	body := `{"query":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/search", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- Chat tests ---

func TestHandleChat_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		chatFunc: func(_ context.Context, _ string, _ int, _ float64) (facade.ChatResponseDTO, error) {
			return facade.ChatResponseDTO{
				Answer:  "42",
				Sources: []facade.SearchResultDTO{{ChunkID: "c-1"}},
			}, nil
		},
	})

	body := `{"query":"meaning of life"}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/chat", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleChat(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp chatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Answer != "42" {
		t.Errorf("expected answer '42', got %q", resp.Answer)
	}
	if len(resp.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(resp.Sources))
	}
}

func TestHandleChat_EmptyQuery(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{})

	body := `{"query":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/rag/chat", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleChat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- DeleteDocument tests ---

func TestHandleDeleteDocument_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{})

	req := httptest.NewRequest(http.MethodDelete, "/api/rag/documents/doc-1", nil)
	req.SetPathValue("documentId", "doc-1")
	w := httptest.NewRecorder()
	h.HandleDeleteDocument(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestHandleDeleteDocument_MissingID(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{})

	req := httptest.NewRequest(http.MethodDelete, "/api/rag/documents/", nil)
	w := httptest.NewRecorder()
	h.HandleDeleteDocument(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDeleteDocument_InternalError(t *testing.T) {
	t.Parallel()
	h := NewHandler(&fakeRAG{
		deleteDocumentFunc: func(_ context.Context, _ string) error {
			return errors.New("db down")
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/rag/documents/doc-1", nil)
	req.SetPathValue("documentId", "doc-1")
	w := httptest.NewRecorder()
	h.HandleDeleteDocument(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
