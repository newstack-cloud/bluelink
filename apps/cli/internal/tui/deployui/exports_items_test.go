package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ExportsItemsTestSuite struct {
	suite.Suite
}

func TestExportsItemsTestSuite(t *testing.T) {
	suite.Run(t, new(ExportsItemsTestSuite))
}

// --- ExportsInstanceItem GetID tests ---

func (s *ExportsItemsTestSuite) Test_GetID_returns_root_for_empty_path() {
	item := &ExportsInstanceItem{
		Name: "root-instance",
		Path: "",
	}
	s.Equal("root", item.GetID())
}

func (s *ExportsItemsTestSuite) Test_GetID_returns_path_for_non_empty_path() {
	item := &ExportsInstanceItem{
		Name: "child-instance",
		Path: "childA/childB",
	}
	s.Equal("childA/childB", item.GetID())
}

// --- ExportsInstanceItem GetName tests ---

func (s *ExportsItemsTestSuite) Test_GetName_returns_name() {
	item := &ExportsInstanceItem{
		Name: "my-instance",
	}
	s.Equal("my-instance", item.GetName())
}

// --- ExportsInstanceItem GetIcon tests ---

func (s *ExportsItemsTestSuite) Test_GetIcon_returns_empty_circle_for_no_exports() {
	item := &ExportsInstanceItem{
		Name:        "instance",
		ExportCount: 0,
	}
	s.Equal("○", item.GetIcon(false))
}

func (s *ExportsItemsTestSuite) Test_GetIcon_returns_filled_circle_for_has_exports() {
	item := &ExportsInstanceItem{
		Name:        "instance",
		ExportCount: 5,
	}
	s.Equal("●", item.GetIcon(false))
}

func (s *ExportsItemsTestSuite) Test_GetIcon_ignores_selected_parameter() {
	item := &ExportsInstanceItem{
		Name:        "instance",
		ExportCount: 3,
	}
	s.Equal(item.GetIcon(false), item.GetIcon(true))
}

// --- ExportsInstanceItem GetAction tests ---

func (s *ExportsItemsTestSuite) Test_GetAction_returns_export_count() {
	item := &ExportsInstanceItem{
		ExportCount: 3,
	}
	s.Equal("3 exports", item.GetAction())
}

func (s *ExportsItemsTestSuite) Test_GetAction_returns_zero_exports() {
	item := &ExportsInstanceItem{
		ExportCount: 0,
	}
	s.Equal("0 exports", item.GetAction())
}

// --- ExportsInstanceItem GetDepth tests ---

func (s *ExportsItemsTestSuite) Test_GetDepth_returns_depth() {
	item := &ExportsInstanceItem{
		Depth: 2,
	}
	s.Equal(2, item.GetDepth())
}

// --- ExportsInstanceItem GetParentID tests ---

func (s *ExportsItemsTestSuite) Test_GetParentID_returns_empty_for_root() {
	item := &ExportsInstanceItem{
		Path: "",
	}
	s.Equal("", item.GetParentID())
}

func (s *ExportsItemsTestSuite) Test_GetParentID_returns_root_for_direct_child() {
	item := &ExportsInstanceItem{
		Path: "childA",
	}
	s.Equal("root", item.GetParentID())
}

func (s *ExportsItemsTestSuite) Test_GetParentID_returns_parent_path_for_nested_child() {
	item := &ExportsInstanceItem{
		Path: "childA/childB",
	}
	s.Equal("childA", item.GetParentID())
}

func (s *ExportsItemsTestSuite) Test_GetParentID_returns_nested_parent_path() {
	item := &ExportsInstanceItem{
		Path: "childA/childB/childC",
	}
	s.Equal("childA/childB", item.GetParentID())
}

// --- ExportsInstanceItem GetItemType tests ---

func (s *ExportsItemsTestSuite) Test_GetItemType_returns_empty_for_root() {
	item := &ExportsInstanceItem{
		Depth: 0,
	}
	s.Equal("", item.GetItemType())
}

func (s *ExportsItemsTestSuite) Test_GetItemType_returns_child_for_non_root() {
	item := &ExportsInstanceItem{
		Depth: 1,
	}
	s.Equal("child", item.GetItemType())
}

func (s *ExportsItemsTestSuite) Test_GetItemType_returns_child_for_deeply_nested() {
	item := &ExportsInstanceItem{
		Depth: 3,
	}
	s.Equal("child", item.GetItemType())
}

// --- ExportsInstanceItem IsExpandable tests ---

func (s *ExportsItemsTestSuite) Test_IsExpandable_returns_false() {
	item := &ExportsInstanceItem{}
	s.False(item.IsExpandable())
}

// --- ExportsInstanceItem CanDrillDown tests ---

func (s *ExportsItemsTestSuite) Test_CanDrillDown_returns_false() {
	item := &ExportsInstanceItem{}
	s.False(item.CanDrillDown())
}

// --- ExportsInstanceItem GetChildren tests ---

func (s *ExportsItemsTestSuite) Test_GetChildren_returns_nil() {
	item := &ExportsInstanceItem{}
	s.Nil(item.GetChildren())
}

