package shared

import (
	"sort"

	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// SectionGrouper provides a generic implementation for grouping items
// into Resources, Child Blueprints, and Links sections.
// It works with any type implementing splitpane.Item.
type SectionGrouper struct {
	MaxExpandDepth int
}

// GroupItems organizes items into sections using the splitpane.Item interface.
func (g *SectionGrouper) GroupItems(items []splitpane.Item, isExpanded func(id string) bool) []splitpane.Section {
	var resources, children, links []splitpane.Item

	for _, item := range items {
		if item.GetParentID() != "" {
			children = append(children, item)
			continue
		}

		switch item.GetItemType() {
		case "resource":
			resources = append(resources, item)
		case "child":
			children = append(children, item)
			children = g.appendExpandedChildren(children, item, isExpanded)
		case "link":
			links = append(links, item)
		}
	}

	SortItems(resources)
	SortItems(links)

	var sections []splitpane.Section

	if len(resources) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Resources",
			Items: resources,
		})
	}

	if len(children) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Child Blueprints",
			Items: children,
		})
	}

	if len(links) > 0 {
		sections = append(sections, splitpane.Section{
			Name:  "Links",
			Items: links,
		})
	}

	return sections
}

// appendExpandedChildren recursively appends children of an expanded item.
func (g *SectionGrouper) appendExpandedChildren(
	children []splitpane.Item,
	item splitpane.Item,
	isExpanded func(id string) bool,
) []splitpane.Item {
	if isExpanded == nil || !isExpanded(item.GetID()) {
		return children
	}

	if item.GetDepth() >= g.MaxExpandDepth {
		return children
	}

	childItems := item.GetChildren()
	SortItems(childItems)

	for _, child := range childItems {
		children = append(children, child)
		if child.IsExpandable() {
			children = g.appendExpandedChildren(children, child, isExpanded)
		}
	}

	return children
}

// SortItems sorts items alphabetically by name.
func SortItems(items []splitpane.Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}
