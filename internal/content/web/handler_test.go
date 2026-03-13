package web

import (
	"testing"
	"time"
)

func TestGreetingForTime(t *testing.T) {
	tests := []struct {
		name     string
		hour     int
		minute   int
		expected string
	}{
		{"morning early", 6, 0, "homepage.greeting.morning"},
		{"morning late", 11, 59, "homepage.greeting.morning"},
		{"afternoon early", 12, 0, "homepage.greeting.afternoon"},
		{"afternoon late", 17, 59, "homepage.greeting.afternoon"},
		{"evening early", 18, 0, "homepage.greeting.evening"},
		{"evening late", 21, 59, "homepage.greeting.evening"},
		{"night early", 22, 0, "homepage.greeting.night"},
		{"night late", 23, 59, "homepage.greeting.night"},
		{"night midnight", 0, 0, "homepage.greeting.night"},
		{"night before morning", 5, 59, "homepage.greeting.night"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Date(2024, 1, 1, tt.hour, tt.minute, 0, 0, time.UTC)
			got := greetingForTime(testTime)
			if got != tt.expected {
				t.Errorf("greetingForTime(%02d:%02d) = %q, want %q", tt.hour, tt.minute, got, tt.expected)
			}
		})
	}
}
