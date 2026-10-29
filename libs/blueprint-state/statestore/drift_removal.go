package statestore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Drift is only reachable through the resource or link it describes, so it is deleted along
// with them, as postgres does by cascading. Nothing else would ever delete it.
type driftedEntities struct {
	resourceIDs []string
	linkIDs     []string
}

// Drops the cached drift of resources and links that are being removed, and returns the ones
// that have stored drift to delete once the removal is persisted. Expects the State write lock
// to be held.
func (s *State) takeDriftOf(
	resources map[string]*state.ResourceState,
	links map[string]*state.LinkState,
) driftedEntities {
	drifted := driftedEntities{}
	for resourceID, resource := range resources {
		_, cached := s.resourceDrift[resourceID]
		delete(s.resourceDrift, resourceID)
		if cached || resource.Drifted {
			drifted.resourceIDs = append(drifted.resourceIDs, resourceID)
		}
	}

	for _, link := range links {
		_, cached := s.linkDrift[link.LinkID]
		delete(s.linkDrift, link.LinkID)
		if cached || link.Drifted {
			drifted.linkIDs = append(drifted.linkIDs, link.LinkID)
		}
	}

	return drifted
}

func deleteStoredDrift(
	ctx context.Context,
	persister *Persister,
	logger core.Logger,
	drifted driftedEntities,
) {
	for _, resourceID := range drifted.resourceIDs {
		drift := &state.ResourceDriftState{ResourceID: resourceID}
		if err := persister.RemoveResourceDrift(ctx, drift); err != nil {
			logger.Warn(
				"failed to delete the drift of a removed resource",
				core.StringLogField("resourceId", resourceID),
				core.ErrorLogField("error", err),
			)
		}
	}

	for _, linkID := range drifted.linkIDs {
		drift := &state.LinkDriftState{LinkID: linkID}
		if err := persister.RemoveLinkDrift(ctx, drift); err != nil {
			logger.Warn(
				"failed to delete the drift of a removed link",
				core.StringLogField("linkId", linkID),
				core.ErrorLogField("error", err),
			)
		}
	}
}
