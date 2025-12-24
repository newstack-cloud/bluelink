package memfile

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type linksContainerImpl struct {
	links            map[string]*state.LinkState
	linkDriftEntries map[string]*state.LinkDriftState
	instances        map[string]*state.InstanceState
	// instance ID -> resourceName -> linkIDs
	resourceDataMappingIDs map[string]map[string][]string
	fs                     afero.Fs
	persister              *statePersister
	logger                 core.Logger
	mu                     *sync.RWMutex
}

func (c *linksContainerImpl) Get(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if linkState, ok := c.links[linkID]; ok {
		return copyLink(linkState), nil
	}

	return state.LinkState{}, state.LinkNotFoundError(linkID)
}

func (c *linksContainerImpl) GetByName(
	ctx context.Context,
	instanceID string,
	linkName string,
) (state.LinkState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		if linkState, ok := instance.Links[linkName]; ok {
			return copyLink(linkState), nil
		}
	}

	elementID := idutils.LinkInBlueprintID(instanceID, linkName)
	return state.LinkState{}, state.LinkNotFoundError(elementID)
}

func (c *linksContainerImpl) ListWithResourceDataMappings(
	ctx context.Context,
	instanceID string,
	resourceName string,
) ([]state.LinkState, error) {
	// Lock for reading and writing as this method builds the resource data mappings
	// on the fly if needed.
	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := c.instances[instanceID]; ok {
		if instance != nil {
			mappings := c.deriveInstanceResourceDataMappings(instanceID)
			if linkIDs, ok := mappings[resourceName]; ok {
				links := []state.LinkState{}
				for _, linkID := range linkIDs {
					if linkState, ok := c.links[linkID]; ok {
						links = append(links, copyLink(linkState))
					}
				}
				return links, nil
			} else {
				return []state.LinkState{}, nil
			}
		}
	}

	return nil, state.InstanceNotFoundError(instanceID)
}

func (c *linksContainerImpl) deriveInstanceResourceDataMappings(
	instanceID string,
) map[string][]string {
	if mappings, ok := c.resourceDataMappingIDs[instanceID]; ok {
		return mappings
	}

	return c.buildResourceDataMappings(instanceID)
}

// A write lock must be held when calling this method.
func (c *linksContainerImpl) buildResourceDataMappings(instanceID string) map[string][]string {
	instanceResourceDataMappings := map[string][]string{}
	for linkID, link := range c.links {
		if link.InstanceID == instanceID {
			for resourceNameFieldPath := range link.ResourceDataMappings {
				// The resourceNameFieldPath is of the form "resourceName::fieldPath"
				// where resourceName is the logical name of the resource in the blueprint instance.
				parts := strings.SplitN(resourceNameFieldPath, "::", 2)
				if len(parts) == 2 {
					resourceName := parts[0]
					if _, ok := instanceResourceDataMappings[resourceName]; !ok {
						instanceResourceDataMappings[resourceName] = []string{}
					}
					instanceResourceDataMappings[resourceName] = append(
						instanceResourceDataMappings[resourceName],
						linkID,
					)
					break
				}
			}
		}
	}
	c.resourceDataMappingIDs[instanceID] = instanceResourceDataMappings
	return instanceResourceDataMappings
}

func (c *linksContainerImpl) Save(
	ctx context.Context,
	linkState state.LinkState,
) error {
	linkLogger := c.logger.WithFields(
		core.StringLogField("linkId", linkState.LinkID),
		core.StringLogField("instanceId", linkState.InstanceID),
		core.StringLogField("linkName", linkState.Name),
	)
	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, linkState.InstanceID); ok {
		instance.Links[linkState.Name] = &linkState
		c.links[linkState.LinkID] = &linkState

		// Build the resource data mappings, it's fine to do this only when saving
		// and removing links directly as the blueprint container will use the link-specific
		// methods to update links in a blueprint instance.
		c.buildResourceDataMappings(linkState.InstanceID)

		linkLogger.Debug("persisting updated or newly created link")
		return c.persister.updateInstance(instance)
	}

	return state.InstanceNotFoundError(linkState.InstanceID)
}

