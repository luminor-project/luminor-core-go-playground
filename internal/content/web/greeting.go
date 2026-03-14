package web

import (
	"crypto/rand"
	"math/big"
)

// GreetingProvider provides greeting messages for the homepage.
type GreetingProvider struct {
	messages []string
}

// NewGreetingProvider creates a new greeting provider with predefined messages.
func NewGreetingProvider() *GreetingProvider {
	return &GreetingProvider{
		messages: []string{
			"Welcome! We promise our code has fewer bugs than your coffee maker.",
			"Hello! Our app is like a good pair of socks – reliable, comfortable, and hard to lose.",
			"Greetings! We've optimized this page so well, it loads before you finish blinking.",
			"Hi there! No cats were harmed in the making of this software. Probably.",
			"Welcome aboard! This app is 100% organic, free-range, and gluten-free.",
			"Hello! If this app were a pizza, it would have all your favorite toppings.",
			"Greetings, traveler! You've reached the digital equivalent of a cozy fireplace.",
			"Welcome! Our servers are running on caffeine and good intentions.",
			"Hi! This application is certified 100% bug-free by our very optimistic QA team.",
			"Hello there! The code behind this page is cleaner than a freshly washed window.",
		},
	}
}

// GetRandomGreeting returns a randomly selected greeting message.
func (g *GreetingProvider) GetRandomGreeting() string {
	if len(g.messages) == 0 {
		return "Welcome!"
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(g.messages))))
	if err != nil {
		return g.messages[0]
	}

	return g.messages[n.Int64()]
}
