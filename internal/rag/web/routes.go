package web

import "net/http"

// RegisterRoutes registers the RAG JSON API routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, rag ragUseCases) {
	h := NewHandler(rag)

	mux.HandleFunc("POST /api/rag/documents", h.HandleIndexDocument)
	mux.HandleFunc("DELETE /api/rag/documents/{documentId}", h.HandleDeleteDocument)
	mux.HandleFunc("POST /api/rag/search", h.HandleSearch)
	mux.HandleFunc("POST /api/rag/chat", h.HandleChat)
}
