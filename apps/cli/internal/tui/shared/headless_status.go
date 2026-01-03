package shared

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// Resource status icon mappings for headless output
var resourceStatusIcons = map[core.ResourceStatus]string{
	core.ResourceStatusCreating:          "...",
	core.ResourceStatusUpdating:          "...",
	core.ResourceStatusDestroying:        "...",
	core.ResourceStatusCreated:           "OK",
	core.ResourceStatusUpdated:           "OK",
	core.ResourceStatusDestroyed:         "OK",
	core.ResourceStatusCreateFailed:      "ERR",
	core.ResourceStatusUpdateFailed:      "ERR",
	core.ResourceStatusDestroyFailed:     "ERR",
	core.ResourceStatusRollingBack:       "<-",
	core.ResourceStatusRollbackFailed:    "!!",
	core.ResourceStatusRollbackComplete:  "RB",
	core.ResourceStatusCreateInterrupted: "INT",
	core.ResourceStatusUpdateInterrupted: "INT",
	core.ResourceStatusDestroyInterrupted: "INT",
}

// Resource status text mappings for headless output
var resourceStatusText = map[core.ResourceStatus]string{
	core.ResourceStatusCreating:           "creating",
	core.ResourceStatusCreated:            "created",
	core.ResourceStatusCreateFailed:       "create failed",
	core.ResourceStatusUpdating:           "updating",
	core.ResourceStatusUpdated:            "updated",
	core.ResourceStatusUpdateFailed:       "update failed",
	core.ResourceStatusDestroying:         "destroying",
	core.ResourceStatusDestroyed:          "destroyed",
	core.ResourceStatusDestroyFailed:      "destroy failed",
	core.ResourceStatusRollingBack:        "rolling back",
	core.ResourceStatusRollbackFailed:     "rollback failed",
	core.ResourceStatusRollbackComplete:   "rolled back",
	core.ResourceStatusCreateInterrupted:  "create interrupted",
	core.ResourceStatusUpdateInterrupted:  "update interrupted",
	core.ResourceStatusDestroyInterrupted: "destroy interrupted",
}

// Instance status icon mappings for headless output
var instanceStatusIcons = map[core.InstanceStatus]string{
	core.InstanceStatusPreparing:              "  ",
	core.InstanceStatusDeploying:              "...",
	core.InstanceStatusUpdating:               "...",
	core.InstanceStatusDestroying:             "...",
	core.InstanceStatusDeployed:               "OK",
	core.InstanceStatusUpdated:                "OK",
	core.InstanceStatusDestroyed:              "OK",
	core.InstanceStatusDeployFailed:           "ERR",
	core.InstanceStatusUpdateFailed:           "ERR",
	core.InstanceStatusDestroyFailed:          "ERR",
	core.InstanceStatusDeployRollingBack:      "<-",
	core.InstanceStatusUpdateRollingBack:      "<-",
	core.InstanceStatusDestroyRollingBack:     "<-",
	core.InstanceStatusDeployRollbackFailed:   "!!",
	core.InstanceStatusUpdateRollbackFailed:   "!!",
	core.InstanceStatusDestroyRollbackFailed:  "!!",
	core.InstanceStatusDeployRollbackComplete: "RB",
	core.InstanceStatusUpdateRollbackComplete: "RB",
	core.InstanceStatusDestroyRollbackComplete: "RB",
	core.InstanceStatusDeployInterrupted:      "INT",
	core.InstanceStatusUpdateInterrupted:      "INT",
	core.InstanceStatusDestroyInterrupted:     "INT",
}

// Instance status text mappings for headless output
var instanceStatusText = map[core.InstanceStatus]string{
	core.InstanceStatusPreparing:               "preparing",
	core.InstanceStatusDeploying:               "deploying",
	core.InstanceStatusDeployed:                "deployed",
	core.InstanceStatusDeployFailed:            "deploy failed",
	core.InstanceStatusUpdating:                "updating",
	core.InstanceStatusUpdated:                 "updated",
	core.InstanceStatusUpdateFailed:            "update failed",
	core.InstanceStatusDestroying:              "destroying",
	core.InstanceStatusDestroyed:               "destroyed",
	core.InstanceStatusDestroyFailed:           "destroy failed",
	core.InstanceStatusDeployRollingBack:       "rolling back deploy",
	core.InstanceStatusDeployRollbackFailed:    "deploy rollback failed",
	core.InstanceStatusDeployRollbackComplete:  "deploy rolled back",
	core.InstanceStatusUpdateRollingBack:       "rolling back update",
	core.InstanceStatusUpdateRollbackFailed:    "update rollback failed",
	core.InstanceStatusUpdateRollbackComplete:  "update rolled back",
	core.InstanceStatusDestroyRollingBack:      "rolling back destroy",
	core.InstanceStatusDestroyRollbackFailed:   "destroy rollback failed",
	core.InstanceStatusDestroyRollbackComplete: "destroy rolled back",
	core.InstanceStatusDeployInterrupted:       "deploy interrupted",
	core.InstanceStatusUpdateInterrupted:       "update interrupted",
	core.InstanceStatusDestroyInterrupted:      "destroy interrupted",
	core.InstanceStatusNotDeployed:             "not deployed",
}

