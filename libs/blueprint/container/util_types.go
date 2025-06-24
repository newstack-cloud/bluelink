package container

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinkPendingCompletion holds information about the completion status of a link
// between two resources for change staging and deployment.
type LinkPendingCompletion struct {
	resourceANode    *links.ChainLinkNode
	resourceBNode    *links.ChainLinkNode
	resourceAPending bool
	resourceBPending bool
	linkPending      bool
}

// CollectedElements holds resources, children and links that have been collected
// for an action in the deployment process for a blueprint instance.
type CollectedElements struct {
	Resources []*ResourceIDInfo
	Children  []*ChildBlueprintIDInfo
	Links     []*LinkIDInfo
	Total     int
}

// LinkIDInfo provides the globally unique ID and logical name of a link.
type LinkIDInfo struct {
	LinkID   string
	LinkName string
}

func (r *LinkIDInfo) ID() string {
	return r.LinkID
}

func (r *LinkIDInfo) LogicalName() string {
	return r.LinkName
}

func (r *LinkIDInfo) Kind() state.ElementKind {
	return state.LinkElement
}

// ResourceIDInfo provides the globally unique ID and logical name of a resource.
type ResourceIDInfo struct {
	ResourceID   string
	ResourceName string
}

func (r *ResourceIDInfo) ID() string {
	return r.ResourceID
}

func (r *ResourceIDInfo) LogicalName() string {
	return r.ResourceName
}

func (r *ResourceIDInfo) Kind() state.ElementKind {
	return state.ResourceElement
}

// ChildBlueprintIDInfo provides the globally unique ID and logical name of a child blueprint.
type ChildBlueprintIDInfo struct {
	ChildInstanceID string
	ChildName       string
}

func (r *ChildBlueprintIDInfo) ID() string {
	return r.ChildInstanceID
}

func (r *ChildBlueprintIDInfo) LogicalName() string {
	return r.ChildName
}

func (r *ChildBlueprintIDInfo) Kind() state.ElementKind {
	return state.ChildElement
}
