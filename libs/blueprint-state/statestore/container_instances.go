package statestore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// ClaimFunc is the backend-specific compare-and-swap used by
// InstancesContainer.ClaimForDeployment. Memfile supplies an in-memory
// mutex-based CAS; objectstore supplies an ETag-based CAS via its Service.
// Implementations must either apply the transition (status change + version
// bump) and return the new version, or return state.ErrVersionConflict with
// the current persisted version.
type ClaimFunc func(
	ctx context.Context,
	instanceID string,
	expectedVersion int64,
	newStatus core.InstanceStatus,
) (int64, error)

// InitialiseAndClaimFunc is the backend-specific atomic-create-at-version-1
// primitive used by InstancesContainer.InitialiseAndClaim. Memfile supplies
// an in-memory check-and-insert under the state's write lock; objectstore
// supplies a conditional Put with IfNoneMatch: "*".
// Implementations must either insert the instance at version 1 with the
// given status and return 1, or return state.ErrInstanceAlreadyExists
// without mutating existing state.
type InitialiseAndClaimFunc func(
	ctx context.Context,
	instanceState state.InstanceState,
	newStatus core.InstanceStatus,
) (int64, error)

// InstancesContainer implements state.InstancesContainer against a shared
// statestore.State and Persister. Concurrency-critical operations
// (ClaimForDeployment, InitialiseAndClaim) delegate to backend-supplied
// funcs; everything else is backend-agnostic.
type InstancesContainer struct {
	state              *State
	persister          *Persister
	claim              ClaimFunc
	initialiseAndClaim InitialiseAndClaimFunc
	logger             core.Logger
}

func NewInstancesContainer(
	st *State,
	persister *Persister,
	claim ClaimFunc,
	initialiseAndClaim InitialiseAndClaimFunc,
	logger core.Logger,
) *InstancesContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &InstancesContainer{
		state:              st,
		persister:          persister,
		claim:              claim,
		initialiseAndClaim: initialiseAndClaim,
		logger:             logger,
	}
}

func (c *InstancesContainer) Get(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}
	if !ok {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}
	// Lookup* methods return shared pointers, so the copy must happen under
	// the state read lock to avoid racing mutators that update the instance
	// and its nested maps in place while holding the write lock.
	c.state.RLock()
	defer c.state.RUnlock()
	return copyInstance(inst, instanceID), nil
}

func (c *InstancesContainer) LookupIDByName(
	ctx context.Context,
	instanceName string,
) (string, error) {
	id, ok, err := c.state.LookupInstanceIDByName(ctx, instanceName)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", state.InstanceNotFoundError(instanceName)
	}
	// A name can resolve to an instance that no longer exists, such as one whose
	// removal was interrupted, so check that the instance is still there.
	if _, exists, err := c.state.LookupInstance(ctx, id); err != nil {
		return "", err
	} else if !exists {
		c.logger.Warn(
			"stale instance ID lookup entry found, instance does not exist",
			core.StringLogField("instanceName", instanceName),
			core.StringLogField("instanceId", id),
		)
		return "", state.InstanceNotFoundError(instanceName)
	}

	return id, nil
}

func (c *InstancesContainer) List(
	ctx context.Context,
	params state.ListInstancesParams,
) (state.ListInstancesResult, error) {
	if err := c.state.preloadAllInstances(ctx); err != nil {
		return state.ListInstancesResult{}, err
	}

	c.state.RLock()
	defer c.state.RUnlock()

	filtered := filterInstances(c.state.instances, params.Search)
	totalCount := len(filtered)
	filtered = applyInstancePagination(filtered, params.Offset, params.Limit)
	return state.ListInstancesResult{
		Instances:  filtered,
		TotalCount: totalCount,
	}, nil
}

func (c *InstancesContainer) Save(
	ctx context.Context,
	instanceState state.InstanceState,
) error {
	return retryOnInstanceChange(func() error {
		return c.saveTree(ctx, instanceState)
	})
}

func (c *InstancesContainer) saveTree(
	ctx context.Context,
	instanceState state.InstanceState,
) error {
	if err := c.state.preloadInstances(
		ctx, instanceTreeIDs(&instanceState)...,
	); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	return c.save(ctx, instanceState)
}

