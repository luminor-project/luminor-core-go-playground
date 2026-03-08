package facade

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/rag/domain"
)

type ragService interface {
	IndexDocument(ctx context.Context, title, sourceType, content string, metadata map[string]string) (domain.Document, error)
	Search(ctx context.Context, query string, limit int, threshold float64) ([]domain.SearchResult, error)
	Chat(ctx context.Context, query string, limit int, threshold float64) (string, []domain.SearchResult, error)
	DeleteDocument(ctx context.Context, documentID string) error
}

// Compile-time interface assertion: facadeImpl satisfies all consumer interfaces.
var _ interface {
	IndexDocument(ctx context.Context, dto IndexDocumentDTO) (DocumentDTO, error)
	Search(ctx context.Context, query string, limit int, threshold float64) ([]SearchResultDTO, error)
	Chat(ctx context.Context, query string, limit int, threshold float64) (ChatResponseDTO, error)
	DeleteDocument(ctx context.Context, documentID string) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	service ragService
	bus     *eventbus.Bus
}

// New creates a new RAG facade implementation.
func New(service ragService, bus *eventbus.Bus) *facadeImpl {
	return &facadeImpl{
		service: service,
		bus:     bus,
	}
}

func (f *facadeImpl) IndexDocument(ctx context.Context, dto IndexDocumentDTO) (DocumentDTO, error) {
	doc, err := f.service.IndexDocument(ctx, dto.Title, dto.SourceType, dto.Content, dto.Metadata)
	if err != nil {
		return DocumentDTO{}, fmt.Errorf("index document: %w", err)
	}

	if err := eventbus.Publish(ctx, f.bus, DocumentIndexedEvent{
		DocumentID: doc.ID,
		Title:      doc.Title,
	}); err != nil {
		slog.Warn("failed to publish DocumentIndexedEvent", "error", err, "document_id", doc.ID)
	}

	return toDocumentDTO(doc), nil
}

func (f *facadeImpl) Search(ctx context.Context, query string, limit int, threshold float64) ([]SearchResultDTO, error) {
	results, err := f.service.Search(ctx, query, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return toSearchResultDTOs(results), nil
}

func (f *facadeImpl) Chat(ctx context.Context, query string, limit int, threshold float64) (ChatResponseDTO, error) {
	answer, results, err := f.service.Chat(ctx, query, limit, threshold)
	if err != nil {
		return ChatResponseDTO{}, fmt.Errorf("chat: %w", err)
	}

	return ChatResponseDTO{
		Answer:  answer,
		Sources: toSearchResultDTOs(results),
	}, nil
}

func (f *facadeImpl) DeleteDocument(ctx context.Context, documentID string) error {
	return f.service.DeleteDocument(ctx, documentID)
}

func toDocumentDTO(doc domain.Document) DocumentDTO {
	return DocumentDTO{
		ID:         doc.ID,
		Title:      doc.Title,
		SourceType: doc.SourceType,
		CreatedAt:  doc.CreatedAt,
	}
}

func toSearchResultDTOs(results []domain.SearchResult) []SearchResultDTO {
	dtos := make([]SearchResultDTO, len(results))
	for i, r := range results {
		dtos[i] = SearchResultDTO{
			ChunkID:    r.ChunkID,
			DocumentID: r.DocumentID,
			Content:    r.Content,
			Score:      r.Score,
			Title:      r.Title,
			SourceType: r.SourceType,
			Metadata:   r.Metadata,
		}
	}
	return dtos
}