func (c *linksContainerImpl) UpdateStatus(
	ctx context.Context,
	linkID string,
	statusInfo state.LinkStatusInfo,
) error {
	linkLogger := c.logger.WithFields(
		core.StringLogField("linkId", linkID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	link, hasLink := c.links[linkID]
	if !hasLink {
		return state.LinkNotFoundError(linkID)
	}

	instance, ok := c.instances[link.InstanceID]
	if !ok {
		// When a link exists but the instance does not,
		// then something has corrupted the state.
		return errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, linkID),
		)
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

	linkLogger.Debug("persisting link status update")
	return c.persister.updateInstance(instance)
}

func (c *linksContainerImpl) Remove(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	linkLogger := c.logger.WithFields(
		core.StringLogField("linkId", linkID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	link, ok := c.links[linkID]
	if !ok {
		return state.LinkState{}, state.LinkNotFoundError(linkID)
	}

	instance, hasInstance := c.instances[link.InstanceID]
	if !hasInstance {
		// When a link exists but the instance does not,
		// then something has corrupted the state.
		return state.LinkState{}, errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, linkID),
		)
	}

	delete(instance.Links, link.Name)
	delete(c.links, linkID)

	// Build the resource data mappings, it's fine to do this only when saving
	// and removing links directly as the blueprint container will use the link-specific
	// methods to update links in a blueprint instance.
	c.buildResourceDataMappings(link.InstanceID)

	linkLogger.Debug("persisting link removal")
	return *link, c.persister.updateInstance(instance)
}

func (c *linksContainerImpl) GetDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, hasLink := c.links[linkID]
	if !hasLink {
		return state.LinkDriftState{}, state.LinkNotFoundError(linkID)
	}

	drift, hasDrift := c.linkDriftEntries[linkID]
	if !hasDrift {
		// An empty drift state is valid for a link that has not drifted.
		return state.LinkDriftState{}, nil
	}

	return copyLinkDrift(drift), nil
}

func (c *linksContainerImpl) SaveDrift(
	ctx context.Context,
	driftState state.LinkDriftState,
) error {
	linkLogger := c.logger.WithFields(
		core.StringLogField("linkId", driftState.LinkID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	link, hasLink := c.links[driftState.LinkID]
	if !hasLink {
		return state.LinkNotFoundError(driftState.LinkID)
	}

	instance, ok := c.instances[link.InstanceID]
	if !ok {
		// When a link exists but the instance does not,
		// then something has corrupted the state.
		return errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, driftState.LinkID),
		)
	}

	link.Drifted = true
	link.LastDriftDetectedTimestamp = driftState.Timestamp

	_, alreadyExists := c.linkDriftEntries[driftState.LinkID]
	c.linkDriftEntries[driftState.LinkID] = &driftState

	linkLogger.Debug("persisting updated or newly created link drift entry")
	err := c.persistLinkDrift(&driftState, alreadyExists)
	if err != nil {
		return err
	}

	linkLogger.Debug("persisting link changes for latest drift state")
	// Ensure that the instance is updated to reflect the drift field
	// updates to the link.
	return c.persister.updateInstance(instance)
}

func (c *linksContainerImpl) persistLinkDrift(
	driftState *state.LinkDriftState,
	alreadyExists bool,
) error {
	if alreadyExists {
		return c.persister.updateLinkDrift(driftState)
	}

	return c.persister.createLinkDrift(driftState)
}

