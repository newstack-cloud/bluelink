package shared

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// RenderSectionHeader renders a styled section header with a separator line.
func RenderSectionHeader(sb *strings.Builder, headerText string, width int, s *styles.Styles) {
	sb.WriteString(s.Header.Render(headerText))
	sb.WriteString("\n")
	sb.WriteString(s.Muted.Render(strings.Repeat("─", ui.SafeWidth(width-4))))
	sb.WriteString("\n\n")
}

// RenderLabelValue renders a label-value pair with muted label styling.
func RenderLabelValue(sb *strings.Builder, label, value string, s *styles.Styles) {
	sb.WriteString(s.Muted.Render(label + ": "))
	sb.WriteString(value)
	sb.WriteString("\n")
}

// FindResourceStateByName finds a resource state by name using the instance state's
// ResourceIDs map to look up the resource ID, then retrieves the state from Resources.
func FindResourceStateByName(instanceState *state.InstanceState, name string) *state.ResourceState {
	if instanceState == nil || instanceState.ResourceIDs == nil || instanceState.Resources == nil {
		return nil
	}
	resourceID, ok := instanceState.ResourceIDs[name]
	if !ok {
		return nil
	}
	return instanceState.Resources[resourceID]
}

// FindChildInstanceIDByPath finds a child blueprint's instance ID by traversing the instance state hierarchy.
// The path format is "childA/childB" where each segment is a child blueprint name.
func FindChildInstanceIDByPath(instanceState *state.InstanceState, path string) string {
	if instanceState == nil || path == "" {
		return ""
	}

	segments := strings.Split(path, "/")
	currentState := instanceState

	for _, childName := range segments {
		if currentState.ChildBlueprints == nil {
			return ""
		}
		childState, ok := currentState.ChildBlueprints[childName]
		if !ok || childState == nil {
			return ""
		}
		currentState = childState
	}

	return currentState.InstanceID
}

// FindLinkIDByPath finds a link's ID by traversing the instance state hierarchy.
// The path format is "childA/childB/linkName" where the preceding segments are child blueprint names.
func FindLinkIDByPath(instanceState *state.InstanceState, path, linkName string) string {
	if instanceState == nil {
		return ""
	}

	segments := strings.Split(path, "/")
	currentState := instanceState

	for i := 0; i < len(segments)-1; i++ {
		childName := segments[i]
		if currentState.ChildBlueprints == nil {
			return ""
		}
		childState, ok := currentState.ChildBlueprints[childName]
		if !ok || childState == nil {
			return ""
		}
		currentState = childState
	}

	if currentState.Links == nil {
		return ""
	}
	linkState, ok := currentState.Links[linkName]
	if !ok || linkState == nil {
		return ""
	}
	return linkState.LinkID
}

// FindResourceIDByPath finds a resource ID by traversing the instance state hierarchy.
// The path format is "childA/childB/resourceName" where the preceding segments are child blueprint names.
func FindResourceIDByPath(instanceState *state.InstanceState, path, resourceName string) string {
	if instanceState == nil {
		return ""
	}

	segments := strings.Split(path, "/")
	currentState := instanceState

	for i := 0; i < len(segments)-1; i++ {
		childName := segments[i]
		if currentState.ChildBlueprints == nil {
			return ""
		}
		childState, ok := currentState.ChildBlueprints[childName]
		if !ok || childState == nil {
			return ""
		}
		currentState = childState
	}

	if currentState.ResourceIDs == nil {
		return ""
	}
	return currentState.ResourceIDs[resourceName]
}
