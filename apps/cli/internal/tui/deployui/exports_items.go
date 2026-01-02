package deployui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure ExportsInstanceItem implements splitpane.Item.
var _ splitpane.Item = (*ExportsInstanceItem)(nil)

// ExportsInstanceItem represents a blueprint instance in the exports view.
// It can be the root instance or any nested child blueprint instance.
type ExportsInstanceItem struct {
	// Name is the display name for the instance
	Name string
	// Path is the full path for lookup (e.g., "childA/childB")
	Path string
	// InstanceID is the blueprint instance ID
	InstanceID string
	// ExportCount is the number of exports for this instance
	ExportCount int
	// Depth is the nesting level for indentation
	Depth int
	// InstanceState holds the full instance state for export access
	InstanceState *state.InstanceState
}

// GetID returns a unique identifier for the item.
func (i *ExportsInstanceItem) GetID() string {
	if i.Path == "" {
		return "root"
	}
	return i.Path
}

// GetName returns the display name for the item.
func (i *ExportsInstanceItem) GetName() string {
	return i.Name
}

// GetIcon returns a status icon for the item.
func (i *ExportsInstanceItem) GetIcon(selected bool) string {
	if i.ExportCount == 0 {
		return "○" // Empty circle for no exports
	}
	return "●" // Filled circle for has exports
}

// GetAction returns action badge text.
func (i *ExportsInstanceItem) GetAction() string {
	return fmt.Sprintf("%d exports", i.ExportCount)
}

// GetDepth returns the nesting depth for indentation.
func (i *ExportsInstanceItem) GetDepth() int {
	return i.Depth
}

// GetParentID returns the parent item ID.
func (i *ExportsInstanceItem) GetParentID() string {
	if i.Path == "" {
		return ""
	}
	lastSlash := strings.LastIndex(i.Path, "/")
	if lastSlash == -1 {
		return "root"
	}
	return i.Path[:lastSlash]
}

// GetItemType returns the type for section grouping.
func (i *ExportsInstanceItem) GetItemType() string {
	if i.Depth == 0 {
		return "" // Root instance has no type indicator
	}
	return "child"
}

// IsExpandable indicates whether the item can be expanded in-place.
func (i *ExportsInstanceItem) IsExpandable() bool {
	return false // Exports view uses a flat list, no expansion
}

// CanDrillDown indicates whether the item can be drilled into.
func (i *ExportsInstanceItem) CanDrillDown() bool {
	return false // No drill-down in exports view
}

// GetChildren returns child items (none for exports view).
func (i *ExportsInstanceItem) GetChildren() []splitpane.Item {
	return nil
}

// BuildInstanceHierarchy builds a flat list of ExportsInstanceItems from
// the instance state hierarchy, suitable for display in the exports view.
func BuildInstanceHierarchy(root *state.InstanceState, rootName string) []splitpane.Item {
	if root == nil {
		return nil
	}

	var items []splitpane.Item

	// Add root instance
	displayName := rootName
	if displayName == "" {
		displayName = "(root)"
	}
	items = append(items, &ExportsInstanceItem{
		Name:          displayName,
		Path:          "",
		InstanceID:    root.InstanceID,
		ExportCount:   len(root.Exports),
		Depth:         0,
		InstanceState: root,
	})

	// Recursively add children
	addChildInstances(&items, root, "", 1)

	return items
}

// addChildInstances recursively adds child instances to the items list.
func addChildInstances(items *[]splitpane.Item, parent *state.InstanceState, parentPath string, depth int) {
	if parent == nil || len(parent.ChildBlueprints) == 0 {
		return
	}

	// Sort child names for consistent ordering
	childNames := make([]string, 0, len(parent.ChildBlueprints))
	for name := range parent.ChildBlueprints {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for _, childName := range childNames {
		child := parent.ChildBlueprints[childName]
		if child == nil {
			continue
		}

		path := joinInstancePath(parentPath, childName)

		*items = append(*items, &ExportsInstanceItem{
			Name:          childName,
			Path:          path,
			InstanceID:    child.InstanceID,
			ExportCount:   len(child.Exports),
			Depth:         depth,
			InstanceState: child,
		})

		// Recurse into grandchildren
		addChildInstances(items, child, path, depth+1)
	}
}

// joinInstancePath joins path segments with a slash.
func joinInstancePath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "/" + child
}