func (c *linksContainerImpl) RemoveDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	linkLogger := c.logger.WithFields(
		core.StringLogField("linkId", linkID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	link, hasLink := c.links[linkID]
	if !hasLink {
		return state.LinkDriftState{}, state.LinkNotFoundError(linkID)
	}

	driftState, hasDrift := c.linkDriftEntries[linkID]
	if !hasDrift {
		return state.LinkDriftState{}, nil
	}

	instance, ok := c.instances[link.InstanceID]
	if !ok {
		// When a link exists but the instance does not,
		// then something has corrupted the state.
		return state.LinkDriftState{}, errMalformedState(
			instanceNotFoundForLinkMessage(link.InstanceID, driftState.LinkID),
		)
	}

	link.Drifted = false
	link.LastDriftDetectedTimestamp = nil
	delete(c.linkDriftEntries, linkID)

	linkLogger.Debug("persisting removal of link drift entry")
	err := c.persister.removeLinkDrift(driftState)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	linkLogger.Debug("persisting link changes for removal of drift state")
	// Ensure that the instance is updated to reflect the drift field
	// updates to the link.
	err = c.persister.updateInstance(instance)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	return *driftState, nil
}

func copyLink(linkState *state.LinkState) state.LinkState {
	if linkState == nil {
		return state.LinkState{}
	}

	return state.LinkState{
		LinkID:                     linkState.LinkID,
		Name:                       linkState.Name,
		InstanceID:                 linkState.InstanceID,
		Status:                     linkState.Status,
		PreciseStatus:              linkState.PreciseStatus,
		LastDeployedTimestamp:      linkState.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: linkState.LastDeployAttemptTimestamp,
		LastStatusUpdateTimestamp:  linkState.LastStatusUpdateTimestamp,
		IntermediaryResourceStates: copyIntermediaryResources(
			linkState.IntermediaryResourceStates,
		),
		Data:                       linkState.Data,
		ResourceDataMappings:       linkState.ResourceDataMappings,
		FailureReasons:             linkState.FailureReasons,
		Drifted:                    linkState.Drifted,
		LastDriftDetectedTimestamp: linkState.LastDriftDetectedTimestamp,
		Durations:                  linkState.Durations,
	}
}

func copyIntermediaryResources(
	intermediaryResourceStates []*state.LinkIntermediaryResourceState,
) []*state.LinkIntermediaryResourceState {
	if intermediaryResourceStates == nil {
		return nil
	}

	intermediaryResourcesCopy := []*state.LinkIntermediaryResourceState{}
	for i, value := range intermediaryResourceStates {
		intermediaryResourcesCopy[i] = &state.LinkIntermediaryResourceState{
			ResourceID:                 value.ResourceID,
			ResourceType:               value.ResourceType,
			InstanceID:                 value.InstanceID,
			LastDeployedTimestamp:      value.LastDeployedTimestamp,
			LastDeployAttemptTimestamp: value.LastDeployAttemptTimestamp,
			ResourceSpecData:           value.ResourceSpecData,
			Status:                     value.Status,
			PreciseStatus:              value.PreciseStatus,
			FailureReasons:             value.FailureReasons,
		}
	}

	return intermediaryResourcesCopy
}

func instanceNotFoundForLinkMessage(
	instanceID string,
	linkID string,
) string {
	return fmt.Sprintf("instance %s not found for link %s", instanceID, linkID)
}

func copyLinkDrift(driftState *state.LinkDriftState) state.LinkDriftState {
	if driftState == nil {
		return state.LinkDriftState{}
	}

	timestampPtr := (*int)(nil)
	if driftState.Timestamp != nil {
		timestampValue := *driftState.Timestamp
		timestampPtr = &timestampValue
	}

	return state.LinkDriftState{
		LinkID:            driftState.LinkID,
		LinkName:          driftState.LinkName,
		ResourceADrift:    copyLinkResourceDrift(driftState.ResourceADrift),
		ResourceBDrift:    copyLinkResourceDrift(driftState.ResourceBDrift),
		IntermediaryDrift: copyIntermediaryDriftMap(driftState.IntermediaryDrift),
		Timestamp:         timestampPtr,
	}
}

func copyLinkResourceDrift(drift *state.LinkResourceDrift) *state.LinkResourceDrift {
	if drift == nil {
		return nil
	}

	fieldChanges := make([]*state.LinkDriftFieldChange, len(drift.MappedFieldChanges))
	for i, fc := range drift.MappedFieldChanges {
		fieldChanges[i] = &state.LinkDriftFieldChange{
			ResourceFieldPath: fc.ResourceFieldPath,
			LinkDataPath:      fc.LinkDataPath,
			// Shallow copy for mapping nodes due to potentially expensive deep copy.
			LinkDataValue: fc.LinkDataValue,
			ExternalValue: fc.ExternalValue,
		}
	}

	return &state.LinkResourceDrift{
		ResourceID:         drift.ResourceID,
		ResourceName:       drift.ResourceName,
		MappedFieldChanges: fieldChanges,
	}
}

func copyIntermediaryDriftMap(
	drift map[string]*state.IntermediaryDriftState,
) map[string]*state.IntermediaryDriftState {
	if drift == nil {
		return nil
	}

	driftCopy := make(map[string]*state.IntermediaryDriftState, len(drift))
	for k, v := range drift {
		driftCopy[k] = copyIntermediaryDriftState(v)
	}
	return driftCopy
}

func copyIntermediaryDriftState(
	drift *state.IntermediaryDriftState,
) *state.IntermediaryDriftState {
	if drift == nil {
		return nil
	}

	timestampPtr := (*int)(nil)
	if drift.Timestamp != nil {
		timestampValue := *drift.Timestamp
		timestampPtr = &timestampValue
	}

	return &state.IntermediaryDriftState{
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		// Shallow copy for mapping nodes due to potentially expensive deep copy.
		PersistedState: drift.PersistedState,
		ExternalState:  drift.ExternalState,
		Changes:        copyIntermediaryDriftChanges(drift.Changes),
		Exists:         drift.Exists,
		Timestamp:      timestampPtr,
	}
}

func copyIntermediaryDriftChanges(
	changes *state.IntermediaryDriftChanges,
) *state.IntermediaryDriftChanges {
	if changes == nil {
		return nil
	}

	return &state.IntermediaryDriftChanges{
		ModifiedFields: copyIntermediaryFieldChanges(changes.ModifiedFields),
		NewFields:      copyIntermediaryFieldChanges(changes.NewFields),
		RemovedFields:  copyIntermediaryFieldChanges(changes.RemovedFields),
	}
}

func copyIntermediaryFieldChanges(
	changes []state.IntermediaryFieldChange,
) []state.IntermediaryFieldChange {
	if changes == nil {
		return nil
	}

	changesCopy := make([]state.IntermediaryFieldChange, len(changes))
	for i, c := range changes {
		changesCopy[i] = state.IntermediaryFieldChange{
			FieldPath: c.FieldPath,
			// Shallow copy for mapping nodes.
			PrevValue: c.PrevValue,
			NewValue:  c.NewValue,
		}
	}
	return changesCopy
}
