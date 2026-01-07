package destroyui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type ViewTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestViewTestSuite(t *testing.T) {
	suite.Run(t, new(ViewTestSuite))
}

func (s *ViewTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		styles.NewBluelinkPalette(),
	)
}

func (s *ViewTestSuite) createTestModel() DestroyModel {
	return DestroyModel{
		styles:                  s.testStyles,
		instanceID:              "inst-123",
		instanceName:            "test-instance",
		changesetID:             "cs-456",
		overviewViewport:        viewport.New(80, 24),
		preDestroyStateViewport: viewport.New(80, 24),
	}
}

// --- overviewFooterHeight tests ---

func (s *ViewTestSuite) Test_overviewFooterHeight_returns_constant() {
	height := overviewFooterHeight()
	s.Equal(4, height)
}

// --- renderError tests ---

func (s *ViewTestSuite) Test_renderError_handles_validation_error() {
	model := s.createTestModel()
	err := &engineerrors.ClientError{
		StatusCode: 422,
		Message:    "Validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "resources.myRes", Message: "invalid field"},
		},
	}
	output := model.renderError(err)
	s.Contains(output, "Validation")
}

func (s *ViewTestSuite) Test_renderError_handles_stream_error() {
	model := s.createTestModel()
	err := &engineerrors.StreamError{
		Event: &types.StreamErrorMessageEvent{
			Message: "Stream error occurred",
		},
	}
	output := model.renderError(err)
	s.Contains(output, "Stream error")
}

func (s *ViewTestSuite) Test_renderError_handles_generic_error() {
	model := s.createTestModel()
	err := &testError{message: "Something went wrong"}
	output := model.renderError(err)
	s.Contains(output, "Something went wrong")
}

// --- renderDeployChangesetError tests ---

func (s *ViewTestSuite) Test_renderDeployChangesetError_shows_mismatch_message() {
	model := s.createTestModel()
	output := model.renderDeployChangesetError()
	s.NotEmpty(output)
}

// --- renderOverviewContent tests ---

