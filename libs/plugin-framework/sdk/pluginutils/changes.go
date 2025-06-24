package pluginutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// GetCurrentResourceStateSpecData returns the spec data for the current
// resource state from the changes object.
func GetCurrentResourceStateSpecData(changes *provider.Changes) *core.MappingNode {
	if changes == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	appliedResourceInfo := changes.AppliedResourceInfo
	if appliedResourceInfo.CurrentResourceState == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	return appliedResourceInfo.CurrentResourceState.SpecData
}

// GetResolvedResourceSpecData returns the resolved spec data for the resource
// from changes.
func GetResolvedResourceSpecData(changes *provider.Changes) *core.MappingNode {
	if changes == nil || changes.AppliedResourceInfo.ResourceWithResolvedSubs == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	return changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec
}

// GetCurrentStateSpecDataFromResourceInfo extracts the current resource state's
// spec data from the resource info.
// If the resource info or current resource state is nil, it returns a MappingNode with an empty set of fields.
func GetCurrentStateSpecDataFromResourceInfo(
	resourceInfo *provider.ResourceInfo,
) *core.MappingNode {
	if resourceInfo == nil || resourceInfo.CurrentResourceState == nil {
		return &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		}
	}

	return resourceInfo.CurrentResourceState.SpecData
}
