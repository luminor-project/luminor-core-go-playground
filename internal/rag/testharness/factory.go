package testharness

import (
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/rag/domain"
)

// MakeDocument creates a Document with sensible defaults for testing.
func MakeDocument(title, content string) domain.Document {
	return domain.NewDocument(title, "text", content, nil, time.Now())
}