func (c *InstancesContainer) SaveBatch(
	ctx context.Context,
	instances []state.InstanceState,
) error {
	return retryOnInstanceChange(func() error {
		return c.saveBatch(ctx, instances)
	})
}

func (c *InstancesContainer) saveBatch(
	ctx context.Context,
	instances []state.InstanceState,
) error {
	for i := range instances {
		if err := c.state.preloadInstances(
			ctx, instanceTreeIDs(&instances[i])...,
		); err != nil {
			return err
		}
	}

	c.state.Lock()
	defer c.state.Unlock()

	for i := range instances {
		if err := c.save(ctx, instances[i]); err != nil {
			return fmt.Errorf(
				"failed to save instance %d/%d (%s): %w",
				i+1, len(instances), instances[i].InstanceID, err,
			)
		}
	}
	return nil
}

func (c *InstancesContainer) GetBatch(
	ctx context.Context,
	instanceIDsOrNames []string,
) ([]state.InstanceState, error) {
	for _, idOrName := range instanceIDsOrNames {
		if err := c.preloadByIDOrName(ctx, idOrName); err != nil {
			return nil, err
		}
	}

	c.state.RLock()
	defer c.state.RUnlock()

	result := make([]state.InstanceState, 0, len(instanceIDsOrNames))
	var missing []string
	for _, idOrName := range instanceIDsOrNames {
		inst := c.findByIDOrName(idOrName)
		if inst == nil {
			missing = append(missing, idOrName)
			continue
		}
		result = append(result, copyInstance(inst, inst.InstanceID))
	}
	if len(missing) > 0 {
		return nil, state.NewInstancesNotFoundError(missing)
	}
	return result, nil
}

func (c *InstancesContainer) preloadByIDOrName(ctx context.Context, idOrName string) error {
	_, found, err := c.state.LookupInstance(ctx, idOrName)
	if err != nil || found {
		return err
	}

	id, found, err := c.state.LookupInstanceIDByName(ctx, idOrName)
	if err != nil || !found {
		return err
	}

	return c.state.preloadInstances(ctx, id)
}

// findByIDOrName expects the read lock to be held by the caller.
func (c *InstancesContainer) findByIDOrName(idOrName string) *state.InstanceState {
	if inst, ok := c.state.instances[idOrName]; ok {
		return inst
	}
	if id, ok := c.state.nameLookup[idOrName]; ok {
		if inst, ok := c.state.instances[id]; ok {
			return inst
		}
	}
	return nil
}

// save expects the state write lock to be held by the caller.
func (c *InstancesContainer) save(
	ctx context.Context,
	instanceState state.InstanceState,
) error {
	logger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceState.InstanceID),
	)

	existing, alreadyExists := c.state.instances[instanceState.InstanceID]
	if alreadyExists && existing.InstanceName != instanceState.InstanceName {
		return state.InstanceNameImmutableError(
			instanceState.InstanceID, existing.InstanceName, instanceState.InstanceName,
		)
	}

	// Children are persisted before the parent that references them, on update as well as
	// create, so a persisted parent never references a child object that was not written.
	logger.Debug("saving child blueprints for instance")
	inst := instanceState
	children, err := c.saveChildBlueprints(ctx, &inst)
	if err != nil {
		return err
	}
	inst.ChildBlueprints = children

	if alreadyExists {
		previousEntities := entityIDsOf(existing)
		cached := c.state.cacheInstance(&inst)
		added, removed := diffEntityIDs(previousEntities, entityIDsOf(cached))
		logger.Debug("persisting instance update")
		return c.persister.UpdateInstanceEntities(ctx, cached, added, removed)
	}

	cached := c.state.cacheInstance(&inst)

	logger.Debug("persisting new instance")
	return c.persister.CreateInstance(ctx, cached)
}

// Every instance in a saved tree is saved, and whether each is created or updated, and
// whether its name may stand, is decided from the cache, so all of them are loaded first.
func instanceTreeIDs(instance *state.InstanceState) []string {
	ids := []string{instance.InstanceID}
	for _, child := range instance.ChildBlueprints {
		ids = append(ids, instanceTreeIDs(child)...)
	}
	return ids
}

