package destroyui

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
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type DestroyItemsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestDestroyItemsTestSuite(t *testing.T) {
	suite.Run(t, new(DestroyItemsTestSuite))
}

func (s *DestroyItemsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		styles.NewBluelinkPalette(),
	)
}

// --- GetID tests ---

func (s *DestroyItemsTestSuite) Test_GetID_returns_resource_name() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name: "myResource",
		},
	}
	s.Equal("myResource", item.GetID())
}

func (s *DestroyItemsTestSuite) Test_GetID_returns_child_name() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name: "myChild",
		},
	}
	s.Equal("myChild", item.GetID())
}

func (s *DestroyItemsTestSuite) Test_GetID_returns_link_name() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "resourceA::resourceB",
		},
	}
	s.Equal("resourceA::resourceB", item.GetID())
}

func (s *DestroyItemsTestSuite) Test_GetID_returns_empty_for_nil_resource() {
	item := &DestroyItem{
		Type:     ItemTypeResource,
		Resource: nil,
	}
	s.Equal("", item.GetID())
}

func (s *DestroyItemsTestSuite) Test_GetID_returns_empty_for_unknown_type() {
	item := &DestroyItem{
		Type: ItemType("unknown"),
	}
	s.Equal("", item.GetID())
}

// --- GetName tests ---

func (s *DestroyItemsTestSuite) Test_GetName_returns_id() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name: "testResource",
		},
	}
	s.Equal("testResource", item.GetName())
}

// --- GetIcon tests ---

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_skipped_icon_for_skipped_resource() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:    "res",
			Skipped: true,
		},
	}
	s.Equal(shared.IconSkipped, item.GetIcon(false))
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_no_change_icon() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:   "res",
			Action: ActionNoChange,
		},
	}
	s.Equal(shared.IconNoChange, item.GetIcon(false))
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_status_icon_for_resource() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:   "res",
			Action: ActionDelete,
			Status: core.ResourceStatusDestroyed,
		},
	}
	icon := item.GetIcon(false)
	s.NotEmpty(icon)
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_skipped_icon_for_skipped_child() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:    "child",
			Skipped: true,
		},
	}
	s.Equal(shared.IconSkipped, item.GetIcon(false))
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_no_change_icon_for_child() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:   "child",
			Action: ActionNoChange,
		},
	}
	s.Equal(shared.IconNoChange, item.GetIcon(false))
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_status_icon_for_child() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:   "child",
			Action: ActionDelete,
			Status: core.InstanceStatusDestroyed,
		},
	}
	icon := item.GetIcon(false)
	s.NotEmpty(icon)
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_skipped_icon_for_skipped_link() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "link",
			Skipped:  true,
		},
	}
	s.Equal(shared.IconSkipped, item.GetIcon(false))
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_no_change_icon_for_link() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "link",
			Action:   ActionNoChange,
		},
	}
	s.Equal(shared.IconNoChange, item.GetIcon(false))
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_status_icon_for_link() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "link",
			Action:   ActionDelete,
			Status:   core.LinkStatusDestroyed,
		},
	}
	icon := item.GetIcon(false)
	s.NotEmpty(icon)
}

func (s *DestroyItemsTestSuite) Test_GetIcon_returns_pending_for_nil_item() {
	item := &DestroyItem{
		Type:     ItemTypeResource,
		Resource: nil,
	}
	s.Equal(shared.IconPending, item.GetIcon(false))
}

// --- GetIconStyled tests ---

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_unstyled_when_not_styled() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:   "res",
			Action: ActionDelete,
		},
	}
	icon := item.GetIcon(false)
	styledIcon := item.GetIconStyled(s.testStyles, false)
	s.Equal(icon, styledIcon)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_skipped_resource() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:    "res",
			Skipped: true,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_no_change_resource() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:   "res",
			Action: ActionNoChange,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_resource_status() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Name:   "res",
			Action: ActionDelete,
			Status: core.ResourceStatusDestroyed,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_skipped_child() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:    "child",
			Skipped: true,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_no_change_child() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:   "child",
			Action: ActionNoChange,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_child_status() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:   "child",
			Action: ActionDelete,
			Status: core.InstanceStatusDestroyed,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_skipped_link() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "link",
			Skipped:  true,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_no_change_link() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "link",
			Action:   ActionNoChange,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_styled_for_link_status() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			LinkName: "link",
			Action:   ActionDelete,
			Status:   core.LinkStatusDestroyed,
		},
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.NotEmpty(result)
}

