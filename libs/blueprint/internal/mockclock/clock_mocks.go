package mockclock

import (
	"sync"
	"time"
)

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
//
// Safe for concurrent use: code under test can poll the clock from one goroutine while
// the test advances it from another, which is how a timeout that is waited on rather
// than simulated has to be exercised.
type AdvanceableClock struct {
	mu  sync.RWMutex
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
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}
