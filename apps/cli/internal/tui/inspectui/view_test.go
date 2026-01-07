package inspectui

import (
	"errors"
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
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

func (s *ViewTestSuite) createTestModel() *InspectModel {
	return &InspectModel{
		styles:           s.testStyles,
		instanceID:       "inst-123",
		instanceName:     "test-instance",
		overviewViewport: viewport.New(80, 24),
		specViewport:     viewport.New(80, 24),
	}
}

// --- renderOverviewView tests ---

func (s *ViewTestSuite) Test_renderOverviewView_shows_header() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
	}
	output := model.renderOverviewView()
	s.Contains(output, "Instance Overview")
}

// --- renderOverviewContent tests ---

func (s *ViewTestSuite) Test_renderOverviewContent_handles_nil_state() {
	model := s.createTestModel()
	model.instanceState = nil
	output := model.renderOverviewContent()
	s.Contains(output, "No instance state available")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_instance_info() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-abc-123",
		InstanceName: "my-instance",
		Status:       core.InstanceStatusDeployed,
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Instance Information")
	s.Contains(output, "inst-abc-123")
	s.Contains(output, "my-instance")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_resources_section() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-123": {
				ResourceID: "res-123",
				Name:       "myBucket",
				Type:       "aws/s3/bucket",
				Status:     core.ResourceStatusCreated,
			},
		},
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Resources (1)")
	s.Contains(output, "myBucket")
	s.Contains(output, "aws/s3/bucket")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_child_blueprints_section() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID:   "child-inst-456",
				InstanceName: "childBlueprint",
				Status:       core.InstanceStatusDeployed,
			},
		},
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Child Blueprints (1)")
	s.Contains(output, "childBlueprint")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_links_section() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Links: map[string]*state.LinkState{
			"resA::resB": {
				LinkID: "link-123",
				Status: core.LinkStatusCreated,
			},
		},
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Links (1)")
	s.Contains(output, "resA::resB")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_exports_section() {
	model := s.createTestModel()
	val := "exported-value"
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Exports: map[string]*state.ExportState{
			"myExport": {
				Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Exports")
	s.Contains(output, "myExport")
}

func (s *ViewTestSuite) Test_renderOverviewContent_shows_timing_section() {
	model := s.createTestModel()
	prepareDuration := float64(5000)
	totalDuration := float64(10000)
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Durations: &state.InstanceCompletionDuration{
			PrepareDuration: &prepareDuration,
			TotalDuration:   &totalDuration,
		},
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Timing")
	s.Contains(output, "Prepare")
	s.Contains(output, "Total")
}

func (s *ViewTestSuite) Test_renderOverviewContent_hides_timing_with_zero_durations() {
	model := s.createTestModel()
	zeroDuration := float64(0)
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Durations: &state.InstanceCompletionDuration{
			PrepareDuration: &zeroDuration,
			TotalDuration:   &zeroDuration,
		},
	}
	output := model.renderOverviewContent()
	s.NotContains(output, "Timing")
}

func (s *ViewTestSuite) Test_renderOverviewContent_hides_timing_with_nil_durations() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Durations:    nil,
	}
	output := model.renderOverviewContent()
	s.NotContains(output, "Timing")
}

// --- renderSpecView tests ---

func (s *ViewTestSuite) Test_renderSpecView_shows_header() {
	model := s.createTestModel()
	output := model.renderSpecView()
	s.Contains(output, "Resource Specification")
}

// --- renderSpecContent tests ---

func (s *ViewTestSuite) Test_renderSpecContent_handles_nil_resource_state() {
	model := s.createTestModel()
	output := model.renderSpecContent(nil, "myResource")
	s.Contains(output, "No specification data available")
}

func (s *ViewTestSuite) Test_renderSpecContent_handles_nil_spec_data() {
	model := s.createTestModel()
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/s3/bucket",
		SpecData:   nil,
	}
	output := model.renderSpecContent(resourceState, "myResource")
	s.Contains(output, "No specification data available")
}

func (s *ViewTestSuite) Test_renderSpecContent_shows_resource_name() {
	model := s.createTestModel()
	val := "some-value"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myBucket",
		Type:       "aws/s3/bucket",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"bucketName": {Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
	}
	output := model.renderSpecContent(resourceState, "myBucket")
	s.Contains(output, "myBucket")
}

func (s *ViewTestSuite) Test_renderSpecContent_shows_specification_section() {
	model := s.createTestModel()
	val := "input-value"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/lambda/function",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"inputField": {Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
		ComputedFields: []string{}, // no computed fields
	}
	output := model.renderSpecContent(resourceState, "myResource")
	s.Contains(output, "Specification")
	s.Contains(output, "inputField")
	s.Contains(output, "input-value")
}

func (s *ViewTestSuite) Test_renderSpecContent_shows_outputs_section() {
	model := s.createTestModel()
	inputVal := "input-value"
	outputVal := "output-value"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/lambda/function",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"inputField":  {Scalar: &core.ScalarValue{StringValue: &inputVal}},
				"outputField": {Scalar: &core.ScalarValue{StringValue: &outputVal}},
			},
		},
		ComputedFields: []string{"outputField"},
	}
	output := model.renderSpecContent(resourceState, "myResource")
	s.Contains(output, "Outputs")
	s.Contains(output, "outputField")
	s.Contains(output, "output-value")
}

