package memfile

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type instancesContainerImpl struct {
	instances        map[string]*state.InstanceState
	instanceIDLookup map[string]string
	// A reference to resources is needed to clean up resources
	// when an instance is removed.
	resources map[string]*state.ResourceState
	// A reference to resource drift entries is needed to clean up
	// resource drift entries when an instance is removed.
	resourceDriftEntries map[string]*state.ResourceDriftState
	// A reference to links is needed to clean up links
	// when an instance is removed.
	links     map[string]*state.LinkState
	fs        afero.Fs
	persister *statePersister
	logger    core.Logger
	mu        *sync.RWMutex
}

func (c *instancesContainerImpl) Get(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instance, hasInstance := c.instances[instanceID]
	if !hasInstance {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}

	return copyInstance(instance, instanceID), nil
}

func (c *instancesContainerImpl) LookupIDByName(
	ctx context.Context,
	instanceName string,
) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	instanceID, hasInstanceID := c.instanceIDLookup[instanceName]
	if hasInstanceID {
		// Verify the instance actually exists in the instances map
		_, instanceExists := c.instances[instanceID]
		if !instanceExists {
			// Stale lookup entry - the instance was deleted but lookup wasn't cleaned up
			c.logger.Warn(
				"stale instance ID lookup entry found, instance does not exist",
				core.StringLogField("instanceName", instanceName),
				core.StringLogField("instanceId", instanceID),
			)
			return "", state.InstanceNotFoundError(instanceName)
		}
		return instanceID, nil
	}

	return "", state.InstanceNotFoundError(instanceName)
}

func (c *instancesContainerImpl) Save(
	ctx context.Context,
	instanceState state.InstanceState,
) error {
	c.mu.Lock()
	// Defer unlock to ensure that modifications are not made to in-memory
	// state during persistence.
	defer c.mu.Unlock()

	return c.save(ctx, instanceState)
}

func (c *instancesContainerImpl) save(
	ctx context.Context,
	instanceState state.InstanceState,
) error {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceState.InstanceID),
	)
	_, alreadyExists := c.instances[instanceState.InstanceID]
	c.instances[instanceState.InstanceID] = &instanceState
	c.instanceIDLookup[instanceState.InstanceName] = instanceState.InstanceID

	if alreadyExists {
		instanceLogger.Debug("persisting instance update")
		return c.persister.updateInstance(&instanceState)
	}

	instanceLogger.Debug(
		"saving child blueprints for instance",
	)
	err := c.saveChildBlueprints(ctx, &instanceState)
	if err != nil {
		return err
	}

	instanceLogger.Debug("persisting new instance")
	return c.persister.createInstance(&instanceState)
}

