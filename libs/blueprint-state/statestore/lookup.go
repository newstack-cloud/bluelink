package statestore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LookupInstance returns the instance for id. Under ModeEager this is a
// pure cache read (loader is the noop). Under ModeLazy, a miss triggers
// loader.LoadInstance and caches the instance with its child blueprints,
// resources and links before returning (see CachePersistedInstance).
//
// Must not be called while holding the State lock.
func (s *State) LookupInstance(ctx context.Context, id string) (*state.InstanceState, bool, error) {
	return s.lookupInstance(ctx, id, map[string]bool{})
}

// LookupInstanceIDByName resolves a name to an instance ID from the name cache,
// which is kept in sync wherever an instance is cached or removed. Under ModeLazy
// a miss reads the instance's name record through the loader.
func (s *State) LookupInstanceIDByName(ctx context.Context, name string) (string, bool, error) {
	s.mu.RLock()
	cachedID, isCached := s.nameLookup[name]
	fresh := isCached && !s.needsRevalidation(cachedID)
	s.mu.RUnlock()
	if fresh {
		return cachedID, true, nil
	}

	// The instance a cached name resolves to may have been removed by another process, and
	// its name taken by a new instance, so it is checked before the name is trusted.
	if isCached {
		inst, found, err := s.LookupInstance(ctx, cachedID)
		if err != nil {
			return "", false, err
		}
		if found && inst.InstanceName == name {
			return cachedID, true, nil
		}
	}

	id, ok, err := s.loader.LoadInstanceIDByName(ctx, name)
	if err != nil || !ok {
		return "", ok, err
	}
	s.mu.Lock()
	s.nameLookup[name] = id
	s.mu.Unlock()
	return id, true, nil
}

