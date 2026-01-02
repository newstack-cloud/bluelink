package stageui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// Ensure StageExportsInstanceItem implements splitpane.Item
var _ splitpane.Item = (*StageExportsInstanceItem)(nil)

// StageExportsInstanceItem represents a blueprint instance in the exports change view.
// It shows export changes for the root instance or any child blueprint.
type StageExportsInstanceItem struct {
	// Name is the display name for the instance
	Name string
	// Path is the full path for hierarchy (e.g., "childA/childB")
	Path string
	// Depth is the nesting level for indentation
	Depth int
	// NewCount is the number of new exports
	NewCount int
	// ModifiedCount is the number of modified exports
	ModifiedCount int
	// RemovedCount is the number of removed exports
	RemovedCount int
	// UnchangedCount is the number of unchanged exports
	UnchangedCount int
	// Changes holds the BlueprintChanges for this instance
	Changes *changes.BlueprintChanges
}

// GetID returns a unique identifier for the item.
func (i *StageExportsInstanceItem) GetID() string {
	if i.Path == "" {
		return "root"
	}
	return i.Path
}

// GetName returns the display name for the item.
func (i *StageExportsInstanceItem) GetName() string {
	return i.Name
}

// GetIcon returns a status icon for the item.
func (i *StageExportsInstanceItem) GetIcon(selected bool) string {
	// Use stage UI consistent icons based on primary action
	if i.NewCount > 0 {
		return "✓" // CREATE icon for new exports
	}
	if i.ModifiedCount > 0 {
		return "±" // UPDATE icon for modified exports
	}
	if i.RemovedCount > 0 {
		return "-" // DELETE icon for removed exports
	}
	return "○" // NO_CHANGE icon
}

// GetAction returns action badge text summarizing export changes.
func (i *StageExportsInstanceItem) GetAction() string {
	var parts []string
	if i.NewCount > 0 {
		parts = append(parts, fmt.Sprintf("%d new", i.NewCount))
	}
	if i.ModifiedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", i.ModifiedCount))
	}
	if i.RemovedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", i.RemovedCount))
	}
	if len(parts) == 0 {
		if i.UnchangedCount > 0 {
			return fmt.Sprintf("%d unchanged", i.UnchangedCount)
		}
		return "no exports"
	}
	return strings.Join(parts, ", ")
}

// GetDepth returns the nesting depth for indentation.
func (i *StageExportsInstanceItem) GetDepth() int {
	return i.Depth
}