// --- BuildInstanceHierarchy tests ---

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_returns_nil_for_nil_root() {
	items := BuildInstanceHierarchy(nil, "test")
	s.Nil(items)
}

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_returns_root_with_provided_name() {
	root := &state.InstanceState{
		InstanceID: "inst-123",
		Exports: map[string]*state.ExportState{
			"export1": {},
			"export2": {},
		},
	}
	items := BuildInstanceHierarchy(root, "my-blueprint")
	s.Require().Len(items, 1)

	rootItem := items[0].(*ExportsInstanceItem)
	s.Equal("my-blueprint", rootItem.Name)
	s.Equal("", rootItem.Path)
	s.Equal("inst-123", rootItem.InstanceID)
	s.Equal(2, rootItem.ExportCount)
	s.Equal(0, rootItem.Depth)
}

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_uses_default_name_when_empty() {
	root := &state.InstanceState{
		InstanceID: "inst-123",
	}
	items := BuildInstanceHierarchy(root, "")
	s.Require().Len(items, 1)

	rootItem := items[0].(*ExportsInstanceItem)
	s.Equal("(root)", rootItem.Name)
}

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_includes_children() {
	root := &state.InstanceState{
		InstanceID: "inst-root",
		Exports: map[string]*state.ExportState{
			"rootExport": {},
		},
		ChildBlueprints: map[string]*state.InstanceState{
			"childA": {
				InstanceID: "inst-child-a",
				Exports: map[string]*state.ExportState{
					"childExport1": {},
					"childExport2": {},
				},
			},
		},
	}
	items := BuildInstanceHierarchy(root, "root")
	s.Require().Len(items, 2)

	rootItem := items[0].(*ExportsInstanceItem)
	s.Equal("root", rootItem.Name)
	s.Equal(0, rootItem.Depth)

	childItem := items[1].(*ExportsInstanceItem)
	s.Equal("childA", childItem.Name)
	s.Equal("childA", childItem.Path)
	s.Equal("inst-child-a", childItem.InstanceID)
	s.Equal(2, childItem.ExportCount)
	s.Equal(1, childItem.Depth)
}

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_sorts_children_alphabetically() {
	root := &state.InstanceState{
		InstanceID: "inst-root",
		ChildBlueprints: map[string]*state.InstanceState{
			"zebra": {InstanceID: "inst-zebra"},
			"alpha": {InstanceID: "inst-alpha"},
			"mike":  {InstanceID: "inst-mike"},
		},
	}
	items := BuildInstanceHierarchy(root, "root")
	s.Require().Len(items, 4)

	s.Equal("root", items[0].(*ExportsInstanceItem).Name)
	s.Equal("alpha", items[1].(*ExportsInstanceItem).Name)
	s.Equal("mike", items[2].(*ExportsInstanceItem).Name)
	s.Equal("zebra", items[3].(*ExportsInstanceItem).Name)
}

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_includes_nested_grandchildren() {
	root := &state.InstanceState{
		InstanceID: "inst-root",
		ChildBlueprints: map[string]*state.InstanceState{
			"childA": {
				InstanceID: "inst-child-a",
				ChildBlueprints: map[string]*state.InstanceState{
					"grandchildA1": {
						InstanceID: "inst-grandchild-a1",
					},
				},
			},
		},
	}
	items := BuildInstanceHierarchy(root, "root")
	s.Require().Len(items, 3)

	rootItem := items[0].(*ExportsInstanceItem)
	s.Equal("root", rootItem.Name)
	s.Equal(0, rootItem.Depth)

	childItem := items[1].(*ExportsInstanceItem)
	s.Equal("childA", childItem.Name)
	s.Equal("childA", childItem.Path)
	s.Equal(1, childItem.Depth)

	grandchildItem := items[2].(*ExportsInstanceItem)
	s.Equal("grandchildA1", grandchildItem.Name)
	s.Equal("childA/grandchildA1", grandchildItem.Path)
	s.Equal(2, grandchildItem.Depth)
}

func (s *ExportsItemsTestSuite) Test_BuildInstanceHierarchy_skips_nil_children() {
	root := &state.InstanceState{
		InstanceID: "inst-root",
		ChildBlueprints: map[string]*state.InstanceState{
			"validChild": {InstanceID: "inst-valid"},
			"nilChild":   nil,
		},
	}
	items := BuildInstanceHierarchy(root, "root")
	s.Require().Len(items, 2)

	s.Equal("root", items[0].(*ExportsInstanceItem).Name)
	s.Equal("validChild", items[1].(*ExportsInstanceItem).Name)
}

// --- joinInstancePath tests ---

func (s *ExportsItemsTestSuite) Test_joinInstancePath_returns_child_for_empty_parent() {
	result := joinInstancePath("", "child")
	s.Equal("child", result)
}

func (s *ExportsItemsTestSuite) Test_joinInstancePath_joins_with_slash() {
	result := joinInstancePath("parent", "child")
	s.Equal("parent/child", result)
}

func (s *ExportsItemsTestSuite) Test_joinInstancePath_handles_nested_paths() {
	result := joinInstancePath("grandparent/parent", "child")
	s.Equal("grandparent/parent/child", result)
}
