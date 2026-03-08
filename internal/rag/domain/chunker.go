package domain

import "strings"

// ChunkText splits text into overlapping chunks using a sliding window approach.
// targetTokens is the approximate number of tokens per chunk (~1 token per 0.75 words).
// overlapTokens is the approximate overlap between consecutive chunks.
func ChunkText(text string, targetTokens, overlapTokens int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	// Convert token counts to word counts (1 token ≈ 0.75 words)
	targetWords := int(float64(targetTokens) * 0.75)
	overlapWords := int(float64(overlapTokens) * 0.75)

	if targetWords < 1 {
		targetWords = 1
	}
	if overlapWords >= targetWords {
		overlapWords = targetWords / 4
	}

	step := targetWords - overlapWords
	if step < 1 {
		step = 1
	}

	var chunks []string
	for start := 0; start < len(words); start += step {
		end := start + targetWords
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, chunk)
		if end == len(words) {
			break
		}
	}

	return chunks
}

// EstimateTokens returns an approximate token count for the given text.
func EstimateTokens(text string) int {
	wordCount := len(strings.Fields(text))
	// ~1 token per 0.75 words → tokens ≈ words / 0.75
	return int(float64(wordCount) / 0.75)
}
