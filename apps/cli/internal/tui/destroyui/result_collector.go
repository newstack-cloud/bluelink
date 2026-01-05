package destroyui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
)

// resultCollector encapsulates the state needed for collecting destroy results.
type resultCollector struct {
	resourcesByName map[string]*ResourceDestroyItem
	childrenByName  map[string]*ChildDestroyItem
	linksByName     map[string]*LinkDestroyItem
	destroyed       []DestroyedElement
	failures        []ElementFailure
	interrupted     []InterruptedElement
}

// collectDestroyResults scans all items to collect destroyed elements,
// failures, and interrupted elements. This provides the data for the destroy overview.
// It traverses the hierarchy to build full element paths.
func (m *DestroyModel) collectDestroyResults() {
	collector := &resultCollector{
		resourcesByName: m.resourcesByName,
		childrenByName:  m.childrenByName,
		linksByName:     m.linksByName,
	}

	collector.collectFromItems(m.items, "")

	m.destroyedElements = collector.destroyed
	m.elementFailures = collector.failures
	m.interruptedElements = collector.interrupted
}

// collectFromItems recursively collects destroyed, failed, and interrupted elements,
// building full paths as it traverses the hierarchy.
func (c *resultCollector) collectFromItems(items []DestroyItem, parentPath string) {
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
func (c *resultCollector) collectFromChanges(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	if bpChanges == nil {
		return
	}

	c.collectNestedResources(bpChanges, parentPath, pathPrefix)
	c.collectNestedLinks(bpChanges, parentPath, pathPrefix)
	c.collectNestedChildren(bpChanges, parentPath, pathPrefix)
}

func (c *resultCollector) collectNestedResources(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for _, resourceName := range bpChanges.RemovedResources {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			c.collectResourceResult(resource, path)
		}
	}
	for resourceName := range bpChanges.ResourceChanges {
		resourceKey := buildMapKey(pathPrefix, resourceName)
		resource := lookupResource(c.resourcesByName, resourceKey, resourceName)
		if resource != nil {
			path := buildElementPath(parentPath, "resources", resourceName)
			c.collectResourceResult(resource, path)
		}
	}
}

func (c *resultCollector) collectNestedLinks(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for _, linkName := range bpChanges.RemovedLinks {
		linkKey := buildMapKey(pathPrefix, linkName)
		link := lookupLink(c.linksByName, linkKey, linkName)
		if link != nil {
			path := buildElementPath(parentPath, "links", linkName)
			c.collectLinkResult(link, path)
		}
	}
}

func (c *resultCollector) collectNestedChildren(bpChanges *changes.BlueprintChanges, parentPath, pathPrefix string) {
	for _, childName := range bpChanges.RemovedChildren {
		childKey := buildMapKey(pathPrefix, childName)
		child := lookupChild(c.childrenByName, childKey, childName)
		if child != nil {
			path := buildElementPath(parentPath, "children", childName)
			c.collectChildResult(child, path)

			if childChanges, ok := bpChanges.ChildChanges[childName]; ok {
				c.collectFromChanges(&childChanges, path, childKey)
			}
		}
	}
	for childName, cc := range bpChanges.ChildChanges {
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

// buildMapKey builds a path-based key for map lookups.
func buildMapKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// lookupResource looks up a resource by path-based key, falling back to simple name.
func lookupResource(m map[string]*ResourceDestroyItem, pathKey, name string) *ResourceDestroyItem {
	if r, ok := m[pathKey]; ok {
		return r
	}
	if r, ok := m[name]; ok {
		return r
	}
	return nil
}

// lookupChild looks up a child by path-based key, falling back to simple name.
func lookupChild(m map[string]*ChildDestroyItem, pathKey, name string) *ChildDestroyItem {
	if ch, ok := m[pathKey]; ok {
		return ch
	}
	if ch, ok := m[name]; ok {
		return ch
	}
	return nil
}

// lookupLink looks up a link by path-based key, falling back to simple name.
func lookupLink(m map[string]*LinkDestroyItem, pathKey, name string) *LinkDestroyItem {
	if l, ok := m[pathKey]; ok {
		return l
	}
	if l, ok := m[name]; ok {
		return l
	}
	return nil
}

// buildElementPath constructs a full path like "children.notifications::resources.queue".
func buildElementPath(parentPath, elementType, elementName string) string {
	segment := elementType + "." + elementName
	if parentPath == "" {
		return segment
	}
	return parentPath + "::" + segment
}

func (c *resultCollector) collectResourceResult(item *ResourceDestroyItem, path string) {
	if IsFailedResourceStatus(item.Status) && len(item.FailureReasons) > 0 {
		c.failures = append(c.failures, ElementFailure{
			ElementName:    item.Name,
			ElementPath:    path,
			ElementType:    item.ResourceType,
			FailureReasons: item.FailureReasons,
		})
		return
	}
	if IsInterruptedResourceStatus(item.Status) {
		c.interrupted = append(c.interrupted, InterruptedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: item.ResourceType,
		})
		return
	}
	if IsSuccessResourceStatus(item.Status) {
		c.destroyed = append(c.destroyed, DestroyedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: item.ResourceType,
		})
	}
}

func (c *resultCollector) collectChildResult(item *ChildDestroyItem, path string) {
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
		c.destroyed = append(c.destroyed, DestroyedElement{
			ElementName: item.Name,
			ElementPath: path,
			ElementType: "child",
		})
	}
}

func (c *resultCollector) collectLinkResult(item *LinkDestroyItem, path string) {
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
		c.destroyed = append(c.destroyed, DestroyedElement{
			ElementName: item.LinkName,
			ElementPath: path,
			ElementType: "link",
		})
	}
}
