package domain

import "time"

// TimeOfDay represents a segment of the day.
type TimeOfDay int

const (
	Morning   TimeOfDay = iota // 06:00 - 12:00
	Afternoon                  // 12:00 - 18:00
	Evening                    // 18:00 - 22:00
	Night                      // 22:00 - 06:00
)

// DetermineTimeOfDay returns the appropriate TimeOfDay for the given time.
func DetermineTimeOfDay(t time.Time) TimeOfDay {
	hour := t.Hour()
	switch {
	case hour >= 6 && hour < 12:
		return Morning
	case hour >= 12 && hour < 18:
		return Afternoon
	case hour >= 18 && hour < 22:
		return Evening
	default:
		return Night
	}
}
