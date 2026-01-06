package shared

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// SkippableResource is an interface for resource items that can be marked as skipped.
type SkippableResource interface {
	GetAction() ActionType
	GetResourceStatus() core.ResourceStatus
	SetSkipped(skipped bool)
}

// SkippableChild is an interface for child items that can be marked as skipped.
type SkippableChild interface {
	GetAction() ActionType
	GetChildStatus() core.InstanceStatus
	SetSkipped(skipped bool)
}

// SkippableLink is an interface for link items that can be marked as skipped.
type SkippableLink interface {
	GetAction() ActionType
	GetLinkStatus() core.LinkStatus
	SetSkipped(skipped bool)
}

// IsPendingResourceStatus returns true if the resource status is a pending/initial state.
func IsPendingResourceStatus(status core.ResourceStatus) bool {
	return status == core.ResourceStatusUnknown
}

// IsPendingChildStatus returns true if the child status is a pending/initial state.
func IsPendingChildStatus(status core.InstanceStatus) bool {
	return status == core.InstanceStatusPreparing || status == core.InstanceStatusNotDeployed
}

// IsPendingLinkStatus returns true if the link status is a pending/initial state.
func IsPendingLinkStatus(status core.LinkStatus) bool {
	return status == core.LinkStatusUnknown
}

// MarkPendingResourcesAsSkipped marks resources that were never attempted as skipped.
func MarkPendingResourcesAsSkipped[T SkippableResource](resources map[string]T) {
	for _, item := range resources {
		if item.GetAction() == ActionNoChange {
			continue
		}
		if IsPendingResourceStatus(item.GetResourceStatus()) {
			item.SetSkipped(true)
		}
	}
}

// MarkPendingChildrenAsSkipped marks children that were never attempted as skipped.
func MarkPendingChildrenAsSkipped[T SkippableChild](children map[string]T) {
	for _, item := range children {
		if item.GetAction() == ActionNoChange {
			continue
		}
		if IsPendingChildStatus(item.GetChildStatus()) {
			item.SetSkipped(true)
		}
	}
}

// MarkPendingLinksAsSkipped marks links that were never attempted as skipped.
func MarkPendingLinksAsSkipped[T SkippableLink](links map[string]T) {
	for _, item := range links {
		if item.GetAction() == ActionNoChange {
			continue
		}
		if IsPendingLinkStatus(item.GetLinkStatus()) {
			item.SetSkipped(true)
		}
	}
}
