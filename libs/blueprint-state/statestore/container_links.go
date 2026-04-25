package statestore

import (
	"context"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinksContainer implements state.LinksContainer against a shared
// statestore.State and Persister. Maintains a container-local
// resourceDataMappingIDs cache (instance → resourceName → []linkID) so
// ListWithResourceDataMappings avoids re-scanning links on every call.
type LinksContainer struct {
	state     *State
	persister *Persister
	// resourceDataMappingIDs maps instanceID → resourceName → []linkID.
	// Built lazily on first use and maintained on Save / Remove.
	resourceDataMappingIDs map[string]map[string][]string
	logger                 core.Logger
}

func NewLinksContainer(st *State, persister *Persister, logger core.Logger) *LinksContainer {
	if logger == nil {
		logger = core.NewNopLogger()
	}
	return &LinksContainer{
		state:                  st,
		persister:              persister,
		resourceDataMappingIDs: map[string]map[string][]string{},
		logger:                 logger,
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
	c.state.Lock()
	defer c.state.Unlock()

	if _, ok := c.state.instances[instanceID]; !ok {
		return nil, state.InstanceNotFoundError(instanceID)
	}
	mappings := c.deriveInstanceResourceDataMappings(instanceID)
	linkIDs, ok := mappings[resourceName]
	if !ok {
		return []state.LinkState{}, nil
	}
	links := make([]state.LinkState, 0, len(linkIDs))
	for _, linkID := range linkIDs {
		if link, ok := c.state.links[linkID]; ok {
			links = append(links, copyLink(link))
		}
	}
	return links, nil
}

func (c *LinksContainer) deriveInstanceResourceDataMappings(instanceID string) map[string][]string {
	if mappings, ok := c.resourceDataMappingIDs[instanceID]; ok {
		return mappings
	}

	return c.buildResourceDataMappings(instanceID)
}

func (c *LinksContainer) buildResourceDataMappings(instanceID string) map[string][]string {
	mappings := map[string][]string{}
	for linkID, link := range c.state.links {
		if link.InstanceID != instanceID {
			continue
		}
		added := map[string]bool{}
		for resourceNameFieldPath := range link.ResourceDataMappings {
			// resourceNameFieldPath is "resourceName::fieldPath".
			parts := strings.SplitN(resourceNameFieldPath, "::", 2)
			if len(parts) != 2 {
				continue
			}
			resourceName := parts[0]
			if added[resourceName] {
				continue
			}
			added[resourceName] = true
			mappings[resourceName] = append(mappings[resourceName], linkID)
		}
	}

	c.resourceDataMappingIDs[instanceID] = mappings

	return mappings
}

func (c *LinksContainer) Save(
	ctx context.Context,
	linkState state.LinkState,
) error {
	c.state.Lock()
	defer c.state.Unlock()

	inst, ok := c.state.instances[linkState.InstanceID]
	if !ok {
		return state.InstanceNotFoundError(linkState.InstanceID)
	}
	if inst.Links == nil {
		inst.Links = map[string]*state.LinkState{}
	}
	inst.Links[linkState.Name] = &linkState
	c.state.links[linkState.LinkID] = &linkState
	c.buildResourceDataMappings(linkState.InstanceID)

	c.logger.Debug(
		"persisting updated or newly created link",
		core.StringLogField("linkId", linkState.LinkID),
		core.StringLogField("instanceId", linkState.InstanceID),
		core.StringLogField("linkName", linkState.Name),
	)
	return c.persister.UpdateInstance(ctx, inst)
}

func (c *LinksContainer) UpdateStatus(
	ctx context.Context,
	linkID string,
	statusInfo state.LinkStatusInfo,
) error {
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
	c.buildResourceDataMappings(link.InstanceID)

	c.logger.Debug(
		"persisting link removal",
		core.StringLogField("linkId", linkID),
	)
	return *link, c.persister.UpdateInstance(ctx, inst)
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

	return copyLinkDrift(drift), nil
}

func (c *LinksContainer) SaveDrift(
	ctx context.Context,
	driftState state.LinkDriftState,
) error {
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
	c.state.Lock()
	defer c.state.Unlock()

	link, ok := c.state.links[linkID]
	if !ok {
		return state.LinkDriftState{}, state.LinkNotFoundError(linkID)
	}
	driftState, hasDrift := c.state.linkDrift[linkID]
	if !hasDrift {
		return state.LinkDriftState{}, nil
	}
	inst, ok := c.state.instances[link.InstanceID]
	if !ok {
		return state.LinkDriftState{}, errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, driftState.LinkID),
		)
	}
	link.Drifted = false
	link.LastDriftDetectedTimestamp = nil
	delete(c.state.linkDrift, linkID)

	c.logger.Debug(
		"persisting removal of link drift entry",
		core.StringLogField("linkId", linkID),
	)
	if err := c.persister.RemoveLinkDrift(ctx, driftState); err != nil {
		return state.LinkDriftState{}, err
	}
	c.logger.Debug(
		"persisting link changes for removal of drift state",
		core.StringLogField("linkId", linkID),
	)
	if err := c.persister.UpdateInstance(ctx, inst); err != nil {
		return state.LinkDriftState{}, err
	}
	return *driftState, nil
}

func instanceNotFoundForLinkMessage(instanceID, linkID string) string {
	return fmt.Sprintf("instance %s not found for link %s", instanceID, linkID)
}
