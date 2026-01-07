package deployui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type DeployItemsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestDeployItemsTestSuite(t *testing.T) {
	suite.Run(t, new(DeployItemsTestSuite))
}

func (s *DeployItemsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

// GetID tests

func (s *DeployItemsTestSuite) Test_GetID_returns_resource_name() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Name: "myResource"},
	}
	s.Equal("myResource", item.GetID())
}

func (s *DeployItemsTestSuite) Test_GetID_returns_child_name() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "myChild"},
	}
	s.Equal("myChild", item.GetID())
}

func (s *DeployItemsTestSuite) Test_GetID_returns_link_name() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{LinkName: "resA::resB"},
	}
	s.Equal("resA::resB", item.GetID())
}

func (s *DeployItemsTestSuite) Test_GetID_returns_empty_for_nil_resource() {
	item := &DeployItem{Type: ItemTypeResource}
	s.Equal("", item.GetID())
}

// GetName tests

func (s *DeployItemsTestSuite) Test_GetName_returns_same_as_GetID() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Name: "myResource"},
	}
	s.Equal(item.GetID(), item.GetName())
}

// GetIcon tests for resources

func (s *DeployItemsTestSuite) Test_GetIcon_resource_pending() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusUnknown},
	}
	s.Equal("○", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_creating() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusCreating},
	}
	s.Equal("◐", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_created() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusCreated},
	}
	s.Equal("✓", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_failed() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusCreateFailed},
	}
	s.Equal("✗", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_rolling_back() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusRollingBack},
	}
	s.Equal("↺", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_rollback_failed() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusRollbackFailed},
	}
	s.Equal("⚠", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_rollback_complete() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusRollbackComplete},
	}
	s.Equal("⟲", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_interrupted() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusCreateInterrupted},
	}
	s.Equal("⏹", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_skipped() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Skipped: true},
	}
	s.Equal("⊘", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_resource_no_change() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Action: ActionNoChange},
	}
	s.Equal("─", item.GetIcon(false))
}

// GetIcon tests for child blueprints

func (s *DeployItemsTestSuite) Test_GetIcon_child_deploying() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Status: core.InstanceStatusDeploying},
	}
	s.Equal("◐", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_child_deployed() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Status: core.InstanceStatusDeployed},
	}
	s.Equal("✓", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_child_failed() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Status: core.InstanceStatusDeployFailed},
	}
	s.Equal("✗", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_child_skipped() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Skipped: true},
	}
	s.Equal("⊘", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_child_no_change() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Action: ActionNoChange},
	}
	s.Equal("─", item.GetIcon(false))
}

// GetIcon tests for links

func (s *DeployItemsTestSuite) Test_GetIcon_link_creating() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Status: core.LinkStatusCreating},
	}
	s.Equal("◐", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_link_created() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Status: core.LinkStatusCreated},
	}
	s.Equal("✓", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_link_failed() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Status: core.LinkStatusCreateFailed},
	}
	s.Equal("✗", item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_link_skipped() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Skipped: true},
	}
	s.Equal("⊘", item.GetIcon(false))
}

// GetIconStyled tests

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_plain_when_not_styled() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusCreated},
	}
	s.Equal("✓", item.GetIconStyled(s.testStyles, false))
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_styled_for_resource() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Status: core.ResourceStatusCreated},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "✓")
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_warning_for_skipped() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Skipped: true},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "⊘")
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_muted_for_no_change() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Action: ActionNoChange},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "─")
}

// GetAction tests

func (s *DeployItemsTestSuite) Test_GetAction_returns_resource_action() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: &ResourceDeployItem{Action: ActionCreate},
	}
	s.Equal("CREATE", item.GetAction())
}

func (s *DeployItemsTestSuite) Test_GetAction_returns_child_action() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Action: ActionUpdate},
	}
	s.Equal("UPDATE", item.GetAction())
}

func (s *DeployItemsTestSuite) Test_GetAction_returns_link_action() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Action: ActionDelete},
	}
	s.Equal("DELETE", item.GetAction())
}

func (s *DeployItemsTestSuite) Test_GetAction_returns_empty_for_nil() {
	item := &DeployItem{Type: ItemTypeResource}
	s.Equal("", item.GetAction())
}

