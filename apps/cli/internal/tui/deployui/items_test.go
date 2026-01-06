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

func (s *DeployItemsTestSuite) Test_buildChildPath_uses_child_name_when_no_path() {
	item := &DeployItem{
		Child: &ChildDeployItem{Name: "parentChild"},
	}
	path := item.buildChildPath("childElement")
	s.Equal("parentChild/childElement", path)
}

func (s *DeployItemsTestSuite) Test_buildChildPath_extends_existing_path() {
	item := &DeployItem{
		Path: "level1/level2",
	}
	path := item.buildChildPath("level3")
	s.Equal("level1/level2/level3", path)
}

func (s *DeployItemsTestSuite) Test_buildChildPath_returns_name_when_no_parent() {
	item := &DeployItem{}
	path := item.buildChildPath("element")
	s.Equal("element", path)
}
