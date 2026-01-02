package memfile

import (
	"context"
	"fmt"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type resourcesContainerImpl struct {
	resources            map[string]*state.ResourceState
	resourceDriftEntries map[string]*state.ResourceDriftState
	instances            map[string]*state.InstanceState
	fs                   afero.Fs
	persister            *statePersister
	logger               core.Logger
	mu                   *sync.RWMutex
}

func (c *resourcesContainerImpl) Get(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resource, hasResource := c.resources[resourceID]
	if !hasResource {
		return state.ResourceState{}, state.ResourceNotFoundError(resourceID)
	}

	return copyResource(resource), nil
}

func (c *resourcesContainerImpl) GetByName(
	ctx context.Context,
	instanceID string,
	resourceName string,
) (state.ResourceState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if instance, ok := getInstance(c.instances, instanceID); ok {
		resourceID, ok := instance.ResourceIDs[resourceName]
		if ok {
			return copyResource(c.resources[resourceID]), nil
		}
	}

	itemID := idutils.ReourceInBlueprintID(instanceID, resourceName)
	return state.ResourceState{}, state.ResourceNotFoundError(itemID)
}

func (c *resourcesContainerImpl) Save(
	ctx context.Context,
	resourceState state.ResourceState,
) error {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("resourceId", resourceState.ResourceID),
		core.StringLogField("resourceName", resourceState.Name),
		core.StringLogField("instanceId", resourceState.InstanceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	if instance, ok := getInstance(c.instances, resourceState.InstanceID); ok {
		instance.ResourceIDs[resourceState.Name] = resourceState.ResourceID
		instance.Resources[resourceState.ResourceID] = &resourceState

		c.resources[resourceState.ResourceID] = &resourceState

		resourceLogger.Debug("persisting updated or newly created resource")
		return c.persister.updateInstance(instance)
	}

	return state.InstanceNotFoundError(resourceState.InstanceID)
}

func (c *resourcesContainerImpl) UpdateStatus(
	ctx context.Context,
	resourceID string,
	statusInfo state.ResourceStatusInfo,
) error {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("resourceId", resourceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	resource, hasResource := c.resources[resourceID]
	if !hasResource {
		return state.ResourceNotFoundError(resourceID)
	}

	instance, ok := c.instances[resource.InstanceID]
	if !ok {
		// When a resource exists but the instance does not,
		// then something has corrupted the state.
		return errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, resourceID),
		)
	}

	resource.Status = statusInfo.Status
	resource.PreciseStatus = statusInfo.PreciseStatus
	resource.FailureReasons = statusInfo.FailureReasons
	if statusInfo.LastDeployAttemptTimestamp != nil {
		resource.LastDeployAttemptTimestamp = *statusInfo.LastDeployAttemptTimestamp
	}
	if statusInfo.LastDeployedTimestamp != nil {
		resource.LastDeployedTimestamp = *statusInfo.LastDeployedTimestamp
	}
	if statusInfo.LastStatusUpdateTimestamp != nil {
		resource.LastStatusUpdateTimestamp = *statusInfo.LastStatusUpdateTimestamp
	}
	if statusInfo.Durations != nil {
		resource.Durations = statusInfo.Durations
	}

	resourceLogger.Debug("persisting updated resource status")
	return c.persister.updateInstance(instance)
}

func (c *resourcesContainerImpl) Remove(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("resourceId", resourceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	resource, ok := c.resources[resourceID]
	if !ok {
		return state.ResourceState{}, state.ResourceNotFoundError(resourceID)
	}

	instance, hasInstance := c.instances[resource.InstanceID]
	if !hasInstance {
		// When a resource exists but the instance does not,
		// then something has corrupted the state.
		return state.ResourceState{}, errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, resourceID),
		)
	}

	delete(instance.Resources, resourceID)
	delete(c.resources, resourceID)

	resourceLogger.Debug("persisting removal of resource")
	return *resource, c.persister.updateInstance(instance)
}

func (c *resourcesContainerImpl) GetDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, hasResource := c.resources[resourceID]
	if !hasResource {
		return state.ResourceDriftState{}, state.ResourceNotFoundError(resourceID)
	}

	drift, hasDrift := c.resourceDriftEntries[resourceID]
	if !hasDrift {
		// An empty drift state is valid for a resource that has not drifted.
		return state.ResourceDriftState{}, nil
	}

	return copyResourceDrift(drift), nil
}

