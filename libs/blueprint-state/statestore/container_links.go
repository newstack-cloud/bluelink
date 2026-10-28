package statestore

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinksContainer implements state.LinksContainer against a shared
// statestore.State and Persister.
type LinksContainer struct {
	state     *State
	persister *Persister
	logger    core.Logger
}

func NewLinksContainer(st *State, persister *Persister, logger core.Logger) *LinksContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &LinksContainer{
		state:     st,
		persister: persister,
		logger:    logger,
	}
}

func (c *LinksContainer) Get(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	l, ok, err := c.state.LookupLink(ctx, linkID)
	if err != nil {
		return state.LinkState{}, err
	}
	if !ok {
		return state.LinkState{}, state.LinkNotFoundError(linkID)
	}
	// Lookup* methods return shared pointers, so the copy must happen under
	// the state read lock to avoid racing mutators that update entities
	// in place while holding the write lock.
	c.state.RLock()
	defer c.state.RUnlock()
	return copyLink(l), nil
}

func (c *LinksContainer) GetByName(
	ctx context.Context,
	instanceID string,
	linkName string,
) (state.LinkState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return state.LinkState{}, err
	}
	if ok {
		c.state.RLock()
		defer c.state.RUnlock()
		if linkState, ok := inst.Links[linkName]; ok {
			return copyLink(linkState), nil
		}
	}
	return state.LinkState{}, state.LinkNotFoundError(
		idutils.LinkInBlueprintID(instanceID, linkName),
	)
}

func (c *LinksContainer) ListWithResourceDataMappings(
	ctx context.Context,
	instanceID string,
	resourceName string,
) ([]state.LinkState, error) {
	inst, ok, err := c.state.LookupInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, state.InstanceNotFoundError(instanceID)
	}

	// Derived from the instance's own links on each call rather than cached, because the
	// links can change through other containers (a whole-instance save, a reload).
	c.state.RLock()
	defer c.state.RUnlock()
	links := []state.LinkState{}
	for _, link := range inst.Links {
		if mapsResourceData(link, resourceName) {
			links = append(links, copyLink(link))
		}
	}
	slices.SortFunc(links, func(a, b state.LinkState) int {
		return strings.Compare(a.LinkID, b.LinkID)
	})

	return links, nil
}

// Mapping keys take the form "resourceName::fieldPath".
func mapsResourceData(link *state.LinkState, resourceName string) bool {
	for resourceNameFieldPath := range link.ResourceDataMappings {
		mappedResource, _, found := strings.Cut(resourceNameFieldPath, "::")
		if found && mappedResource == resourceName {
			return true
		}
	}
	return false
}

func (c *LinksContainer) Save(
	ctx context.Context,
	linkState state.LinkState,
) error {
	return retryOnInstanceChange(func() error {
		return c.save(ctx, linkState)
	})
}

