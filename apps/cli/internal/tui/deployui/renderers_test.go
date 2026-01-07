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
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
	"github.com/stretchr/testify/suite"
)

type DeployRenderersTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
	renderer   *DeployDetailsRenderer
}

func TestDeployRenderersTestSuite(t *testing.T) {
	suite.Run(t, new(DeployRenderersTestSuite))
}

func (s *DeployRenderersTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		styles.NewBluelinkPalette(),
	)
	s.renderer = &DeployDetailsRenderer{
		MaxExpandDepth:       3,
		NavigationStackDepth: 0,
	}
}

// --- RenderDetails tests ---

func (s *DeployRenderersTestSuite) Test_RenderDetails_returns_unknown_for_wrong_type() {
	result := s.renderer.RenderDetails(&mockItem{}, 80, s.testStyles)
	s.Contains(result, "Unknown item type")
}

func (s *DeployRenderersTestSuite) Test_RenderDetails_renders_resource_details() {
	item := &DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:         "myResource",
			ResourceType: "aws/s3/bucket",
			Action:       shared.ActionCreate,
			Status:       core.ResourceStatusCreated,
		},
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "myResource")
	s.Contains(result, "aws/s3/bucket")
}

func (s *DeployRenderersTestSuite) Test_RenderDetails_renders_child_details() {
	item := &DeployItem{
		Type: ItemTypeChild,
		Child: &ChildDeployItem{
			Name:   "childBlueprint",
			Action: shared.ActionUpdate,
			Status: core.InstanceStatusDeployed,
		},
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "childBlueprint")
}

func (s *DeployRenderersTestSuite) Test_RenderDetails_renders_link_details() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{
			LinkName:      "resourceA::resourceB",
			ResourceAName: "resourceA",
			ResourceBName: "resourceB",
			Action:        shared.ActionCreate,
			Status:        core.LinkStatusCreated,
		},
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "resourceA::resourceB")
}

func (s *DeployRenderersTestSuite) Test_RenderDetails_returns_unknown_for_unknown_type() {
	item := &DeployItem{
		Type: ItemType("unknown_type"),
	}
	result := s.renderer.RenderDetails(item, 80, s.testStyles)
	s.Contains(result, "Unknown item type")
}

// --- renderResourceDetails tests ---

func (s *DeployRenderersTestSuite) Test_renderResourceDetails_returns_no_data_for_nil_resource() {
	item := &DeployItem{
		Type:     ItemTypeResource,
		Resource: nil,
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "No resource data")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDetails_shows_display_name() {
	item := &DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:        "myResource",
			DisplayName: "My Display Name",
			Action:      shared.ActionCreate,
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "My Display Name")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDetails_shows_skipped_status() {
	item := &DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:    "skippedResource",
			Action:  shared.ActionCreate,
			Skipped: true,
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "Skipped")
	s.Contains(result, "deployment failure")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDetails_shows_attempt_info() {
	item := &DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:     "retriedResource",
			Action:   shared.ActionCreate,
			Attempt:  3,
			CanRetry: true,
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "Attempt:")
	s.Contains(result, "3")
	s.Contains(result, "can retry")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDetails_shows_failure_reasons() {
	item := &DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:           "failedResource",
			Action:         shared.ActionCreate,
			Status:         core.ResourceStatusCreateFailed,
			FailureReasons: []string{"Connection timeout", "Permission denied"},
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "Connection timeout")
	s.Contains(result, "Permission denied")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDetails_shows_durations() {
	configDuration := float64(5000)
	totalDuration := float64(10000)
	item := &DeployItem{
		Type: ItemTypeResource,
		Resource: &ResourceDeployItem{
			Name:   "timedResource",
			Action: shared.ActionCreate,
			Status: core.ResourceStatusCreated,
			Durations: &state.ResourceCompletionDurations{
				ConfigCompleteDuration: &configDuration,
				TotalDuration:          &totalDuration,
			},
		},
	}
	result := s.renderer.renderResourceDetails(item, 80, s.testStyles)
	s.Contains(result, "Timing")
	s.Contains(result, "Config Complete")
	s.Contains(result, "Total")
}

// --- getResourceState tests ---

func (s *DeployRenderersTestSuite) Test_getResourceState_returns_post_deploy_state_first() {
	postDeployState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-123": {ResourceID: "res-123", Name: "myResource"},
		},
		ResourceIDs: map[string]string{"myResource": "res-123"},
	}
	renderer := &DeployDetailsRenderer{
		PostDeployInstanceState: postDeployState,
	}
	res := &ResourceDeployItem{Name: "myResource"}

	result := renderer.getResourceState(res, "myResource")
	s.NotNil(result)
	s.Equal("res-123", result.ResourceID)
}