func (s *ViewTestSuite) Test_renderSpecContent_handles_multiline_spec_values() {
	model := s.createTestModel()
	multilineVal := "line1\nline2\nline3"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/lambda/function",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"multilineField": {Scalar: &core.ScalarValue{StringValue: &multilineVal}},
			},
		},
	}
	output := model.renderSpecContent(resourceState, "myResource")
	s.Contains(output, "multilineField")
	s.Contains(output, "line1")
	s.Contains(output, "line2")
	s.Contains(output, "line3")
}

func (s *ViewTestSuite) Test_renderSpecContent_handles_multiline_output_values() {
	model := s.createTestModel()
	multilineVal := "{\n  \"key\": \"value\"\n}"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/lambda/function",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"jsonOutput": {Scalar: &core.ScalarValue{StringValue: &multilineVal}},
			},
		},
		ComputedFields: []string{"jsonOutput"},
	}
	output := model.renderSpecContent(resourceState, "myResource")
	s.Contains(output, "jsonOutput")
	s.Contains(output, "key")
	s.Contains(output, "value")
}

// --- renderError tests ---

func (s *ViewTestSuite) Test_renderError_shows_error_header() {
	model := s.createTestModel()
	err := errors.New("Something went wrong")
	output := model.renderError(err)
	s.Contains(output, "Error")
}

func (s *ViewTestSuite) Test_renderError_shows_error_message() {
	model := s.createTestModel()
	err := errors.New("Failed to fetch instance state")
	output := model.renderError(err)
	s.Contains(output, "Failed to fetch instance state")
}

func (s *ViewTestSuite) Test_renderError_shows_quit_instruction() {
	model := s.createTestModel()
	err := errors.New("Some error")
	output := model.renderError(err)
	s.Contains(output, "Press q to quit")
}

// --- Edge cases for sections ---

func (s *ViewTestSuite) Test_renderOverviewContent_empty_resources_hidden() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Resources:    map[string]*state.ResourceState{},
	}
	output := model.renderOverviewContent()
	s.NotContains(output, "Resources (0)")
}

func (s *ViewTestSuite) Test_renderOverviewContent_empty_children_hidden() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:      "inst-123",
		InstanceName:    "test",
		Status:          core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{},
	}
	output := model.renderOverviewContent()
	s.NotContains(output, "Child Blueprints (0)")
}

func (s *ViewTestSuite) Test_renderOverviewContent_empty_links_hidden() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Links:        map[string]*state.LinkState{},
	}
	output := model.renderOverviewContent()
	s.NotContains(output, "Links (0)")
}

func (s *ViewTestSuite) Test_renderOverviewContent_empty_exports_hidden() {
	model := s.createTestModel()
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Exports:      map[string]*state.ExportState{},
	}
	output := model.renderOverviewContent()
	// When exports is empty, the exports section header should not appear
	// Check that there's no "Exports" header without any items
	s.NotContains(output, "Exports\n\n\n") // would be empty exports section
}

func (s *ViewTestSuite) Test_renderOverviewContent_with_all_sections() {
	model := s.createTestModel()
	val := "export-val"
	prepareDuration := float64(5000)
	totalDuration := float64(10000)
	model.instanceState = &state.InstanceState{
		InstanceID:   "inst-123",
		InstanceName: "test",
		Status:       core.InstanceStatusDeployed,
		Resources: map[string]*state.ResourceState{
			"res-1": {ResourceID: "res-1", Name: "resource1", Type: "aws/s3/bucket", Status: core.ResourceStatusCreated},
		},
		ChildBlueprints: map[string]*state.InstanceState{
			"child1": {InstanceID: "child-1", InstanceName: "child1", Status: core.InstanceStatusDeployed},
		},
		Links: map[string]*state.LinkState{
			"a::b": {LinkID: "link-1", Status: core.LinkStatusCreated},
		},
		Exports: map[string]*state.ExportState{
			"export1": {Value: &core.MappingNode{Scalar: &core.ScalarValue{StringValue: &val}}},
		},
		Durations: &state.InstanceCompletionDuration{
			PrepareDuration: &prepareDuration,
			TotalDuration:   &totalDuration,
		},
	}
	output := model.renderOverviewContent()
	s.Contains(output, "Instance Information")
	s.Contains(output, "Resources (1)")
	s.Contains(output, "Child Blueprints (1)")
	s.Contains(output, "Links (1)")
	s.Contains(output, "Exports")
	s.Contains(output, "Timing")
}

// --- renderSpecContent edge cases ---

func (s *ViewTestSuite) Test_renderSpecContent_no_computed_fields() {
	model := s.createTestModel()
	val := "value"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/s3/bucket",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"field1": {Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
		ComputedFields: nil, // no computed fields
	}
	output := model.renderSpecContent(resourceState, "myResource")
	s.Contains(output, "Specification")
	s.Contains(output, "field1")
}

func (s *ViewTestSuite) Test_renderSpecContent_only_computed_fields() {
	model := s.createTestModel()
	val := "computed"
	resourceState := &state.ResourceState{
		ResourceID: "res-123",
		Name:       "myResource",
		Type:       "aws/s3/bucket",
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"computedField": {Scalar: &core.ScalarValue{StringValue: &val}},
			},
		},
		ComputedFields: []string{"computedField"},
	}
	output := model.renderSpecContent(resourceState, "myResource")
	// No non-computed fields, so Specification section should be empty or not shown
	s.Contains(output, "Outputs")
	s.Contains(output, "computedField")
}
