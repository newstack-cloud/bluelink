package stageui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type StageRenderersTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
	renderer   *StageDetailsRenderer
}

func TestStageRenderersTestSuite(t *testing.T) {
	suite.Run(t, new(StageRenderersTestSuite))
}

func (s *StageRenderersTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		styles.NewBluelinkPalette(),
	)
	s.renderer = &StageDetailsRenderer{
		MaxExpandDepth:       3,
		NavigationStackDepth: 0,
	}
}

// --- RenderDetails tests ---

func (s *StageRenderersTestSuite) Test_RenderDetails_returns_unknown_for_wrong_type() {
	// Create a mock item that isn't a *StageItem
	result := s.renderer.RenderDetails(&mockItem{}, 80, s.testStyles)
	s.Contains(result, "Unknown item type")
}

func (s *StageRenderersTestSuite) Test_RenderDetails_renders_resource_details() {
	item := &StageItem{
		Type:         ItemTypeResource,
		Name:         "myResource",
		ResourceType: "aws/s3/bucket",
		Action:       ActionCreate,
		Changes:      &provider.Changes{},
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "myResource")
	s.Contains(result, "aws/s3/bucket")
}

func (s *StageRenderersTestSuite) Test_RenderDetails_renders_child_details() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "childBlueprint",
		Action:  ActionUpdate,
		Changes: &changes.BlueprintChanges{},
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "childBlueprint")
}

func (s *StageRenderersTestSuite) Test_RenderDetails_renders_link_details() {
	item := &StageItem{
		Type:    ItemTypeLink,
		Name:    "resourceA::resourceB",
		Action:  ActionCreate,
		Changes: &provider.LinkChanges{},
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "resourceA::resourceB")
}

func (s *StageRenderersTestSuite) Test_RenderDetails_returns_unknown_for_unknown_type() {
	item := &StageItem{
		Type: ItemType("unknown_type"), // Unknown type
		Name: "unknown",
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "Unknown item type")
}

// --- renderResourceDetails tests ---

func (s *StageRenderersTestSuite) Test_renderResourceDetails_shows_display_name_in_header() {
	item := &StageItem{
		Type:         ItemTypeResource,
		Name:         "myResource",
		DisplayName:  "My Display Name",
		ResourceType: "aws/s3/bucket",
		Action:       ActionCreate,
		Changes:      &provider.Changes{},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "My Display Name")
	s.Contains(result, "Name:")
	s.Contains(result, "myResource")
}

func (s *StageRenderersTestSuite) Test_renderResourceDetails_shows_removed_message() {
	item := &StageItem{
		Type:    ItemTypeResource,
		Name:    "deletedResource",
		Action:  ActionDelete,
		Removed: true,
		Changes: &provider.Changes{},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "will be destroyed")
}

func (s *StageRenderersTestSuite) Test_renderResourceDetails_shows_no_changes() {
	item := &StageItem{
		Type:    ItemTypeResource,
		Name:    "unchangedResource",
		Action:  ActionNoChange,
		Changes: &provider.Changes{},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "No changes")
}

func (s *StageRenderersTestSuite) Test_renderResourceDetails_shows_resource_id_from_changes() {
	item := &StageItem{
		Type:   ItemTypeResource,
		Name:   "myResource",
		Action: ActionUpdate,
		Changes: &provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				ResourceID: "res-123-from-changes",
			},
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "res-123-from-changes")
}

func (s *StageRenderersTestSuite) Test_renderResourceDetails_shows_resource_id_from_state() {
	item := &StageItem{
		Type:    ItemTypeResource,
		Name:    "myResource",
		Action:  ActionUpdate,
		Changes: &provider.Changes{},
		ResourceState: &state.ResourceState{
			ResourceID: "res-456-from-state",
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "res-456-from-state")
}

func (s *StageRenderersTestSuite) Test_renderResourceDetails_shows_outputs_from_state() {
	item := &StageItem{
		Type:    ItemTypeResource,
		Name:    "myResource",
		Action:  ActionUpdate,
		Changes: &provider.Changes{},
		ResourceState: &state.ResourceState{
			ResourceID:     "res-123",
			SpecData:       &core.MappingNode{},
			ComputedFields: []string{"arn"},
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	// Output section should be rendered (even if empty when SpecData has no matching fields)
	s.NotEmpty(result)
}

// --- renderResourceChanges tests ---

func (s *StageRenderersTestSuite) Test_renderResourceChanges_shows_new_fields() {
	resourceChanges := &provider.Changes{
		NewFields: []provider.FieldChange{
			{FieldPath: "spec.bucketName", NewValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("my-bucket")}}},
		},
	}
	result := s.renderer.renderResourceChanges(resourceChanges, 80, s.testStyles)
	s.Contains(result, "Field Changes")
	s.Contains(result, "spec.bucketName")
	s.Contains(result, "my-bucket")
}

func (s *StageRenderersTestSuite) Test_renderResourceChanges_shows_modified_fields() {
	resourceChanges := &provider.Changes{
		ModifiedFields: []provider.FieldChange{
			{
				FieldPath: "spec.size",
				PrevValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("10")}},
				NewValue:  &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("20")}},
			},
		},
	}
	result := s.renderer.renderResourceChanges(resourceChanges, 80, s.testStyles)
	s.Contains(result, "Field Changes")
	s.Contains(result, "spec.size")
	s.Contains(result, "10")
	s.Contains(result, "20")
}