func (s *DestroyItemsTestSuite) Test_GetIconStyled_returns_icon_for_nil_resource() {
	item := &DestroyItem{
		Type:     ItemTypeResource,
		Resource: nil,
	}
	result := item.GetIconStyled(s.testStyles, true)
	s.Equal(shared.IconPending, result)
}

// --- GetAction tests ---

func (s *DestroyItemsTestSuite) Test_GetAction_returns_resource_action() {
	item := &DestroyItem{
		Type: ItemTypeResource,
		Resource: &ResourceDestroyItem{
			Action: ActionDelete,
		},
	}
	s.Equal(string(ActionDelete), item.GetAction())
}

func (s *DestroyItemsTestSuite) Test_GetAction_returns_child_action() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Action: ActionDelete,
		},
	}
	s.Equal(string(ActionDelete), item.GetAction())
}

func (s *DestroyItemsTestSuite) Test_GetAction_returns_link_action() {
	item := &DestroyItem{
		Type: ItemTypeLink,
		Link: &LinkDestroyItem{
			Action: ActionDelete,
		},
	}
	s.Equal(string(ActionDelete), item.GetAction())
}

func (s *DestroyItemsTestSuite) Test_GetAction_returns_empty_for_nil() {
	item := &DestroyItem{
		Type:     ItemTypeResource,
		Resource: nil,
	}
	s.Equal("", item.GetAction())
}

// --- GetDepth tests ---

func (s *DestroyItemsTestSuite) Test_GetDepth_returns_depth() {
	item := &DestroyItem{
		Depth: 3,
	}
	s.Equal(3, item.GetDepth())
}

// --- GetParentID tests ---

func (s *DestroyItemsTestSuite) Test_GetParentID_returns_parent_child() {
	item := &DestroyItem{
		ParentChild: "parentName",
	}
	s.Equal("parentName", item.GetParentID())
}

// --- GetItemType tests ---

func (s *DestroyItemsTestSuite) Test_GetItemType_returns_type() {
	item := &DestroyItem{
		Type: ItemTypeResource,
	}
	s.Equal(string(ItemTypeResource), item.GetItemType())
}

// --- IsExpandable tests ---

func (s *DestroyItemsTestSuite) Test_IsExpandable_returns_true_for_child_with_changes() {
	item := &DestroyItem{
		Type:    ItemTypeChild,
		Changes: &changes.BlueprintChanges{},
	}
	s.True(item.IsExpandable())
}

func (s *DestroyItemsTestSuite) Test_IsExpandable_returns_true_for_child_with_instance_state() {
	item := &DestroyItem{
		Type:          ItemTypeChild,
		InstanceState: &state.InstanceState{},
	}
	s.True(item.IsExpandable())
}

func (s *DestroyItemsTestSuite) Test_IsExpandable_returns_false_for_child_without_data() {
	item := &DestroyItem{
		Type: ItemTypeChild,
	}
	s.False(item.IsExpandable())
}

func (s *DestroyItemsTestSuite) Test_IsExpandable_returns_false_for_resource() {
	item := &DestroyItem{
		Type:    ItemTypeResource,
		Changes: &changes.BlueprintChanges{},
	}
	s.False(item.IsExpandable())
}

// --- CanDrillDown tests ---

func (s *DestroyItemsTestSuite) Test_CanDrillDown_returns_true_for_child_with_changes() {
	item := &DestroyItem{
		Type:    ItemTypeChild,
		Changes: &changes.BlueprintChanges{},
	}
	s.True(item.CanDrillDown())
}

