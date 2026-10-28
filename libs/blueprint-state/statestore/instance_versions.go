package statestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// InstanceVersioner is implemented by an EntityLoader that can report the stored version of an
// instance object without reading it. With it, a State configured WithRevalidation checks
// cached instances against storage, so it notices changes made by other processes sharing
// the same storage.
type InstanceVersioner interface {
	// InstanceVersion returns the version of the stored instance object, and false if there
	// is none.
	InstanceVersion(ctx context.Context, instanceID string) (version string, found bool, err error)
}

// WithRevalidation makes the State check a cached instance against storage before using it,
// once maxAge has passed since it was last checked. A changed instance is reloaded and one
// that has gone is dropped. A zero maxAge checks on every use. Ages are measured with clock.
//
// It only has an effect with an EntityLoader that implements InstanceVersioner, and is meant
// for processes that share storage with others. A process alone on its storage can leave it
// out, since its own writes keep its cache current.
func WithRevalidation(maxAge time.Duration, clock commoncore.Clock) StateOption {
	return func(s *State) {
		s.revalidation = &revalidation{maxAge: maxAge, clock: clock}
	}
}

type revalidation struct {
	maxAge time.Duration
	clock  commoncore.Clock
}

type instanceVersion struct {
	value     string
	checkedAt time.Time
	// Set when a write was rejected because the stored instance changed, so the cached copy
	// is reloaded on next use however recently it was checked.
	stale bool
}

// Expects the write lock to be held.
func (s *State) recordInstanceVersion(instanceID string, version string) {
	s.instanceVersions[instanceID] = &instanceVersion{value: version, checkedAt: s.now()}
}

// Returns the version an instance write is conditional on, whether one is known at all, and
// whether the cached instance is stale, in which case no write may be based on it until it has
// been reloaded. Expects at least the read lock to be held.
func (s *State) cachedInstanceVersion(instanceID string) (value string, known bool, stale bool) {
	version, ok := s.instanceVersions[instanceID]
	if !ok {
		return "", false, false
	}
	return version.value, true, version.stale
}

// Expects the write lock to be held.
func (s *State) markInstanceStale(instanceID string) {
	if version, ok := s.instanceVersions[instanceID]; ok {
		version.stale = true
		return
	}
	s.instanceVersions[instanceID] = &instanceVersion{stale: true}
}

func (s *State) now() time.Time {
	if s.revalidation == nil {
		return time.Time{}
	}
	return s.revalidation.clock.Now()
}

// Expects at least the read lock to be held.
func (s *State) needsRevalidation(instanceID string) bool {
	version, ok := s.instanceVersions[instanceID]
	if ok && version.stale {
		return true
	}

	if s.revalidation == nil || s.versioner() == nil {
		return false
	}
	return !ok || s.revalidation.clock.Now().Sub(version.checkedAt) >= s.revalidation.maxAge
}

func (s *State) versioner() InstanceVersioner {
	versioner, _ := s.loader.(InstanceVersioner)
	return versioner
}

// Checks a cached instance against storage, reloading it if it changed and dropping it if it
// has gone, and returns the instance to use.
func (s *State) revalidateInstance(
	ctx context.Context,
	instanceID string,
	visiting map[string]bool,
) (*state.InstanceState, bool, error) {
	s.mu.RLock()
	checked := s.instanceVersions[instanceID]
	s.mu.RUnlock()

	unchanged, err := s.isStoredVersion(ctx, instanceID, checked)
	if err != nil {
		return nil, false, err
	}
	if unchanged {
		s.mu.Lock()
		defer s.mu.Unlock()
		checked.checkedAt = s.now()
		inst, ok := s.instances[instanceID]
		return inst, ok, nil
	}

	persisted, found, err := s.loader.LoadInstance(ctx, instanceID)
	if err != nil {
		return nil, false, err
	}
	if !found || persisted.Removing {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.uncacheInstance(instanceID)
		return nil, false, nil
	}

	return s.refreshInstance(ctx, persisted, checked, visiting)
}

func (s *State) isStoredVersion(
	ctx context.Context,
	instanceID string,
	checked *instanceVersion,
) (bool, error) {
	versioner := s.versioner()
	if checked == nil || checked.stale || versioner == nil {
		return false, nil
	}

	stored, found, err := versioner.InstanceVersion(ctx, instanceID)
	if err != nil {
		return false, err
	}
	return found && stored == checked.value, nil
}

