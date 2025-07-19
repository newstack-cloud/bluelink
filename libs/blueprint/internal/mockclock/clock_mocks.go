package mockclock

import "time"

// Thursday, 7th September 2023 14:43:44
const CurrentTimeUnixMock int64 = 1694097824

type StaticClock struct{}

func (c *StaticClock) Now() time.Time {
	return time.Unix(CurrentTimeUnixMock, 0)
}

func (c *StaticClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

// AdvanceableClock is a mock implementation of the core.Clock interface
// that allows for advancing the clock by a specified duration.
type AdvanceableClock struct {
	now time.Time
}

// NewAdvanceableClock creates a new instance of AdvanceableClock
// with the current time set to the provided time.
// This is useful for testing scenarios where the clock needs to be advanced
// to simulate the passage of time.
func NewAdvanceableClock(
	currentTime time.Time,
) *AdvanceableClock {
	return &AdvanceableClock{
		now: currentTime,
	}
}

func (c *AdvanceableClock) Now() time.Time {
	return c.now
}

func (c *AdvanceableClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

// Advance advances the clock by the given duration, this is not
// a part of the Clock interface, but is useful for testing
// scenarios where the clock needs to be advanced to simulate
// the passage of time.
func (c *AdvanceableClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}
