package facade

import "time"

// IndexDocumentDTO holds data for indexing a new document.
type IndexDocumentDTO struct {
	Title      string
	SourceType string
	Content    string
	Metadata   map[string]string
}

// DocumentDTO represents an indexed document.
type DocumentDTO struct {
	ID         string
	Title      string
	SourceType string
	CreatedAt  time.Time
}

// SearchResultDTO represents a search hit with relevance score.
type SearchResultDTO struct {
	ChunkID    string
	DocumentID string
	Content    string
	Score      float64
	Title      string
	SourceType string
	Metadata   map[string]string
}

// ChatResponseDTO holds the generated answer and its supporting sources.
type ChatResponseDTO struct {
	Answer  string
	Sources []SearchResultDTO
}