// LookupResource returns a resource by ID, materialising it on cache miss.
//
// A cached resource is only as current as the instance that holds it, so that instance is
// revalidated (see WithRevalidation) before the resource is returned.
func (s *State) LookupResource(ctx context.Context, id string) (*state.ResourceState, bool, error) {
	s.mu.RLock()
	cached, isCached := s.resources[id]
	fresh := isCached && !s.needsRevalidation(cached.InstanceID)
	s.mu.RUnlock()
	if fresh {
		return cached, true, nil
	}

	instanceID, ok, err := s.resourceInstanceID(ctx, id, cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	// Loading the instance caches its resources. A record whose instance no longer holds
	// the resource is left behind by an interrupted write and reads as not found.
	if _, _, err := s.LookupInstance(ctx, instanceID); err != nil {
		return nil, false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.resources[id]
	if !ok || r.InstanceID != instanceID {
		return nil, false, nil
	}

	return r, true, nil
}

// Resource IDs never move between instances, so a cached resource names the instance that holds
// it without reading its record.
func (s *State) resourceInstanceID(
	ctx context.Context,
	id string,
	cached *state.ResourceState,
) (string, bool, error) {
	if cached != nil {
		return cached.InstanceID, true, nil
	}
	return s.loader.LoadResourceInstanceID(ctx, id)
}

// LookupResourceDrift returns a resource-drift entry by resource ID.
//
// Drift is saved together with a rewrite of the instance that holds the resource, so the
// resource is looked up first, which drops cached drift when that instance changed in storage.
func (s *State) LookupResourceDrift(ctx context.Context, id string) (*state.ResourceDriftState, bool, error) {
	if _, _, err := s.LookupResource(ctx, id); err != nil {
		return nil, false, err
	}

	s.mu.RLock()
	if d, ok := s.resourceDrift[id]; ok {
		s.mu.RUnlock()
		return d, true, nil
	}
	s.mu.RUnlock()

	d, ok, err := s.loader.LoadResourceDrift(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.resourceDrift[id] = d
	s.mu.Unlock()
	return d, true, nil
}

// LookupLink returns a link by ID. As for resources, a cached link's instance is revalidated
// before the link is returned.
func (s *State) LookupLink(ctx context.Context, id string) (*state.LinkState, bool, error) {
	s.mu.RLock()
	cached, isCached := s.links[id]
	fresh := isCached && !s.needsRevalidation(cached.InstanceID)
	s.mu.RUnlock()
	if fresh {
		return cached, true, nil
	}

	instanceID, ok, err := s.linkInstanceID(ctx, id, cached)
	if err != nil || !ok {
		return nil, ok, err
	}
	// As for resources, the record is only a hint and the loaded instance decides.
	if _, _, err := s.LookupInstance(ctx, instanceID); err != nil {
		return nil, false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.links[id]
	if !ok || l.InstanceID != instanceID {
		return nil, false, nil
	}

	return l, true, nil
}

func (s *State) linkInstanceID(ctx context.Context, id string, cached *state.LinkState) (string, bool, error) {
	if cached != nil {
		return cached.InstanceID, true, nil
	}
	return s.loader.LoadLinkInstanceID(ctx, id)
}

// LookupLinkDrift returns a link-drift entry by link ID. As for resources, the link is looked
// up first so that drift cached for a changed instance is dropped.
func (s *State) LookupLinkDrift(ctx context.Context, id string) (*state.LinkDriftState, bool, error) {
	if _, _, err := s.LookupLink(ctx, id); err != nil {
		return nil, false, err
	}

	s.mu.RLock()
	if d, ok := s.linkDrift[id]; ok {
		s.mu.RUnlock()
		return d, true, nil
	}
	s.mu.RUnlock()

	d, ok, err := s.loader.LoadLinkDrift(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.linkDrift[id] = d
	s.mu.Unlock()
	return d, true, nil
}

// LookupEvent returns an event by ID.
func (s *State) LookupEvent(ctx context.Context, id string) (*manage.Event, bool, error) {
	s.mu.RLock()
	if e, ok := s.events[id]; ok {
		s.mu.RUnlock()
		return e, true, nil
	}
	s.mu.RUnlock()

	e, ok, err := s.loader.LoadEvent(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.events[id] = e
	s.mu.Unlock()
	return e, true, nil
}

// LookupChangeset returns a changeset by ID.
func (s *State) LookupChangeset(ctx context.Context, id string) (*manage.Changeset, bool, error) {
	s.mu.RLock()
	if c, ok := s.changesets[id]; ok {
		s.mu.RUnlock()
		return c, true, nil
	}
	s.mu.RUnlock()

	c, ok, err := s.loader.LoadChangeset(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.changesets[id] = c
	s.mu.Unlock()
	return c, true, nil
}

// LookupValidation returns a blueprint validation by ID.
func (s *State) LookupValidation(ctx context.Context, id string) (*manage.BlueprintValidation, bool, error) {
	s.mu.RLock()
	if v, ok := s.validations[id]; ok {
		s.mu.RUnlock()
		return v, true, nil
	}
	s.mu.RUnlock()

	v, ok, err := s.loader.LoadValidation(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.validations[id] = v
	s.mu.Unlock()
	return v, true, nil
}

// LookupAuditRecord returns an audit record by ID.
func (s *State) LookupAuditRecord(ctx context.Context, id string) (*manage.AuditRecord, bool, error) {
	s.mu.RLock()
	if r, ok := s.auditRecords[id]; ok {
		s.mu.RUnlock()
		return r, true, nil
	}
	s.mu.RUnlock()

	r, ok, err := s.loader.LoadAuditRecord(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.auditRecords[id] = r
	s.mu.Unlock()
	return r, true, nil
}

// LookupReconciliation returns a reconciliation result by ID.
func (s *State) LookupReconciliation(ctx context.Context, id string) (*manage.ReconciliationResult, bool, error) {
	s.mu.RLock()
	if r, ok := s.reconciliations[id]; ok {
		s.mu.RUnlock()
		return r, true, nil
	}
	s.mu.RUnlock()

	r, ok, err := s.loader.LoadReconciliation(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.reconciliations[id] = r
	s.mu.Unlock()
	return r, true, nil
}

// LookupCleanupOperation returns a cleanup operation by ID.
func (s *State) LookupCleanupOperation(ctx context.Context, id string) (*manage.CleanupOperation, bool, error) {
	s.mu.RLock()
	if op, ok := s.cleanupOps[id]; ok {
		s.mu.RUnlock()
		return op, true, nil
	}
	s.mu.RUnlock()

	op, ok, err := s.loader.LoadCleanupOperation(ctx, id)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.mu.Lock()
	s.cleanupOps[id] = op
	s.mu.Unlock()
	return op, true, nil
}