func (s *DeployRenderersTestSuite) Test_getResourceState_falls_back_to_pre_deploy_state() {
	preDeployState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-456": {ResourceID: "res-456", Name: "myResource"},
		},
		ResourceIDs: map[string]string{"myResource": "res-456"},
	}
	renderer := &DeployDetailsRenderer{
		PreDeployInstanceState: preDeployState,
	}
	res := &ResourceDeployItem{Name: "myResource"}

	result := renderer.getResourceState(res, "myResource")
	s.NotNil(result)
	s.Equal("res-456", result.ResourceID)
}

func (s *DeployRenderersTestSuite) Test_getResourceState_falls_back_to_resource_state_field() {
	resourceState := &state.ResourceState{ResourceID: "res-789", Name: "myResource"}
	renderer := &DeployDetailsRenderer{}
	res := &ResourceDeployItem{
		Name:          "myResource",
		ResourceState: resourceState,
	}

	result := renderer.getResourceState(res, "myResource")
	s.NotNil(result)
	s.Equal("res-789", result.ResourceID)
}

func (s *DeployRenderersTestSuite) Test_getResourceState_falls_back_to_changeset() {
	changesetState := &state.ResourceState{ResourceID: "res-999", Name: "myResource"}
	renderer := &DeployDetailsRenderer{}
	res := &ResourceDeployItem{
		Name: "myResource",
		Changes: &provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				CurrentResourceState: changesetState,
			},
		},
	}

	result := renderer.getResourceState(res, "myResource")
	s.NotNil(result)
	s.Equal("res-999", result.ResourceID)
}

func (s *DeployRenderersTestSuite) Test_getResourceState_returns_nil_when_not_found() {
	renderer := &DeployDetailsRenderer{}
	res := &ResourceDeployItem{Name: "missingResource"}

	result := renderer.getResourceState(res, "missingResource")
	s.Nil(result)
}

// --- findResourceStateByPath tests ---

func (s *DeployRenderersTestSuite) Test_findResourceStateByPath_finds_top_level_resource() {
	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-123": {ResourceID: "res-123", Name: "myResource"},
		},
		ResourceIDs: map[string]string{"myResource": "res-123"},
	}

	result := findResourceStateByPath(instanceState, "myResource", "myResource")
	s.NotNil(result)
	s.Equal("res-123", result.ResourceID)
}

func (s *DeployRenderersTestSuite) Test_findResourceStateByPath_finds_nested_resource() {
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childA": {
				Resources: map[string]*state.ResourceState{
					"res-nested": {ResourceID: "res-nested", Name: "nestedResource"},
				},
				ResourceIDs: map[string]string{"nestedResource": "res-nested"},
			},
		},
	}

	result := findResourceStateByPath(instanceState, "childA/nestedResource", "nestedResource")
	s.NotNil(result)
	s.Equal("res-nested", result.ResourceID)
}

func (s *DeployRenderersTestSuite) Test_findResourceStateByPath_returns_nil_for_missing_child() {
	instanceState := &state.InstanceState{}

	result := findResourceStateByPath(instanceState, "missingChild/resource", "resource")
	s.Nil(result)
}

func (s *DeployRenderersTestSuite) Test_findResourceStateByPath_returns_nil_for_nil_state() {
	result := findResourceStateByPath(nil, "path", "resource")
	s.Nil(result)
}

// --- renderChildDetails tests ---

func (s *DeployRenderersTestSuite) Test_renderChildDetails_returns_no_data_for_nil_child() {
	item := &DeployItem{
		Type:  ItemTypeChild,
		Child: nil,
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "No child data")
}

func (s *DeployRenderersTestSuite) Test_renderChildDetails_shows_instance_ids() {
	item := &DeployItem{
		Type: ItemTypeChild,
		Child: &ChildDeployItem{
			Name:             "childBlueprint",
			ChildInstanceID:  "child-123",
			ParentInstanceID: "parent-456",
			Action:           shared.ActionCreate,
			Status:           core.InstanceStatusDeployed,
		},
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "child-123")
	s.Contains(result, "parent-456")
}

