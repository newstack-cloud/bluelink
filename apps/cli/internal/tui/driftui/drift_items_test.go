package driftui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type DriftItemsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestDriftItemsTestSuite(t *testing.T) {
	suite.Run(t, new(DriftItemsTestSuite))
}

func (s *DriftItemsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

// DriftItem tests

func (s *DriftItemsTestSuite) Test_DriftItem_GetID_without_child_path() {
	item := &DriftItem{Name: "resource1"}
	s.Equal("resource1", item.GetID())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetID_with_child_path() {
	item := &DriftItem{Name: "resource1", ChildPath: "child1.child2"}
	s.Equal("child1.child2:resource1", item.GetID())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetName() {
	item := &DriftItem{Name: "myResource"}
	s.Equal("myResource", item.GetName())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetIcon_drift() {
	item := &DriftItem{DriftType: container.ReconciliationTypeDrift}
	s.Equal("⚠", item.GetIcon(false))
	s.Equal("⚠", item.GetIcon(true))
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetIcon_interrupted() {
	item := &DriftItem{DriftType: container.ReconciliationTypeInterrupted}
	s.Equal("!", item.GetIcon(false))
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetIcon_default() {
	item := &DriftItem{DriftType: "unknown"}
	s.Equal("○", item.GetIcon(false))
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetIconStyled_returns_plain_when_not_styled() {
	item := &DriftItem{DriftType: container.ReconciliationTypeDrift}
	icon := item.GetIconStyled(s.testStyles, false)
	s.Equal("⚠", icon)
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetIconStyled_returns_styled_for_drift() {
	item := &DriftItem{DriftType: container.ReconciliationTypeDrift}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "⚠")
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetIconStyled_returns_styled_for_interrupted() {
	item := &DriftItem{DriftType: container.ReconciliationTypeInterrupted}
	icon := item.GetIconStyled(s.testStyles, true)
	s.Contains(icon, "!")
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetAction() {
	item := &DriftItem{DriftType: container.ReconciliationTypeDrift}
	s.Equal("DRIFT", item.GetAction())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetDepth() {
	item := &DriftItem{Depth: 2}
	s.Equal(2, item.GetDepth())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetParentID() {
	item := &DriftItem{ParentChild: "parent1"}
	s.Equal("parent1", item.GetParentID())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetItemType() {
	item := &DriftItem{Type: DriftItemTypeResource}
	s.Equal("resource", item.GetItemType())
}

func (s *DriftItemsTestSuite) Test_DriftItem_IsExpandable_true_for_child_with_children() {
	item := &DriftItem{
		Type:     DriftItemTypeChild,
		Children: []*DriftItem{{Name: "child"}},
	}
	s.True(item.IsExpandable())
}

func (s *DriftItemsTestSuite) Test_DriftItem_IsExpandable_false_for_resource() {
	item := &DriftItem{Type: DriftItemTypeResource}
	s.False(item.IsExpandable())
}

func (s *DriftItemsTestSuite) Test_DriftItem_IsExpandable_false_for_child_without_children() {
	item := &DriftItem{Type: DriftItemTypeChild}
	s.False(item.IsExpandable())
}

func (s *DriftItemsTestSuite) Test_DriftItem_CanDrillDown_true_for_child_with_children() {
	item := &DriftItem{
		Type:     DriftItemTypeChild,
		Children: []*DriftItem{{Name: "child"}},
	}
	s.True(item.CanDrillDown())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetChildren_returns_nil_for_non_child() {
	item := &DriftItem{Type: DriftItemTypeResource}
	s.Nil(item.GetChildren())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetChildren_returns_nil_for_empty_children() {
	item := &DriftItem{Type: DriftItemTypeChild}
	s.Nil(item.GetChildren())
}

func (s *DriftItemsTestSuite) Test_DriftItem_GetChildren_returns_children_as_splitpane_items() {
	child1 := &DriftItem{Name: "child1"}
	child2 := &DriftItem{Name: "child2"}
	item := &DriftItem{
		Type:     DriftItemTypeChild,
		Children: []*DriftItem{child1, child2},
	}
	children := item.GetChildren()
	s.Len(children, 2)
	s.Equal("child1", children[0].GetName())
	s.Equal("child2", children[1].GetName())
}

// BuildDriftItems tests

func (s *DriftItemsTestSuite) Test_BuildDriftItems_returns_empty_for_empty_result() {
	result := &container.ReconciliationCheckResult{}
	items := BuildDriftItems(result, nil)
	s.Empty(items)
}

func (s *DriftItemsTestSuite) Test_BuildDriftItems_includes_parent_level_resources() {
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "resource1",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	items := BuildDriftItems(result, nil)
	s.Len(items, 1)
	driftItem := items[0].(*DriftItem)
	s.Equal("resource1", driftItem.Name)
	s.Equal(DriftItemTypeResource, driftItem.Type)
}

func (s *DriftItemsTestSuite) Test_BuildDriftItems_includes_parent_level_links() {
	result := &container.ReconciliationCheckResult{
		Links: []container.LinkReconcileResult{
			{
				LinkName: "link1",
				Type:     container.ReconciliationTypeDrift,
			},
		},
	}
	items := BuildDriftItems(result, nil)
	s.Len(items, 1)
	driftItem := items[0].(*DriftItem)
	s.Equal("link1", driftItem.Name)
	s.Equal(DriftItemTypeLink, driftItem.Type)
}

func (s *DriftItemsTestSuite) Test_BuildDriftItems_groups_child_resources_under_child_item() {
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "childResource",
				ResourceType: "aws/lambda/function",
				ChildPath:    "child1",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	items := BuildDriftItems(result, nil)
	s.Len(items, 1)
	childItem := items[0].(*DriftItem)
	s.Equal("child1", childItem.Name)
	s.Equal(DriftItemTypeChild, childItem.Type)
	s.Len(childItem.Children, 1)
	s.Equal("childResource", childItem.Children[0].Name)
}

func (s *DriftItemsTestSuite) Test_BuildDriftItems_handles_nested_child_paths() {
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "deepResource",
				ResourceType: "aws/dynamodb/table",
				ChildPath:    "child1.child2",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	items := BuildDriftItems(result, nil)
	s.Len(items, 1)
	child1 := items[0].(*DriftItem)
	s.Equal("child1", child1.Name)
	s.Equal("child1", child1.ChildPath)
}

func (s *DriftItemsTestSuite) Test_BuildDriftItems_includes_resource_state_when_available() {
	resourceState := &state.ResourceState{
		Name: "resource1",
		Type: "aws/s3/bucket",
	}
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"resource1": "res-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-123": resourceState,
		},
	}
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "resource1",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	items := BuildDriftItems(result, instanceState)
	s.Len(items, 1)
	driftItem := items[0].(*DriftItem)
	s.Equal(resourceState, driftItem.ResourceState)
}

// Human readable functions tests

func (s *DriftItemsTestSuite) Test_HumanReadableDriftType_drift() {
	s.Equal("DRIFT", HumanReadableDriftType(container.ReconciliationTypeDrift))
}

func (s *DriftItemsTestSuite) Test_HumanReadableDriftType_interrupted() {
	s.Equal("INTERRUPTED", HumanReadableDriftType(container.ReconciliationTypeInterrupted))
}

func (s *DriftItemsTestSuite) Test_HumanReadableDriftType_state_refresh() {
	s.Equal("STATE REFRESH", HumanReadableDriftType(container.ReconciliationTypeStateRefresh))
}

func (s *DriftItemsTestSuite) Test_HumanReadableDriftType_unknown() {
	s.Equal("unknown", HumanReadableDriftType("unknown"))
}

func (s *DriftItemsTestSuite) Test_HumanReadableAction_accept_external() {
	s.Equal("Accept external state", HumanReadableAction(container.ReconciliationActionAcceptExternal))
}

func (s *DriftItemsTestSuite) Test_HumanReadableAction_update_status() {
	s.Equal("Update status only", HumanReadableAction(container.ReconciliationActionUpdateStatus))
}

func (s *DriftItemsTestSuite) Test_HumanReadableAction_manual_cleanup() {
	s.Equal("Manual cleanup required", HumanReadableAction(container.ReconciliationActionManualCleanupRequired))
}

func (s *DriftItemsTestSuite) Test_HumanReadableDriftTypeLabel_drift() {
	s.Equal("Drift", HumanReadableDriftTypeLabel(container.ReconciliationTypeDrift))
}

func (s *DriftItemsTestSuite) Test_HumanReadableDriftTypeLabel_interrupted() {
	s.Equal("Interrupted", HumanReadableDriftTypeLabel(container.ReconciliationTypeInterrupted))
}

func (s *DriftItemsTestSuite) Test_HumanReadableDriftTypeLabel_state_refresh() {
	s.Equal("State refresh", HumanReadableDriftTypeLabel(container.ReconciliationTypeStateRefresh))
}

// DriftDetailsRenderer tests

func (s *DriftItemsTestSuite) Test_DriftDetailsRenderer_RenderDetails_resource() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myBucket",
		ResourceType: "aws/s3/bucket",
		DriftType:    container.ReconciliationTypeDrift,
		Recommended:  container.ReconciliationActionAcceptExternal,
	}
	result := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "myBucket")
	s.Contains(result, "aws/s3/bucket")
	s.Contains(result, "Drift")
}

func (s *DriftItemsTestSuite) Test_DriftDetailsRenderer_RenderDetails_link() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:        DriftItemTypeLink,
		Name:        "myLink",
		DriftType:   container.ReconciliationTypeDrift,
		Recommended: container.ReconciliationActionAcceptExternal,
	}
	result := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "myLink")
	s.Contains(result, "Drift")
}

func (s *DriftItemsTestSuite) Test_DriftDetailsRenderer_RenderDetails_child() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:      DriftItemTypeChild,
		Name:      "childBlueprint",
		ChildPath: "child1",
		Children: []*DriftItem{
			{Type: DriftItemTypeResource, Name: "res1"},
			{Type: DriftItemTypeResource, Name: "res2"},
		},
	}
	result := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "childBlueprint")
	s.Contains(result, "child1")
	s.Contains(result, "2 resources with drift")
}

func (s *DriftItemsTestSuite) Test_DriftDetailsRenderer_RenderDetails_unknown_type() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{Type: "unknown"}
	result := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "Unknown item type")
}

func (s *DriftItemsTestSuite) Test_DriftDetailsRenderer_RenderDetails_shows_changes() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/s3/bucket",
		DriftType:    container.ReconciliationTypeDrift,
		Recommended:  container.ReconciliationActionAcceptExternal,
		ResourceResult: &container.ResourceReconcileResult{
			Changes: &provider.Changes{
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.tags", PrevValue: stringNode("old"), NewValue: stringNode("new")},
				},
			},
		},
	}
	result := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "Changes")
	s.Contains(result, "spec.tags")
}

