// Package outpututil provides shared utilities for rendering resource outputs
// (computed fields from spec data) across different UI components.
package outpututil

import (
	"fmt"
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// FormatDuration formats a duration in milliseconds to a human-friendly string.
// - < 1000ms: "XXXms"
// - < 60s: "X.XXs"
// - >= 60s: "Xm Ys"
func FormatDuration(milliseconds float64) string {
	if milliseconds < 1000 {
		return fmt.Sprintf("%.0fms", milliseconds)
	}

	seconds := milliseconds / 1000
	if seconds < 60 {
		return fmt.Sprintf("%.2fs", seconds)
	}

	minutes := int(seconds) / 60
	remainingSeconds := int(seconds) % 60
	return fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
}

// OutputField holds a name-value pair for output rendering.
type OutputField struct {
	Name  string
	Value string
}

// RenderOutputsFromState renders outputs from ResourceState.
// Uses ResourceState.ComputedFields to determine which fields to display.
func RenderOutputsFromState(
	resourceState *state.ResourceState,
	width int,
	s *styles.Styles,
) string {
	if resourceState == nil || resourceState.SpecData == nil {
		return ""
	}

	fields := CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
	if len(fields) == 0 {
		return ""
	}

	return RenderOutputFields(fields, width, s)
}

// CollectOutputFields extracts field entries from spec data.
// Uses computedFields paths if provided, otherwise extracts all top-level fields.
func CollectOutputFields(specData *core.MappingNode, computedFields []string) []OutputField {
	if len(computedFields) > 0 {
		return collectFieldsFromPaths(specData, computedFields)
	}
	return collectTopLevelFields(specData)
}

// collectFieldsFromPaths extracts field values using the provided paths.
func collectFieldsFromPaths(specData *core.MappingNode, fieldPaths []string) []OutputField {
	var fields []OutputField
	for _, fieldPath := range fieldPaths {
		lookupPath := ConvertSpecPathToLookupPath(fieldPath)
		value, err := core.GetPathValue(lookupPath, specData, 10)
		if err != nil || value == nil {
			continue
		}

		displayName := strings.TrimPrefix(fieldPath, "spec.")
		formattedValue := headless.FormatMappingNode(value)
		if IsValidOutputValue(formattedValue) {
			fields = append(fields, OutputField{Name: displayName, Value: formattedValue})
		}
	}
	return fields
}

// ConvertSpecPathToLookupPath converts a spec field path (e.g., "spec.id")
// to a lookup path for core.GetPathValue (e.g., "$.id").
func ConvertSpecPathToLookupPath(fieldPath string) string {
	// Strip "spec." prefix since SpecData already represents the spec
	path := strings.TrimPrefix(fieldPath, "spec.")
	// Prepend "$." for the root accessor
	return "$." + path
}

// collectTopLevelFields extracts all top-level fields from spec data.
func collectTopLevelFields(specData *core.MappingNode) []OutputField {
	if specData.Fields == nil {
		return nil
	}

	var fields []OutputField
	for fieldName, fieldValue := range specData.Fields {
		formattedValue := headless.FormatMappingNode(fieldValue)
		if IsValidOutputValue(formattedValue) {
			fields = append(fields, OutputField{Name: fieldName, Value: formattedValue})
		}
	}
	return fields
}

// IsValidOutputValue checks if a formatted value should be displayed.
func IsValidOutputValue(value string) bool {
	return value != "" && value != "null" && value != "<nil>"
}

// RenderOutputFields renders a sorted list of output fields with wrapping support.
// Uses "Current Outputs:" as the default label.
func RenderOutputFields(fields []OutputField, width int, s *styles.Styles) string {
	return RenderOutputFieldsWithLabel(fields, "Current Outputs:", width, s)
}

// RenderOutputFieldsWithLabel renders a sorted list of output fields with a custom label.
func RenderOutputFieldsWithLabel(fields []OutputField, label string, width int, s *styles.Styles) string {
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	sb := strings.Builder{}
	sb.WriteString(s.Category.Render(label))
	sb.WriteString("\n")

	for _, field := range fields {
		renderOutputField(&sb, field, width, s)
	}

	return sb.String()
}

// renderOutputField renders a single output field with optional text wrapping.
func renderOutputField(sb *strings.Builder, field OutputField, width int, s *styles.Styles) {
	prefix := fmt.Sprintf("  %s: ", field.Name)
	valueWidth := width - len(prefix) - 4 // margin for borders/padding

	if valueWidth > 20 && len(field.Value) > valueWidth {
		renderWrappedField(sb, prefix, field.Value, valueWidth, s)
	} else {
		sb.WriteString(s.Muted.Render(fmt.Sprintf("%s%s", prefix, field.Value)))
		sb.WriteString("\n")
	}
}

// renderWrappedField renders a field value with text wrapping.
func renderWrappedField(sb *strings.Builder, prefix, value string, valueWidth int, s *styles.Styles) {
	sb.WriteString(s.Muted.Render(prefix))
	wrappedValue := WrapText(value, valueWidth)
	lines := strings.Split(wrappedValue, "\n")
	for i, line := range lines {
		if i > 0 {
			sb.WriteString(s.Muted.Render(strings.Repeat(" ", len(prefix))))
		}
		sb.WriteString(s.Muted.Render(line))
		sb.WriteString("\n")
	}
}

// WrapText wraps text to fit within the specified width.
func WrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var result strings.Builder
	remaining := text

	for len(remaining) > width {
		// Find a good break point (last space within width)
		breakPoint := width
		for i := width; i > 0; i-- {
			if remaining[i] == ' ' {
				breakPoint = i
				break
			}
		}

		// If no space found, break at width
		if breakPoint == width && remaining[width] != ' ' {
			// Check if we're in the middle of a word - just break at width
			breakPoint = width
		}

		result.WriteString(remaining[:breakPoint])
		result.WriteString("\n")

		// Skip the space if we broke at one
		if breakPoint < len(remaining) && remaining[breakPoint] == ' ' {
			remaining = remaining[breakPoint+1:]
		} else {
			remaining = remaining[breakPoint:]
		}
	}

	result.WriteString(remaining)
	return result.String()
}

