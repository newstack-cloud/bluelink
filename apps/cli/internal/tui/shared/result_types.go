package shared

// ElementFailure represents a failure for a specific element with its root cause reasons.
type ElementFailure struct {
	ElementName    string
	ElementPath    string // Full path like "children.notifications::resources.notificationQueue"
	ElementType    string // "resource", "child", or "link"
	FailureReasons []string
}

// InterruptedElement represents an element that was interrupted during an operation.
type InterruptedElement struct {
	ElementName string
	ElementPath string // Full path like "children.notifications::resources.notificationQueue"
	ElementType string // "resource", "child", or "link"
}

// SuccessfulElement represents an element that completed successfully.
type SuccessfulElement struct {
	ElementName string
	ElementPath string // Full path like "children.notifications::resources.notificationQueue"
	ElementType string // "resource", "child", or "link"
	Action      string // "created", "updated", "destroyed", etc.
}

// BuildMapKey builds a path-based key for map lookups.
func BuildMapKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// BuildElementPath constructs a full path like "children.notifications::resources.queue".
func BuildElementPath(parentPath, elementType, elementName string) string {
	segment := elementType + "." + elementName
	if parentPath == "" {
		return segment
	}
	return parentPath + "::" + segment
}

// LookupByKey looks up an item by path-based key, falling back to simple name.
// This is a generic helper used by the result collectors.
func LookupByKey[T any](m map[string]*T, pathKey, name string) *T {
	if item, ok := m[pathKey]; ok {
		return item
	}
	if item, ok := m[name]; ok {
		return item
	}
	return nil
}
