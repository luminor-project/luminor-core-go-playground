package clock

import "time"

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// RealClock returns the actual current time.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time { return time.Now() }

// New creates a RealClock.
func New() RealClock { return RealClock{} }

// FixedClock always returns the same time. Useful for tests.
type FixedClock struct{ T time.Time }

// Now returns the fixed time.
func (c FixedClock) Now() time.Time { return c.T }

// NewFixed creates a FixedClock with the given time.
func NewFixed(t time.Time) FixedClock { return FixedClock{T: t} }
