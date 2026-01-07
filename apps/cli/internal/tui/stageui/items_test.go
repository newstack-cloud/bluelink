package stageui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type StageItemsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestStageItemsTestSuite(t *testing.T) {
	suite.Run(t, new(StageItemsTestSuite))
}

func (s *StageItemsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		styles.NewBluelinkPalette(),
	)
}

// --- splitpane.Item interface tests ---

func (s *StageItemsTestSuite) Test_GetID_returns_name() {
	item := &StageItem{Name: "test-resource"}
	s.Equal("test-resource", item.GetID())
}

func (s *StageItemsTestSuite) Test_GetName_returns_name() {
	item := &StageItem{Name: "my-child"}
	s.Equal("my-child", item.GetName())
}

func (s *StageItemsTestSuite) Test_GetAction_returns_action_string() {
	testCases := []struct {
		action   ActionType
		expected string
	}{
		{ActionCreate, "CREATE"},
		{ActionUpdate, "UPDATE"},
		{ActionDelete, "DELETE"},
		{ActionRecreate, "RECREATE"},
		{ActionNoChange, "NO CHANGE"},
	}
	for _, tc := range testCases {
		s.Run(tc.expected, func() {
			item := &StageItem{Action: tc.action}
			s.Equal(tc.expected, item.GetAction())
		})
	}
}

func (s *StageItemsTestSuite) Test_GetDepth_returns_depth() {
	item := &StageItem{Depth: 3}
	s.Equal(3, item.GetDepth())
}

func (s *StageItemsTestSuite) Test_GetParentID_returns_parent_child() {
	item := &StageItem{ParentChild: "parent-blueprint"}
	s.Equal("parent-blueprint", item.GetParentID())
}

func (s *StageItemsTestSuite) Test_GetItemType_returns_type_string() {
	testCases := []struct {
		itemType ItemType
		expected string
	}{
		{ItemTypeResource, "resource"},
		{ItemTypeChild, "child"},
		{ItemTypeLink, "link"},
	}
	for _, tc := range testCases {
		s.Run(tc.expected, func() {
			item := &StageItem{Type: tc.itemType}
			s.Equal(tc.expected, item.GetItemType())
		})
	}
}

// --- GetIcon tests ---

func (s *StageItemsTestSuite) Test_GetIcon_returns_icon_for_create() {
	item := &StageItem{Action: ActionCreate}
	s.Equal("✓", item.GetIcon(false))
}

func (s *StageItemsTestSuite) Test_GetIcon_returns_icon_for_update() {
	item := &StageItem{Action: ActionUpdate}
	s.Equal("±", item.GetIcon(false))
}

func (s *StageItemsTestSuite) Test_GetIcon_returns_icon_for_delete() {
	item := &StageItem{Action: ActionDelete}
	s.Equal("-", item.GetIcon(false))
}

func (s *StageItemsTestSuite) Test_GetIcon_returns_icon_for_recreate() {
	item := &StageItem{Action: ActionRecreate}
	s.Equal("↻", item.GetIcon(false))
}

func (s *StageItemsTestSuite) Test_GetIcon_returns_icon_for_no_change() {
	item := &StageItem{Action: ActionNoChange}
	s.Equal("○", item.GetIcon(false))
}

// --- GetIconStyled tests ---

func (s *StageItemsTestSuite) Test_GetIconStyled_returns_plain_icon_when_not_styled() {
	item := &StageItem{Action: ActionCreate}
	icon := item.GetIconStyled(s.testStyles, false)
	s.Equal("✓", icon)
}

func (s *StageItemsTestSuite) Test_GetIconStyled_returns_styled_icon_for_create() {
	item := &StageItem{Action: ActionCreate}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "✓")
}

func (s *StageItemsTestSuite) Test_GetIconStyled_returns_styled_icon_for_update() {
	item := &StageItem{Action: ActionUpdate}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "±")
}

func (s *StageItemsTestSuite) Test_GetIconStyled_returns_styled_icon_for_delete() {
	item := &StageItem{Action: ActionDelete}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "-")
}

func (s *StageItemsTestSuite) Test_GetIconStyled_returns_styled_icon_for_recreate() {
	item := &StageItem{Action: ActionRecreate}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "↻")
}

func (s *StageItemsTestSuite) Test_GetIconStyled_returns_styled_icon_for_no_change() {
	item := &StageItem{Action: ActionNoChange}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "○")
}

// --- IsExpandable tests ---

func (s *StageItemsTestSuite) Test_IsExpandable_true_for_child_with_changes() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Changes: &changes.BlueprintChanges{},
	}
	s.True(item.IsExpandable())
}

