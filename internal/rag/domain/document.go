package domain

import (
	"time"

	"github.com/google/uuid"
)

// Document represents an indexed document.
type Document struct {
	ID         string
	Title      string
	SourceType string
	Content    string
	Metadata   map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewDocument creates a new Document with a generated ID.
func NewDocument(title, sourceType, content string, metadata map[string]string, now time.Time) Document {
	if metadata == nil {
		metadata = map[string]string{}
	}
	return Document{
		ID:         uuid.New().String(),
		Title:      title,
		SourceType: sourceType,
		Content:    content,
		Metadata:   metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Chunk represents a text fragment of a document with its embedding.
type Chunk struct {
	ID         string
	DocumentID string
	ChunkIndex int
	Content    string
	TokenCount int
	Embedding  []float32
	CreatedAt  time.Time
}

// NewChunk creates a new Chunk with a generated ID.
func NewChunk(documentID string, index int, content string, tokenCount int, embedding []float32, now time.Time) Chunk {
	return Chunk{
		ID:         uuid.New().String(),
		DocumentID: documentID,
		ChunkIndex: index,
		Content:    content,
		TokenCount: tokenCount,
		Embedding:  embedding,
		CreatedAt:  now,
	}
}

// SearchResult pairs a chunk with its similarity score and parent document info.
type SearchResult struct {
	ChunkID    string
	DocumentID string
	Content    string
	Score      float64
	Title      string
	SourceType string
	Metadata   map[string]string
}