func (s *DestroyItemsTestSuite) Test_CanDrillDown_returns_false_for_resource() {
	item := &DestroyItem{
		Type: ItemTypeResource,
	}
	s.False(item.CanDrillDown())
}

// --- GetChildren tests ---

func (s *DestroyItemsTestSuite) Test_GetChildren_returns_nil_for_resource() {
	item := &DestroyItem{
		Type: ItemTypeResource,
	}
	s.Nil(item.GetChildren())
}

func (s *DestroyItemsTestSuite) Test_GetChildren_returns_nil_for_child_without_data() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "child"},
	}
	s.Nil(item.GetChildren())
}

func (s *DestroyItemsTestSuite) Test_GetChildren_returns_removed_resources() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "child"},
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"res1", "res2"},
		},
		resourcesByName: make(map[string]*ResourceDestroyItem),
		childrenByName:  make(map[string]*ChildDestroyItem),
		linksByName:     make(map[string]*LinkDestroyItem),
	}
	children := item.GetChildren()
	s.Len(children, 2)
}

func (s *DestroyItemsTestSuite) Test_GetChildren_returns_resource_changes() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "child"},
		Changes: &changes.BlueprintChanges{
			ResourceChanges: map[string]provider.Changes{
				"res1": {},
			},
		},
		resourcesByName: make(map[string]*ResourceDestroyItem),
		childrenByName:  make(map[string]*ChildDestroyItem),
		linksByName:     make(map[string]*LinkDestroyItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
}

func (s *DestroyItemsTestSuite) Test_GetChildren_returns_removed_children() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "parent"},
		Changes: &changes.BlueprintChanges{
			RemovedChildren: []string{"child1"},
		},
		resourcesByName: make(map[string]*ResourceDestroyItem),
		childrenByName:  make(map[string]*ChildDestroyItem),
		linksByName:     make(map[string]*LinkDestroyItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
}

func (s *DestroyItemsTestSuite) Test_GetChildren_returns_child_changes() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "parent"},
		Changes: &changes.BlueprintChanges{
			ChildChanges: map[string]changes.BlueprintChanges{
				"nestedChild": {},
			},
		},
		resourcesByName: make(map[string]*ResourceDestroyItem),
		childrenByName:  make(map[string]*ChildDestroyItem),
		linksByName:     make(map[string]*LinkDestroyItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
}

func (s *DestroyItemsTestSuite) Test_GetChildren_propagates_skipped() {
	item := &DestroyItem{
		Type: ItemTypeChild,
		Child: &ChildDestroyItem{
			Name:    "child",
			Skipped: true,
		},
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"res1"},
		},
		resourcesByName: make(map[string]*ResourceDestroyItem),
		childrenByName:  make(map[string]*ChildDestroyItem),
		linksByName:     make(map[string]*LinkDestroyItem),
	}
	children := item.GetChildren()
	s.Len(children, 1)
	destroyItem := children[0].(*DestroyItem)
	s.True(destroyItem.Resource.Skipped)
}

// --- getOrCreateResourceItem tests ---

func (s *DestroyItemsTestSuite) Test_GetOrCreateResourceItem_creates_new_item() {
	item := &DestroyItem{
		resourcesByName: make(map[string]*ResourceDestroyItem),
	}
	result := item.GetOrCreateResourceItem("newRes", ActionDelete, false)
	s.Equal("newRes", result.Name)
	s.Equal(ActionDelete, result.Action)
	s.False(result.Skipped)
}

func (s *DestroyItemsTestSuite) Test_GetOrCreateResourceItem_returns_existing_by_path() {
	existing := &ResourceDestroyItem{
		Name:   "res",
		Action: ActionDelete,
	}
	item := &DestroyItem{
		Path: "parent",
		resourcesByName: map[string]*ResourceDestroyItem{
			"parent/res": existing,
		},
	}
	result := item.GetOrCreateResourceItem("res", ActionUpdate, true)
	s.Same(existing, result)
	s.True(result.Skipped)
}

