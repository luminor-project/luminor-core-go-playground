package config

import (
	"testing"
)

func TestDefaultGreetings(t *testing.T) {
	greetings := DefaultGreetings()

	if len(greetings) != 5 {
		t.Errorf("Expected 5 default greetings, got %d", len(greetings))
	}

	for i, g := range greetings {
		if g.Text == "" {
			t.Errorf("Greeting at index %d has empty text", i)
		}
	}
}

func TestNewGreetingsConfiguration_UsesDefaultsWhenEmpty(t *testing.T) {
	cfg := NewGreetingsConfiguration()

	if len(cfg.greetings) != 5 {
		t.Errorf("Expected 5 default greetings, got %d", len(cfg.greetings))
	}
}

func TestNewGreetingsConfiguration_UsesProvidedGreetings(t *testing.T) {
	custom := []Greeting{
		{Text: "Custom greeting 1"},
		{Text: "Custom greeting 2"},
	}
	cfg := NewGreetingsConfiguration(custom...)

	if len(cfg.greetings) != 2 {
		t.Errorf("Expected 2 custom greetings, got %d", len(cfg.greetings))
	}

	if cfg.greetings[0].Text != "Custom greeting 1" {
		t.Errorf("Expected first greeting to be 'Custom greeting 1', got %s", cfg.greetings[0].Text)
	}
}

func TestGetGreeting_EmptyCollection(t *testing.T) {
	cfg := &GreetingsConfiguration{
		greetings: []Greeting{},
		rng:       func(_ int) int { return 0 },
	}

	greeting := cfg.GetGreeting()

	if greeting.Text != "" {
		t.Errorf("Expected empty greeting, got %s", greeting.Text)
	}
}

func TestGetGreeting_SingleGreeting(t *testing.T) {
	cfg := NewGreetingsConfiguration(Greeting{Text: "Only greeting"})
	cfg.rng = func(_ int) int { return 0 }

	greeting := cfg.GetGreeting()

	if greeting.Text != "Only greeting" {
		t.Errorf("Expected 'Only greeting', got %s", greeting.Text)
	}
}

func TestGetGreeting_ReturnsGreetingFromCollection(t *testing.T) {
	custom := []Greeting{
		{Text: "First"},
		{Text: "Second"},
		{Text: "Third"},
	}
	cfg := NewGreetingsConfiguration(custom...)

	valid := map[string]bool{
		"First":  true,
		"Second": true,
		"Third":  true,
	}

	for i := 0; i < 10; i++ {
		greeting := cfg.GetGreeting()
		if !valid[greeting.Text] {
			t.Errorf("Iteration %d: Got unexpected greeting '%s'", i, greeting.Text)
		}
	}
}

func TestGetGreeting_RandomDistribution(t *testing.T) {
	custom := []Greeting{
		{Text: "A"},
		{Text: "B"},
		{Text: "C"},
	}
	cfg := NewGreetingsConfiguration(custom...)

	counts := map[string]int{
		"A": 0,
		"B": 0,
		"C": 0,
	}

	iterations := 1000
	for i := 0; i < iterations; i++ {
		greeting := cfg.GetGreeting()
		counts[greeting.Text]++
	}

	for text, count := range counts {
		// With proper randomness, each greeting should appear roughly 1/3 of the time
		// Allow for statistical variance: each should appear at least 5% of iterations
		// This test has a 1 - (0.95^1000) ~ 100% chance of passing with proper randomness
		minExpected := iterations * 5 / 100
		if count < minExpected {
			t.Errorf("Greeting '%s' appeared only %d times out of %d (expected at least %d). "+
				"This suggests poor random distribution.", text, count, iterations, minExpected)
		}
	}
}