func (s *StageItemsTestSuite) Test_IsExpandable_false_for_child_without_changes() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Changes: nil,
	}
	s.False(item.IsExpandable())
}

func (s *StageItemsTestSuite) Test_IsExpandable_false_for_resource() {
	item := &StageItem{
		Type:    ItemTypeResource,
		Changes: &provider.Changes{},
	}
	s.False(item.IsExpandable())
}

func (s *StageItemsTestSuite) Test_IsExpandable_false_for_link() {
	item := &StageItem{
		Type:    ItemTypeLink,
		Changes: &provider.LinkChanges{},
	}
	s.False(item.IsExpandable())
}

// --- CanDrillDown tests ---

func (s *StageItemsTestSuite) Test_CanDrillDown_true_for_child_with_blueprint_changes() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Changes: &changes.BlueprintChanges{},
	}
	s.True(item.CanDrillDown())
}

func (s *StageItemsTestSuite) Test_CanDrillDown_false_for_child_without_blueprint_changes() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Changes: &provider.Changes{}, // wrong type
	}
	s.False(item.CanDrillDown())
}

func (s *StageItemsTestSuite) Test_CanDrillDown_false_for_resource() {
	item := &StageItem{
		Type:    ItemTypeResource,
		Changes: &provider.Changes{},
	}
	s.False(item.CanDrillDown())
}

func (s *StageItemsTestSuite) Test_CanDrillDown_false_for_link() {
	item := &StageItem{
		Type:    ItemTypeLink,
		Changes: &provider.LinkChanges{},
	}
	s.False(item.CanDrillDown())
}

// --- GetChildren tests ---