// Replaces a cached instance with what was reloaded from storage. A write this process made
// while the reload was in flight is newer than what the reload read, so the cached copy is
// kept when the version it was checked at has since been replaced.
func (s *State) refreshInstance(
	ctx context.Context,
	persisted *PersistedInstanceState,
	checked *instanceVersion,
	visiting map[string]bool,
) (*state.InstanceState, bool, error) {
	visiting[persisted.InstanceID] = true
	defer delete(visiting, persisted.InstanceID)

	children, err := s.lookupChildren(ctx, persisted, visiting)
	if err != nil {
		return nil, false, err
	}
	loaded := persisted.ToInstanceState()
	loaded.ChildBlueprints = children

	s.mu.Lock()
	defer s.mu.Unlock()
	cached, isCached := s.instances[persisted.InstanceID]
	if isCached && s.instanceVersions[persisted.InstanceID] != checked {
		return cached, true, nil
	}

	if isCached {
		s.forgetDriftOf(cached)
	}
	refreshed := s.cacheInstance(loaded)
	s.forgetDriftOf(refreshed)
	s.recordLoadedVersion(persisted)
	return refreshed, true, nil
}

// Expects the write lock to be held.
func (s *State) recordLoadedVersion(persisted *PersistedInstanceState) {
	if persisted.ETag != "" {
		s.recordInstanceVersion(persisted.InstanceID, persisted.ETag)
	}
}

// Drops an instance that another process removed, along with the resources, links and drift
// cached for it. A parent that still lists it as a child keeps the child until the parent is
// itself reloaded, which it is once the parent's changed object is noticed. Expects the write
// lock to be held.
func (s *State) uncacheInstance(instanceID string) {
	delete(s.instanceVersions, instanceID)
	cached, ok := s.instances[instanceID]
	if !ok {
		return
	}

	s.forgetDriftOf(cached)
	s.indexInstanceEntities(cached, &state.InstanceState{})
	if s.nameLookup[cached.InstanceName] == instanceID {
		delete(s.nameLookup, cached.InstanceName)
	}
	delete(s.instances, instanceID)
}

// Drift is saved and removed together with a rewrite of the instance that holds the resource
// or link, so an instance that changed in storage may also have changed drift. Expects the
// write lock to be held.
func (s *State) forgetDriftOf(instance *state.InstanceState) {
	for resourceID := range instance.Resources {
		delete(s.resourceDrift, resourceID)
	}
	for _, link := range instance.Links {
		delete(s.linkDrift, link.LinkID)
	}
}

// Bounds how often a mutator applies its change again after its write was rejected because
// another process changed the instance. A conflict that outlasts this is reported as
// state.ErrInstanceWriteConflict.
const maxInstanceWriteAttempts = 3

// Runs a mutator's attempt, which preloads, changes the cached instance and persists it, again
// while its write is rejected because another process changed the instance. The rejected write
// marks the instance as stale, so the next attempt's preload reloads it and the change is applied
// to what is now stored.
func retryOnInstanceChange(attempt func() error) error {
	_, err := retryOnInstanceChangeWithResult(func() (struct{}, error) {
		return struct{}{}, attempt()
	})
	return err
}

// As retryOnInstanceChange, for mutators that return what they changed.
func retryOnInstanceChangeWithResult[Result any](attempt func() (Result, error)) (Result, error) {
	var changed *instanceChangedError
	for range maxInstanceWriteAttempts {
		result, err := attempt()
		if !errors.As(err, &changed) {
			return result, err
		}
	}

	var none Result
	return none, state.InstanceWriteConflictError(changed.instanceID, maxInstanceWriteAttempts)
}

// instanceChangedError is returned by the Persister when a conditional write of an instance
// object is rejected because the stored instance changed since it was cached. Mutators apply
// their change again to the reloaded instance (see retryOnInstanceChange).
type instanceChangedError struct {
	instanceID string
}

func (e *instanceChangedError) Error() string {
	return fmt.Sprintf("statestore: instance %q changed in storage since it was cached", e.instanceID)
}

func (e *instanceChangedError) Unwrap() error {
	return ErrPreconditionFailed
}
