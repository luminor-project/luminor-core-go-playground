package domain

import (
	"testing"
	"time"
)

func TestDetermineTimeOfDay(t *testing.T) {
	tests := []struct {
		name     string
		t        time.Time
		expected TimeOfDay
	}{
		{"morning start", time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC), Morning},
		{"morning end", time.Date(2024, 1, 1, 11, 59, 59, 0, time.UTC), Morning},
		{"afternoon start", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), Afternoon},
		{"afternoon end", time.Date(2024, 1, 1, 17, 59, 59, 0, time.UTC), Afternoon},
		{"evening start", time.Date(2024, 1, 1, 18, 0, 0, 0, time.UTC), Evening},
		{"evening end", time.Date(2024, 1, 1, 21, 59, 59, 0, time.UTC), Evening},
		{"night start", time.Date(2024, 1, 1, 22, 0, 0, 0, time.UTC), Night},
		{"night before morning", time.Date(2024, 1, 1, 5, 59, 59, 0, time.UTC), Night},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineTimeOfDay(tt.t)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