func (c *resourcesContainerImpl) SaveDrift(
	ctx context.Context,
	driftState state.ResourceDriftState,
) error {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("resourceId", driftState.ResourceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	resource, hasResource := c.resources[driftState.ResourceID]
	if !hasResource {
		return state.ResourceNotFoundError(driftState.ResourceID)
	}

	instance, ok := c.instances[resource.InstanceID]
	if !ok {
		// When a resource exists but the instance does not,
		// then something has corrupted the state.
		return errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, driftState.ResourceID),
		)
	}

	resource.Drifted = true
	resource.LastDriftDetectedTimestamp = driftState.Timestamp

	_, alreadyExists := c.resourceDriftEntries[driftState.ResourceID]
	c.resourceDriftEntries[driftState.ResourceID] = &driftState

	resourceLogger.Debug("persisting updated or newly created resource drift entry")
	err := c.persistResourceDrift(&driftState, alreadyExists)
	if err != nil {
		return err
	}

	resourceLogger.Debug("persisting resource changes for latest drift state")
	// Ensure that the instance is updated to reflect the drift field
	// updates to the resource.
	return c.persister.updateInstance(instance)
}

func (c *resourcesContainerImpl) persistResourceDrift(
	driftState *state.ResourceDriftState,
	alreadyExists bool,
) error {
	if alreadyExists {
		return c.persister.updateResourceDrift(driftState)
	}

	return c.persister.createResourceDrift(driftState)
}

func (c *resourcesContainerImpl) RemoveDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("resourceId", resourceID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	resource, hasResource := c.resources[resourceID]
	if !hasResource {
		return state.ResourceDriftState{}, state.ResourceNotFoundError(resourceID)
	}

	driftState, hasDrift := c.resourceDriftEntries[resourceID]
	if !hasDrift {
		return state.ResourceDriftState{}, nil
	}

	instance, ok := c.instances[resource.InstanceID]
	if !ok {
		// When a resource exists but the instance does not,
		// then something has corrupted the state.
		return state.ResourceDriftState{}, errMalformedState(
			instanceNotFoundForResourceMessage(resource.InstanceID, driftState.ResourceID),
		)
	}

	resource.Drifted = false
	resource.LastDriftDetectedTimestamp = nil
	delete(c.resourceDriftEntries, resourceID)

	resourceLogger.Debug("persisting removal of resource drift entry")
	err := c.persister.removeResourceDrift(driftState)
	if err != nil {
		return state.ResourceDriftState{}, err
	}

	resourceLogger.Debug("persisting resource changes for removal of drift state")
	// Ensure that the instance is updated to reflect the drift field
	// updates to the resource.
	err = c.persister.updateInstance(instance)
	if err != nil {
		return state.ResourceDriftState{}, err
	}

	return *driftState, nil
}

func copyResource(resourceState *state.ResourceState) state.ResourceState {
	if resourceState == nil {
		return state.ResourceState{}
	}

	metadataCopy := copyResourceMetadata(resourceState.Metadata)
	systemMetadataCopy := copySystemMetadata(resourceState.SystemMetadata)

	dependsOnResources := make([]string, len(resourceState.DependsOnResources))
	copy(dependsOnResources, resourceState.DependsOnResources)

	dependsOnChildren := make([]string, len(resourceState.DependsOnChildren))
	copy(dependsOnChildren, resourceState.DependsOnChildren)

	computedFields := make([]string, len(resourceState.ComputedFields))
	copy(computedFields, resourceState.ComputedFields)

	return state.ResourceState{
		ResourceID:         resourceState.ResourceID,
		Name:               resourceState.Name,
		Type:               resourceState.Type,
		TemplateName:       resourceState.TemplateName,
		InstanceID:         resourceState.InstanceID,
		Status:             resourceState.Status,
		PreciseStatus:      resourceState.PreciseStatus,
		Description:        resourceState.Description,
		Metadata:           &metadataCopy,
		SystemMetadata:     systemMetadataCopy,
		ComputedFields:     computedFields,
		DependsOnResources: dependsOnResources,
		DependsOnChildren:  dependsOnChildren,
		FailureReasons:     resourceState.FailureReasons,
		// The spec data pointer will be copied, no part of the blueprint container
		// implementation should modify the spec data in instance state so it is safe
		// to copy the pointer instead of making a deep copy.
		SpecData:                   resourceState.SpecData,
		LastDeployedTimestamp:      resourceState.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: resourceState.LastDeployAttemptTimestamp,
		LastStatusUpdateTimestamp:  resourceState.LastStatusUpdateTimestamp,
		Drifted:                    resourceState.Drifted,
		LastDriftDetectedTimestamp: resourceState.LastDriftDetectedTimestamp,
		Durations:                  resourceState.Durations,
	}
}