// GetDepth tests

func (s *DeployItemsTestSuite) Test_GetDepth_returns_depth() {
	item := &DeployItem{Depth: 3}
	s.Equal(3, item.GetDepth())
}

// GetParentID tests

func (s *DeployItemsTestSuite) Test_GetParentID_returns_parent_child() {
	item := &DeployItem{ParentChild: "parentBlueprint"}
	s.Equal("parentBlueprint", item.GetParentID())
}

// GetItemType tests

func (s *DeployItemsTestSuite) Test_GetItemType_returns_type() {
	item := &DeployItem{Type: ItemTypeResource}
	s.Equal("resource", item.GetItemType())
}

// IsExpandable tests

func (s *DeployItemsTestSuite) Test_IsExpandable_true_for_child_with_changes() {
	item := &DeployItem{
		Type:    ItemTypeChild,
		Changes: &changes.BlueprintChanges{},
	}
	s.True(item.IsExpandable())
}

func (s *DeployItemsTestSuite) Test_IsExpandable_true_for_child_with_instance_state() {
	item := &DeployItem{
		Type:          ItemTypeChild,
		InstanceState: &state.InstanceState{},
	}
	s.True(item.IsExpandable())
}

func (s *DeployItemsTestSuite) Test_IsExpandable_false_for_resource() {
	item := &DeployItem{Type: ItemTypeResource}
	s.False(item.IsExpandable())
}

func (s *DeployItemsTestSuite) Test_IsExpandable_false_for_child_without_changes_or_state() {
	item := &DeployItem{Type: ItemTypeChild}
	s.False(item.IsExpandable())
}

// CanDrillDown tests

func (s *DeployItemsTestSuite) Test_CanDrillDown_same_as_IsExpandable() {
	item := &DeployItem{
		Type:    ItemTypeChild,
		Changes: &changes.BlueprintChanges{},
	}
	s.Equal(item.IsExpandable(), item.CanDrillDown())
}

// GetChildren tests

func (s *DeployItemsTestSuite) Test_GetChildren_returns_nil_for_non_child() {
	item := &DeployItem{Type: ItemTypeResource}
	s.Nil(item.GetChildren())
}

func (s *DeployItemsTestSuite) Test_GetChildren_returns_nil_for_child_without_changes_or_state() {
	item := &DeployItem{Type: ItemTypeChild}
	s.Nil(item.GetChildren())
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_from_new_resources() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"newResource": {},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeResource, childItem.Type)
	s.Equal("newResource", childItem.Resource.Name)
	s.Equal(ActionCreate, childItem.Resource.Action)
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_from_resource_changes() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"changedResource": {
					ModifiedFields: []provider.FieldChange{
						{FieldPath: "spec.replicas"},
					},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("changedResource", childItem.Resource.Name)
	s.Equal(ActionUpdate, childItem.Resource.Action)
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_from_removed_resources() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"removedResource"},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("removedResource", childItem.Resource.Name)
	s.Equal(ActionDelete, childItem.Resource.Action)
}

func (s *DeployItemsTestSuite) Test_GetChildren_sets_recreate_for_must_recreate() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"recreateResource": {
					MustRecreate: true,
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ActionRecreate, childItem.Resource.Action)
}

func (s *DeployItemsTestSuite) Test_GetChildren_inherits_skipped_status() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint", Skipped: true},
		Changes: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"newResource": {},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.True(childItem.Resource.Skipped)
}