func (s *StageItemsTestSuite) Test_GetChildren_returns_nil_for_resource() {
	item := &StageItem{
		Type: ItemTypeResource,
	}
	s.Nil(item.GetChildren())
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_nil_for_link() {
	item := &StageItem{
		Type: ItemTypeLink,
	}
	s.Nil(item.GetChildren())
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_nil_for_child_without_blueprint_changes() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Changes: &provider.Changes{}, // wrong type
	}
	s.Nil(item.GetChildren())
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_nil_for_child_with_nil_changes() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Changes: nil,
	}
	s.Nil(item.GetChildren())
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_new_resources() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "child-blueprint",
		Changes: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"newResource": {},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	resourceItem := children[0].(*StageItem)
	s.Equal("newResource", resourceItem.Name)
	s.Equal(ItemTypeResource, resourceItem.Type)
	s.Equal(ActionCreate, resourceItem.Action)
	s.True(resourceItem.New)
	s.Equal("child-blueprint", resourceItem.ParentChild)
	s.Equal(1, resourceItem.Depth)
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_changed_resources() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "child-blueprint",
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"changedResource": {
					ModifiedFields: []provider.FieldChange{
						{FieldPath: "spec.field1"},
					},
				},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	resourceItem := children[0].(*StageItem)
	s.Equal("changedResource", resourceItem.Name)
	s.Equal(ItemTypeResource, resourceItem.Type)
	s.Equal(ActionUpdate, resourceItem.Action)
	s.Equal("child-blueprint", resourceItem.ParentChild)
	s.Equal(1, resourceItem.Depth)
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_removed_resources() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "child-blueprint",
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"removedResource"},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	resourceItem := children[0].(*StageItem)
	s.Equal("removedResource", resourceItem.Name)
	s.Equal(ItemTypeResource, resourceItem.Type)
	s.Equal(ActionDelete, resourceItem.Action)
	s.True(resourceItem.Removed)
	s.Equal("child-blueprint", resourceItem.ParentChild)
	s.Equal(1, resourceItem.Depth)
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_recreate_resources() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "child-blueprint",
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"recreateResource": {
					MustRecreate: true,
				},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	resourceItem := children[0].(*StageItem)
	s.Equal("recreateResource", resourceItem.Name)
	s.Equal(ItemTypeResource, resourceItem.Type)
	s.Equal(ActionRecreate, resourceItem.Action)
	s.True(resourceItem.Recreate)
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_new_children() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "parent-child",
		Changes: &changes.BlueprintChanges{
			NewChildren: map[string]changes.NewBlueprintDefinition{
				"newChild": {
					NewResources: map[string]provider.Changes{
						"nestedResource": {},
					},
				},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	childItem := children[0].(*StageItem)
	s.Equal("newChild", childItem.Name)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal(ActionCreate, childItem.Action)
	s.True(childItem.New)
	s.Equal("parent-child", childItem.ParentChild)
	s.Equal(1, childItem.Depth)

	// Verify the changes contain the nested resources
	nestedChanges := childItem.Changes.(*changes.BlueprintChanges)
	s.Len(nestedChanges.NewResources, 1)
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_changed_children() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "parent-child",
		Changes: &changes.BlueprintChanges{
			ChildChanges: map[string]changes.BlueprintChanges{
				"changedChild": {
					ResourceChanges: map[string]provider.Changes{
						"nestedResource": {
							ModifiedFields: []provider.FieldChange{
								{FieldPath: "spec.field1"},
							},
						},
					},
				},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	childItem := children[0].(*StageItem)
	s.Equal("changedChild", childItem.Name)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal(ActionUpdate, childItem.Action)
	s.Equal("parent-child", childItem.ParentChild)
	s.Equal(1, childItem.Depth)
}

func (s *StageItemsTestSuite) Test_GetChildren_returns_removed_children() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "parent-child",
		Changes: &changes.BlueprintChanges{
			RemovedChildren: []string{"removedChild"},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	childItem := children[0].(*StageItem)
	s.Equal("removedChild", childItem.Name)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal(ActionDelete, childItem.Action)
	s.True(childItem.Removed)
	s.Equal("parent-child", childItem.ParentChild)
	s.Equal(1, childItem.Depth)
}

func (s *StageItemsTestSuite) Test_GetChildren_looks_up_resource_state_from_instance_state() {
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "changedResource",
		Type:       "aws/lambda/function",
	}

	item := &StageItem{
		Type: ItemTypeChild,
		Name: "child-blueprint",
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"changedResource": {
					ModifiedFields: []provider.FieldChange{
						{FieldPath: "spec.field1"},
					},
				},
			},
		},
		InstanceState: &state.InstanceState{
			Resources: map[string]*state.ResourceState{
				"res-123": resourceState,
			},
			// ResourceIDs maps resource name -> resource ID
			ResourceIDs: map[string]string{
				"changedResource": "res-123",
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	resourceItem := children[0].(*StageItem)
	s.NotNil(resourceItem.ResourceState)
	s.Equal("res-123", resourceItem.ResourceState.ResourceID)
}

func (s *StageItemsTestSuite) Test_GetChildren_looks_up_child_instance_state() {
	nestedState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"nested-res-123": {
				ResourceID: "nested-res-123",
				Name:       "nestedResource",
			},
		},
	}

	item := &StageItem{
		Type: ItemTypeChild,
		Name: "parent-child",
		Changes: &changes.BlueprintChanges{
			ChildChanges: map[string]changes.BlueprintChanges{
				"changedChild": {},
			},
		},
		InstanceState: &state.InstanceState{
			ChildBlueprints: map[string]*state.InstanceState{
				"changedChild": nestedState,
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	childItem := children[0].(*StageItem)
	s.NotNil(childItem.InstanceState)
	s.Len(childItem.InstanceState.Resources, 1)
}

func (s *StageItemsTestSuite) Test_GetChildren_adds_no_change_resources_from_state() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "child-blueprint",
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
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	resourceItem := children[0].(*StageItem)
	s.Equal("unchangedResource", resourceItem.Name)
	s.Equal(ItemTypeResource, resourceItem.Type)
	s.Equal(ActionNoChange, resourceItem.Action)
	s.Equal("aws/s3/bucket", resourceItem.ResourceType)
	s.NotNil(resourceItem.ResourceState)
}

func (s *StageItemsTestSuite) Test_GetChildren_adds_no_change_children_from_state() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "parent-child",
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			ChildBlueprints: map[string]*state.InstanceState{
				"unchangedChild": {
					Resources: map[string]*state.ResourceState{},
				},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	s.Len(children, 1)

	childItem := children[0].(*StageItem)
	s.Equal("unchangedChild", childItem.Name)
	s.Equal(ItemTypeChild, childItem.Type)
	s.Equal(ActionNoChange, childItem.Action)
	s.NotNil(childItem.InstanceState)
	// No-change children get empty changes so they can still be expanded
	s.NotNil(childItem.Changes)
}

func (s *StageItemsTestSuite) Test_GetChildren_skips_resources_already_in_changes() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "child-blueprint",
		Changes: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"resourceA": {},
			},
		},
		InstanceState: &state.InstanceState{
			Resources: map[string]*state.ResourceState{
				"res-123": {
					ResourceID: "res-123",
					Name:       "resourceA", // Same name as new resource
				},
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	// Should only have 1 item (the new resource), not 2
	s.Len(children, 1)
	s.Equal(ActionCreate, children[0].(*StageItem).Action)
}

func (s *StageItemsTestSuite) Test_GetChildren_skips_children_already_in_changes() {
	item := &StageItem{
		Type: ItemTypeChild,
		Name: "parent-child",
		Changes: &changes.BlueprintChanges{
			NewChildren: map[string]changes.NewBlueprintDefinition{
				"childA": {},
			},
		},
		InstanceState: &state.InstanceState{
			ChildBlueprints: map[string]*state.InstanceState{
				"childA": {}, // Same name as new child
			},
		},
		Depth: 0,
	}

	children := item.GetChildren()
	// Should only have 1 item (the new child), not 2
	s.Len(children, 1)
	s.Equal(ActionCreate, children[0].(*StageItem).Action)
}

// --- determineResourceActionFromChanges tests ---

func (s *StageItemsTestSuite) Test_determineResourceActionFromChanges_returns_recreate_when_must_recreate() {
	changes := &provider.Changes{
		MustRecreate: true,
	}
	s.Equal(ActionRecreate, determineResourceActionFromChanges(changes))
}

func (s *StageItemsTestSuite) Test_determineResourceActionFromChanges_returns_update_when_has_changes() {
	changes := &provider.Changes{
		ModifiedFields: []provider.FieldChange{
			{FieldPath: "spec.field1"},
		},
	}
	s.Equal(ActionUpdate, determineResourceActionFromChanges(changes))
}

func (s *StageItemsTestSuite) Test_determineResourceActionFromChanges_returns_update_when_has_new_fields() {
	changes := &provider.Changes{
		NewFields: []provider.FieldChange{
			{FieldPath: "spec.newField"},
		},
	}
	s.Equal(ActionUpdate, determineResourceActionFromChanges(changes))
}

func (s *StageItemsTestSuite) Test_determineResourceActionFromChanges_returns_update_when_has_removed_fields() {
	changes := &provider.Changes{
		RemovedFields: []string{"spec.oldField"},
	}
	s.Equal(ActionUpdate, determineResourceActionFromChanges(changes))
}

func (s *StageItemsTestSuite) Test_determineResourceActionFromChanges_returns_update_when_has_outbound_link_changes() {
	changes := &provider.Changes{
		NewOutboundLinks: map[string]provider.LinkChanges{
			"targetResource": {},
		},
	}
	s.Equal(ActionUpdate, determineResourceActionFromChanges(changes))
}

func (s *StageItemsTestSuite) Test_determineResourceActionFromChanges_returns_no_change_when_empty() {
	changes := &provider.Changes{}
	s.Equal(ActionNoChange, determineResourceActionFromChanges(changes))
}

// --- ToSplitPaneItems tests ---

func (s *StageItemsTestSuite) Test_ToSplitPaneItems_converts_items() {
	items := []StageItem{
		{Name: "item1", Type: ItemTypeResource},
		{Name: "item2", Type: ItemTypeChild},
		{Name: "item3", Type: ItemTypeLink},
	}

	result := ToSplitPaneItems(items)

	s.Len(result, 3)
	for i, item := range result {
		stageItem := item.(*StageItem)
		s.Equal(items[i].Name, stageItem.Name)
	}
}

func (s *StageItemsTestSuite) Test_ToSplitPaneItems_returns_empty_slice_for_empty_input() {
	result := ToSplitPaneItems([]StageItem{})
	s.Empty(result)
	s.NotNil(result)
}

func (s *StageItemsTestSuite) Test_ToSplitPaneItems_preserves_item_properties() {
	items := []StageItem{
		{
			Type:         ItemTypeResource,
			Name:         "myResource",
			ResourceType: "aws/s3/bucket",
			DisplayName:  "My Bucket",
			Action:       ActionUpdate,
			Depth:        2,
			ParentChild:  "parentBlueprint",
			New:          false,
			Removed:      false,
			Recreate:     false,
		},
	}

	result := ToSplitPaneItems(items)
	stageItem := result[0].(*StageItem)

	s.Equal(ItemTypeResource, stageItem.Type)
	s.Equal("myResource", stageItem.Name)
	s.Equal("aws/s3/bucket", stageItem.ResourceType)
	s.Equal("My Bucket", stageItem.DisplayName)
	s.Equal(ActionUpdate, stageItem.Action)
	s.Equal(2, stageItem.Depth)
	s.Equal("parentBlueprint", stageItem.ParentChild)
}

// --- Interface compliance test ---

func (s *StageItemsTestSuite) Test_StageItem_implements_splitpane_Item() {
	var _ splitpane.Item = (*StageItem)(nil)
}
