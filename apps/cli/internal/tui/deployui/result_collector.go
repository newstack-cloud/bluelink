package deployui

import (
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
)

// Result collection methods for DeployModel.
// These methods scan deployment items to collect successful operations,
// failures, and interrupted elements for the deployment overview.

// resultCollector encapsulates the state needed for collecting deployment results.
// This pattern reduces parameter counts by grouping related data together.
type resultCollector struct {
	resourcesByName map[string]*ResourceDeployItem
	childrenByName  map[string]*ChildDeployItem
	linksByName     map[string]*LinkDeployItem
	successful      []SuccessfulElement
	failures        []ElementFailure
	interrupted     []InterruptedElement
}

// collectDeploymentResults scans all items to collect successful operations,
// failures, and interrupted elements. This provides the data for the deployment overview.
// It traverses the hierarchy to build full element paths.
func (m *DeployModel) collectDeploymentResults() {
	collector := &resultCollector{
		resourcesByName: m.resourcesByName,
		childrenByName:  m.childrenByName,
		linksByName:     m.linksByName,
	}

	collector.collectFromItems(m.items, "")

	m.successfulElements = collector.successful
	m.elementFailures = collector.failures
	m.interruptedElements = collector.interrupted
}

// collectFromItems recursively collects successful operations, failures, and interruptions from items,
// building full paths as it traverses the hierarchy.
func (c *resultCollector) collectFromItems(items []DeployItem, parentPath string) {
	for _, item := range items {
		switch item.Type {
		case ItemTypeResource:
			if item.Resource != nil {
				path := buildElementPath(parentPath, "resources", item.Resource.Name)
				c.collectResourceResult(item.Resource, path)
			}
		case ItemTypeChild:
			if item.Child != nil {
				path := buildElementPath(parentPath, "children", item.Child.Name)
				c.collectChildResult(item.Child, path)

				if item.Changes != nil {
					c.collectFromChanges(item.Changes, path, item.Child.Name)
				}
			}
		case ItemTypeLink:
			if item.Link != nil {
				path := buildElementPath(parentPath, "links", item.Link.LinkName)
				c.collectLinkResult(item.Link, path)
			}
		}
	}
}

// collectFromChanges recursively collects results from nested blueprint changes.
// The pathPrefix is used for map key lookups (e.g., "parentChild/childName"),
// while parentPath is used for display (e.g., "children.parentChild::children.childName").
func (c *resultCollector) collectFromChanges(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	if blueprintChanges == nil {
		return
	}

	c.collectNestedResources(blueprintChanges, parentPath, pathPrefix)
	c.collectNestedChildren(blueprintChanges, parentPath, pathPrefix)
}

func (c *resultCollector) collectNestedResources(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for resourceName := range blueprintChanges.NewResources {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			c.collectResourceResult(resource, path)
		}
	}
	for resourceName := range blueprintChanges.ResourceChanges {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			c.collectResourceResult(resource, path)
		}
	}
}

func (c *resultCollector) collectNestedChildren(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for childName, nc := range blueprintChanges.NewChildren {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(c.childrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			c.collectChildResult(child, path)

			childChanges := &changes.BlueprintChanges{
				NewResources: nc.NewResources,
				NewChildren:  nc.NewChildren,
			}
			c.collectFromChanges(childChanges, path, childKey)
		}
	}
	for childName, cc := range blueprintChanges.ChildChanges {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(c.childrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			c.collectChildResult(child, path)

			ccCopy := cc
			c.collectFromChanges(&ccCopy, path, childKey)
		}
	}
}

// Re-export shared helper functions for internal use.
var (
	buildMapKey      = shared.BuildMapKey
	buildElementPath = shared.BuildElementPath
)

// lookupResource looks up a resource by path-based key, falling back to simple name.
func lookupResource(m map[string]*ResourceDeployItem, pathKey, name string) *ResourceDeployItem {
	return shared.LookupByKey(m, pathKey, name)
}

// lookupChild looks up a child by path-based key, falling back to simple name.
func lookupChild(m map[string]*ChildDeployItem, pathKey, name string) *ChildDeployItem {
	return shared.LookupByKey(m, pathKey, name)
}

func (c *resultCollector) collectResourceResult(item *ResourceDeployItem, path string) {
	if IsFailedResourceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "resource",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedResourceStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "resource",
		})
		return
	}
	if IsSuccessResourceStatus(item.Status) {
		c.successful = append(c.successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "resource",
			Action:      ResourceStatusToAction(item.Status),
		})
	}
}

func (c *resultCollector) collectChildResult(item *ChildDeployItem, path string) {
	if IsFailedInstanceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "child",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedInstanceStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
		return
	}
	if IsSuccessInstanceStatus(item.Status) {
		c.successful = append(c.successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
			Action:      InstanceStatusToAction(item.Status),
		})
	}
}

func (c *resultCollector) collectLinkResult(item *LinkDeployItem, path string) {
	if IsFailedLinkStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.LinkName,
			ElementPath:    path,
			ElementType:    "link",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedLinkStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
		return
	}
	if IsSuccessLinkStatus(item.Status) {
		c.successful = append(c.successful, SuccessfulElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
			Action:      LinkStatusToAction(item.Status),
		})
	}
}

// Helper functions for link name parsing.

func extractResourceAFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func extractResourceBFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