func (s *ViewTestSuite) Test_renderOverviewContent_shows_header() {
	model := s.createTestModel()
	model.finalStatus = core.InstanceStatusDestroyed
	output := model.renderOverviewContent()
	s.Contains(output, "Destroy Summary")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_failed_header() {
	model := s.createTestModel()
	model.finalStatus = core.InstanceStatusDestroyFailed
	output := model.renderOverviewContent()
	s.Contains(output, "Destroy Failed")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_rollback_header() {
	model := s.createTestModel()
	model.finalStatus = core.InstanceStatusDestroyRollbackComplete
	output := model.renderOverviewContent()
	s.Contains(output, "Destroy Rolled Back")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_interrupted_header() {
	model := s.createTestModel()
	model.finalStatus = core.InstanceStatusDestroyInterrupted
	output := model.renderOverviewContent()
	s.Contains(output, "Destroy Interrupted")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_instance_info() {
	model := s.createTestModel()
	model.instanceName = "my-instance"
	model.instanceID = "inst-abc-123"
	model.changesetID = "cs-xyz-789"
	output := model.renderOverviewContent()
	s.Contains(output, "my-instance")
	s.Contains(output, "inst-abc-123")
	s.Contains(output, "cs-xyz-789")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_failure_reasons() {
	model := s.createTestModel()
	model.failureReasons = []string{"First error", "Second error"}
	output := model.renderOverviewContent()
	s.Contains(output, "Root cause")
	s.Contains(output, "First error")
	s.Contains(output, "Second error")
}

// --- renderDestroyedElements tests ---

func (s *ViewTestSuite) Test_renderDestroyedElements_empty_when_no_elements() {
	model := s.createTestModel()
	model.destroyedElements = []DestroyedElement{}
	var sb strings.Builder
	successStyle := lipgloss.NewStyle()
	model.renderDestroyedElements(&sb, successStyle)
	s.Empty(sb.String())
}

func (s *ViewTestSuite) Test_renderDestroyedElements_shows_count_and_elements() {
	model := s.createTestModel()
	model.destroyedElements = []DestroyedElement{
		{ElementName: "res1", ElementPath: "resources/res1", ElementType: "aws/s3/bucket"},
		{ElementName: "res2", ElementPath: "resources/res2", ElementType: "aws/lambda/function"},
	}
	var sb strings.Builder
	successStyle := lipgloss.NewStyle()
	model.renderDestroyedElements(&sb, successStyle)
	output := sb.String()
	s.Contains(output, "2 Destroyed")
	s.Contains(output, "resources/res1")
	s.Contains(output, "resources/res2")
}

func (s *ViewTestSuite) Test_renderDestroyedElements_shows_resource_type() {
	model := s.createTestModel()
	model.destroyedElements = []DestroyedElement{
		{ElementName: "res1", ElementPath: "resources/res1", ElementType: "aws/s3/bucket"},
	}
	var sb strings.Builder
	successStyle := lipgloss.NewStyle()
	model.renderDestroyedElements(&sb, successStyle)
	output := sb.String()
	s.Contains(output, "aws/s3/bucket")
}

func (s *ViewTestSuite) Test_renderDestroyedElements_hides_child_and_link_types() {
	model := s.createTestModel()
	model.destroyedElements = []DestroyedElement{
		{ElementName: "childBlueprint", ElementPath: "children/childBlueprint", ElementType: "child"},
		{ElementName: "link1", ElementPath: "links/link1", ElementType: "link"},
	}
	var sb strings.Builder
	successStyle := lipgloss.NewStyle()
	model.renderDestroyedElements(&sb, successStyle)
	output := sb.String()
	// child and link types should not be shown in parentheses
	s.NotContains(output, "(child)")
	s.NotContains(output, "(link)")
}

// --- renderInterruptedElements tests ---

func (s *ViewTestSuite) Test_renderInterruptedElements_empty_when_no_elements() {
	model := s.createTestModel()
	model.interruptedElements = []shared.InterruptedElement{}
	var sb strings.Builder
	model.renderInterruptedElements(&sb)
	s.Empty(sb.String())
}

func (s *ViewTestSuite) Test_renderInterruptedElements_shows_count_and_elements() {
	model := s.createTestModel()
	model.interruptedElements = []shared.InterruptedElement{
		{ElementName: "res1", ElementPath: "resources/res1", ElementType: "aws/s3/bucket"},
		{ElementName: "res2", ElementPath: "resources/res2", ElementType: "aws/lambda/function"},
	}
	var sb strings.Builder
	model.renderInterruptedElements(&sb)
	output := sb.String()
	s.Contains(output, "2")
	s.Contains(output, "Interrupted")
	s.Contains(output, "resources/res1")
	s.Contains(output, "resources/res2")
}

func (s *ViewTestSuite) Test_renderInterruptedElements_shows_unknown_state_message() {
	model := s.createTestModel()
	model.interruptedElements = []shared.InterruptedElement{
		{ElementName: "res1", ElementPath: "resources/res1", ElementType: "aws/s3/bucket"},
	}
	var sb strings.Builder
	model.renderInterruptedElements(&sb)
	output := sb.String()
	s.Contains(output, "state is unknown")
}

// --- renderPreDestroyStateContent tests ---

func (s *ViewTestSuite) Test_renderPreDestroyStateContent_handles_nil_state() {
	model := s.createTestModel()
	model.preDestroyInstanceState = nil
	output := model.renderPreDestroyStateContent()
	s.Contains(output, "No pre-destroy state available")
}

func (s *ViewTestSuite) Test_renderPreDestroyStateContent_shows_header() {
	model := s.createTestModel()
	model.preDestroyInstanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
	}
	output := model.renderPreDestroyStateContent()
	s.Contains(output, "Pre-Destroy Instance State")
}

func (s *ViewTestSuite) Test_renderPreDestroyStateContent_shows_instance_info() {
	model := s.createTestModel()
	model.instanceName = "my-instance"
	model.preDestroyInstanceState = &state.InstanceState{
		InstanceID:   "inst-abc",
		InstanceName: "my-instance",
	}
	output := model.renderPreDestroyStateContent()
	s.Contains(output, "my-instance")
	s.Contains(output, "inst-abc")
}

// --- renderInstanceStateHierarchy tests ---

func (s *ViewTestSuite) Test_renderInstanceStateHierarchy_renders_resources() {
	model := s.createTestModel()
	model.preDestroyStateViewport = viewport.New(100, 30)
	instanceState := &state.InstanceState{
		InstanceID: "inst-123",
		ResourceIDs: map[string]string{
			"myBucket": "res-bucket-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-bucket-123": {
				ResourceID: "res-bucket-123",
				Name:       "myBucket",
				Type:       "aws/s3/bucket",
			},
		},
	}
	var sb strings.Builder
	model.renderInstanceStateHierarchy(&sb, instanceState, "", 0, 80)
	output := sb.String()
	s.Contains(output, "Resources")
	s.Contains(output, "myBucket")
}

func (s *ViewTestSuite) Test_renderInstanceStateHierarchy_renders_links() {
	model := s.createTestModel()
	instanceState := &state.InstanceState{
		InstanceID: "inst-123",
		Links: map[string]*state.LinkState{
			"resA::resB": {
				LinkID: "link-123",
			},
		},
	}
	var sb strings.Builder
	model.renderInstanceStateHierarchy(&sb, instanceState, "", 0, 80)
	output := sb.String()
	s.Contains(output, "Links")
	s.Contains(output, "resA::resB")
}

func (s *ViewTestSuite) Test_renderInstanceStateHierarchy_renders_exports() {
	model := s.createTestModel()
	val := "exported-value"
	instanceState := &state.InstanceState{
		InstanceID: "inst-123",
		Exports: map[string]*state.ExportState{
			"myExport": {
				Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
	}
	var sb strings.Builder
	model.renderInstanceStateHierarchy(&sb, instanceState, "", 0, 80)
	output := sb.String()
	s.Contains(output, "Exports")
	s.Contains(output, "myExport")
}

func (s *ViewTestSuite) Test_renderInstanceStateHierarchy_renders_children() {
	model := s.createTestModel()
	instanceState := &state.InstanceState{
		InstanceID: "inst-123",
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID:   "child-inst-456",
				InstanceName: "childBlueprint",
			},
		},
	}
	var sb strings.Builder
	model.renderInstanceStateHierarchy(&sb, instanceState, "", 0, 80)
	output := sb.String()
	s.Contains(output, "Child Blueprints")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "child-inst-456")
}

// --- filterSpecFields tests ---

func (s *ViewTestSuite) Test_filterSpecFields_returns_nil_for_nil_specData() {
	result := filterSpecFields(nil, nil, false)
	s.Nil(result)
}

func (s *ViewTestSuite) Test_filterSpecFields_returns_nil_for_nil_fields() {
	specData := &core.MappingNode{Scalar: &core.ScalarValue{}}
	result := filterSpecFields(specData, nil, false)
	s.Nil(result)
}

func (s *ViewTestSuite) Test_filterSpecFields_filters_computed_fields() {
	val1 := "input-value"
	val2 := "computed-value"
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"inputField":    {Scalar: &core.ScalarValue{StringValue: &val1}},
			"computedField": {Scalar: &core.ScalarValue{StringValue: &val2}},
		},
	}
	computedFields := map[string]bool{"computedField": true}

	// Get non-computed fields only
	nonComputed := filterSpecFields(specData, computedFields, false)
	s.Len(nonComputed, 1)
	s.Equal("inputField", nonComputed[0].Name)

	// Get computed fields only
	computed := filterSpecFields(specData, computedFields, true)
	s.Len(computed, 1)
	s.Equal("computedField", computed[0].Name)
}

func (s *ViewTestSuite) Test_filterSpecFields_excludes_null_values() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"nullField": nil,
		},
	}
	result := filterSpecFields(specData, nil, false)
	s.Empty(result)
}

