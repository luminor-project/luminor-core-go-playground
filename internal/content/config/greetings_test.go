package config

import (
	"math/rand"
	"sync"
	"testing"
	"time"
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

func TestGetGreeting_ThreadSafety(t *testing.T) {
	custom := []Greeting{
		{Text: "First"},
		{Text: "Second"},
		{Text: "Third"},
	}
	cfg := NewGreetingsConfiguration(custom...)

	var wg sync.WaitGroup
	iterations := 100
	goroutines := 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				greeting := cfg.GetGreeting()
				if greeting.Text != "First" && greeting.Text != "Second" && greeting.Text != "Third" {
					t.Errorf("Got unexpected greeting: %s", greeting.Text)
				}
			}
		}()
	}

	wg.Wait()
}

func TestGetGreeting_DifferentSeedsProduceDifferentSelections(t *testing.T) {
	custom := []Greeting{
		{Text: "A"},
		{Text: "B"},
		{Text: "C"},
		{Text: "D"},
		{Text: "E"},
	}

	seeds := []int64{1, 42, 123, 456, 789}
	selections := make(map[string]bool)

	for _, seed := range seeds {
		rng := rand.New(rand.NewSource(seed))
		cfg := &GreetingsConfiguration{
			greetings: custom,
			rng:       rng.Intn,
		}

		for i := 0; i < 10; i++ {
			greeting := cfg.GetGreeting()
			selections[greeting.Text] = true
		}
	}

	if len(selections) < 3 {
		t.Errorf("Expected diverse selections across different seeds, got only %d unique greetings", len(selections))
	}
}

func TestGetGreeting_LargeCollection(t *testing.T) {
	largeCollection := make([]Greeting, 1000)
	for i := 0; i < 1000; i++ {
		largeCollection[i] = Greeting{Text: "Greeting " + string(rune('0'+i%10))}
	}

	cfg := NewGreetingsConfiguration(largeCollection...)

	counts := make(map[string]int)
	iterations := 5000

	for i := 0; i < iterations; i++ {
		greeting := cfg.GetGreeting()
		counts[greeting.Text]++
	}

	// With 1000 items and 10 distinct greetings, each should appear roughly 1/10 of the time
	for text, count := range counts {
		expected := iterations / 10
		variance := float64(count-expected) / float64(expected)
		if variance < -0.3 || variance > 0.3 {
			t.Errorf("Greeting '%s' distribution skewed: got %d, expected ~%d (variance: %.2f%%)",
				text, count, expected, variance*100)
		}
	}
}

func TestGetGreeting_RNGReturnsUnexpectedValues(t *testing.T) {
	custom := []Greeting{
		{Text: "First"},
		{Text: "Second"},
		{Text: "Third"},
	}

	// Test with deterministic RNG that returns boundary values
	testCases := []struct {
		name     string
		rngFunc  func(int) int
		expected string
	}{
		{
			name:     "returns zero",
			rngFunc:  func(n int) int { return 0 },
			expected: "First",
		},
		{
			name:     "returns max index",
			rngFunc:  func(n int) int { return n - 1 },
			expected: "Third",
		},
		{
			name:     "returns middle index",
			rngFunc:  func(n int) int { return n / 2 },
			expected: "Second",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &GreetingsConfiguration{
				greetings: custom,
				rng:       tc.rngFunc,
			}

			greeting := cfg.GetGreeting()
			if greeting.Text != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, greeting.Text)
			}
		})
	}
}

func TestNewGreetingsConfiguration_WithNilGreetings(t *testing.T) {
	cfg := NewGreetingsConfiguration(nil...)

	if len(cfg.greetings) != 5 {
		t.Errorf("Expected 5 default greetings when nil slice provided, got %d", len(cfg.greetings))
	}
}

func TestGetGreeting_ConsistentWithSeededRNG(t *testing.T) {
	custom := []Greeting{
		{Text: "Alpha"},
		{Text: "Beta"},
		{Text: "Gamma"},
		{Text: "Delta"},
		{Text: "Epsilon"},
	}

	seed := time.Now().UnixNano()
	rng1 := rand.New(rand.NewSource(seed))
	rng2 := rand.New(rand.NewSource(seed))

	cfg1 := &GreetingsConfiguration{greetings: custom, rng: rng1.Intn}
	cfg2 := &GreetingsConfiguration{greetings: custom, rng: rng2.Intn}

	for i := 0; i < 20; i++ {
		g1 := cfg1.GetGreeting()
		g2 := cfg2.GetGreeting()
		if g1.Text != g2.Text {
			t.Errorf("Same seed produced different greetings at iteration %d: '%s' vs '%s'",
				i, g1.Text, g2.Text)
		}
	}
}
