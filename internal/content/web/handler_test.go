package web

import (
	"strings"
	"testing"
	"time"
)

func TestGreetingForTime(t *testing.T) {
	// Helper to check if a key is in a slice
	contains := func(slice []string, key string) bool {
		for _, item := range slice {
			if item == key {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name           string
		hour           int
		minute         int
		expectedPool   []string
		expectedPrefix string
	}{
		{"morning early", 6, 0, greetingPools["morning"], "homepage.greeting.morning"},
		{"morning late", 11, 59, greetingPools["morning"], "homepage.greeting.morning"},
		{"afternoon early", 12, 0, greetingPools["afternoon"], "homepage.greeting.afternoon"},
		{"afternoon late", 17, 59, greetingPools["afternoon"], "homepage.greeting.afternoon"},
		{"evening early", 18, 0, greetingPools["evening"], "homepage.greeting.evening"},
		{"evening late", 21, 59, greetingPools["evening"], "homepage.greeting.evening"},
		{"night early", 22, 0, greetingPools["night"], "homepage.greeting.night"},
		{"night late", 23, 59, greetingPools["night"], "homepage.greeting.night"},
		{"night midnight", 0, 0, greetingPools["night"], "homepage.greeting.night"},
		{"night before morning", 5, 59, greetingPools["night"], "homepage.greeting.night"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Date(2024, 1, 1, tt.hour, tt.minute, 0, 0, time.UTC)
			got := greetingForTime(testTime)

			// Check that the returned key is in the expected pool
			if !contains(tt.expectedPool, got) {
				t.Errorf("greetingForTime(%02d:%02d) = %q, want one of %v", tt.hour, tt.minute, got, tt.expectedPool)
			}

			// Check that the key has the expected prefix
			if !strings.HasPrefix(got, tt.expectedPrefix) {
				t.Errorf("greetingForTime(%02d:%02d) = %q, want prefix %q", tt.hour, tt.minute, got, tt.expectedPrefix)
			}
		})
	}
}

func TestGreetingPoolsNotEmpty(t *testing.T) {
	// Ensure all pools have at least one greeting
	for period, pool := range greetingPools {
		if len(pool) == 0 {
			t.Errorf("greeting pool for %s is empty", period)
		}
	}
}