func (s *ViewTestSuite) Test_filterSpecFields_sorts_fields_alphabetically() {
	valA := "a"
	valB := "b"
	valC := "c"
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"zField": {Scalar: &core.ScalarValue{StringValue: &valC}},
			"aField": {Scalar: &core.ScalarValue{StringValue: &valA}},
			"mField": {Scalar: &core.ScalarValue{StringValue: &valB}},
		},
	}
	result := filterSpecFields(specData, nil, false)
	s.Len(result, 3)
	s.Equal("aField", result[0].Name)
	s.Equal("mField", result[1].Name)
	s.Equal("zField", result[2].Name)
}

// --- renderLinkStates tests ---

func (s *ViewTestSuite) Test_renderLinkStates_renders_links() {
	model := s.createTestModel()
	links := map[string]*state.LinkState{
		"resA::resB": {LinkID: "link-123"},
		"resC::resD": {LinkID: "link-456"},
	}
	var sb strings.Builder
	model.renderLinkStates(&sb, links, "  ")
	output := sb.String()
	s.Contains(output, "resA::resB")
	s.Contains(output, "resC::resD")
	s.Contains(output, "link-123")
	s.Contains(output, "link-456")
}

func (s *ViewTestSuite) Test_renderLinkStates_sorts_links_alphabetically() {
	model := s.createTestModel()
	links := map[string]*state.LinkState{
		"z::link": {LinkID: "link-z"},
		"a::link": {LinkID: "link-a"},
	}
	var sb strings.Builder
	model.renderLinkStates(&sb, links, "  ")
	output := sb.String()
	// a::link should appear before z::link
	aIndex := strings.Index(output, "a::link")
	zIndex := strings.Index(output, "z::link")
	s.True(aIndex < zIndex, "a::link should appear before z::link")
}

