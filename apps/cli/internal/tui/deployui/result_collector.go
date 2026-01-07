package deployui

import (
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
)

// Result collection methods for DeployModel.
// These methods scan deployment items to collect successful operations,
// failures, and interrupted elements for the deployment overview.

// ResultCollector encapsulates the state needed for collecting deployment results.
// This pattern reduces parameter counts by grouping related data together.
// Exported for testing purposes.
type ResultCollector struct {
	ResourcesByName map[string]*ResourceDeployItem
	ChildrenByName  map[string]*ChildDeployItem
	LinksByName     map[string]*LinkDeployItem
	Successful      []SuccessfulElement
	Failures        []ElementFailure
	Interrupted     []InterruptedElement
}

// resultCollector is an alias for internal use within this package.
type resultCollector = ResultCollector

// collectDeploymentResults scans all items to collect successful operations,
// failures, and interrupted elements. This provides the data for the deployment overview.
// It traverses the hierarchy to build full element paths.
func (m *DeployModel) collectDeploymentResults() {
	collector := &resultCollector{
		ResourcesByName: m.resourcesByName,
		ChildrenByName:  m.childrenByName,
		LinksByName:     m.linksByName,
	}

	collector.CollectFromItems(m.items, "")

	m.successfulElements = collector.Successful
	m.elementFailures = collector.Failures
	m.interruptedElements = collector.Interrupted
}

// collectFromItems recursively collects successful operations, failures, and interruptions from items,
// building full paths as it traverses the hierarchy.
func (c *ResultCollector) CollectFromItems(items []DeployItem, parentPath string) {
	for _, item := range items {
		switch item.Type {
		case ItemTypeResource:
			if item.Resource != nil {
				path := buildElementPath(parentPath, "resources", item.Resource.Name)
				c.CollectResourceResult(item.Resource, path)
			}
		case ItemTypeChild:
			if item.Child != nil {
				path := buildElementPath(parentPath, "children", item.Child.Name)
				c.CollectChildResult(item.Child, path)

				if item.Changes != nil {
					c.CollectFromChanges(item.Changes, path, item.Child.Name)
				}
			}
		case ItemTypeLink:
			if item.Link != nil {
				path := buildElementPath(parentPath, "links", item.Link.LinkName)
				c.CollectLinkResult(item.Link, path)
			}
		}
	}
}

// collectFromChanges recursively collects results from nested blueprint changes.
// The pathPrefix is used for map key lookups (e.g., "parentChild/childName"),
// while parentPath is used for display (e.g., "children.parentChild::children.childName").
func (c *ResultCollector) CollectFromChanges(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	if blueprintChanges == nil {
		return
	}

	c.collectNestedResources(blueprintChanges, parentPath, pathPrefix)
	c.collectNestedChildren(blueprintChanges, parentPath, pathPrefix)
}

func (c *ResultCollector) collectNestedResources(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for resourceName := range blueprintChanges.NewResources {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.ResourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			c.CollectResourceResult(resource, path)
		}
	}
	for resourceName := range blueprintChanges.ResourceChanges {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.ResourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			c.CollectResourceResult(resource, path)
		}
	}
}

func (c *ResultCollector) collectNestedChildren(blueprintChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for childName, nc := range blueprintChanges.NewChildren {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(c.ChildrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			c.CollectChildResult(child, path)

			childChanges := &changes.BlueprintChanges{
				NewResources: nc.NewResources,
				NewChildren:  nc.NewChildren,
			}
			c.CollectFromChanges(childChanges, path, childKey)
		}
	}
	for childName, cc := range blueprintChanges.ChildChanges {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(c.ChildrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			c.CollectChildResult(child, path)

			ccCopy := cc
			c.CollectFromChanges(&ccCopy, path, childKey)
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

func (c *ResultCollector) CollectResourceResult(item *ResourceDeployItem, path string) {
	if IsFailedResourceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.Failures = append(c.Failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "resource",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedResourceStatus(item.Status) {
		c.Interrupted = append(c.Interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "resource",
		})
		return
	}
	if IsSuccessResourceStatus(item.Status) {
		c.Successful = append(c.Successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "resource",
			Action:      ResourceStatusToAction(item.Status),
		})
	}
}

func (c *ResultCollector) CollectChildResult(item *ChildDeployItem, path string) {
	if IsFailedInstanceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.Failures = append(c.Failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    "child",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedInstanceStatus(item.Status) {
		c.Interrupted = append(c.Interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
		return
	}
	if IsSuccessInstanceStatus(item.Status) {
		c.Successful = append(c.Successful, SuccessfulElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
			Action:      InstanceStatusToAction(item.Status),
		})
	}
}

func (c *ResultCollector) CollectLinkResult(item *LinkDeployItem, path string) {
	if IsFailedLinkStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.Failures = append(c.Failures, ElementFailure{
			ElementName:    item.LinkName,
			ElementPath:    path,
			ElementType:    "link",
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedLinkStatus(item.Status) {
		c.Interrupted = append(c.Interrupted, InterruptedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
		return
	}
	if IsSuccessLinkStatus(item.Status) {
		c.Successful = append(c.Successful, SuccessfulElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
			Action:      LinkStatusToAction(item.Status),
		})
	}
}

// Helper functions for link name parsing.

// ExtractResourceAFromLinkName extracts the first resource name from a link name (format: "resourceA::resourceB").
func ExtractResourceAFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// ExtractResourceBFromLinkName extracts the second resource name from a link name (format: "resourceA::resourceB").
func ExtractResourceBFromLinkName(linkName string) string {
	parts := strings.Split(linkName, "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