func (s *DriftItemsTestSuite) Test_DriftDetailsRenderer_shows_interrupted_resource_state() {
	renderer := &DriftDetailsRenderer{}
	item := &DriftItem{
		Type:         DriftItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/s3/bucket",
		DriftType:    container.ReconciliationTypeInterrupted,
		Recommended:  container.ReconciliationActionManualCleanupRequired,
		ResourceResult: &container.ResourceReconcileResult{
			ResourceExists: false,
		},
	}
	result := renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "Resource exists: No")
}

// DriftSectionGrouper tests

func (s *DriftItemsTestSuite) Test_DriftSectionGrouper_groups_by_type() {
	grouper := &DriftSectionGrouper{}
	items := []splitpane.Item{
		&DriftItem{Type: DriftItemTypeResource, Name: "res1"},
		&DriftItem{Type: DriftItemTypeLink, Name: "link1"},
		&DriftItem{Type: DriftItemTypeChild, Name: "child1"},
	}
	sections := grouper.GroupItems(items, nil)
	s.Len(sections, 3)

	s.Equal("Resources", sections[0].Name)
	s.Len(sections[0].Items, 1)

	s.Equal("Links", sections[1].Name)
	s.Len(sections[1].Items, 1)

	s.Equal("Child Blueprints", sections[2].Name)
	s.Len(sections[2].Items, 1)
}