// Link status icon mappings for headless output
var linkStatusIcons = map[core.LinkStatus]string{
	core.LinkStatusCreating:               "...",
	core.LinkStatusUpdating:               "...",
	core.LinkStatusDestroying:             "...",
	core.LinkStatusCreated:                "OK",
	core.LinkStatusUpdated:                "OK",
	core.LinkStatusDestroyed:              "OK",
	core.LinkStatusCreateFailed:           "ERR",
	core.LinkStatusUpdateFailed:           "ERR",
	core.LinkStatusDestroyFailed:          "ERR",
	core.LinkStatusCreateRollingBack:      "<-",
	core.LinkStatusUpdateRollingBack:      "<-",
	core.LinkStatusDestroyRollingBack:     "<-",
	core.LinkStatusCreateRollbackFailed:   "!!",
	core.LinkStatusUpdateRollbackFailed:   "!!",
	core.LinkStatusDestroyRollbackFailed:  "!!",
	core.LinkStatusCreateRollbackComplete: "RB",
	core.LinkStatusUpdateRollbackComplete: "RB",
	core.LinkStatusDestroyRollbackComplete: "RB",
	core.LinkStatusCreateInterrupted:      "INT",
	core.LinkStatusUpdateInterrupted:      "INT",
	core.LinkStatusDestroyInterrupted:     "INT",
}

// Link status text mappings for headless output
var linkStatusText = map[core.LinkStatus]string{
	core.LinkStatusCreating:               "creating",
	core.LinkStatusCreated:                "created",
	core.LinkStatusCreateFailed:           "create failed",
	core.LinkStatusUpdating:               "updating",
	core.LinkStatusUpdated:                "updated",
	core.LinkStatusUpdateFailed:           "update failed",
	core.LinkStatusDestroying:             "destroying",
	core.LinkStatusDestroyed:              "destroyed",
	core.LinkStatusDestroyFailed:          "destroy failed",
	core.LinkStatusCreateRollingBack:      "rolling back create",
	core.LinkStatusCreateRollbackFailed:   "create rollback failed",
	core.LinkStatusCreateRollbackComplete: "create rolled back",
	core.LinkStatusUpdateRollingBack:      "rolling back update",
	core.LinkStatusUpdateRollbackFailed:   "update rollback failed",
	core.LinkStatusUpdateRollbackComplete: "update rolled back",
	core.LinkStatusDestroyRollingBack:     "rolling back destroy",
	core.LinkStatusDestroyRollbackFailed:  "destroy rollback failed",
	core.LinkStatusDestroyRollbackComplete: "destroy rolled back",
	core.LinkStatusCreateInterrupted:      "create interrupted",
	core.LinkStatusUpdateInterrupted:      "update interrupted",
	core.LinkStatusDestroyInterrupted:     "destroy interrupted",
}

// ResourceStatusHeadlessIcon returns the icon for a resource status in headless mode.
func ResourceStatusHeadlessIcon(status core.ResourceStatus) string {
	if icon, ok := resourceStatusIcons[status]; ok {
		return icon
	}
	return "  "
}

// ResourceStatusHeadlessText returns the text for a resource status in headless mode.
func ResourceStatusHeadlessText(status core.ResourceStatus) string {
	if text, ok := resourceStatusText[status]; ok {
		return text
	}
	return "pending"
}

// InstanceStatusHeadlessIcon returns the icon for an instance status in headless mode.
func InstanceStatusHeadlessIcon(status core.InstanceStatus) string {
	if icon, ok := instanceStatusIcons[status]; ok {
		return icon
	}
	return "  "
}

// InstanceStatusHeadlessText returns the text for an instance status in headless mode.
func InstanceStatusHeadlessText(status core.InstanceStatus) string {
	if text, ok := instanceStatusText[status]; ok {
		return text
	}
	return "unknown"
}

// LinkStatusHeadlessIcon returns the icon for a link status in headless mode.
func LinkStatusHeadlessIcon(status core.LinkStatus) string {
	if icon, ok := linkStatusIcons[status]; ok {
		return icon
	}
	return "  "
}

// LinkStatusHeadlessText returns the text for a link status in headless mode.
func LinkStatusHeadlessText(status core.LinkStatus) string {
	if text, ok := linkStatusText[status]; ok {
		return text
	}
	return "pending"
}