// WrapTextLines wraps text to fit within the specified width, returning lines as a slice.
// Uses word-boundary wrapping: breaks at spaces when possible, otherwise at width.
func WrapTextLines(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return lines
}

// CollectNonComputedFields extracts all fields from spec data excluding computed fields.
// This is used to show the user-provided spec without the computed outputs.
// Fields are returned sorted alphabetically by name for consistent display.
func CollectNonComputedFields(specData *core.MappingNode, computedFields []string) []OutputField {
	if specData == nil || specData.Fields == nil {
		return nil
	}

	// Build a set of computed field names for quick lookup
	computedSet := make(map[string]bool, len(computedFields))
	for _, path := range computedFields {
		// Strip "spec." prefix to get the field name
		fieldName := strings.TrimPrefix(path, "spec.")
		// Only consider top-level fields for exclusion
		if !strings.Contains(fieldName, ".") {
			computedSet[fieldName] = true
		}
	}

	// Collect field names and sort them for consistent ordering
	var fieldNames []string
	for fieldName := range specData.Fields {
		if !computedSet[fieldName] {
			fieldNames = append(fieldNames, fieldName)
		}
	}
	sort.Strings(fieldNames)

	var fields []OutputField
	for _, fieldName := range fieldNames {
		fieldValue := specData.Fields[fieldName]
		formattedValue := headless.FormatMappingNode(fieldValue)
		if IsValidOutputValue(formattedValue) {
			fields = append(fields, OutputField{Name: fieldName, Value: formattedValue})
		}
	}
	return fields
}

// CountNonComputedFields counts the number of non-computed fields in spec data.
func CountNonComputedFields(specData *core.MappingNode, computedFields []string) int {
	fields := CollectNonComputedFields(specData, computedFields)
	return len(fields)
}