func (s *DriftItemsTestSuite) Test_DriftSectionGrouper_omits_empty_sections() {
	grouper := &DriftSectionGrouper{}
	items := []splitpane.Item{
		&DriftItem{Type: DriftItemTypeResource, Name: "res1"},
	}
	sections := grouper.GroupItems(items, nil)
	s.Len(sections, 1)
	s.Equal("Resources", sections[0].Name)
}

func (s *DriftItemsTestSuite) Test_DriftSectionGrouper_sorts_items_alphabetically() {
	grouper := &DriftSectionGrouper{}
	items := []splitpane.Item{
		&DriftItem{Type: DriftItemTypeResource, Name: "zebra"},
		&DriftItem{Type: DriftItemTypeResource, Name: "alpha"},
		&DriftItem{Type: DriftItemTypeResource, Name: "beta"},
	}
	sections := grouper.GroupItems(items, nil)
	s.Equal("alpha", sections[0].Items[0].GetName())
	s.Equal("beta", sections[0].Items[1].GetName())
	s.Equal("zebra", sections[0].Items[2].GetName())
}

// SortDriftItems tests

func (s *DriftItemsTestSuite) Test_SortDriftItems_sorts_alphabetically() {
	items := []splitpane.Item{
		&DriftItem{Name: "zebra"},
		&DriftItem{Name: "alpha"},
		&DriftItem{Name: "beta"},
	}
	SortDriftItems(items)
	s.Equal("alpha", items[0].GetName())
	s.Equal("beta", items[1].GetName())
	s.Equal("zebra", items[2].GetName())
}

// Helper path functions tests

func (s *DriftItemsTestSuite) Test_splitDriftChildPath_empty() {
	result := splitDriftChildPath("")
	s.Nil(result)
}

func (s *DriftItemsTestSuite) Test_splitDriftChildPath_single() {
	result := splitDriftChildPath("child1")
	s.Equal([]string{"child1"}, result)
}

func (s *DriftItemsTestSuite) Test_splitDriftChildPath_nested() {
	result := splitDriftChildPath("child1.child2.child3")
	s.Equal([]string{"child1", "child2", "child3"}, result)
}

func (s *DriftItemsTestSuite) Test_joinDriftChildPath() {
	result := joinDriftChildPath([]string{"child1", "child2"})
	s.Equal("child1.child2", result)
}

// Helper function
func stringNode(val string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &val},
	}
}