func (s *DeployItemsTestSuite) Test_GetChildren_adds_unchanged_resources_from_instance_state() {
	// Set Action on the parent child to simulate deploy mode (non-inspect)
	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "childBlueprint", Action: ActionUpdate},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			Resources: map[string]*state.ResourceState{
				"res-123": {
					ResourceID: "res-123",
					Name:       "unchangedResource",
					Type:       "aws/s3/bucket",
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("unchangedResource", childItem.Resource.Name)
	s.Equal(ActionNoChange, childItem.Resource.Action)
}

func (s *DeployItemsTestSuite) Test_GetChildren_discovers_items_from_shared_maps_during_streaming() {
	// This test simulates the streaming scenario where:
	// 1. A child blueprint item exists with empty Changes and nil InstanceState
	// 2. Resources have been added to the shared maps via streaming events
	// 3. GetChildren should discover these resources from the maps

	resourcesByName := make(map[string]*ResourceDeployItem)
	childrenByName := make(map[string]*ChildDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	// Simulate a resource added via streaming event with path-based key
	resourcesByName["streamingChild/streamedResource"] = &ResourceDeployItem{
		Name:         "streamedResource",
		ResourceID:   "res-streaming-123",
		ResourceType: "aws/lambda/function",
		Action:       ActionInspect,
		Status:       core.ResourceStatusCreating,
	}

	// Simulate a nested child added via streaming event
	childrenByName["streamingChild/nestedChild"] = &ChildDeployItem{
		Name:    "nestedChild",
		Action:  ActionInspect,
		Changes: &changes.BlueprintChanges{},
	}

	// Simulate a link added via streaming event
	linksByName["streamingChild/resourceA::resourceB"] = &LinkDeployItem{
		LinkName:      "resourceA::resourceB",
		ResourceAName: "resourceA",
		ResourceBName: "resourceB",
		Action:        ActionInspect,
		Status:        core.LinkStatusCreating,
	}

	// Create a child item representing a child blueprint during streaming
	// Note: both Changes and InstanceState can be nil/empty in streaming
	item := &DeployItem{
		Type:            ItemTypeChild,
		Child:           &ChildDeployItem{Name: "streamingChild", Action: ActionInspect},
		Changes:         &changes.BlueprintChanges{}, // Empty changes
		InstanceState:   nil,                         // No state yet during streaming
		resourcesByName: resourcesByName,
		childrenByName:  childrenByName,
		linksByName:     linksByName,
	}

	children := item.GetChildren()

	// Should find all 3 items from the shared maps
	s.Len(children, 3)

	// Verify the items were discovered correctly
	var foundResource, foundChild, foundLink bool
	for _, child := range children {
		childItem := child.(*DeployItem)
		switch childItem.Type {
		case ItemTypeResource:
			s.Equal("streamedResource", childItem.Resource.Name)
			s.Equal(ActionInspect, childItem.Resource.Action)
			s.Equal(core.ResourceStatusCreating, childItem.Resource.Status)
			foundResource = true
		case ItemTypeChild:
			s.Equal("nestedChild", childItem.Child.Name)
			s.Equal(ActionInspect, childItem.Child.Action)
			foundChild = true
		case ItemTypeLink:
			s.Equal("resourceA::resourceB", childItem.Link.LinkName)
			s.Equal(ActionInspect, childItem.Link.Action)
			foundLink = true
		}
	}
	s.True(foundResource, "should find resource from shared map")
	s.True(foundChild, "should find child from shared map")
	s.True(foundLink, "should find link from shared map")
}

func (s *DeployItemsTestSuite) Test_GetChildren_only_discovers_direct_children_from_shared_maps() {
	// This test ensures that GetChildren only discovers direct children,
	// not grandchildren or items from other parent paths

	resourcesByName := make(map[string]*ResourceDeployItem)

	// Direct child - should be discovered
	resourcesByName["parentChild/directResource"] = &ResourceDeployItem{
		Name:   "directResource",
		Action: ActionInspect,
	}

	// Grandchild - should NOT be discovered
	resourcesByName["parentChild/nestedChild/grandchildResource"] = &ResourceDeployItem{
		Name:   "grandchildResource",
		Action: ActionInspect,
	}

	// Sibling's child - should NOT be discovered
	resourcesByName["otherChild/siblingResource"] = &ResourceDeployItem{
		Name:   "siblingResource",
		Action: ActionInspect,
	}

	item := &DeployItem{
		Type:            ItemTypeChild,
		Child:           &ChildDeployItem{Name: "parentChild", Action: ActionInspect},
		Changes:         &changes.BlueprintChanges{},
		resourcesByName: resourcesByName,
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}

	children := item.GetChildren()

	// Should only find the direct child
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("directResource", childItem.Resource.Name)
}

// resourceStatusIcon tests

func (s *DeployItemsTestSuite) Test_resourceStatusIcon_all_statuses() {
	testCases := []struct {
		status   core.ResourceStatus
		expected string
	}{
		{core.ResourceStatusCreating, "◐"},
		{core.ResourceStatusUpdating, "◐"},
		{core.ResourceStatusDestroying, "◐"},
		{core.ResourceStatusCreated, "✓"},
		{core.ResourceStatusUpdated, "✓"},
		{core.ResourceStatusDestroyed, "✓"},
		{core.ResourceStatusCreateFailed, "✗"},
		{core.ResourceStatusUpdateFailed, "✗"},
		{core.ResourceStatusDestroyFailed, "✗"},
		{core.ResourceStatusRollingBack, "↺"},
		{core.ResourceStatusRollbackFailed, "⚠"},
		{core.ResourceStatusRollbackComplete, "⟲"},
		{core.ResourceStatusCreateInterrupted, "⏹"},
		{core.ResourceStatusUpdateInterrupted, "⏹"},
		{core.ResourceStatusDestroyInterrupted, "⏹"},
		{core.ResourceStatusUnknown, "○"},
	}

	for _, tc := range testCases {
		s.Equal(tc.expected, shared.ResourceStatusIcon(tc.status), "Status: %s", tc.status)
	}
}

// instanceStatusIcon tests

func (s *DeployItemsTestSuite) Test_instanceStatusIcon_all_statuses() {
	testCases := []struct {
		status   core.InstanceStatus
		expected string
	}{
		{core.InstanceStatusPreparing, "○"},
		{core.InstanceStatusDeploying, "◐"},
		{core.InstanceStatusUpdating, "◐"},
		{core.InstanceStatusDestroying, "◐"},
		{core.InstanceStatusDeployed, "✓"},
		{core.InstanceStatusUpdated, "✓"},
		{core.InstanceStatusDestroyed, "✓"},
		{core.InstanceStatusDeployFailed, "✗"},
		{core.InstanceStatusUpdateFailed, "✗"},
		{core.InstanceStatusDestroyFailed, "✗"},
		{core.InstanceStatusDeployRollingBack, "↺"},
		{core.InstanceStatusDeployRollbackFailed, "⚠"},
		{core.InstanceStatusDeployRollbackComplete, "⟲"},
		{core.InstanceStatusDeployInterrupted, "⏹"},
		{core.InstanceStatus(999), "○"}, // Unknown/default case
	}

	for _, tc := range testCases {
		s.Equal(tc.expected, shared.InstanceStatusIcon(tc.status), "Status: %s", tc.status)
	}
}

// linkStatusIcon tests

func (s *DeployItemsTestSuite) Test_linkStatusIcon_all_statuses() {
	testCases := []struct {
		status   core.LinkStatus
		expected string
	}{
		{core.LinkStatusCreating, "◐"},
		{core.LinkStatusUpdating, "◐"},
		{core.LinkStatusDestroying, "◐"},
		{core.LinkStatusCreated, "✓"},
		{core.LinkStatusUpdated, "✓"},
		{core.LinkStatusDestroyed, "✓"},
		{core.LinkStatusCreateFailed, "✗"},
		{core.LinkStatusUpdateFailed, "✗"},
		{core.LinkStatusDestroyFailed, "✗"},
		{core.LinkStatusCreateRollingBack, "↺"},
		{core.LinkStatusCreateRollbackFailed, "⚠"},
		{core.LinkStatusCreateRollbackComplete, "⟲"},
		{core.LinkStatusCreateInterrupted, "⏹"},
		{core.LinkStatusUnknown, "○"},
	}

	for _, tc := range testCases {
		s.Equal(tc.expected, shared.LinkStatusIcon(tc.status), "Status: %s", tc.status)
	}
}

// ToSplitPaneItems tests

func (s *DeployItemsTestSuite) Test_ToSplitPaneItems_converts_slice() {
	items := []DeployItem{
		{Type: ItemTypeResource, Resource: &ResourceDeployItem{Name: "res1"}},
		{Type: ItemTypeResource, Resource: &ResourceDeployItem{Name: "res2"}},
	}
	result := ToSplitPaneItems(items)
	s.Len(result, 2)
	s.Equal("res1", result[0].GetName())
	s.Equal("res2", result[1].GetName())
}

// buildChildPath tests

func (s *DeployItemsTestSuite) Test_BuildChildPath_uses_child_name_when_no_path() {
	item := &DeployItem{
		Child: &ChildDeployItem{Name: "parentChild"},
	}
	path := item.BuildChildPath("childElement")
	s.Equal("parentChild/childElement", path)
}

func (s *DeployItemsTestSuite) Test_BuildChildPath_extends_existing_path() {
	item := &DeployItem{
		Path: "level1/level2",
	}
	path := item.BuildChildPath("level3")
	s.Equal("level1/level2/level3", path)
}

func (s *DeployItemsTestSuite) Test_BuildChildPath_returns_name_when_no_parent() {
	item := &DeployItem{}
	path := item.BuildChildPath("element")
	s.Equal("element", path)
}

// isDirectChild tests

func (s *DeployItemsTestSuite) Test_IsDirectChild_returns_true_for_direct_child() {
	item := &DeployItem{}
	s.True(item.IsDirectChild("parent/child", "parent/"))
}

func (s *DeployItemsTestSuite) Test_IsDirectChild_returns_false_for_grandchild() {
	item := &DeployItem{}
	s.False(item.IsDirectChild("parent/child/grandchild", "parent/"))
}

func (s *DeployItemsTestSuite) Test_IsDirectChild_returns_false_for_different_prefix() {
	item := &DeployItem{}
	s.False(item.IsDirectChild("other/child", "parent/"))
}

func (s *DeployItemsTestSuite) Test_IsDirectChild_returns_false_for_same_length_path() {
	item := &DeployItem{}
	s.False(item.IsDirectChild("parent/", "parent/"))
}

func (s *DeployItemsTestSuite) Test_IsDirectChild_returns_false_for_shorter_path() {
	item := &DeployItem{}
	s.False(item.IsDirectChild("par", "parent/"))
}

// getDefaultChildAction tests

func (s *DeployItemsTestSuite) Test_GetDefaultChildAction_returns_inspect_when_parent_is_inspect() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "testChild", Action: ActionInspect},
	}
	s.Equal(ActionInspect, item.GetDefaultChildAction())
}

