package statestore

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// copyInstance returns a deep-ish copy of instanceState: resources, links,
// exports, metadata, and child dependency info are copied; child blueprints
// are copied recursively with a path guard against cycles. Spec/mapping
// node values that can be expensive to deep-copy remain shared references.
// path carries the traversal trail of instance IDs to detect cycles.
func copyInstance(instanceState *state.InstanceState, path string) state.InstanceState {
	instanceCopy := *instanceState
	if instanceState.Resources != nil {
		instanceCopy.Resources = make(map[string]*state.ResourceState, len(instanceState.Resources))
		for resourceID, resource := range instanceState.Resources {
			resCopy := copyResource(resource)
			instanceCopy.Resources[resourceID] = &resCopy
		}
	}
	if instanceState.ResourceIDs != nil {
		instanceCopy.ResourceIDs = make(map[string]string, len(instanceState.ResourceIDs))
		maps.Copy(instanceCopy.ResourceIDs, instanceState.ResourceIDs)
	}
	if instanceState.Links != nil {
		instanceCopy.Links = make(map[string]*state.LinkState, len(instanceState.Links))
		for linkName, link := range instanceState.Links {
			linkCopy := copyLink(link)
			instanceCopy.Links[linkName] = &linkCopy
		}
	}
	if instanceState.Metadata != nil {
		instanceCopy.Metadata = make(map[string]*core.MappingNode, len(instanceState.Metadata))
		maps.Copy(instanceCopy.Metadata, instanceState.Metadata)
	}
	if instanceState.Exports != nil {
		instanceCopy.Exports = make(map[string]*state.ExportState, len(instanceState.Exports))
		for exportName, export := range instanceState.Exports {
			instanceCopy.Exports[exportName] = copyExport(export)
		}
	}
	copyChildBlueprintComponents(&instanceCopy, instanceState, path)
	return instanceCopy
}

func copyChildBlueprintComponents(dest, src *state.InstanceState, path string) {
	if src.ChildBlueprints != nil {
		dest.ChildBlueprints = make(map[string]*state.InstanceState, len(src.ChildBlueprints))
		for childName, childState := range src.ChildBlueprints {
			if instancePathContains(path, childState.InstanceID) {
				// Cycle — skip to avoid infinite recursion.
				continue
			}
			copied := copyInstance(childState, fmt.Sprintf("%s/%s", path, childState.InstanceID))
			dest.ChildBlueprints[childName] = &copied
		}
	}
	if src.ChildDependencies != nil {
		dest.ChildDependencies = make(map[string]*state.DependencyInfo, len(src.ChildDependencies))
		for childName, dependencyInfo := range src.ChildDependencies {
			dest.ChildDependencies[childName] = copyDependencyInfo(dependencyInfo)
		}
	}
}

func copyDependencyInfo(dependencyInfo *state.DependencyInfo) *state.DependencyInfo {
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

func instancePathContains(path, instanceID string) bool {
	return slices.Contains(strings.Split(path, "/"), instanceID)
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
		ResourceID:                 resourceState.ResourceID,
		Name:                       resourceState.Name,
		Type:                       resourceState.Type,
		TemplateName:               resourceState.TemplateName,
		InstanceID:                 resourceState.InstanceID,
		Status:                     resourceState.Status,
		PreciseStatus:              resourceState.PreciseStatus,
		Description:                resourceState.Description,
		Metadata:                   &metadataCopy,
		SystemMetadata:             systemMetadataCopy,
		ComputedFields:             computedFields,
		DependsOnResources:         dependsOnResources,
		DependsOnChildren:          dependsOnChildren,
		FailureReasons:             resourceState.FailureReasons,
		SpecData:                   resourceState.SpecData,
		LastDeployedTimestamp:      resourceState.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: resourceState.LastDeployAttemptTimestamp,
		LastStatusUpdateTimestamp:  resourceState.LastStatusUpdateTimestamp,
		Drifted:                    resourceState.Drifted,
		LastDriftDetectedTimestamp: resourceState.LastDriftDetectedTimestamp,
		Durations:                  resourceState.Durations,
		RemovalPolicy:              resourceState.RemovalPolicy,
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
		IntermediaryResourceStates: copyIntermediaryResources(linkState.IntermediaryResourceStates),
		Data:                       linkState.Data,
		ResourceDataMappings:       linkState.ResourceDataMappings,
		FailureReasons:             linkState.FailureReasons,
		Drifted:                    linkState.Drifted,
		LastDriftDetectedTimestamp: linkState.LastDriftDetectedTimestamp,
		Durations:                  linkState.Durations,
	}
}

