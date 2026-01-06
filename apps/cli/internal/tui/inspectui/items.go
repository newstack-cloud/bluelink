package inspectui

import (
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/deployui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// buildItemsFromInstanceState creates DeployItems from the current instance state.
// This is used for the inspect view to display all resources, links, and children.
func buildItemsFromInstanceState(
	instanceState *state.InstanceState,
	resourcesByName map[string]*deployui.ResourceDeployItem,
	childrenByName map[string]*deployui.ChildDeployItem,
	linksByName map[string]*deployui.LinkDeployItem,
) []deployui.DeployItem {
	if instanceState == nil {
		return nil
	}

	var items []deployui.DeployItem

	items = appendResourcesFromState(items, instanceState, resourcesByName)
	items = appendChildrenFromState(items, instanceState, childrenByName, resourcesByName, linksByName)
	items = appendLinksFromState(items, instanceState, linksByName)

	return items
}

func appendResourcesFromState(
	items []deployui.DeployItem,
	instanceState *state.InstanceState,
	resourcesByName map[string]*deployui.ResourceDeployItem,
) []deployui.DeployItem {
	for _, resourceState := range instanceState.Resources {
		item := &deployui.ResourceDeployItem{
			Name:          resourceState.Name,
			ResourceID:    resourceState.ResourceID,
			ResourceType:  resourceState.Type,
			Action:        shared.ActionInspect,
			Status:        resourceState.Status,
			ResourceState: resourceState,
		}
		resourcesByName[resourceState.Name] = item
		items = append(items, deployui.DeployItem{
			Type:          deployui.ItemTypeResource,
			Resource:      item,
			InstanceState: instanceState,
		})
	}
	return items
}

func appendChildrenFromState(
	items []deployui.DeployItem,
	instanceState *state.InstanceState,
	childrenByName map[string]*deployui.ChildDeployItem,
	resourcesByName map[string]*deployui.ResourceDeployItem,
	linksByName map[string]*deployui.LinkDeployItem,
) []deployui.DeployItem {
	for name, childState := range instanceState.ChildBlueprints {
		item := &deployui.ChildDeployItem{
			Name:            name,
			ChildInstanceID: childState.InstanceID,
			Action:          shared.ActionInspect,
			Status:          childState.Status,
			Changes:         &changes.BlueprintChanges{},
		}
		childrenByName[name] = item
		items = append(items, deployui.MakeChildDeployItem(
			item,
			&changes.BlueprintChanges{},
			childState,
			childrenByName,
			resourcesByName,
			linksByName,
		))
	}
	return items
}

func appendLinksFromState(
	items []deployui.DeployItem,
	instanceState *state.InstanceState,
	linksByName map[string]*deployui.LinkDeployItem,
) []deployui.DeployItem {
	for linkName, linkState := range instanceState.Links {
		item := &deployui.LinkDeployItem{
			LinkID:        linkState.LinkID,
			LinkName:      linkName,
			ResourceAName: extractResourceAFromLinkName(linkName),
			ResourceBName: extractResourceBFromLinkName(linkName),
			Action:        shared.ActionInspect,
			Status:        linkState.Status,
		}
		linksByName[linkName] = item
		items = append(items, deployui.DeployItem{
			Type:          deployui.ItemTypeLink,
			Link:          item,
			InstanceState: instanceState,
		})
	}
	return items
}

func extractResourceAFromLinkName(linkName string) string {
	if idx := strings.Index(linkName, "::"); idx >= 0 {
		return linkName[:idx]
	}
	return linkName
}

func extractResourceBFromLinkName(linkName string) string {
	if idx := strings.Index(linkName, "::"); idx >= 0 {
		return linkName[idx+2:]
	}
	return ""
}

// ToSplitPaneItems converts a slice of DeployItems to splitpane.Items.
func ToSplitPaneItems(items []deployui.DeployItem) []splitpane.Item {
	return deployui.ToSplitPaneItems(items)
}
