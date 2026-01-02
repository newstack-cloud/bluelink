package outpututil

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type OutputsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestOutputsTestSuite(t *testing.T) {
	suite.Run(t, new(OutputsTestSuite))
}

func (s *OutputsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

func (s *OutputsTestSuite) Test_FormatDuration_milliseconds() {
	s.Equal("500ms", FormatDuration(500))
	s.Equal("0ms", FormatDuration(0))
	s.Equal("999ms", FormatDuration(999))
}

func (s *OutputsTestSuite) Test_FormatDuration_seconds() {
	s.Equal("1.00s", FormatDuration(1000))
	s.Equal("1.50s", FormatDuration(1500))
	s.Equal("59.99s", FormatDuration(59990))
}

func (s *OutputsTestSuite) Test_FormatDuration_minutes() {
	s.Equal("1m 0s", FormatDuration(60000))
	s.Equal("1m 30s", FormatDuration(90000))
	s.Equal("5m 30s", FormatDuration(330000))
}

func (s *OutputsTestSuite) Test_WrapText_no_wrap_needed() {
	s.Equal("short text", WrapText("short text", 50))
}

func (s *OutputsTestSuite) Test_WrapText_wraps_at_space() {
	result := WrapText("hello world test", 12)
	s.Contains(result, "\n")
}

func (s *OutputsTestSuite) Test_WrapText_zero_width() {
	s.Equal("text", WrapText("text", 0))
}

func (s *OutputsTestSuite) Test_WrapText_negative_width() {
	s.Equal("text", WrapText("text", -10))
}

func (s *OutputsTestSuite) Test_WrapText_exact_width() {
	s.Equal("hello", WrapText("hello", 5))
}

func (s *OutputsTestSuite) Test_IsValidOutputValue_returns_true_for_valid() {
	s.True(IsValidOutputValue("valid"))
	s.True(IsValidOutputValue("123"))
	s.True(IsValidOutputValue("some value"))
}

func (s *OutputsTestSuite) Test_IsValidOutputValue_returns_false_for_empty() {
	s.False(IsValidOutputValue(""))
}

func (s *OutputsTestSuite) Test_IsValidOutputValue_returns_false_for_null() {
	s.False(IsValidOutputValue("null"))
}

func (s *OutputsTestSuite) Test_IsValidOutputValue_returns_false_for_nil() {
	s.False(IsValidOutputValue("<nil>"))
}

func (s *OutputsTestSuite) Test_ConvertSpecPathToLookupPath_strips_spec_prefix() {
	s.Equal("$.id", ConvertSpecPathToLookupPath("spec.id"))
	s.Equal("$.nested.field", ConvertSpecPathToLookupPath("spec.nested.field"))
}

func (s *OutputsTestSuite) Test_ConvertSpecPathToLookupPath_handles_no_prefix() {
	s.Equal("$.field", ConvertSpecPathToLookupPath("field"))
}

func (s *OutputsTestSuite) Test_CollectOutputFields_returns_empty_for_nil_computed_fields() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"field1": stringNode("value1"),
		},
	}
	fields := CollectOutputFields(specData, nil)
	s.Len(fields, 1)
	s.Equal("field1", fields[0].Name)
}

func (s *OutputsTestSuite) Test_CollectOutputFields_uses_computed_field_paths() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id":   stringNode("123"),
			"name": stringNode("test"),
		},
	}
	fields := CollectOutputFields(specData, []string{"spec.id"})
	s.Len(fields, 1)
	s.Equal("id", fields[0].Name)
	// FormatMappingNode returns JSON-formatted strings with quotes
	s.Equal(`"123"`, fields[0].Value)
}

func (s *OutputsTestSuite) Test_CollectNonComputedFields_excludes_computed() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id":      stringNode("123"),
			"name":    stringNode("test"),
			"created": stringNode("2024-01-01"),
		},
	}
	fields := CollectNonComputedFields(specData, []string{"spec.id", "spec.created"})
	s.Len(fields, 1)
	s.Equal("name", fields[0].Name)
}

func (s *OutputsTestSuite) Test_CollectNonComputedFields_returns_nil_for_nil_spec() {
	fields := CollectNonComputedFields(nil, nil)
	s.Nil(fields)
}

func (s *OutputsTestSuite) Test_CollectNonComputedFields_returns_nil_for_nil_fields() {
	specData := &core.MappingNode{}
	fields := CollectNonComputedFields(specData, nil)
	s.Nil(fields)
}

func (s *OutputsTestSuite) Test_CollectNonComputedFields_sorted_alphabetically() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"zebra": stringNode("z"),
			"alpha": stringNode("a"),
			"beta":  stringNode("b"),
		},
	}
	fields := CollectNonComputedFields(specData, nil)
	s.Len(fields, 3)
	s.Equal("alpha", fields[0].Name)
	s.Equal("beta", fields[1].Name)
	s.Equal("zebra", fields[2].Name)
}

func (s *OutputsTestSuite) Test_CountNonComputedFields_returns_count() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"field1": stringNode("value1"),
			"field2": stringNode("value2"),
			"field3": stringNode("value3"),
		},
	}
	count := CountNonComputedFields(specData, []string{"spec.field1"})
	s.Equal(2, count)
}