// Returns the parent's children as the cached instances, so the parent's ChildBlueprints map
// shares pointers with the cache.
func (c *InstancesContainer) saveChildBlueprints(
	ctx context.Context,
	instance *state.InstanceState,
) (map[string]*state.InstanceState, error) {
	children := make(map[string]*state.InstanceState, len(instance.ChildBlueprints))
	for childName, child := range instance.ChildBlueprints {
		if err := c.save(ctx, *child); err != nil {
			return nil, err
		}
		children[childName] = c.state.instances[child.InstanceID]
	}
	return children, nil
}

func (c *InstancesContainer) UpdateStatus(
	ctx context.Context,
	instanceID string,
	statusInfo state.InstanceStatusInfo,
) error {
	return retryOnInstanceChange(func() error {
		return c.updateStatus(ctx, instanceID, statusInfo)
	})
}

func (c *InstancesContainer) updateStatus(
	ctx context.Context,
	instanceID string,
	statusInfo state.InstanceStatusInfo,
) error {
	if err := c.state.preloadInstances(ctx, instanceID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceNotFoundError(instanceID)
	}
	inst.Status = statusInfo.Status
	if statusInfo.LastDeployedTimestamp != nil {
		inst.LastDeployedTimestamp = *statusInfo.LastDeployedTimestamp
	}
	if statusInfo.LastDeployAttemptTimestamp != nil {
		inst.LastDeployAttemptTimestamp = *statusInfo.LastDeployAttemptTimestamp
	}
	if statusInfo.LastStatusUpdateTimestamp != nil {
		inst.LastStatusUpdateTimestamp = *statusInfo.LastStatusUpdateTimestamp
	}
	if statusInfo.Durations != nil {
		inst.Durations = statusInfo.Durations
	}
	c.logger.Debug(
		"persisting instance status update",
		core.StringLogField("instanceId", instanceID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

// ClaimForDeployment delegates to the backend-supplied ClaimFunc. Memfile's
// ClaimFunc performs an in-memory mutex-based CAS; objectstore's performs
// an ETag-based CAS via its Service.
func (c *InstancesContainer) ClaimForDeployment(
	ctx context.Context,
	instanceID string,
	expectedVersion int64,
	newStatus core.InstanceStatus,
) (int64, error) {
	return c.claim(ctx, instanceID, expectedVersion, newStatus)
}

// InitialiseAndClaim delegates to the backend-supplied InitialiseAndClaimFunc.
// Memfile performs an in-memory check-and-insert under the state's write lock;
// objectstore performs a conditional Put with IfNoneMatch: "*".
func (c *InstancesContainer) InitialiseAndClaim(
	ctx context.Context,
	instanceState state.InstanceState,
	newStatus core.InstanceStatus,
) (int64, error) {
	return c.initialiseAndClaim(ctx, instanceState, newStatus)
}

func (c *InstancesContainer) Remove(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	return retryOnInstanceChangeWithResult(func() (state.InstanceState, error) {
		return c.remove(ctx, instanceID)
	})
}

func (c *InstancesContainer) remove(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	_, found, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}
	if !found {
		return c.resumeRemoval(ctx, instanceID)
	}

	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[instanceID]
	if !ok {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}
	delete(c.state.instances, instanceID)
	if inst.InstanceName != "" {
		delete(c.state.nameLookup, inst.InstanceName)
	}
	c.state.forgetDriftOf(inst)
	c.cleanupResources(inst.ResourceIDs)
	c.cleanupLinks(inst.InstanceID)

	c.logger.Debug(
		"persisting removal of blueprint instance",
		core.StringLogField("instanceId", instanceID),
	)
	return *inst, c.persister.RemoveInstance(ctx, inst)
}

// An instance whose removal was interrupted reads as not found, so removing it again is
// what finishes the removal.
func (c *InstancesContainer) resumeRemoval(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	persisted, found, err := c.state.loadInstanceBeingRemoved(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}
	if !found {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}

	c.state.Lock()
	defer c.state.Unlock()

	// The removal is resumed from the marked object just read, so its deletes are conditional
	// on that version, as they would have been had the removal not been interrupted.
	c.state.recordLoadedVersion(persisted)
	beingRemoved := persisted.ToInstanceState()
	c.logger.Debug(
		"resuming interrupted removal of blueprint instance",
		core.StringLogField("instanceId", instanceID),
	)
	return *beingRemoved, c.persister.RemoveInstance(ctx, beingRemoved)
}

func (c *InstancesContainer) cleanupResources(resourceIDs map[string]string) {
	for _, resourceID := range resourceIDs {
		delete(c.state.resources, resourceID)
	}
}

func (c *InstancesContainer) cleanupLinks(instanceID string) {
	for linkID, link := range c.state.links {
		if link.InstanceID == instanceID {
			delete(c.state.links, linkID)
		}
	}
}

// SingleProcessClaimFunc returns a ClaimFunc suitable for backends that
// run in a single deploy-engine process: an in-memory compare-and-swap
// under the state's write lock. Appropriate for memfile-style backends
// that own the state exclusively. Multi-writer backends (e.g. shared
// object stores) must supply their own ClaimFunc backed by a
// distributed CAS primitive — the in-memory mutex offers no protection
// against concurrent writers in other processes.
func SingleProcessClaimFunc(st *State, persister *Persister) ClaimFunc {
	return func(
		ctx context.Context,
		instanceID string,
		expectedVersion int64,
		newStatus core.InstanceStatus,
	) (int64, error) {
		st.Lock()
		defer st.Unlock()

		inst, ok := st.instances[instanceID]
		if !ok {
			return 0, state.InstanceNotFoundError(instanceID)
		}
		if inst.Version != expectedVersion {
			return inst.Version, state.ErrVersionConflict
		}
		inst.Status = newStatus
		inst.Version++
		inst.LastStatusUpdateTimestamp = int(time.Now().Unix())
		if err := persister.UpdateInstance(ctx, inst); err != nil {
			return 0, err
		}
		return inst.Version, nil
	}
}

// SingleProcessInitialiseAndClaimFunc returns an InitialiseAndClaimFunc
// suitable for backends that run in a single deploy-engine process: an
// in-memory check-and-insert under the state's write lock. Appropriate
// for memfile-style backends that own the state exclusively.
// Multi-writer backends (e.g. shared object stores) must supply their
// own InitialiseAndClaimFunc backed by an atomic create-if-absent
// primitive (e.g. IfNoneMatch:"*") — the in-memory mutex offers no
// protection against concurrent writers in other processes.
func SingleProcessInitialiseAndClaimFunc(st *State, persister *Persister) InitialiseAndClaimFunc {
	return func(
		ctx context.Context,
		instanceState state.InstanceState,
		newStatus core.InstanceStatus,
	) (int64, error) {
		st.Lock()
		defer st.Unlock()

		if _, ok := st.instances[instanceState.InstanceID]; ok {
			return 0, state.ErrInstanceAlreadyExists
		}
		now := int(time.Now().Unix())
		instanceState.Status = newStatus
		instanceState.Version = 1
		instanceState.LastStatusUpdateTimestamp = now
		inst := &instanceState
		st.instances[inst.InstanceID] = inst
		if inst.InstanceName != "" {
			st.nameLookup[inst.InstanceName] = inst.InstanceID
		}
		if err := persister.CreateInstance(ctx, inst); err != nil {
			delete(st.instances, inst.InstanceID)
			if inst.InstanceName != "" {
				delete(st.nameLookup, inst.InstanceName)
			}
			return 0, err
		}
		return inst.Version, nil
	}
}

func filterInstances(
	instances map[string]*state.InstanceState,
	search string,
) []state.InstanceSummary {
	var filtered []state.InstanceSummary
	searchLower := strings.ToLower(search)
	for _, inst := range instances {
		if search != "" && !strings.Contains(strings.ToLower(inst.InstanceName), searchLower) {
			continue
		}
		filtered = append(filtered, state.InstanceSummary{
			InstanceID:            inst.InstanceID,
			InstanceName:          inst.InstanceName,
			Status:                inst.Status,
			LastDeployedTimestamp: int64(inst.LastDeployedTimestamp),
		})
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].InstanceName < filtered[j].InstanceName
	})
	return filtered
}

func applyInstancePagination(
	items []state.InstanceSummary,
	offset, limit int,
) []state.InstanceSummary {
	if offset > 0 {
		if offset >= len(items) {
			return nil
		}
		items = items[offset:]
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
