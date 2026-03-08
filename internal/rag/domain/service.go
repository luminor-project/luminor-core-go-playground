package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrEmptyContent     = errors.New("document content is empty")
	ErrEmptyQuery       = errors.New("search query is empty")
)

// Repository defines the persistence interface for RAG documents and chunks.
type Repository interface {
	CreateDocument(ctx context.Context, doc Document) error
	CreateChunks(ctx context.Context, chunks []Chunk) error
	FindSimilarChunks(ctx context.Context, embedding []float32, limit int, threshold float64) ([]SearchResult, error)
	DeleteDocumentAndChunks(ctx context.Context, documentID string) error
	ExecuteInTx(ctx context.Context, fn func(repo Repository) error) error
}

// Embedder generates vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, model, text string) ([]float32, error)
}

// Generator produces chat completions.
type Generator interface {
	Chat(ctx context.Context, model string, messages []Message) (string, error)
}

// Message represents a chat message for the generator.
type Message struct {
	Role    string
	Content string
}

// RAGService orchestrates document indexing, search, and retrieval-augmented generation.
type RAGService struct {
	repo       Repository
	embedder   Embedder
	generator  Generator
	embedModel string
	chatModel  string
}

// NewRAGService creates a new RAGService.
func NewRAGService(repo Repository, embedder Embedder, generator Generator, embedModel, chatModel string) *RAGService {
	return &RAGService{
		repo:       repo,
		embedder:   embedder,
		generator:  generator,
		embedModel: embedModel,
		chatModel:  chatModel,
	}
}

// IndexDocument chunks, embeds, and stores a document with its vector embeddings.
func (s *RAGService) IndexDocument(ctx context.Context, title, sourceType, content string, metadata map[string]string) (Document, error) {
	if strings.TrimSpace(content) == "" {
		return Document{}, ErrEmptyContent
	}

	doc := NewDocument(title, sourceType, content, metadata)
	texts := ChunkText(content, 500, 50)

	// Embed all chunks concurrently (bounded).
	embeddings := make([][]float32, len(texts))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for i, text := range texts {
		g.Go(func() error {
			emb, err := s.embedder.Embed(gctx, s.embedModel, text)
			if err != nil {
				return fmt.Errorf("embed chunk %d: %w", i, err)
			}
			embeddings[i] = emb
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return Document{}, err
	}

	// Build chunk entities.
	chunks := make([]Chunk, len(texts))
	for i, text := range texts {
		chunks[i] = NewChunk(doc.ID, i, text, EstimateTokens(text), embeddings[i])
	}

	// Store document + chunks in a single transaction.
	if err := s.repo.ExecuteInTx(ctx, func(txRepo Repository) error {
		if err := txRepo.CreateDocument(ctx, doc); err != nil {
			return fmt.Errorf("create document: %w", err)
		}
		if err := txRepo.CreateChunks(ctx, chunks); err != nil {
			return fmt.Errorf("create chunks: %w", err)
		}
		return nil
	}); err != nil {
		return Document{}, err
	}

	return doc, nil
}

// Search finds the most relevant chunks for a query using cosine similarity.
func (s *RAGService) Search(ctx context.Context, query string, limit int, threshold float64) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}
	if limit <= 0 {
		limit = 5
	}
	if threshold <= 0 {
		threshold = 0.3
	}

	embedding, err := s.embedder.Embed(ctx, s.embedModel, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	results, err := s.repo.FindSimilarChunks(ctx, embedding, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("find similar chunks: %w", err)
	}

	return results, nil
}

// Chat performs retrieval-augmented generation: search for context, then generate an answer.
func (s *RAGService) Chat(ctx context.Context, query string, limit int, threshold float64) (string, []SearchResult, error) {
	results, err := s.Search(ctx, query, limit, threshold)
	if err != nil {
		return "", nil, err
	}

	// Build context from retrieved chunks.
	var contextParts []string
	for i, r := range results {
		contextParts = append(contextParts, fmt.Sprintf("[Source %d: %s]\n%s", i+1, r.Title, r.Content))
	}
	contextText := strings.Join(contextParts, "\n\n")

	systemPrompt := "You are a helpful assistant. Answer the user's question based on the provided context. " +
		"If the context doesn't contain enough information to answer, say so clearly. " +
		"Cite your sources by referencing the source numbers."

	userPrompt := query
	if contextText != "" {
		userPrompt = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", contextText, query)
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	answer, err := s.generator.Chat(ctx, s.chatModel, messages)
	if err != nil {
		return "", results, fmt.Errorf("generate answer: %w", err)
	}

	return answer, results, nil
}

// DeleteDocument removes a document and all its chunks.
func (s *RAGService) DeleteDocument(ctx context.Context, documentID string) error {
	return s.repo.DeleteDocumentAndChunks(ctx, documentID)
}