func (c *instancesContainerImpl) saveChildBlueprints(
	ctx context.Context,
	instance *state.InstanceState,
) error {
	for _, childInstance := range instance.ChildBlueprints {
		err := c.save(ctx, *childInstance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *instancesContainerImpl) UpdateStatus(
	ctx context.Context,
	instanceID string,
	statusInfo state.InstanceStatusInfo,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	instance, hasInstance := c.instances[instanceID]
	if !hasInstance {
		return state.InstanceNotFoundError(instanceID)
	}

	instance.Status = statusInfo.Status
	if statusInfo.LastDeployedTimestamp != nil {
		instance.LastDeployedTimestamp = *statusInfo.LastDeployedTimestamp
	}
	if statusInfo.LastDeployAttemptTimestamp != nil {
		instance.LastDeployAttemptTimestamp = *statusInfo.LastDeployAttemptTimestamp
	}
	if statusInfo.LastStatusUpdateTimestamp != nil {
		instance.LastStatusUpdateTimestamp = *statusInfo.LastStatusUpdateTimestamp
	}
	if statusInfo.Durations != nil {
		instance.Durations = statusInfo.Durations
	}

	c.logger.Debug(
		"persisting instance status update",
		core.StringLogField("instanceId", instanceID),
	)
	return c.persister.updateInstance(instance)
}

func (c *instancesContainerImpl) Remove(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)
	c.mu.Lock()
	defer c.mu.Unlock()

	instance, hasInstance := c.instances[instanceID]
	if !hasInstance {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}

	delete(c.instances, instanceID)
	// Clean up the name-to-ID lookup to prevent stale lookups
	// after the instance is removed.
	delete(c.instanceIDLookup, instance.InstanceName)
	instanceLogger.Debug(
		"cleaning up resource drift entries for instance being removed",
	)
	c.cleanupResourceDriftEntries(instance.ResourceIDs)
	instanceLogger.Debug(
		"cleaning up resources for instance being removed",
	)
	c.cleanupResources(instance.ResourceIDs)
	instanceLogger.Debug(
		"cleaning up links for instance being removed",
	)
	c.cleanupLinks(instance.InstanceID)

	instanceLogger.Debug(
		"persisting removal of blueprint instance",
	)
	return *instance, c.persister.removeInstance(instance)
}

// A lock must be held when calling this method.
func (c *instancesContainerImpl) cleanupResourceDriftEntries(resourceIDs map[string]string) {
	for _, resourceID := range resourceIDs {
		delete(c.resourceDriftEntries, resourceID)
	}
}

// A lock must be held when calling this method.
func (c *instancesContainerImpl) cleanupResources(resourceIDs map[string]string) {
	for _, resourceID := range resourceIDs {
		delete(c.resources, resourceID)
	}
}

// A lock must be held when calling this method.
func (c *instancesContainerImpl) cleanupLinks(instanceID string) {
	for _, link := range c.links {
		if link.InstanceID == instanceID {
			delete(c.links, link.LinkID)
		}
	}
}

func copyInstance(instanceState *state.InstanceState, path string) state.InstanceState {
	instanceCopy := *instanceState
	if instanceState.Resources != nil {
		instanceCopy.Resources = make(map[string]*state.ResourceState)
		for resourceID, resource := range instanceState.Resources {
			resCopy := copyResource(resource)
			instanceCopy.Resources[resourceID] = &resCopy
		}
	}
	if instanceState.ResourceIDs != nil {
		instanceCopy.ResourceIDs = make(map[string]string)
		for resourceName, resourceID := range instanceState.ResourceIDs {
			instanceCopy.ResourceIDs[resourceName] = resourceID
		}
	}
	if instanceState.Links != nil {
		instanceCopy.Links = make(map[string]*state.LinkState)
		for linkName, link := range instanceState.Links {
			linkCopy := copyLink(link)
			instanceCopy.Links[linkName] = &linkCopy
		}
	}
	if instanceState.Metadata != nil {
		instanceCopy.Metadata = make(map[string]*core.MappingNode)
		for key, value := range instanceState.Metadata {
			instanceCopy.Metadata[key] = value
		}
	}
	if instanceState.Exports != nil {
		instanceCopy.Exports = make(map[string]*state.ExportState)
		for exportName, export := range instanceState.Exports {
			exportCopy := copyExport(export)
			instanceCopy.Exports[exportName] = exportCopy
		}
	}
	copyChildBlueprintComponents(&instanceCopy, instanceState, path)
	return instanceCopy
}

func copyChildBlueprintComponents(
	dest *state.InstanceState,
	src *state.InstanceState,
	path string,
) {
	if src.ChildBlueprints != nil {
		dest.ChildBlueprints = make(map[string]*state.InstanceState)
		for childName, childState := range src.ChildBlueprints {
			if instancePathContains(path, childState.InstanceID) {
				// Avoid circular references
				continue
			}
			copy := copyInstance(childState, fmt.Sprintf("%s/%s", path, childState.InstanceID))
			dest.ChildBlueprints[childName] = &copy
		}
	}
	if src.ChildDependencies != nil {
		dest.ChildDependencies = make(map[string]*state.DependencyInfo)
		for childName, dependencyInfo := range src.ChildDependencies {
			dest.ChildDependencies[childName] = copyDependencyInfo(dependencyInfo)
		}
	}
}

func copyDependencyInfo(
	dependencyInfo *state.DependencyInfo,
) *state.DependencyInfo {
	if dependencyInfo == nil {
		return nil
	}

	dependsOnResources := make([]string, len(dependencyInfo.DependsOnResources))
	copy(dependsOnResources, dependencyInfo.DependsOnResources)

	dependsOnChildren := make([]string, len(dependencyInfo.DependsOnChildren))
	copy(dependsOnChildren, dependencyInfo.DependsOnChildren)

	return &state.DependencyInfo{
		DependsOnResources: dependsOnResources,
		DependsOnChildren:  dependsOnChildren,
	}
}

func instancePathContains(path string, instanceID string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == instanceID {
			return true
		}
	}
	return false
}

func getInstance(
	instances map[string]*state.InstanceState,
	instanceID string,
) (*state.InstanceState, bool) {
	instance, ok := instances[instanceID]
	if ok && instance != nil {
		return instance, true
	}

	return nil, false
}
