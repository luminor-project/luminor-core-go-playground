package web

import (
	"testing"
)

func TestNewGreetingProvider(t *testing.T) {
	t.Parallel()

	provider := NewGreetingProvider()

	if provider == nil {
		t.Fatal("expected provider to not be nil")
	}

	if len(provider.messages) == 0 {
		t.Fatal("expected messages to not be empty")
	}
}

func TestGetRandomGreeting(t *testing.T) {
	t.Parallel()

	provider := NewGreetingProvider()

	greeting := provider.GetRandomGreeting()

	if greeting == "" {
		t.Fatal("expected greeting to not be empty")
	}

	found := false
	for _, msg := range provider.messages {
		if msg == greeting {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected greeting to be one of the predefined messages, got: %s", greeting)
	}
}

func TestGetRandomGreeting_ReturnsWelcomeWhenEmpty(t *testing.T) {
	t.Parallel()

	provider := &GreetingProvider{
		messages: []string{},
	}

	greeting := provider.GetRandomGreeting()

	if greeting != "Welcome!" {
		t.Fatalf("expected 'Welcome!' when messages are empty, got: %s", greeting)
	}
}

func TestGetRandomGreeting_Distribution(t *testing.T) {
	t.Parallel()

	provider := NewGreetingProvider()

	seen := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		greeting := provider.GetRandomGreeting()
		seen[greeting]++
	}

	if len(seen) == 0 {
		t.Fatal("expected to see at least one greeting after multiple calls")
	}

	if len(seen) == 1 && len(provider.messages) > 1 {
		t.Log("Warning: Only one unique greeting seen in 100 iterations, randomness may be limited")
	}
}