func (s *DeployItemsTestSuite) Test_GetDefaultChildAction_returns_no_change_when_parent_is_not_inspect() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "testChild", Action: ActionUpdate},
	}
	s.Equal(ActionNoChange, item.GetDefaultChildAction())
}

func (s *DeployItemsTestSuite) Test_GetDefaultChildAction_returns_no_change_when_child_is_nil() {
	item := &DeployItem{Type: ItemTypeChild}
	s.Equal(ActionNoChange, item.GetDefaultChildAction())
}

// MakeChildDeployItem tests

func (s *DeployItemsTestSuite) Test_MakeChildDeployItem_creates_item_with_all_fields() {
	child := &ChildDeployItem{Name: "testChild", Action: ActionCreate}
	childChanges := &changes.BlueprintChanges{}
	instanceState := &state.InstanceState{}
	childrenByName := make(map[string]*ChildDeployItem)
	resourcesByName := make(map[string]*ResourceDeployItem)
	linksByName := make(map[string]*LinkDeployItem)

	result := MakeChildDeployItem(child, childChanges, instanceState, childrenByName, resourcesByName, linksByName)

	s.Equal(ItemTypeChild, result.Type)
	s.Equal(child, result.Child)
	s.Equal(childChanges, result.Changes)
	s.Equal(instanceState, result.InstanceState)
	s.Equal(childrenByName, result.childrenByName)
	s.Equal(resourcesByName, result.resourcesByName)
	s.Equal(linksByName, result.linksByName)
}