func (s *StageRenderersTestSuite) Test_renderResourceChanges_shows_removed_fields() {
	resourceChanges := &provider.Changes{
		RemovedFields: []string{"spec.oldField"},
	}
	result := s.renderer.renderResourceChanges(resourceChanges, 80, s.testStyles)
	s.Contains(result, "Field Changes")
	s.Contains(result, "spec.oldField")
}

func (s *StageRenderersTestSuite) Test_renderResourceChanges_shows_no_changes_when_empty() {
	resourceChanges := &provider.Changes{}
	result := s.renderer.renderResourceChanges(resourceChanges, 80, s.testStyles)
	s.Contains(result, "No changes")
}

func (s *StageRenderersTestSuite) Test_renderResourceChanges_shows_outbound_link_changes() {
	resourceChanges := &provider.Changes{
		NewOutboundLinks: map[string]provider.LinkChanges{
			"targetResource": {},
		},
	}
	result := s.renderer.renderResourceChanges(resourceChanges, 80, s.testStyles)
	s.Contains(result, "Outbound Link Changes")
	s.Contains(result, "targetResource")
	s.Contains(result, "new link")
}

// --- renderOutboundLinkChanges tests ---

func (s *StageRenderersTestSuite) Test_renderOutboundLinkChanges_shows_new_links() {
	resourceChanges := &provider.Changes{
		NewOutboundLinks: map[string]provider.LinkChanges{
			"newTarget": {
				NewFields: []*provider.FieldChange{
					{FieldPath: "linkField", NewValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("value")}}},
				},
			},
		},
	}
	result := s.renderer.renderOutboundLinkChanges(resourceChanges, s.testStyles)
	s.Contains(result, "newTarget")
	s.Contains(result, "new link")
	s.Contains(result, "linkField")
}

func (s *StageRenderersTestSuite) Test_renderOutboundLinkChanges_shows_modified_links() {
	resourceChanges := &provider.Changes{
		OutboundLinkChanges: map[string]provider.LinkChanges{
			"modifiedTarget": {},
		},
	}
	result := s.renderer.renderOutboundLinkChanges(resourceChanges, s.testStyles)
	s.Contains(result, "modifiedTarget")
	s.Contains(result, "link updated")
}

func (s *StageRenderersTestSuite) Test_renderOutboundLinkChanges_shows_removed_links() {
	resourceChanges := &provider.Changes{
		RemovedOutboundLinks: []string{"removedTarget"},
	}
	result := s.renderer.renderOutboundLinkChanges(resourceChanges, s.testStyles)
	s.Contains(result, "removedTarget")
	s.Contains(result, "link removed")
}

// --- renderChildDetails tests ---

func (s *StageRenderersTestSuite) Test_renderChildDetails_shows_basic_info() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "childBlueprint",
		Action:  ActionUpdate,
		Changes: &changes.BlueprintChanges{},
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "childBlueprint")
	s.Contains(result, "Changes computed")
}

func (s *StageRenderersTestSuite) Test_renderChildDetails_shows_removed_message() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "deletedChild",
		Action:  ActionDelete,
		Removed: true,
		Changes: &changes.BlueprintChanges{},
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "will be destroyed")
}

func (s *StageRenderersTestSuite) Test_renderChildDetails_shows_instance_id() {
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "childBlueprint",
		Action:  ActionUpdate,
		Changes: &changes.BlueprintChanges{},
		InstanceState: &state.InstanceState{
			InstanceID: "child-instance-123",
		},
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "child-instance-123")
}

func (s *StageRenderersTestSuite) Test_renderChildDetails_shows_drill_down_hint_at_max_depth() {
	renderer := &StageDetailsRenderer{
		MaxExpandDepth:       2,
		NavigationStackDepth: 0,
	}
	item := &StageItem{
		Type:    ItemTypeChild,
		Name:    "deepChild",
		Action:  ActionUpdate,
		Changes: &changes.BlueprintChanges{},
		Depth:   2, // At max depth
	}
	result := renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "Press enter to inspect")
}