func (s *DeployRenderersTestSuite) Test_renderChildDetails_shows_skipped_status() {
	item := &DeployItem{
		Type: ItemTypeChild,
		Child: &ChildDeployItem{
			Name:    "skippedChild",
			Action:  shared.ActionCreate,
			Skipped: true,
		},
	}
	result := s.renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "Skipped")
	s.Contains(result, "deployment failure")
}

func (s *DeployRenderersTestSuite) Test_renderChildDetails_shows_drill_down_hint_at_max_depth() {
	renderer := &DeployDetailsRenderer{
		MaxExpandDepth:       2,
		NavigationStackDepth: 0,
	}
	item := &DeployItem{
		Type: ItemTypeChild,
		Child: &ChildDeployItem{
			Name:   "deepChild",
			Action: shared.ActionUpdate,
			Status: core.InstanceStatusDeployed,
		},
		Changes: &changes.BlueprintChanges{}, // Non-nil changes
		Depth:   2,
	}
	result := renderer.renderChildDetails(item, 80, s.testStyles)
	s.Contains(result, "Press enter to inspect")
}

// --- renderLinkDetails tests ---

func (s *DeployRenderersTestSuite) Test_renderLinkDetails_returns_no_data_for_nil_link() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: nil,
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "No link data")
}

func (s *DeployRenderersTestSuite) Test_renderLinkDetails_shows_link_info() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{
			LinkID:        "link-123",
			LinkName:      "resourceA::resourceB",
			ResourceAName: "resourceA",
			ResourceBName: "resourceB",
			Action:        shared.ActionCreate,
			Status:        core.LinkStatusCreated,
		},
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "resourceA::resourceB")
	s.Contains(result, "link-123")
}

func (s *DeployRenderersTestSuite) Test_renderLinkDetails_shows_skipped_status() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{
			LinkName: "resourceA::resourceB",
			Action:   shared.ActionCreate,
			Skipped:  true,
		},
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "Skipped")
	s.Contains(result, "deployment failure")
}

func (s *DeployRenderersTestSuite) Test_renderLinkDetails_shows_stage_attempt() {
	item := &DeployItem{
		Type: ItemTypeLink,
		Link: &LinkDeployItem{
			LinkName:             "resourceA::resourceB",
			Action:               shared.ActionCreate,
			Status:               core.LinkStatusCreating,
			CurrentStageAttempt:  3,
			CanRetryCurrentStage: true,
		},
	}
	result := s.renderer.renderLinkDetails(item, 80, s.testStyles)
	s.Contains(result, "Stage Attempt:")
	s.Contains(result, "3")
	s.Contains(result, "can retry")
}

// --- renderResourceDurations tests ---

func (s *DeployRenderersTestSuite) Test_renderResourceDurations_returns_empty_for_nil() {
	result := renderResourceDurations(nil, s.testStyles)
	s.Empty(result)
}

