package manage

import "context"

// Housekeeping is the upkeep a state backend needs to carry out on its own storage, beyond
// the retention of any one category of state. What it does is up to the backend, such as
// reclaiming storage that interrupted writes left behind, and a backend that needs no upkeep
// does not provide it.
//
// Housekeeping must never change what the backend reads back, only how much storage it uses,
// so it is safe to run at any time, including while other processes are writing to the same
// storage.
type Housekeeping interface {
	// Run carries out one round of housekeeping and returns how many stored items it
	// deleted. A round that finds nothing to do deletes nothing and is not an error.
	Run(ctx context.Context) (int64, error)
}

// HousekeepingProvider is implemented by state containers whose backend needs housekeeping,
// so a caller can find out without knowing which backend it was given.
type HousekeepingProvider interface {
	Housekeeping() Housekeeping
}
