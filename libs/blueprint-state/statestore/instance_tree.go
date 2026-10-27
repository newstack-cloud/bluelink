package statestore

import (
	"context"
	"fmt"
	"maps"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// CachePersistedInstance caches an instance read from storage together with everything
// under it, and returns the cached instance. An instance that is already cached is
// replaced in place with what was read.
//
// Child blueprints are persisted as ID references, so each referenced child is resolved
// through LookupInstance, which loads it (and its own children) on a cache miss. A
// referenced child that no longer exists, or whose removal has started, is treated as
// detached where destroying a blueprint removes a child before its parent drops the reference.
//
// Backends call this after a write they made outside the Persister, such as an atomic
// claim, so the cache reflects storage without dropping the instance's children.
func (s *State) CachePersistedInstance(
	ctx context.Context,
	persisted *PersistedInstanceState,
) (*state.InstanceState, error) {
	return s.cachePersistedInstance(ctx, persisted, cacheReplace, map[string]bool{})
}

type cacheMode int

const (
	// A lookup that loaded an instance keeps whatever another goroutine cached meanwhile,
	// because that may already carry changes made after the load read storage.
	cacheIfAbsent cacheMode = iota
	// A backend's own write is newer than anything cached, so it replaces the cached copy.
	cacheReplace
)

// Mutators work on the cache under the write lock, and loading cannot happen under it, so they
// call this first. An instance removed between the two is then reported as not found by the
// mutator, as it would have been had the removal come first.
func (s *State) preloadInstances(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		if _, _, err := s.LookupInstance(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// As preloadInstances, for mutators that address a resource by ID. Loading the resource
// loads the instance that holds it.
func (s *State) preloadResource(ctx context.Context, resourceID string) error {
	_, _, err := s.LookupResource(ctx, resourceID)
	return err
}

func (s *State) preloadResourceWithDrift(ctx context.Context, resourceID string) error {
	if err := s.preloadResource(ctx, resourceID); err != nil {
		return err
	}

	_, _, err := s.LookupResourceDrift(ctx, resourceID)
	return err
}

// As preloadResource, for links.
func (s *State) preloadLink(ctx context.Context, linkID string) error {
	_, _, err := s.LookupLink(ctx, linkID)
	return err
}

func (s *State) preloadLinkWithDrift(ctx context.Context, linkID string) error {
	if err := s.preloadLink(ctx, linkID); err != nil {
		return err
	}
	_, _, err := s.LookupLinkDrift(ctx, linkID)
	return err
}

// visiting holds the instances on the path being loaded, so a child that is its own
// ancestor is reported rather than recursed into.
func (s *State) lookupInstance(
	ctx context.Context,
	id string,
	visiting map[string]bool,
) (*state.InstanceState, bool, error) {
	s.mu.RLock()
	if inst, ok := s.instances[id]; ok {
		s.mu.RUnlock()
		return inst, true, nil
	}
	s.mu.RUnlock()

	persisted, ok, err := s.loader.LoadInstance(ctx, id)
	if err != nil || !ok || persisted.Removing {
		return nil, false, err
	}

	inst, err := s.cachePersistedInstance(ctx, persisted, cacheIfAbsent, visiting)
	if err != nil {
		return nil, false, err
	}

	return inst, true, nil
}

// Returns an instance whose removal started but did not finish, which lookups otherwise
// report as not found. It is not cached.
func (s *State) loadInstanceBeingRemoved(ctx context.Context, id string) (*state.InstanceState, bool, error) {
	persisted, ok, err := s.loader.LoadInstance(ctx, id)
	if err != nil || !ok || !persisted.Removing {
		return nil, false, err
	}

	return persisted.ToInstanceState(), true, nil
}

func (s *State) cachePersistedInstance(
	ctx context.Context,
	persisted *PersistedInstanceState,
	mode cacheMode,
	visiting map[string]bool,
) (*state.InstanceState, error) {
	visiting[persisted.InstanceID] = true
	defer delete(visiting, persisted.InstanceID)

	children, err := s.lookupChildren(ctx, persisted, visiting)
	if err != nil {
		return nil, err
	}

	loaded := persisted.ToInstanceState()
	loaded.ChildBlueprints = children

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, isCached := s.instances[loaded.InstanceID]; isCached && mode == cacheIfAbsent {
		return cached, nil
	}
	return s.cacheInstance(loaded), nil
}

func (s *State) lookupChildren(
	ctx context.Context,
	persisted *PersistedInstanceState,
	visiting map[string]bool,
) (map[string]*state.InstanceState, error) {
	children := make(map[string]*state.InstanceState, len(persisted.ChildBlueprints))
	for childName, childID := range persisted.ChildBlueprints {
		if visiting[childID] {
			return nil, errMalformedState(fmt.Sprintf(
				"instance %q is its own ancestor through child %q of instance %q",
				childID, childName, persisted.InstanceID,
			))
		}

		child, found, err := s.lookupInstance(ctx, childID, visiting)
		if err != nil {
			return nil, err
		}
		if found {
			children[childName] = child
		}
	}
	return children, nil
}

// Expects the write lock to be held.
//
// An instance that is already cached is updated in place rather than replaced, because its
// parent's ChildBlueprints map holds the existing pointer.
func (s *State) cacheInstance(instance *state.InstanceState) *state.InstanceState {
	cached, isCached := s.instances[instance.InstanceID]
	if !isCached {
		s.instances[instance.InstanceID] = instance
		s.indexInstanceEntities(nil, instance)
		s.indexInstanceName(nil, instance)
		return instance
	}

	previous := *cached
	*cached = *instance
	s.indexInstanceEntities(&previous, cached)
	s.indexInstanceName(&previous, cached)
	return cached
}

// Mutators change the cache before persisting, so after a failed instance write the cache
// holds a change storage does not. Under ModeLazy the cached instances are dropped and
// reloaded on next access; otherwise a retry would diff against the unsaved change and skip
// the ID records it still needs. All of them go, not just the one written, because a parent's
// ChildBlueprints holds pointers to its cached children. Under ModeEager there is nothing to
// reload from, so the cache is kept. Expects the write lock to be held.
func (s *State) forgetCachedInstances() {
	if !s.isLazilyListed() {
		return
	}

	// Cleared in place, since a backend may share the instances map with State.
	clear(s.instances)
	clear(s.resources)
	clear(s.links)
	clear(s.nameLookup)
}

// Keeps the resource and link caches, which are keyed by globally unique ID, in step with
// the instance that holds those entities. Expects the write lock to be held.
func (s *State) indexInstanceEntities(previous, current *state.InstanceState) {
	if previous != nil {
		currentLinkIDs := map[string]bool{}
		for _, link := range current.Links {
			currentLinkIDs[link.LinkID] = true
		}

		for resourceID := range previous.Resources {
			if _, kept := current.Resources[resourceID]; !kept {
				delete(s.resources, resourceID)
			}
		}

		for _, link := range previous.Links {
			if !currentLinkIDs[link.LinkID] {
				delete(s.links, link.LinkID)
			}
		}
	}

	maps.Copy(s.resources, current.Resources)
	// Source instance state links are keyed by name,
	// target storage links are keyed by ID.
	for _, link := range current.Links {
		s.links[link.LinkID] = link
	}
}

func (s *State) indexInstanceName(previous, current *state.InstanceState) {
	if previous != nil && previous.InstanceName != current.InstanceName {
		delete(s.nameLookup, previous.InstanceName)
	}

	if current.InstanceName != "" {
		s.nameLookup[current.InstanceName] = current.InstanceID
	}
}