func (c *LinksContainer) save(
	ctx context.Context,
	linkState state.LinkState,
) error {
	if err := c.state.preloadInstances(ctx, linkState.InstanceID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[linkState.InstanceID]
	if !ok {
		return state.InstanceNotFoundError(linkState.InstanceID)
	}
	if inst.Links == nil {
		inst.Links = map[string]*state.LinkState{}
	}
	added, removed := linkEntityChange(inst.Links[linkState.Name], linkState.LinkID)
	for _, replacedLinkID := range removed.Links {
		delete(c.state.links, replacedLinkID)
	}

	inst.Links[linkState.Name] = &linkState
	c.state.links[linkState.LinkID] = &linkState

	c.logger.Debug(
		"persisting updated or newly created link",
		core.StringLogField("linkId", linkState.LinkID),
		core.StringLogField("instanceId", linkState.InstanceID),
		core.StringLogField("linkName", linkState.Name),
	)
	return c.persister.UpdateInstanceEntities(ctx, inst, added, removed)
}

// Links are held by name, so saving one can replace a link with a different ID under the
// same name.
func linkEntityChange(previous *state.LinkState, linkID string) (added, removed InstanceEntityIDs) {
	if previous != nil && previous.LinkID == linkID {
		return added, removed
	}

	added.Links = []string{linkID}
	if previous != nil {
		removed.Links = []string{previous.LinkID}
	}

	return added, removed
}

func (c *LinksContainer) UpdateStatus(
	ctx context.Context,
	linkID string,
	statusInfo state.LinkStatusInfo,
) error {
	return retryOnInstanceChange(func() error {
		return c.updateStatus(ctx, linkID, statusInfo)
	})
}

func (c *LinksContainer) updateStatus(
	ctx context.Context,
	linkID string,
	statusInfo state.LinkStatusInfo,
) error {
	if err := c.state.preloadLink(ctx, linkID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	link, ok := c.state.links[linkID]
	if !ok {
		return state.LinkNotFoundError(linkID)
	}
	inst, ok := c.state.instances[link.InstanceID]
	if !ok {
		return errMalformedState(instanceNotFoundForLinkMessage(link.InstanceID, linkID))
	}

	link.Status = statusInfo.Status
	link.PreciseStatus = statusInfo.PreciseStatus
	link.FailureReasons = statusInfo.FailureReasons
	if statusInfo.LastDeployAttemptTimestamp != nil {
		link.LastDeployAttemptTimestamp = *statusInfo.LastDeployAttemptTimestamp
	}
	if statusInfo.LastDeployedTimestamp != nil {
		link.LastDeployedTimestamp = *statusInfo.LastDeployedTimestamp
	}
	if statusInfo.LastStatusUpdateTimestamp != nil {
		link.LastStatusUpdateTimestamp = *statusInfo.LastStatusUpdateTimestamp
	}
	if statusInfo.Durations != nil {
		link.Durations = statusInfo.Durations
	}

	c.logger.Debug(
		"persisting link status update",
		core.StringLogField("linkId", linkID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *LinksContainer) Remove(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	return retryOnInstanceChangeWithResult(func() (state.LinkState, error) {
		return c.remove(ctx, linkID)
	})
}

func (c *LinksContainer) remove(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	if err := c.state.preloadLink(ctx, linkID); err != nil {
		return state.LinkState{}, err
	}

	c.state.Lock()
	defer c.state.Unlock()

	link, ok := c.state.links[linkID]
	if !ok {
		return state.LinkState{}, state.LinkNotFoundError(linkID)
	}
	inst, ok := c.state.instances[link.InstanceID]
	if !ok {
		return state.LinkState{}, errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, linkID),
		)
	}
	delete(inst.Links, link.Name)
	delete(c.state.links, linkID)

	c.logger.Debug(
		"persisting link removal",
		core.StringLogField("linkId", linkID),
	)

	removed := InstanceEntityIDs{Links: []string{linkID}}
	return *link, c.persister.UpdateInstanceEntities(
		ctx, inst, InstanceEntityIDs{}, removed,
	)
}

func (c *LinksContainer) GetDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	if _, ok, err := c.state.LookupLink(ctx, linkID); err != nil {
		return state.LinkDriftState{}, err
	} else if !ok {
		return state.LinkDriftState{}, state.LinkNotFoundError(linkID)
	}
	drift, ok, err := c.state.LookupLinkDrift(ctx, linkID)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	if !ok {
		return state.LinkDriftState{}, nil
	}

	c.state.RLock()
	defer c.state.RUnlock()
	return copyLinkDrift(drift), nil
}

func (c *LinksContainer) SaveDrift(
	ctx context.Context,
	driftState state.LinkDriftState,
) error {
	return retryOnInstanceChange(func() error {
		return c.saveDrift(ctx, driftState)
	})
}

func (c *LinksContainer) saveDrift(
	ctx context.Context,
	driftState state.LinkDriftState,
) error {
	if err := c.state.preloadLinkWithDrift(ctx, driftState.LinkID); err != nil {
		return err
	}

	c.state.Lock()
	defer c.state.Unlock()

	link, ok := c.state.links[driftState.LinkID]
	if !ok {
		return state.LinkNotFoundError(driftState.LinkID)
	}
	inst, ok := c.state.instances[link.InstanceID]
	if !ok {
		return errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, driftState.LinkID),
		)
	}
	link.Drifted = true
	link.LastDriftDetectedTimestamp = driftState.Timestamp

	_, alreadyExists := c.state.linkDrift[driftState.LinkID]
	c.state.linkDrift[driftState.LinkID] = &driftState

	c.logger.Debug(
		"persisting updated or newly created link drift entry",
		core.StringLogField("linkId", driftState.LinkID),
	)
	if err := c.persistLinkDrift(ctx, &driftState, alreadyExists); err != nil {
		return err
	}
	c.logger.Debug(
		"persisting link changes for latest drift state",
		core.StringLogField("linkId", driftState.LinkID),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *LinksContainer) persistLinkDrift(
	ctx context.Context,
	driftState *state.LinkDriftState,
	alreadyExists bool,
) error {
	if alreadyExists {
		return c.persister.UpdateLinkDrift(ctx, driftState)
	}
	return c.persister.CreateLinkDrift(ctx, driftState)
}

func (c *LinksContainer) RemoveDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	return retryOnInstanceChangeWithResult(func() (state.LinkDriftState, error) {
		return c.removeDrift(ctx, linkID)
	})
}

// As for resources, a retried attempt can find the drift entry already deleted and still has
// to clear the link's drifted flag.
func (c *LinksContainer) removeDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	if err := c.state.preloadLinkWithDrift(ctx, linkID); err != nil {
		return state.LinkDriftState{}, err
	}

	c.state.Lock()
	defer c.state.Unlock()

	link, ok := c.state.links[linkID]
	if !ok {
		return state.LinkDriftState{}, state.LinkNotFoundError(linkID)
	}
	driftState, hasDrift := c.state.linkDrift[linkID]
	if !hasDrift && !link.Drifted {
		return state.LinkDriftState{}, nil
	}
	inst, ok := c.state.instances[link.InstanceID]
	if !ok {
		return state.LinkDriftState{}, errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, linkID),
		)
	}
	link.Drifted = false
	link.LastDriftDetectedTimestamp = nil
	delete(c.state.linkDrift, linkID)

	if hasDrift {
		c.logger.Debug(
			"persisting removal of link drift entry",
			core.StringLogField("linkId", linkID),
		)
		if err := c.persister.RemoveLinkDrift(ctx, driftState); err != nil {
			return state.LinkDriftState{}, err
		}
	}
	c.logger.Debug(
		"persisting link changes for removal of drift state",
		core.StringLogField("linkId", linkID),
	)
	if err := c.persister.UpdateInstance(ctx, inst); err != nil {
		return state.LinkDriftState{}, err
	}
	if !hasDrift {
		return state.LinkDriftState{}, nil
	}
	return *driftState, nil
}

func instanceNotFoundForLinkMessage(instanceID, linkID string) string {
	return fmt.Sprintf("instance %s not found for link %s", instanceID, linkID)
}