func copyResourceDrift(driftState *state.ResourceDriftState) state.ResourceDriftState {
	if driftState == nil {
		return state.ResourceDriftState{}
	}
	var timestampPtr *int
	if driftState.Timestamp != nil {
		v := *driftState.Timestamp
		timestampPtr = &v
	}
	return state.ResourceDriftState{
		ResourceID:   driftState.ResourceID,
		ResourceName: driftState.ResourceName,
		SpecData:     driftState.SpecData,
		Difference:   copyResourceDriftDifference(driftState.Difference),
		Timestamp:    timestampPtr,
	}
}

func copyResourceDriftDifference(difference *state.ResourceDriftChanges) *state.ResourceDriftChanges {
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
	out := make([]*state.ResourceDriftFieldChange, len(fieldChanges))
	for i, v := range fieldChanges {
		out[i] = &state.ResourceDriftFieldChange{
			FieldPath:    v.FieldPath,
			StateValue:   v.StateValue,
			DriftedValue: v.DriftedValue,
		}
	}
	return out
}

func copyLinkDrift(driftState *state.LinkDriftState) state.LinkDriftState {
	if driftState == nil {
		return state.LinkDriftState{}
	}
	var timestampPtr *int
	if driftState.Timestamp != nil {
		v := *driftState.Timestamp
		timestampPtr = &v
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
			LinkDataValue:     fc.LinkDataValue,
			ExternalValue:     fc.ExternalValue,
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
	out := make(map[string]*state.IntermediaryDriftState, len(drift))
	for k, v := range drift {
		out[k] = copyIntermediaryDriftState(v)
	}
	return out
}

func copyIntermediaryDriftState(drift *state.IntermediaryDriftState) *state.IntermediaryDriftState {
	if drift == nil {
		return nil
	}
	var timestampPtr *int
	if drift.Timestamp != nil {
		v := *drift.Timestamp
		timestampPtr = &v
	}
	return &state.IntermediaryDriftState{
		ResourceID:     drift.ResourceID,
		ResourceType:   drift.ResourceType,
		PersistedState: drift.PersistedState,
		ExternalState:  drift.ExternalState,
		Changes:        copyIntermediaryDriftChanges(drift.Changes),
		Exists:         drift.Exists,
		Timestamp:      timestampPtr,
	}
}

func copyIntermediaryDriftChanges(changes *state.IntermediaryDriftChanges) *state.IntermediaryDriftChanges {
	if changes == nil {
		return nil
	}
	return &state.IntermediaryDriftChanges{
		ModifiedFields: copyIntermediaryFieldChanges(changes.ModifiedFields),
		NewFields:      copyIntermediaryFieldChanges(changes.NewFields),
		RemovedFields:  copyIntermediaryFieldChanges(changes.RemovedFields),
	}
}

func copyIntermediaryFieldChanges(changes []state.IntermediaryFieldChange) []state.IntermediaryFieldChange {
	if changes == nil {
		return nil
	}
	out := make([]state.IntermediaryFieldChange, len(changes))
	for i, c := range changes {
		out[i] = state.IntermediaryFieldChange{
			FieldPath: c.FieldPath,
			PrevValue: c.PrevValue,
			NewValue:  c.NewValue,
		}
	}
	return out
}

func copyIntermediaryResources(
	resources []*state.LinkIntermediaryResourceState,
) []*state.LinkIntermediaryResourceState {
	if resources == nil {
		return nil
	}
	out := make([]*state.LinkIntermediaryResourceState, len(resources))
	for i, value := range resources {
		out[i] = &state.LinkIntermediaryResourceState{
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
	return out
}