func (s *StageRenderersTestSuite) Test_renderChildDetails_shows_changes_summary() {
	item := &StageItem{
		Type:   ItemTypeChild,
		Name:   "childBlueprint",
		Action: ActionUpdate,
		Changes: &changes.BlueprintChanges{
			NewResources: map[string]provider.Changes{
				"newRes1": {},
				"newRes2": {},
			},
			ResourceChanges: map[string]provider.Changes{
				"updatedRes": {},
			},
			RemovedResources: []string{"deletedRes"},
		},
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "Summary")
	s.Contains(result, "2")
	s.Contains(result, "to be created")
	s.Contains(result, "to be updated")
	s.Contains(result, "to be removed")
}

// --- renderChildChangesSummary tests ---

func (s *StageRenderersTestSuite) Test_renderChildChangesSummary_shows_child_blueprint_changes() {
	childChanges := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"newChild1": {},
			"newChild2": {},
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"updatedChild": {},
		},
		RemovedChildren: []string{"deletedChild"},
	}
	result := s.renderer.renderChildChangesSummary(childChanges, s.testStyles)
	s.Contains(result, "2 child blueprints to be created")
	s.Contains(result, "1 child blueprint to be updated")
	s.Contains(result, "1 child blueprint to be removed")
}

func (s *StageRenderersTestSuite) Test_renderChildChangesSummary_returns_empty_for_no_changes() {
	childChanges := &changes.BlueprintChanges{}
	result := s.renderer.renderChildChangesSummary(childChanges, s.testStyles)
	s.Empty(result)
}

// --- renderLinkDetails tests ---

func (s *StageRenderersTestSuite) Test_renderLinkDetails_shows_basic_info() {
	item := &StageItem{
		Type:    ItemTypeLink,
		Name:    "resourceA::resourceB",
		Action:  ActionCreate,
		Changes: &provider.LinkChanges{},
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "resourceA::resourceB")
	s.Contains(result, "Changes computed")
}

func (s *StageRenderersTestSuite) Test_renderLinkDetails_shows_removed_message() {
	item := &StageItem{
		Type:    ItemTypeLink,
		Name:    "resourceA::resourceB",
		Action:  ActionDelete,
		Removed: true,
		Changes: &provider.LinkChanges{},
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "will be destroyed")
}

func (s *StageRenderersTestSuite) Test_renderLinkDetails_shows_link_id() {
	item := &StageItem{
		Type:    ItemTypeLink,
		Name:    "resourceA::resourceB",
		Action:  ActionUpdate,
		Changes: &provider.LinkChanges{},
		LinkState: &state.LinkState{
			LinkID: "link-456",
		},
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "link-456")
}

// --- renderLinkChanges tests ---

func (s *StageRenderersTestSuite) Test_renderLinkChanges_shows_new_fields() {
	linkChanges := &provider.LinkChanges{
		NewFields: []*provider.FieldChange{
			{FieldPath: "linkData.field1", NewValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("value1")}}},
		},
	}
	result := s.renderer.renderLinkChanges(linkChanges, s.testStyles)
	s.Contains(result, "Changes")
	s.Contains(result, "linkData.field1")
	s.Contains(result, "value1")
}

func (s *StageRenderersTestSuite) Test_renderLinkChanges_shows_modified_fields() {
	linkChanges := &provider.LinkChanges{
		ModifiedFields: []*provider.FieldChange{
			{
				FieldPath: "linkData.field1",
				PrevValue: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("old")}},
				NewValue:  &core.MappingNode{Scalar: &core.ScalarValue{StringValue: strPtr("new")}},
			},
		},
	}
	result := s.renderer.renderLinkChanges(linkChanges, s.testStyles)
	s.Contains(result, "old")
	s.Contains(result, "new")
}

func (s *StageRenderersTestSuite) Test_renderLinkChanges_shows_removed_fields() {
	linkChanges := &provider.LinkChanges{
		RemovedFields: []string{"linkData.oldField"},
	}
	result := s.renderer.renderLinkChanges(linkChanges, s.testStyles)
	s.Contains(result, "linkData.oldField")
}

func (s *StageRenderersTestSuite) Test_renderLinkChanges_shows_no_field_changes() {
	linkChanges := &provider.LinkChanges{}
	result := s.renderer.renderLinkChanges(linkChanges, s.testStyles)
	s.Contains(result, "No field changes")
}

// --- forceWrap tests ---

func (s *StageRenderersTestSuite) Test_forceWrap_breaks_long_text() {
	result := s.renderer.forceWrap("abcdefghij", 3)
	s.Equal([]string{"abc", "def", "ghi", "j"}, result)
}

func (s *StageRenderersTestSuite) Test_forceWrap_returns_original_for_short_text() {
	result := s.renderer.forceWrap("abc", 10)
	s.Equal([]string{"abc"}, result)
}