// RenderSpecHint renders a hint line showing the spec field count and keyboard shortcut.
func RenderSpecHint(specData *core.MappingNode, computedFields []string, s *styles.Styles) string {
	count := CountNonComputedFields(specData, computedFields)
	if count == 0 {
		return ""
	}
	return s.Muted.Render(fmt.Sprintf("Spec: (%d fields) - press 's' to view", count))
}

// CollectNonComputedFieldsPretty extracts all fields from spec data excluding computed fields,
// using pretty-printed JSON for complex nested structures.
// Fields are returned sorted alphabetically by name for consistent display.
func CollectNonComputedFieldsPretty(specData *core.MappingNode, computedFields []string) []OutputField {
	if specData == nil || specData.Fields == nil {
		return nil
	}

	// Build a set of computed field names for quick lookup
	computedSet := make(map[string]bool, len(computedFields))
	for _, path := range computedFields {
		// Strip "spec." prefix to get the field name
		fieldName := strings.TrimPrefix(path, "spec.")
		// Only consider top-level fields for exclusion
		if !strings.Contains(fieldName, ".") {
			computedSet[fieldName] = true
		}
	}

	// Collect field names and sort them for consistent ordering
	var fieldNames []string
	for fieldName := range specData.Fields {
		if !computedSet[fieldName] {
			fieldNames = append(fieldNames, fieldName)
		}
	}
	sort.Strings(fieldNames)

	prettyOpts := headless.FormatMappingNodeOptions{PrettyPrint: true}
	var fields []OutputField
	for _, fieldName := range fieldNames {
		fieldValue := specData.Fields[fieldName]
		formattedValue := headless.FormatMappingNodeWithOptions(fieldValue, prettyOpts)
		if IsValidOutputValue(formattedValue) {
			fields = append(fields, OutputField{Name: fieldName, Value: formattedValue})
		}
	}
	return fields
}

// RenderSpecFields renders non-computed spec fields with a label.
func RenderSpecFields(specData *core.MappingNode, computedFields []string, width int, s *styles.Styles) string {
	fields := CollectNonComputedFieldsPretty(specData, computedFields)
	if len(fields) == 0 {
		return ""
	}
	return RenderOutputFieldsWithLabel(fields, "Spec:", width, s)
}

// ExportField holds export data for rendering, including metadata about the export.
type ExportField struct {
	Name        string
	Value       string
	Type        string
	Description string
	Field       string
}

// CollectExportFields extracts exports from instance state, sorted alphabetically by name.
func CollectExportFields(exports map[string]*state.ExportState) []ExportField {
	if len(exports) == 0 {
		return nil
	}

	// Collect and sort export names for consistent ordering
	var exportNames []string
	for name := range exports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)

	var fields []ExportField
	for _, name := range exportNames {
		export := exports[name]
		if export == nil {
			continue
		}
		formattedValue := headless.FormatMappingNode(export.Value)
		fields = append(fields, ExportField{
			Name:        name,
			Value:       formattedValue,
			Type:        string(export.Type),
			Description: export.Description,
			Field:       export.Field,
		})
	}
	return fields
}

// CollectExportFieldsPretty extracts exports with pretty-printed JSON values.
func CollectExportFieldsPretty(exports map[string]*state.ExportState) []ExportField {
	if len(exports) == 0 {
		return nil
	}

	// Collect and sort export names for consistent ordering
	var exportNames []string
	for name := range exports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)

	prettyOpts := headless.FormatMappingNodeOptions{PrettyPrint: true}
	var fields []ExportField
	for _, name := range exportNames {
		export := exports[name]
		if export == nil {
			continue
		}
		formattedValue := headless.FormatMappingNodeWithOptions(export.Value, prettyOpts)
		fields = append(fields, ExportField{
			Name:        name,
			Value:       formattedValue,
			Type:        string(export.Type),
			Description: export.Description,
			Field:       export.Field,
		})
	}
	return fields
}