func copyResourceMetadata(metadata *state.ResourceMetadataState) state.ResourceMetadataState {
	if metadata == nil {
		return state.ResourceMetadataState{}
	}

	return state.ResourceMetadataState{
		DisplayName: metadata.DisplayName,
		Annotations: metadata.Annotations,
		Labels:      metadata.Labels,
		Custom:      metadata.Custom,
	}
}

func copySystemMetadata(systemMetadata *state.SystemMetadataState) *state.SystemMetadataState {
	if systemMetadata == nil {
		return nil
	}

	return &state.SystemMetadataState{
		Provenance: copyProvenance(systemMetadata.Provenance),
	}
}

func copyProvenance(provenance *state.ProvenanceState) *state.ProvenanceState {
	if provenance == nil {
		return nil
	}

	return &state.ProvenanceState{
		ProvisionedBy:         provenance.ProvisionedBy,
		DeployEngineVersion:   provenance.DeployEngineVersion,
		ProviderPluginID:      provenance.ProviderPluginID,
		ProviderPluginVersion: provenance.ProviderPluginVersion,
		ProvisionedAt:         provenance.ProvisionedAt,
	}
}

func copyResourceDrift(driftState *state.ResourceDriftState) state.ResourceDriftState {
	if driftState == nil {
		return state.ResourceDriftState{}
	}

	timestampPtr := (*int)(nil)
	if driftState.Timestamp != nil {
		timestampValue := *driftState.Timestamp
		timestampPtr = &timestampValue
	}

	return state.ResourceDriftState{
		ResourceID:   driftState.ResourceID,
		ResourceName: driftState.ResourceName,
		// The spec data pointer will be copied, no part of the blueprint container
		// implementation should modify the spec data in instance state so it is safe
		// to copy the pointer instead of making a deep copy.
		// Spec data size is variable depending on the resource implementation so it
		// can be especially expensive to deep copy in comparison to other fields.
		SpecData:   driftState.SpecData,
		Difference: copyResourceDriftDifference(driftState.Difference),
		Timestamp:  timestampPtr,
	}
}

func copyResourceDriftDifference(
	difference *state.ResourceDriftChanges,
) *state.ResourceDriftChanges {
	if difference == nil {
		return nil
	}

	removedFields := make([]string, len(difference.RemovedFields))
	copy(removedFields, difference.RemovedFields)

	unchangedFields := make([]string, len(difference.UnchangedFields))
	copy(unchangedFields, difference.UnchangedFields)

	return &state.ResourceDriftChanges{
		ModifiedFields:  copyResourceDriftFieldChanges(difference.ModifiedFields),
		NewFields:       copyResourceDriftFieldChanges(difference.NewFields),
		RemovedFields:   removedFields,
		UnchangedFields: unchangedFields,
	}
}

func copyResourceDriftFieldChanges(
	fieldChanges []*state.ResourceDriftFieldChange,
) []*state.ResourceDriftFieldChange {
	if fieldChanges == nil {
		return nil
	}

	fieldChangesCopy := make([]*state.ResourceDriftFieldChange, len(fieldChanges))
	for i, value := range fieldChanges {
		fieldChangesCopy[i] = &state.ResourceDriftFieldChange{
			FieldPath: value.FieldPath,
			// Shallow copy for mapping nodes due to potentially expensive to deep copy.
			StateValue:   value.StateValue,
			DriftedValue: value.DriftedValue,
		}
	}

	return fieldChangesCopy
}

func instanceNotFoundForResourceMessage(
	instanceID string,
	resourceID string,
) string {
	return fmt.Sprintf("instance %s not found for resource %s", instanceID, resourceID)
}
