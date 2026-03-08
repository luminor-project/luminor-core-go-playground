package domain

import (
	"context"
	"errors"
	"testing"
)

// --- Mock Embedder ---

type mockEmbedder struct {
	embedding []float32
	err       error
}

func (m *mockEmbedder) Embed(_ context.Context, _, _ string) ([]float32, error) {
	return m.embedding, m.err
}

// --- Mock Generator ---

type mockGenerator struct {
	response string
	err      error
}

func (m *mockGenerator) Chat(_ context.Context, _ string, _ []Message) (string, error) {
	return m.response, m.err
}

// --- Mock Repository ---

type mockRepository struct {
	createDocumentFn     func(ctx context.Context, doc Document) error
	createChunksFn       func(ctx context.Context, chunks []Chunk) error
	findSimilarChunksFn  func(ctx context.Context, embedding []float32, limit int, threshold float64) ([]SearchResult, error)
	deleteDocAndChunksFn func(ctx context.Context, documentID string) error
}

func (m *mockRepository) CreateDocument(ctx context.Context, doc Document) error {
	if m.createDocumentFn != nil {
		return m.createDocumentFn(ctx, doc)
	}
	return nil
}

func (m *mockRepository) CreateChunks(ctx context.Context, chunks []Chunk) error {
	if m.createChunksFn != nil {
		return m.createChunksFn(ctx, chunks)
	}
	return nil
}

func (m *mockRepository) FindSimilarChunks(ctx context.Context, embedding []float32, limit int, threshold float64) ([]SearchResult, error) {
	if m.findSimilarChunksFn != nil {
		return m.findSimilarChunksFn(ctx, embedding, limit, threshold)
	}
	return nil, nil
}

func (m *mockRepository) DeleteDocumentAndChunks(ctx context.Context, documentID string) error {
	if m.deleteDocAndChunksFn != nil {
		return m.deleteDocAndChunksFn(ctx, documentID)
	}
	return nil
}

func (m *mockRepository) ExecuteInTx(_ context.Context, fn func(repo Repository) error) error {
	return fn(m)
}

// --- Tests ---

func TestIndexDocument_EmptyContent(t *testing.T) {
	svc := NewRAGService(&mockRepository{}, &mockEmbedder{}, &mockGenerator{}, "model", "model")

	_, err := svc.IndexDocument(context.Background(), "title", "text", "", nil)
	if !errors.Is(err, ErrEmptyContent) {
		t.Fatalf("expected ErrEmptyContent, got %v", err)
	}
}

func TestIndexDocument_Success(t *testing.T) {
	var createdDoc Document
	var createdChunks []Chunk

	repo := &mockRepository{
		createDocumentFn: func(_ context.Context, doc Document) error {
			createdDoc = doc
			return nil
		},
		createChunksFn: func(_ context.Context, chunks []Chunk) error {
			createdChunks = chunks
			return nil
		},
	}

	embedder := &mockEmbedder{embedding: make([]float32, 768)}
	svc := NewRAGService(repo, embedder, &mockGenerator{}, "nomic-embed-text", "llama3")

	doc, err := svc.IndexDocument(context.Background(), "Test Doc", "text", "Hello world from the test document.", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Title != "Test Doc" {
		t.Fatalf("expected title 'Test Doc', got %q", doc.Title)
	}
	if createdDoc.ID == "" {
		t.Fatal("document was not persisted")
	}
	if len(createdChunks) == 0 {
		t.Fatal("no chunks were created")
	}
}

func TestIndexDocument_EmbedError(t *testing.T) {
	repo := &mockRepository{}
	embedder := &mockEmbedder{err: errors.New("ollama down")}
	svc := NewRAGService(repo, embedder, &mockGenerator{}, "model", "model")

	_, err := svc.IndexDocument(context.Background(), "title", "text", "some content here", nil)
	if err == nil {
		t.Fatal("expected error from embedder")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	svc := NewRAGService(&mockRepository{}, &mockEmbedder{}, &mockGenerator{}, "model", "model")

	_, err := svc.Search(context.Background(), "", 5, 0.3)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearch_Success(t *testing.T) {
	expected := []SearchResult{
		{ChunkID: "c1", Content: "matching chunk", Score: 0.95},
	}

	repo := &mockRepository{
		findSimilarChunksFn: func(_ context.Context, _ []float32, _ int, _ float64) ([]SearchResult, error) {
			return expected, nil
		},
	}
	embedder := &mockEmbedder{embedding: make([]float32, 768)}
	svc := NewRAGService(repo, embedder, &mockGenerator{}, "model", "model")

	results, err := svc.Search(context.Background(), "test query", 5, 0.3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ChunkID != "c1" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestChat_Success(t *testing.T) {
	repo := &mockRepository{
		findSimilarChunksFn: func(_ context.Context, _ []float32, _ int, _ float64) ([]SearchResult, error) {
			return []SearchResult{{Content: "context chunk", Title: "Doc1"}}, nil
		},
	}
	embedder := &mockEmbedder{embedding: make([]float32, 768)}
	generator := &mockGenerator{response: "Here is the answer based on the context."}
	svc := NewRAGService(repo, embedder, generator, "model", "model")

	answer, sources, err := svc.Chat(context.Background(), "What is this about?", 5, 0.3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}