func (s *DestroyItemsTestSuite) Test_GetOrCreateResourceItem_returns_existing_by_name() {
	existing := &ResourceDestroyItem{
		Name:   "res",
		Action: ActionDelete,
	}
	item := &DestroyItem{
		resourcesByName: map[string]*ResourceDestroyItem{
			"res": existing,
		},
	}
	result := item.GetOrCreateResourceItem("res", ActionUpdate, false)
	s.Same(existing, result)
}

// --- getOrCreateChildItem tests ---

func (s *DestroyItemsTestSuite) Test_GetOrCreateChildItem_creates_new_item() {
	item := &DestroyItem{
		childrenByName: make(map[string]*ChildDestroyItem),
	}
	result := item.GetOrCreateChildItem("newChild", ActionDelete, nil, false)
	s.Equal("newChild", result.Name)
	s.Equal(ActionDelete, result.Action)
	s.False(result.Skipped)
}

func (s *DestroyItemsTestSuite) Test_GetOrCreateChildItem_returns_existing_by_path() {
	existing := &ChildDestroyItem{
		Name:   "child",
		Action: ActionDelete,
	}
	item := &DestroyItem{
		Path: "parent",
		childrenByName: map[string]*ChildDestroyItem{
			"parent/child": existing,
		},
	}
	result := item.GetOrCreateChildItem("child", ActionUpdate, nil, true)
	s.Same(existing, result)
	s.True(result.Skipped)
}

func (s *DestroyItemsTestSuite) Test_GetOrCreateChildItem_returns_existing_by_name() {
	existing := &ChildDestroyItem{
		Name:   "child",
		Action: ActionDelete,
	}
	item := &DestroyItem{
		childrenByName: map[string]*ChildDestroyItem{
			"child": existing,
		},
	}
	result := item.GetOrCreateChildItem("child", ActionUpdate, nil, false)
	s.Same(existing, result)
}

// --- buildChildPath tests ---

func (s *DestroyItemsTestSuite) Test_BuildChildPath_with_existing_path() {
	item := &DestroyItem{
		Path: "parent/child",
	}
	result := item.BuildChildPath("grandchild")
	s.Equal("parent/child/grandchild", result)
}

func (s *DestroyItemsTestSuite) Test_BuildChildPath_without_path_with_child() {
	item := &DestroyItem{
		Child: &ChildDestroyItem{Name: "parent"},
	}
	result := item.BuildChildPath("nested")
	s.Equal("parent/nested", result)
}

func (s *DestroyItemsTestSuite) Test_BuildChildPath_without_path_without_child() {
	item := &DestroyItem{}
	result := item.BuildChildPath("simple")
	s.Equal("simple", result)
}

// --- appendResourceItems tests ---

func (s *DestroyItemsTestSuite) Test_AppendResourceItems_handles_removed_and_changed() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "parent"},
		Changes: &changes.BlueprintChanges{
			RemovedResources: []string{"removed1"},
			ResourceChanges: map[string]provider.Changes{
				"changed1": {},
			},
		},
		resourcesByName: make(map[string]*ResourceDestroyItem),
	}
	var items []splitpane.Item
	result := item.AppendResourceItems(items, false)
	s.Len(result, 2)
}

// --- appendChildItems tests ---

func (s *DestroyItemsTestSuite) Test_AppendChildItems_handles_removed_and_changed() {
	item := &DestroyItem{
		Type:  ItemTypeChild,
		Child: &ChildDestroyItem{Name: "parent"},
		Changes: &changes.BlueprintChanges{
			RemovedChildren: []string{"removed1"},
			ChildChanges: map[string]changes.BlueprintChanges{
				"changed1": {},
			},
		},
		childrenByName: make(map[string]*ChildDestroyItem),
	}
	var items []splitpane.Item
	result := item.AppendChildItems(items, false)
	s.Len(result, 2)
}