func (s *DeployRenderersTestSuite) Test_renderResourceDurations_shows_config_complete_duration() {
	duration := float64(5000)
	durations := &state.ResourceCompletionDurations{
		ConfigCompleteDuration: &duration,
	}
	result := renderResourceDurations(durations, s.testStyles)
	s.Contains(result, "Config Complete")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDurations_shows_total_duration() {
	duration := float64(10000)
	durations := &state.ResourceCompletionDurations{
		TotalDuration: &duration,
	}
	result := renderResourceDurations(durations, s.testStyles)
	s.Contains(result, "Total")
}

func (s *DeployRenderersTestSuite) Test_renderResourceDurations_skips_zero_durations() {
	zeroDuration := float64(0)
	durations := &state.ResourceCompletionDurations{
		ConfigCompleteDuration: &zeroDuration,
		TotalDuration:          &zeroDuration,
	}
	result := renderResourceDurations(durations, s.testStyles)
	s.Empty(result)
}

// --- DeployFooterRenderer tests ---

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_deploying_when_not_finished() {
	footer := &DeployFooterRenderer{
		InstanceName: "my-instance",
		ChangesetID:  "cs-123",
		Finished:     false,
		SpinnerView:  "⠋",
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "Deploying")
	s.Contains(result, "my-instance")
}

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_complete_when_finished() {
	footer := &DeployFooterRenderer{
		InstanceName: "my-instance",
		FinalStatus:  core.InstanceStatusDeployed,
		Finished:     true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "Deployment")
	s.Contains(result, "complete")
}

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_failed_status() {
	footer := &DeployFooterRenderer{
		InstanceName: "my-instance",
		FinalStatus:  core.InstanceStatusDeployFailed,
		Finished:     true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "failed")
}

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_rolled_back_status() {
	footer := &DeployFooterRenderer{
		InstanceName: "my-instance",
		FinalStatus:  core.InstanceStatusDeployRollbackComplete,
		Finished:     true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "rolled back")
}

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_exports_hint_when_available() {
	footer := &DeployFooterRenderer{
		InstanceName:     "my-instance",
		FinalStatus:      core.InstanceStatusDeployed,
		Finished:         true,
		HasInstanceState: true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "exports")
}

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_pre_rollback_hint_when_available() {
	footer := &DeployFooterRenderer{
		InstanceName:        "my-instance",
		FinalStatus:         core.InstanceStatusDeployRollbackComplete,
		Finished:            true,
		HasPreRollbackState: true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "pre-rollback")
}

func (s *DeployRenderersTestSuite) Test_DeployFooterRenderer_shows_element_summary() {
	footer := &DeployFooterRenderer{
		InstanceName: "my-instance",
		FinalStatus:  core.InstanceStatusDeployed,
		Finished:     true,
		SuccessfulElements: []SuccessfulElement{
			{ElementName: "res1"},
			{ElementName: "res2"},
		},
		ElementFailures: []ElementFailure{
			{ElementName: "res3"},
		},
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "2")
	s.Contains(result, "successful")
}

// --- renderFinalStatus tests ---

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_deployed() {
	result := renderFinalStatus(core.InstanceStatusDeployed, s.testStyles)
	s.Contains(result, "complete")
}

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_updated() {
	result := renderFinalStatus(core.InstanceStatusUpdated, s.testStyles)
	s.Contains(result, "complete")
}

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_destroyed() {
	result := renderFinalStatus(core.InstanceStatusDestroyed, s.testStyles)
	s.Contains(result, "complete")
}

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_deploy_failed() {
	result := renderFinalStatus(core.InstanceStatusDeployFailed, s.testStyles)
	s.Contains(result, "failed")
}

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_rollback_complete() {
	result := renderFinalStatus(core.InstanceStatusDeployRollbackComplete, s.testStyles)
	s.Contains(result, "rolled back")
}

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_rollback_failed() {
	result := renderFinalStatus(core.InstanceStatusDeployRollbackFailed, s.testStyles)
	s.Contains(result, "rollback failed")
}

func (s *DeployRenderersTestSuite) Test_renderFinalStatus_unknown() {
	result := renderFinalStatus(core.InstanceStatus(999), s.testStyles)
	s.Contains(result, "unknown")
}

// --- DeployStagingFooterRenderer tests ---

func (s *DeployRenderersTestSuite) Test_DeployStagingFooterRenderer_shows_changeset_id() {
	footer := &DeployStagingFooterRenderer{
		ChangesetID: "cs-123",
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "cs-123")
}

func (s *DeployRenderersTestSuite) Test_DeployStagingFooterRenderer_shows_change_summary() {
	footer := &DeployStagingFooterRenderer{
		ChangesetID: "cs-123",
		Summary: ChangeSummary{
			Create:   2,
			Update:   1,
			Delete:   1,
			Recreate: 1,
		},
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "2 to create")
	s.Contains(result, "1 to update")
	s.Contains(result, "1 to delete")
	s.Contains(result, "1 to recreate")
}

func (s *DeployRenderersTestSuite) Test_DeployStagingFooterRenderer_shows_no_changes() {
	footer := &DeployStagingFooterRenderer{
		ChangesetID: "cs-123",
		Summary:     ChangeSummary{},
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "No changes")
}

func (s *DeployRenderersTestSuite) Test_DeployStagingFooterRenderer_shows_confirmation_prompt() {
	footer := &DeployStagingFooterRenderer{
		ChangesetID: "cs-123",
		Summary:     ChangeSummary{Create: 1},
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "Apply these changes")
}

func (s *DeployRenderersTestSuite) Test_DeployStagingFooterRenderer_shows_exports_hint() {
	footer := &DeployStagingFooterRenderer{
		ChangesetID:      "cs-123",
		HasExportChanges: true,
	}
	result := footer.RenderFooter(&splitpane.Model{}, s.testStyles)
	s.Contains(result, "exports")
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
