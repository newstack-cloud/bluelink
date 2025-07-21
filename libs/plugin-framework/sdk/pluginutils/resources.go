package pluginutils

import "github.com/newstack-cloud/bluelink/libs/blueprint/provider"

// GetResourceName safely retrieves the resource name from the provided
// resource info struct. If the resource info is nil, it returns "unknown".
func GetResourceName(resourceInfo *provider.ResourceInfo) string {
	if resourceInfo == nil {
		return "unknown"
	}

	return resourceInfo.ResourceName
}

// GetInstanceID retrieves the instance ID from the resource info.
// If the resource info is nil, it returns "unknown".
func GetInstanceID(resourceInfo *provider.ResourceInfo) string {
	if resourceInfo == nil {
		return "unknown"
	}

	return resourceInfo.InstanceID
}

// IsResourceNew checks if the provided changes indicate that a new resource
// is being created or an existing resource is being recreated.
// This is useful for tasks like determining the changes that should be reported
// for a link implementation where changes to the link data are determined
// by the changes to the resources that are linked.
func IsResourceNew(changes *provider.Changes) bool {
	return (len(changes.ModifiedFields) == 0 &&
		len(changes.RemovedFields) == 0 &&
		len(changes.UnchangedFields) == 0 &&
		len(changes.NewFields) > 0) || changes.MustRecreate
}