// --- renderExports tests ---

func (s *ViewTestSuite) Test_renderExports_renders_export_values() {
	model := s.createTestModel()
	val := "exported-value"
	exports := map[string]*state.ExportState{
		"myExport": {
			Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &val}},
		},
	}
	var sb strings.Builder
	model.renderExports(&sb, exports, "  ", 80)
	output := sb.String()
	s.Contains(output, "myExport")
	s.Contains(output, "exported-value")
}

func (s *ViewTestSuite) Test_renderExports_handles_nil_export_value() {
	model := s.createTestModel()
	exports := map[string]*state.ExportState{
		"nilExport": nil,
	}
	var sb strings.Builder
	model.renderExports(&sb, exports, "  ", 80)
	output := sb.String()
	s.Contains(output, "nilExport")
}

func (s *ViewTestSuite) Test_renderExports_sorts_exports_alphabetically() {
	model := s.createTestModel()
	valA := "a"
	valZ := "z"
	exports := map[string]*state.ExportState{
		"zExport": {Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &valZ}}},
		"aExport": {Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &valA}}},
	}
	var sb strings.Builder
	model.renderExports(&sb, exports, "  ", 80)
	output := sb.String()
	aIndex := strings.Index(output, "aExport")
	zIndex := strings.Index(output, "zExport")
	s.True(aIndex < zIndex, "aExport should appear before zExport")
}

// --- renderFieldsWithWrapping tests ---

func (s *ViewTestSuite) Test_renderFieldsWithWrapping_renders_simple_field() {
	model := s.createTestModel()
	fields := []fieldInfo{
		{Name: "field1", Value: "value1"},
	}
	var sb strings.Builder
	model.renderFieldsWithWrapping(&sb, fields, "  ", 80)
	output := sb.String()
	s.Contains(output, "field1")
	s.Contains(output, "value1")
}

func (s *ViewTestSuite) Test_renderFieldsWithWrapping_handles_multiline_values() {
	model := s.createTestModel()
	fields := []fieldInfo{
		{Name: "jsonField", Value: "{\n  \"key\": \"value\"\n}"},
	}
	var sb strings.Builder
	model.renderFieldsWithWrapping(&sb, fields, "  ", 80)
	output := sb.String()
	s.Contains(output, "jsonField")
	s.Contains(output, "key")
	s.Contains(output, "value")
}

// --- renderResourceStates tests ---

func (s *ViewTestSuite) Test_renderResourceStates_renders_resource_info() {
	model := s.createTestModel()
	bucketName := "my-bucket"
	instanceState := &state.InstanceState{
		InstanceID: "inst-123",
		ResourceIDs: map[string]string{
			"myBucket": "res-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "myBucket",
				Type:       "aws/s3/bucket",
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"bucketName": {Scalar: &core.ScalarValue{StringValue: &bucketName}},
					},
				},
			},
		},
	}
	var sb strings.Builder
	model.renderResourceStates(&sb, instanceState, "  ", 80)
	output := sb.String()
	s.Contains(output, "myBucket")
	s.Contains(output, "aws/s3/bucket")
	s.Contains(output, "res-123")
}

func (s *ViewTestSuite) Test_renderResourceStates_shows_spec_and_outputs() {
	model := s.createTestModel()
	inputVal := "input-value"
	outputVal := "output-value"
	instanceState := &state.InstanceState{
		InstanceID: "inst-123",
		ResourceIDs: map[string]string{
			"myResource": "res-456",
		},
		Resources: map[string]*state.ResourceState{
			"res-456": {
				ResourceID: "res-456",
				Name:       "myResource",
				Type:       "aws/lambda/function",
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"inputField":  {Scalar: &core.ScalarValue{StringValue: &inputVal}},
						"outputField": {Scalar: &core.ScalarValue{StringValue: &outputVal}},
					},
				},
				ComputedFields: []string{"outputField"},
			},
		},
	}
	var sb strings.Builder
	model.renderResourceStates(&sb, instanceState, "  ", 80)
	output := sb.String()
	s.Contains(output, "Spec")
	s.Contains(output, "Outputs")
	s.Contains(output, "inputField")
	s.Contains(output, "outputField")
}

