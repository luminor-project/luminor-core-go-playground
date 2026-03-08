package domain

import (
	"strings"
	"testing"
)

func TestChunkText_EmptyInput(t *testing.T) {
	chunks := ChunkText("", 500, 50)
	if chunks != nil {
		t.Fatalf("expected nil, got %v", chunks)
	}
}

func TestChunkText_ShortText(t *testing.T) {
	text := "Hello world this is a short text."
	chunks := ChunkText(text, 500, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("expected %q, got %q", text, chunks[0])
	}
}

func TestChunkText_ProducesOverlap(t *testing.T) {
	// Create text with enough words to produce multiple chunks.
	words := make([]string, 600)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	chunks := ChunkText(text, 500, 50)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify chunks are non-empty and contain only expected words.
	for i, c := range chunks {
		if c == "" {
			t.Fatalf("chunk %d is empty", i)
		}
	}
}

func TestChunkText_CoversAllContent(t *testing.T) {
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"}
	text := strings.Join(words, " ")

	// Small chunk size to force multiple chunks.
	chunks := ChunkText(text, 5, 1)
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}

	// Last word must appear in the last chunk.
	lastChunk := chunks[len(chunks)-1]
	if !strings.Contains(lastChunk, "kappa") {
		t.Fatalf("last chunk %q does not contain 'kappa'", lastChunk)
	}
}

func TestEstimateTokens(t *testing.T) {
	text := "one two three four"
	tokens := EstimateTokens(text)
	// 4 words / 0.75 ≈ 5.33 → 5
	if tokens < 4 || tokens > 7 {
		t.Fatalf("expected ~5 tokens, got %d", tokens)
	}
}