func (s *OutputsTestSuite) Test_RenderSpecHint_returns_empty_for_zero_count() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id": stringNode("123"),
		},
	}
	hint := RenderSpecHint(specData, []string{"spec.id"}, s.testStyles)
	s.Empty(hint)
}

func (s *OutputsTestSuite) Test_RenderSpecHint_returns_hint_with_count() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id":   stringNode("123"),
			"name": stringNode("test"),
		},
	}
	hint := RenderSpecHint(specData, []string{"spec.id"}, s.testStyles)
	s.Contains(hint, "1 fields")
	s.Contains(hint, "press 's' to view")
}

func (s *OutputsTestSuite) Test_RenderOutputsFromState_returns_empty_for_nil_state() {
	result := RenderOutputsFromState(nil, 80, s.testStyles)
	s.Empty(result)
}

func (s *OutputsTestSuite) Test_RenderOutputsFromState_returns_empty_for_nil_spec_data() {
	resourceState := &state.ResourceState{}
	result := RenderOutputsFromState(resourceState, 80, s.testStyles)
	s.Empty(result)
}

func (s *OutputsTestSuite) Test_RenderOutputsFromState_renders_computed_fields() {
	resourceState := &state.ResourceState{
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"id":   stringNode("resource-123"),
				"name": stringNode("test-resource"),
			},
		},
		ComputedFields: []string{"spec.id"},
	}
	result := RenderOutputsFromState(resourceState, 80, s.testStyles)
	s.Contains(result, "Current Outputs:")
	s.Contains(result, "id")
	s.Contains(result, "resource-123")
}

func (s *OutputsTestSuite) Test_CollectExportFields_returns_nil_for_empty_exports() {
	fields := CollectExportFields(nil)
	s.Nil(fields)

	fields = CollectExportFields(map[string]*state.ExportState{})
	s.Nil(fields)
}

func (s *OutputsTestSuite) Test_CollectExportFields_collects_exports() {
	exports := map[string]*state.ExportState{
		"export1": {
			Value:       stringNode("value1"),
			Type:        "string",
			Description: "First export",
		},
		"export2": {
			Value:       stringNode("value2"),
			Type:        "string",
			Description: "Second export",
		},
	}
	fields := CollectExportFields(exports)
	s.Len(fields, 2)
	s.Equal("export1", fields[0].Name)
	s.Equal("export2", fields[1].Name)
}

func (s *OutputsTestSuite) Test_CollectExportFields_sorted_alphabetically() {
	exports := map[string]*state.ExportState{
		"zebra": {Value: stringNode("z")},
		"alpha": {Value: stringNode("a")},
	}
	fields := CollectExportFields(exports)
	s.Len(fields, 2)
	s.Equal("alpha", fields[0].Name)
	s.Equal("zebra", fields[1].Name)
}

func (s *OutputsTestSuite) Test_CollectExportFields_skips_nil_exports() {
	exports := map[string]*state.ExportState{
		"valid":   {Value: stringNode("value")},
		"invalid": nil,
	}
	fields := CollectExportFields(exports)
	s.Len(fields, 1)
	s.Equal("valid", fields[0].Name)
}

func (s *OutputsTestSuite) Test_CollectExportFieldsPretty_returns_nil_for_empty() {
	fields := CollectExportFieldsPretty(nil)
	s.Nil(fields)
}

func (s *OutputsTestSuite) Test_RenderOutputFields_renders_label_and_fields() {
	fields := []OutputField{
		{Name: "field1", Value: "value1"},
		{Name: "field2", Value: "value2"},
	}
	result := RenderOutputFields(fields, 80, s.testStyles)
	s.Contains(result, "Current Outputs:")
	s.Contains(result, "field1")
	s.Contains(result, "value1")
}

func (s *OutputsTestSuite) Test_RenderOutputFieldsWithLabel_uses_custom_label() {
	fields := []OutputField{
		{Name: "field1", Value: "value1"},
	}
	result := RenderOutputFieldsWithLabel(fields, "Custom Label:", 80, s.testStyles)
	s.Contains(result, "Custom Label:")
}

func (s *OutputsTestSuite) Test_RenderOutputFieldsWithLabel_sorts_fields() {
	fields := []OutputField{
		{Name: "zebra", Value: "z"},
		{Name: "alpha", Value: "a"},
	}
	result := RenderOutputFieldsWithLabel(fields, "Label:", 80, s.testStyles)
	alphaIdx := indexOf(result, "alpha")
	zebraIdx := indexOf(result, "zebra")
	s.True(alphaIdx < zebraIdx, "alpha should come before zebra")
}

func (s *OutputsTestSuite) Test_RenderSpecFields_returns_empty_for_no_fields() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id": stringNode("123"),
		},
	}
	result := RenderSpecFields(specData, []string{"spec.id"}, 80, s.testStyles)
	s.Empty(result)
}

func (s *OutputsTestSuite) Test_RenderSpecFields_renders_non_computed_fields() {
	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id":   stringNode("123"),
			"name": stringNode("test"),
		},
	}
	result := RenderSpecFields(specData, []string{"spec.id"}, 80, s.testStyles)
	s.Contains(result, "Spec:")
	s.Contains(result, "name")
	s.Contains(result, "test")
}

// Helper functions

func stringNode(val string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &val},
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