// GetParentID returns the parent item ID.
func (i *StageExportsInstanceItem) GetParentID() string {
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
func (i *StageExportsInstanceItem) GetItemType() string {
	if i.Depth == 0 {
		return "" // Root instance has no type indicator
	}
	return "child"
}

// IsExpandable indicates whether the item can be expanded in-place.
func (i *StageExportsInstanceItem) IsExpandable() bool {
	return false // Exports view uses a flat list, no expansion
}

// CanDrillDown indicates whether the item can be drilled into.
func (i *StageExportsInstanceItem) CanDrillDown() bool {
	return false // No drill-down in exports view
}

// GetChildren returns child items (none for exports view).
func (i *StageExportsInstanceItem) GetChildren() []splitpane.Item {
	return nil
}

// HasExportChanges returns true if there are any export changes.
func (i *StageExportsInstanceItem) HasExportChanges() bool {
	return i.NewCount > 0 || i.ModifiedCount > 0 || i.RemovedCount > 0
}

// countExportChanges counts exports by type, correctly categorizing exports
// in ExportChanges with nil prevValue as "new" exports (for new deployments).
// Exports that are resolve-on-deploy placeholders (NewValue is nil and in ResolveOnDeploy list)
// are counted as unchanged when there are no other actual changes in the changeset,
// since they represent values that will be re-resolved to the same values on deploy.
// When there are real changes, resolve-on-deploy exports are counted as modified
// since they represent values that will change upon deployment.
func countExportChanges(bc *changes.BlueprintChanges) (newCount, modifiedCount, removedCount, unchangedCount int) {
	if bc == nil {
		return 0, 0, 0, 0
	}

	// Check if there are any actual changes in the changeset
	hasActualChanges := hasActualChangesInChangeset(bc)

	// Count explicit new exports
	newCount = len(bc.NewExports)

	// Count exports in ExportChanges, but those with nil prevValue are actually new.
	// Resolve-on-deploy placeholders are counted as unchanged when there are no actual changes.
	for exportName, change := range bc.ExportChanges {
		if change.PrevValue == nil {
			newCount++
		} else if !hasActualChanges && isResolveOnDeployPlaceholder(exportName, &change, bc.ResolveOnDeploy) {
			// Count resolve-on-deploy placeholders as unchanged when there are no actual changes
			unchangedCount++
		} else {
			modifiedCount++
		}
	}

	removedCount = len(bc.RemovedExports)
	unchangedCount += len(bc.UnchangedExports)

	return newCount, modifiedCount, removedCount, unchangedCount
}

// hasActualChangesInChangeset checks if there are any actual changes in the changeset
// (new resources, resource changes, removed resources, new/removed children, etc.)
// excluding resolve-on-deploy placeholders. This recursively checks child changesets.
func hasActualChangesInChangeset(bc *changes.BlueprintChanges) bool {
	if bc == nil {
		return false
	}

	// Check for resource changes
	if len(bc.NewResources) > 0 || len(bc.RemovedResources) > 0 {
		return true
	}
	for _, rc := range bc.ResourceChanges {
		if provider.HasAnyChanges(&rc) {
			return true
		}
	}

	// Check for new or removed children
	if len(bc.NewChildren) > 0 || len(bc.RemovedChildren) > 0 {
		return true
	}

	// Recursively check child changesets for actual changes
	for _, childChanges := range bc.ChildChanges {
		if hasActualChangesInChangeset(&childChanges) {
			return true
		}
	}

	// Check for link changes
	if len(bc.RemovedLinks) > 0 {
		return true
	}

	// Check for export changes (excluding resolve-on-deploy placeholders)
	if len(bc.NewExports) > 0 || len(bc.RemovedExports) > 0 {
		return true
	}
	for exportName, change := range bc.ExportChanges {
		if change.PrevValue == nil {
			// This is actually a new export
			return true
		}
		if !isResolveOnDeployPlaceholder(exportName, &change, bc.ResolveOnDeploy) {
			// This is an actual modification, not just a resolve-on-deploy placeholder
			return true
		}
	}

	return false
}

// isResolveOnDeployPlaceholder checks if an export change is just a placeholder
// for a value that will be resolved on deploy, not an actual change.
// This is the case when newValue is nil and the export path is in ResolveOnDeploy.
func isResolveOnDeployPlaceholder(exportName string, change *provider.FieldChange, resolveOnDeploy []string) bool {
	if change.NewValue != nil {
		return false
	}
	exportPath := "exports." + exportName
	for _, rod := range resolveOnDeploy {
		if rod == exportPath {
			return true
		}
	}
	return false
}

// BuildExportChangeHierarchy builds a flat list of StageExportsInstanceItems
// from BlueprintChanges, suitable for display in the exports view.
func BuildExportChangeHierarchy(
	rootChanges *changes.BlueprintChanges,
	rootName string,
) []splitpane.Item {
	if rootChanges == nil {
		return nil
	}

	var items []splitpane.Item

	// Add root instance
	displayName := rootName
	if displayName == "" {
		displayName = "(root)"
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(rootChanges)

	rootItem := &StageExportsInstanceItem{
		Name:           displayName,
		Path:           "",
		Depth:          0,
		NewCount:       newCount,
		ModifiedCount:  modifiedCount,
		RemovedCount:   removedCount,
		UnchangedCount: unchangedCount,
		Changes:        rootChanges,
	}
	items = append(items, rootItem)

	// Recursively add children
	addChildExportItems(&items, rootChanges, "", 1)

	return items
}

// addChildExportItems recursively adds child instances to the items list.
func addChildExportItems(
	items *[]splitpane.Item,
	parentChanges *changes.BlueprintChanges,
	parentPath string,
	depth int,
) {
	if parentChanges == nil {
		return
	}

	// Collect all child names from NewChildren and ChildChanges
	childNames := make(map[string]bool)
	for name := range parentChanges.NewChildren {
		childNames[name] = true
	}
	for name := range parentChanges.ChildChanges {
		childNames[name] = true
	}

	// Sort child names for consistent ordering
	sortedNames := make([]string, 0, len(childNames))
	for name := range childNames {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, childName := range sortedNames {
		path := joinExportPath(parentPath, childName)

		// Check if this is a new child or an existing one with changes
		if newChild, isNew := parentChanges.NewChildren[childName]; isNew {
			// New child blueprint - wrap in BlueprintChanges for consistency
			childChanges := &changes.BlueprintChanges{
				NewResources:    newChild.NewResources,
				NewChildren:     newChild.NewChildren,
				NewExports:      newChild.NewExports,
				ResolveOnDeploy: newChild.ResolveOnDeploy,
			}

			item := &StageExportsInstanceItem{
				Name:          childName,
				Path:          path,
				Depth:         depth,
				NewCount:      len(newChild.NewExports),
				ModifiedCount: 0,
				RemovedCount:  0,
				Changes:       childChanges,
			}
			*items = append(*items, item)

			// Recurse into new children's nested children
			addNewChildExportItems(items, newChild.NewChildren, path, depth+1)
		} else if childChanges, hasChanges := parentChanges.ChildChanges[childName]; hasChanges {
			// Existing child with changes - use countExportChanges to properly categorize
			newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(&childChanges)
			item := &StageExportsInstanceItem{
				Name:           childName,
				Path:           path,
				Depth:          depth,
				NewCount:       newCount,
				ModifiedCount:  modifiedCount,
				RemovedCount:   removedCount,
				UnchangedCount: unchangedCount,
				Changes:        &childChanges,
			}
			*items = append(*items, item)

			// Recurse into child's nested children
			addChildExportItems(items, &childChanges, path, depth+1)
		}
	}
}

// addNewChildExportItems recursively adds new child blueprint items.
func addNewChildExportItems(
	items *[]splitpane.Item,
	newChildren map[string]changes.NewBlueprintDefinition,
	parentPath string,
	depth int,
) {
	if len(newChildren) == 0 {
		return
	}

	// Sort child names for consistent ordering
	childNames := make([]string, 0, len(newChildren))
	for name := range newChildren {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for _, childName := range childNames {
		child := newChildren[childName]
		path := joinExportPath(parentPath, childName)

		childChanges := &changes.BlueprintChanges{
			NewResources:    child.NewResources,
			NewChildren:     child.NewChildren,
			NewExports:      child.NewExports,
			ResolveOnDeploy: child.ResolveOnDeploy,
		}

		item := &StageExportsInstanceItem{
			Name:     childName,
			Path:     path,
			Depth:    depth,
			NewCount: len(child.NewExports),
			Changes:  childChanges,
		}
		*items = append(*items, item)

		// Recurse into nested new children
		addNewChildExportItems(items, child.NewChildren, path, depth+1)
	}
}

// joinExportPath joins path segments with a slash.
func joinExportPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "/" + child
}

// HasAnyExportChanges checks if BlueprintChanges has any real export changes
// (either directly or in child blueprints).
// Export changes that are just "resolve on deploy" placeholders (newValue is nil
// and path is in ResolveOnDeploy) are not considered actual changes.
func HasAnyExportChanges(bc *changes.BlueprintChanges) bool {
	if bc == nil {
		return false
	}

	// Check root-level exports
	if len(bc.NewExports) > 0 || len(bc.RemovedExports) > 0 {
		return true
	}

	// Check ExportChanges, but filter out resolve-on-deploy placeholders
	for name, change := range bc.ExportChanges {
		if !isResolveOnDeployPlaceholder(name, &change, bc.ResolveOnDeploy) {
			return true
		}
	}

	// Check new children
	for _, newChild := range bc.NewChildren {
		if len(newChild.NewExports) > 0 {
			return true
		}
		if hasNewChildExports(newChild.NewChildren) {
			return true
		}
	}

	// Check existing children with changes
	for _, childChanges := range bc.ChildChanges {
		if HasAnyExportChanges(&childChanges) {
			return true
		}
	}

	return false
}

// hasNewChildExports recursively checks if any new children have exports.
func hasNewChildExports(newChildren map[string]changes.NewBlueprintDefinition) bool {
	for _, child := range newChildren {
		if len(child.NewExports) > 0 {
			return true
		}
		if hasNewChildExports(child.NewChildren) {
			return true
		}
	}
	return false
}

// HasAnyExportsToShow checks if BlueprintChanges has any exports to display
// (including resolve-on-deploy placeholders). This is used when there are
// actual deployment changes and we want to show all exports.
func HasAnyExportsToShow(bc *changes.BlueprintChanges) bool {
	if bc == nil {
		return false
	}

	// Check root-level exports (including ExportChanges which may have placeholders)
	if len(bc.NewExports) > 0 || len(bc.ExportChanges) > 0 || len(bc.RemovedExports) > 0 {
		return true
	}

	// Check new children
	for _, newChild := range bc.NewChildren {
		if len(newChild.NewExports) > 0 {
			return true
		}
		if hasNewChildExports(newChild.NewChildren) {
			return true
		}
	}

	// Check existing children with changes
	for _, childChanges := range bc.ChildChanges {
		if HasAnyExportsToShow(&childChanges) {
			return true
		}
	}

	return false
}
