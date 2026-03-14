// Package config provides configuration for content web features.
package config

import (
	"math/rand"
)

// Greeting represents a single funny greeting message.
type Greeting struct {
	Text string
}

// GreetingsConfiguration holds the collection of greetings and provides methods to retrieve them.
type GreetingsConfiguration struct {
	greetings []Greeting
	rng       func(int) int
}

// DefaultGreetings returns a curated collection of finance-themed humorous greetings.
func DefaultGreetings() []Greeting {
	return []Greeting{
		{Text: "Your rent is due, but your smile is free!"},
		{Text: "Where property management meets perfectly timed coffee breaks."},
		{Text: "Turning 'Have you paid the rent?' into 'Thanks for being awesome!'"},
		{Text: "Because spreadsheets deserve a sense of humor too."},
		{Text: "Property management: now with 47% less paperwork stress."},
	}
}

// NewGreetingsConfiguration creates a new configuration with the provided greetings.
// If no greetings are provided, it uses the default collection.
func NewGreetingsConfiguration(greetings ...Greeting) *GreetingsConfiguration {
	var g []Greeting
	if len(greetings) == 0 {
		g = DefaultGreetings()
	} else {
		g = make([]Greeting, len(greetings))
		copy(g, greetings)
	}

	return &GreetingsConfiguration{
		greetings: g,
		rng:       rand.Intn,
	}
}

// GetGreeting returns a random greeting from the collection.
// If the collection is empty, it returns an empty greeting (no error since this is a display feature).
func (c *GreetingsConfiguration) GetGreeting() Greeting {
	if len(c.greetings) == 0 {
		return Greeting{}
	}
	return c.greetings[c.rng(len(c.greetings))]
}
