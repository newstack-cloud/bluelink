package state

import "slices"

// UpsertContributionFailure returns the given failures with the one held for the new
// failure's layer depth replaced, or the new failure appended when no failure is held for
// that layer yet.
//
// A layer is identified purely by its depth, so a layer that fails, is retried and fails
// again is held once rather than accumulating an entry per attempt.
//
// The result is ordered by layer depth, so a resource's failures are read back in the order
// the layers would have been applied in regardless of the order they failed in.
func UpsertContributionFailure(
	failures []ResourceLinkContributionFailure,
	failure ResourceLinkContributionFailure,
) []ResourceLinkContributionFailure {
	updated := make([]ResourceLinkContributionFailure, 0, len(failures)+1)
	replaced := false

	for _, current := range failures {
		if current.LayerDepth == failure.LayerDepth {
			updated = append(updated, failure)
			replaced = true
			continue
		}

		updated = append(updated, current)
	}

	if !replaced {
		updated = append(updated, failure)
	}

	slices.SortFunc(
		updated,
		func(a, b ResourceLinkContributionFailure) int {
			return a.LayerDepth - b.LayerDepth
		},
	)

	return updated
}

// RemoveContributionFailureForLayer returns the given failures without the one held for the
// given layer depth, and nil once no failure is left.
//
// Nil rather than an empty slice, so that a resource which has no outstanding contribution
// failures is indistinguishable from one that never had any, in state and in what a client
// is given.
func RemoveContributionFailureForLayer(
	failures []ResourceLinkContributionFailure,
	layerDepth int,
) []ResourceLinkContributionFailure {
	remaining := make([]ResourceLinkContributionFailure, 0, len(failures))
	for _, current := range failures {
		if current.LayerDepth != layerDepth {
			remaining = append(remaining, current)
		}
	}

	if len(remaining) == 0 {
		return nil
	}

	return remaining
}
