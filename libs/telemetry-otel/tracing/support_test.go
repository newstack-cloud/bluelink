package tracing_test

import (
	"time"
)

// A clock the tests fix in place, so that an identifier carrying a timestamp can be
// asserted against a known value rather than against whatever the host clock said.
type staticClock struct {
	at time.Time
}

func (c staticClock) Now() time.Time {
	return c.at
}