func (s *StageRenderersTestSuite) Test_forceWrap_handles_zero_width() {
	result := s.renderer.forceWrap("abc", 0)
	s.Equal([]string{"abc"}, result)
}

func (s *StageRenderersTestSuite) Test_forceWrap_handles_negative_width() {
	result := s.renderer.forceWrap("abc", -1)
	s.Equal([]string{"abc"}, result)
}

// --- StageFooterRenderer tests ---

func (s *StageRenderersTestSuite) Test_StageFooterRenderer_uses_delegate_when_set() {
	delegate := &mockFooterRenderer{output: "delegate output"}
	footer := &StageFooterRenderer{
		ChangesetID: "cs-123",
		Delegate:    delegate,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Equal("delegate output", result)
}

func (s *StageRenderersTestSuite) Test_StageFooterRenderer_shows_changeset_id() {
	footer := &StageFooterRenderer{
		ChangesetID: "cs-123",
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "cs-123")
}

func (s *StageRenderersTestSuite) Test_StageFooterRenderer_shows_deploy_instruction() {
	footer := &StageFooterRenderer{
		ChangesetID: "cs-123",
		CreateCount: 1,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "bluelink deploy")
}

func (s *StageRenderersTestSuite) Test_StageFooterRenderer_shows_destroy_instruction_when_destroy() {
	footer := &StageFooterRenderer{
		ChangesetID: "cs-123",
		Destroy:     true,
		DeleteCount: 1,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "bluelink destroy")
}

func (s *StageRenderersTestSuite) Test_StageFooterRenderer_shows_no_changes_message() {
	footer := &StageFooterRenderer{
		ChangesetID: "cs-123",
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "No changes to apply")
}

func (s *StageRenderersTestSuite) Test_StageFooterRenderer_shows_exports_key_hint() {
	footer := &StageFooterRenderer{
		ChangesetID:      "cs-123",
		HasExportChanges: true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "exports")
}

// --- renderChangeSummary tests ---

func (s *StageRenderersTestSuite) Test_renderChangeSummary_shows_create_count() {
	footer := &StageFooterRenderer{CreateCount: 3}
	result := footer.renderChangeSummary(s.testStyles)
	s.Contains(result, "3 creates")
}

func (s *StageRenderersTestSuite) Test_renderChangeSummary_shows_update_count() {
	footer := &StageFooterRenderer{UpdateCount: 2}
	result := footer.renderChangeSummary(s.testStyles)
	s.Contains(result, "2 updates")
}

func (s *StageRenderersTestSuite) Test_renderChangeSummary_shows_recreate_count() {
	footer := &StageFooterRenderer{RecreateCount: 1}
	result := footer.renderChangeSummary(s.testStyles)
	s.Contains(result, "1 recreate")
}

func (s *StageRenderersTestSuite) Test_renderChangeSummary_shows_delete_count() {
	footer := &StageFooterRenderer{DeleteCount: 4}
	result := footer.renderChangeSummary(s.testStyles)
	s.Contains(result, "4 deletes")
}

func (s *StageRenderersTestSuite) Test_renderChangeSummary_shows_all_counts() {
	footer := &StageFooterRenderer{
		CreateCount:   1,
		UpdateCount:   2,
		RecreateCount: 3,
		DeleteCount:   4,
	}
	result := footer.renderChangeSummary(s.testStyles)
	s.Contains(result, "1 create")
	s.Contains(result, "2 updates")
	s.Contains(result, "3 recreates")
	s.Contains(result, "4 deletes")
}

// --- sortedMapKeys tests ---

func (s *StageRenderersTestSuite) Test_sortedMapKeys_returns_sorted_keys() {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	result := sortedMapKeys(m)
	s.Equal([]string{"a", "b", "c"}, result)
}

func (s *StageRenderersTestSuite) Test_sortedMapKeys_handles_empty_map() {
	m := map[string]int{}
	result := sortedMapKeys(m)
	s.Empty(result)
}

// --- Helper types ---

type mockItem struct{}

func (m *mockItem) GetID() string                                   { return "mock" }
func (m *mockItem) GetName() string                                 { return "mock" }
func (m *mockItem) GetIcon(bool) string                             { return "" }
func (m *mockItem) GetIconStyled(*styles.Styles, bool) string       { return "" }
func (m *mockItem) GetAction() string                               { return "" }
func (m *mockItem) GetDepth() int                                   { return 0 }
func (m *mockItem) GetParentID() string                             { return "" }
func (m *mockItem) GetItemType() string                             { return "" }
func (m *mockItem) IsExpandable() bool                              { return false }
func (m *mockItem) CanDrillDown() bool                              { return false }
func (m *mockItem) GetChildren() []splitpane.Item                   { return nil }

type mockFooterRenderer struct {
	output string
}

func (m *mockFooterRenderer) RenderFooter(*splitpane.Model, *styles.Styles) string {
	return m.output
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