// GetChildren with links tests

func (s *DeployItemsTestSuite) Test_GetChildren_builds_links_from_new_resources() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"resourceA": {
					NewOutboundLinks: map[string]provider.LinkChanges{
						"resourceB": {},
					},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	// Should have both the resource and the link
	s.Len(children, 2)

	var foundResource, foundLink bool
	for _, child := range children {
		childItem := child.(*DeployItem)
		if childItem.Type == ItemTypeResource {
			s.Equal("resourceA", childItem.Resource.Name)
			foundResource = true
		}
		if childItem.Type == ItemTypeLink {
			s.Equal("resourceA::resourceB", childItem.Link.LinkName)
			s.Equal(ActionCreate, childItem.Link.Action)
			foundLink = true
		}
	}
	s.True(foundResource)
	s.True(foundLink)
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_links_from_resource_changes() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"resourceA": {
					NewOutboundLinks: map[string]provider.LinkChanges{
						"resourceB": {},
					},
					OutboundLinkChanges: map[string]provider.LinkChanges{
						"resourceC": {},
					},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	// Should have the resource and 2 links
	s.Len(children, 3)

	var linkActions []ActionType
	for _, child := range children {
		childItem := child.(*DeployItem)
		if childItem.Type == ItemTypeLink {
			linkActions = append(linkActions, childItem.Link.Action)
		}
	}
	s.Contains(linkActions, ActionCreate) // New link
	s.Contains(linkActions, ActionUpdate) // Changed link
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_removed_links_from_resource_changes() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"resourceA": {
					RemovedOutboundLinks: []string{"resourceA::resourceB"},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	// Should have resource and removed link
	s.Len(children, 2)

	var foundRemovedLink bool
	for _, child := range children {
		childItem := child.(*DeployItem)
		if childItem.Type == ItemTypeLink {
			s.Equal("resourceA::resourceB", childItem.Link.LinkName)
			s.Equal(ActionDelete, childItem.Link.Action)
			foundRemovedLink = true
		}
	}
	s.True(foundRemovedLink)
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_top_level_removed_links() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			RemovedLinks: []string{"resX::resY"},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeLink, childItem.Type)
	s.Equal("resX::resY", childItem.Link.LinkName)
	s.Equal(ActionDelete, childItem.Link.Action)
}

// GetChildren with nested children tests

func (s *DeployItemsTestSuite) Test_GetChildren_builds_new_nested_children() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "parentChild"},
		Changes: &changes.BlueprintChanges{
			NewChildren: map[string]changes.NewBlueprintDefinition{
				"nestedChild": {
					NewResources: map[string]provider.Changes{
						"nestedResource": {},
					},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal("nestedChild", childItem.Child.Name)
	s.Equal(ActionCreate, childItem.Child.Action)
	// Should have nested Changes converted from NewBlueprintDefinition
	s.NotNil(childItem.Changes)
	s.Len(childItem.Changes.NewResources, 1)
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_changed_nested_children() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "parentChild"},
		Changes: &changes.BlueprintChanges{
			ChildChanges: map[string]changes.BlueprintChanges{
				"nestedChild": {
					ResourceChanges: map[string]provider.Changes{
						"changedResource": {
							ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
						},
					},
				},
			},
		},
		InstanceState: &state.InstanceState{
			ChildBlueprints: map[string]*state.InstanceState{
				"nestedChild": {
					InstanceID: "nested-instance-123",
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal("nestedChild", childItem.Child.Name)
	s.Equal(ActionUpdate, childItem.Child.Action)
	// Should have nested InstanceState
	s.NotNil(childItem.InstanceState)
	s.Equal("nested-instance-123", childItem.InstanceState.InstanceID)
}

func (s *DeployItemsTestSuite) Test_GetChildren_builds_removed_nested_children() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "parentChild"},
		Changes: &changes.BlueprintChanges{
			RemovedChildren: []string{"removedChild"},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal("removedChild", childItem.Child.Name)
	s.Equal(ActionDelete, childItem.Child.Action)
}

// GetChildren with unchanged items from instance state tests

func (s *DeployItemsTestSuite) Test_GetChildren_adds_unchanged_children_from_instance_state() {
	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "parentChild", Action: ActionUpdate},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			ChildBlueprints: map[string]*state.InstanceState{
				"unchangedChild": {
					InstanceID: "unchanged-instance-123",
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal("unchangedChild", childItem.Child.Name)
	s.Equal(ActionNoChange, childItem.Child.Action)
}

func (s *DeployItemsTestSuite) Test_GetChildren_adds_unchanged_links_from_instance_state() {
	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "parentChild", Action: ActionUpdate},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			Links: map[string]*state.LinkState{
				"resA::resB": {
					LinkID: "link-123",
					Status: core.LinkStatusCreated,
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal(ItemTypeLink, childItem.Type)
	s.Equal("resA::resB", childItem.Link.LinkName)
	s.Equal(ActionNoChange, childItem.Link.Action)
	s.Equal("link-123", childItem.Link.LinkID)
}

func (s *DeployItemsTestSuite) Test_GetChildren_inherits_inspect_action_for_unchanged_items() {
	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "parentChild", Action: ActionInspect},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			Resources: map[string]*state.ResourceState{
				"res-123": {
					ResourceID: "res-123",
					Name:       "inspectResource",
					Type:       "aws/s3/bucket",
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("inspectResource", childItem.Resource.Name)
	s.Equal(ActionInspect, childItem.Resource.Action)
}

// GetChildren with resource changes that have only link changes

func (s *DeployItemsTestSuite) Test_GetChildren_sets_no_change_for_resource_with_only_link_changes() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"resourceA": {
					// No ModifiedFields, only link changes
					NewOutboundLinks: map[string]provider.LinkChanges{
						"resourceB": {},
					},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()

	var resourceItem *DeployItem
	for _, child := range children {
		childItem := child.(*DeployItem)
		if childItem.Type == ItemTypeResource {
			resourceItem = childItem
			break
		}
	}
	s.NotNil(resourceItem)
	s.Equal("resourceA", resourceItem.Resource.Name)
	s.Equal(ActionNoChange, resourceItem.Resource.Action)
}

// populateResourceItemFromChanges tests (via getOrCreateResourceItemWithChanges)

func (s *DeployItemsTestSuite) Test_GetChildren_populates_resource_from_changes_data() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Name: "childBlueprint"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"changedResource": {
					ModifiedFields: []provider.FieldChange{{FieldPath: "spec.value"}},
					AppliedResourceInfo: provider.ResourceInfo{
						ResourceID: "res-from-changes",
						CurrentResourceState: &state.ResourceState{
							ResourceID: "res-from-changes",
							Type:       "aws/lambda/function",
						},
					},
				},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("changedResource", childItem.Resource.Name)
	s.Equal("res-from-changes", childItem.Resource.ResourceID)
	s.Equal("aws/lambda/function", childItem.Resource.ResourceType)
	s.NotNil(childItem.Resource.ResourceState)
}

// getOrCreateResourceItemFromState tests (via GetChildren with instance state)

func (s *DeployItemsTestSuite) Test_GetChildren_hydrates_existing_resource_from_state() {
	// Pre-populate the shared map with a resource item (simulating streaming scenario)
	resourcesByName := make(map[string]*ResourceDeployItem)
	resourcesByName["childBlueprint/existingResource"] = &ResourceDeployItem{
		Name:   "existingResource",
		Action: ActionInspect,
		Status: core.ResourceStatusCreating, // Status from streaming
	}

	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "childBlueprint", Action: ActionUpdate},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			Resources: map[string]*state.ResourceState{
				"res-456": {
					ResourceID: "res-456",
					Name:       "existingResource",
					Type:       "aws/sqs/queue",
				},
			},
		},
		resourcesByName: resourcesByName,
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	// Should have hydrated the existing resource with state data
	s.Equal("existingResource", childItem.Resource.Name)
	s.Equal("res-456", childItem.Resource.ResourceID)
	s.Equal("aws/sqs/queue", childItem.Resource.ResourceType)
	// Should preserve the streaming status
	s.Equal(core.ResourceStatusCreating, childItem.Resource.Status)
}

// getOrCreateChildItemFromState tests

func (s *DeployItemsTestSuite) Test_GetChildren_migrates_child_from_simple_key_to_path_key() {
	// Pre-populate the shared map with a simple name key (backwards compat scenario)
	childrenByName := make(map[string]*ChildDeployItem)
	childrenByName["existingChild"] = &ChildDeployItem{
		Name:   "existingChild",
		Action: ActionInspect,
	}

	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "parentChild", Action: ActionUpdate},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			ChildBlueprints: map[string]*state.InstanceState{
				"existingChild": {InstanceID: "child-instance-789"},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  childrenByName,
		linksByName:     make(map[string]*LinkDeployItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("existingChild", childItem.Child.Name)
	// Should have migrated from simple key to path-based key
	s.Nil(childrenByName["existingChild"])
	s.NotNil(childrenByName["parentChild/existingChild"])
}

// getOrCreateLinkItemFromState tests

func (s *DeployItemsTestSuite) Test_GetChildren_migrates_link_from_simple_key_to_path_key() {
	// Pre-populate the shared map with a simple name key
	linksByName := make(map[string]*LinkDeployItem)
	linksByName["resA::resB"] = &LinkDeployItem{
		LinkName: "resA::resB",
		Action:   ActionInspect,
	}

	item := &DeployItem{
		Type:    ItemTypeChild,
		Child:   &ChildDeployItem{Name: "parentChild", Action: ActionUpdate},
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			Links: map[string]*state.LinkState{
				"resA::resB": {LinkID: "link-migrated"},
			},
		},
		resourcesByName: make(map[string]*ResourceDeployItem),
		childrenByName:  make(map[string]*ChildDeployItem),
		linksByName:     linksByName,
	}
	children := item.GetChildren()
	s.Len(children, 1)
	childItem := children[0].(*DeployItem)
	s.Equal("resA::resB", childItem.Link.LinkName)
	s.Equal("link-migrated", childItem.Link.LinkID)
	// Should have migrated from simple key to path-based key
	s.Nil(linksByName["resA::resB"])
	s.NotNil(linksByName["parentChild/resA::resB"])
}

// GetID edge cases for nil pointers

func (s *DeployItemsTestSuite) Test_GetID_returns_empty_for_nil_child() {
	item := &DeployItem{Type: ItemTypeChild}
	s.Equal("", item.GetID())
}

func (s *DeployItemsTestSuite) Test_GetID_returns_empty_for_nil_link() {
	item := &DeployItem{Type: ItemTypeLink}
	s.Equal("", item.GetID())
}

func (s *DeployItemsTestSuite) Test_GetID_returns_empty_for_unknown_type() {
	item := &DeployItem{Type: "unknown"}
	s.Equal("", item.GetID())
}

// GetIcon edge cases for nil pointers

func (s *DeployItemsTestSuite) Test_GetIcon_returns_pending_for_nil_child() {
	item := &DeployItem{Type: ItemTypeChild}
	s.Equal(shared.IconPending, item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_returns_pending_for_nil_link() {
	item := &DeployItem{Type: ItemTypeLink}
	s.Equal(shared.IconPending, item.GetIcon(false))
}

func (s *DeployItemsTestSuite) Test_GetIcon_returns_pending_for_unknown_type() {
	item := &DeployItem{Type: "unknown"}
	s.Equal(shared.IconPending, item.GetIcon(false))
}

// GetIconStyled edge cases for nil pointers

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_icon_for_nil_resource() {
	item := &DeployItem{Type: ItemTypeResource}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Equal(shared.IconPending, icon)
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_icon_for_nil_child() {
	item := &DeployItem{Type: ItemTypeChild}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Equal(shared.IconPending, icon)
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_returns_icon_for_nil_link() {
	item := &DeployItem{Type: ItemTypeLink}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Equal(shared.IconPending, icon)
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_child_skipped() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Skipped: true},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, shared.IconSkipped)
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_child_no_change() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: &ChildDeployItem{Action: ActionNoChange},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, shared.IconNoChange)
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_link_skipped() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Skipped: true},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, shared.IconSkipped)
}

func (s *DeployItemsTestSuite) Test_GetIconStyled_link_no_change() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Action: ActionNoChange},
	}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, shared.IconNoChange)
}

// GetAction edge cases for nil pointers

func (s *DeployItemsTestSuite) Test_GetAction_returns_empty_for_nil_child() {
	item := &DeployItem{Type: ItemTypeChild}
	s.Equal("", item.GetAction())
}

func (s *DeployItemsTestSuite) Test_GetAction_returns_empty_for_nil_link() {
	item := &DeployItem{Type: ItemTypeLink}
	s.Equal("", item.GetAction())
}

func (s *DeployItemsTestSuite) Test_GetAction_returns_empty_for_unknown_type() {
	item := &DeployItem{Type: "unknown"}
	s.Equal("", item.GetAction())
}

// Link no change icon test

func (s *DeployItemsTestSuite) Test_GetIcon_link_no_change() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{Action: ActionNoChange},
	}
	s.Equal(shared.IconNoChange, item.GetIcon(false))
}
