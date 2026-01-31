package helpinfo

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// RenderResourceHoverInfo renders a schema definition for a resource
// combined with contextual type and description info.
func RenderResourceHoverInfo(name string, resource *schema.Resource) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"**Resource** `%s`\n\n"+
			"Defines an infrastructure component to deploy and manage.\n\n",
		name,
	))

	if resource.Type != nil {
		sb.WriteString(fmt.Sprintf("**type:** `%s`\n\n", resource.Type.Value))
	}

	if resource.Description != nil {
		desc, _ := substitutions.SubstitutionsToString("", resource.Description)
		if desc != "" {
			sb.WriteString(desc)
		}
	}

	return sb.String()
}

// RenderVariableHoverInfo renders a schema definition for a variable
// combined with contextual type and description info.
func RenderVariableHoverInfo(name string, variable *schema.Variable) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"**Variable** `%s`\n\n"+
			"An input parameter that can be provided at deploy time.\n\n",
		name,
	))

	if variable.Type != nil {
		sb.WriteString(fmt.Sprintf("**type:** `%s`\n\n", string(variable.Type.Value)))
	}

	if variable.Description != nil && variable.Description.StringValue != nil {
		sb.WriteString(*variable.Description.StringValue)
	}

	return sb.String()
}

// RenderValueHoverInfo renders a schema definition for a value
// combined with contextual type and description info.
func RenderValueHoverInfo(name string, value *schema.Value) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"**Value** `%s`\n\n"+
			"A computed or static value used throughout the blueprint.\n\n",
		name,
	))

	if value.Type != nil {
		sb.WriteString(fmt.Sprintf("**type:** `%s`\n\n", string(value.Type.Value)))
	}

	if value.Description != nil {
		desc, _ := substitutions.SubstitutionsToString("", value.Description)
		if desc != "" {
			sb.WriteString(desc)
		}
	}

	return sb.String()
}

// RenderDataSourceHoverInfo renders a schema definition for a data source
// combined with contextual type and description info.
func RenderDataSourceHoverInfo(name string, ds *schema.DataSource) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"**Data Source** `%s`\n\n"+
			"Fetches external data for use in the blueprint.\n\n",
		name,
	))

	if ds.Type != nil {
		sb.WriteString(fmt.Sprintf("**type:** `%s`\n\n", string(ds.Type.Value)))
	}

	if ds.Description != nil {
		desc, _ := substitutions.SubstitutionsToString("", ds.Description)
		if desc != "" {
			sb.WriteString(desc)
		}
	}

	return sb.String()
}

// RenderIncludeHoverInfo renders a schema definition for an include (child blueprint)
// combined with contextual path and description info.
func RenderIncludeHoverInfo(name string, include *schema.Include) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"**Include** `%s`\n\n"+
			"A child blueprint included in this blueprint.\n\n",
		name,
	))

	if include.Path != nil {
		path, _ := substitutions.SubstitutionsToString("", include.Path)
		if path != "" {
			sb.WriteString(fmt.Sprintf("**path:** `%s`\n\n", path))
		}
	}

	if include.Description != nil {
		desc, _ := substitutions.SubstitutionsToString("", include.Description)
		if desc != "" {
			sb.WriteString(desc)
		}
	}

	return sb.String()
}

// RenderExportHoverInfo renders a schema definition for an export
// combined with contextual type, field, and description info.
func RenderExportHoverInfo(name string, export *schema.Export) string {
	return fmt.Sprintf(
		"**Export** `%s`\n\n"+
			"A value exported from this blueprint for use by other blueprints and external systems.",
		name,
	)
}

var sectionDefinitions = map[string]string{
	"resources": "**resources**\n\nDefines infrastructure resources to deploy and manage. " +
		"Each resource has a type, spec, and optional metadata.",
	"variables": "**variables**\n\nInput parameters provided at deploy time. " +
		"Variables can have types, defaults, and allowed values.",
	"values": "**values**\n\nComputed or static values used throughout the blueprint. " +
		"Values can reference variables and other values.",
	"datasources": "**datasources**\n\nExternal data sources queried at deploy time. " +
		"Data sources provide exports that can be referenced in substitutions.",
	"includes": "**includes**\n\nChild blueprints included in this blueprint. " +
		"Includes can pass variables and receive exports.",
	"exports":   "**exports**\n\nValues exported from this blueprint for use by parent blueprints.",
	"version":   "**version**\n\nThe blueprint specification version.",
	"transform": "**transform**\n\nTransformers that modify the blueprint before validation and deployment.",
	"metadata":  "**metadata**\n\nBlueprint-level metadata containing arbitrary key-value data.",
}

// RenderSectionDefinition renders a schema definition for a top-level section.
func RenderSectionDefinition(sectionName string) string {
	if def, ok := sectionDefinitions[sectionName]; ok {
		return def
	}
	return ""
}

var resourceFieldDefinitions = map[string]string{
	"type":         "**type**\n\nThe resource type identifier from a registered provider (e.g. `aws/dynamodb/table`).",
	"spec":         "**spec**\n\nThe provider-specific configuration for this resource.",
	"metadata":     "**metadata**\n\nContains annotations, labels, display name, and custom metadata.",
	"custom":       "**custom**\n\nArbitrary key-value metadata for use by tools and systems built around blueprints.",
	"linkSelector": "**linkSelector**\n\nConfigures which resources this resource links to. Resources are matched by label selectors.",
	"condition":    "**condition**\n\nA condition that determines whether this resource is deployed.",
	"each":         "**each**\n\nA substitution that resolves to a list, creating one resource instance per element.",
	"description":  "**description**\n\nA human-readable description of this resource.",
	"dependsOn":    "**dependsOn**\n\nExplicit dependencies on other resources for deployment ordering.",
}

var variableFieldDefinitions = map[string]string{
	"type":          "**type**\n\nThe variable data type (e.g. `string`, `integer`, `float`, `boolean`).",
	"description":   "**description**\n\nA human-readable description of this variable.",
	"secret":        "**secret**\n\nWhether this variable contains sensitive data that should be masked.",
	"default":       "**default**\n\nThe default value used when no value is provided at deploy time.",
	"allowedValues": "**allowedValues**\n\nA list of values that are valid for this variable.",
}

var valueFieldDefinitions = map[string]string{
	"type":        "**type**\n\nThe value data type (e.g. `string`, `integer`, `float`, `boolean`, `array`, `object`).",
	"value":       "**value**\n\nThe value content, which can include substitutions referencing variables and other values.",
	"description": "**description**\n\nA human-readable description of this value.",
	"secret":      "**secret**\n\nWhether this value contains sensitive data that should be masked.",
}

var dataSourceFieldDefinitions = map[string]string{
	"type":        "**type**\n\nThe data source type identifier from a registered provider.",
	"metadata":    "**metadata**\n\nContains display name, annotations, and custom metadata for this data source.",
	"custom":      "**custom**\n\nArbitrary key-value metadata for use by tools and systems built around blueprints.",
	"filter":      "**filter**\n\nFilter criteria for querying the data source.",
	"exports":     "**exports**\n\nFields exported from the data source that can be referenced in substitutions.",
	"description": "**description**\n\nA human-readable description of this data source.",
}

var includeFieldDefinitions = map[string]string{
	"path":        "**path**\n\nThe file path to the child blueprint.",
	"variables":   "**variables**\n\nVariables passed to the child blueprint.",
	"metadata":    "**metadata**\n\nMetadata passed to the child blueprint.",
	"description": "**description**\n\nA human-readable description of this include.",
}

var exportFieldDefinitions = map[string]string{
	"type":        "**type**\n\nThe export data type.",
	"field":       "**field**\n\nThe field path that resolves to the exported value.",
	"description": "**description**\n\nA human-readable description of this export.",
}

var fieldDefinitionsByParent = map[string]map[string]string{
	"resource":   resourceFieldDefinitions,
	"variable":   variableFieldDefinitions,
	"value":      valueFieldDefinitions,
	"datasource": dataSourceFieldDefinitions,
	"include":    includeFieldDefinitions,
	"export":     exportFieldDefinitions,
}

// RenderFieldDefinition renders a schema definition for a known blueprint field,
// contextualised by the parent element type (e.g. "resource", "variable").
func RenderFieldDefinition(fieldName string, parentContext string) string {
	if defs, ok := fieldDefinitionsByParent[parentContext]; ok {
		if def, ok := defs[fieldName]; ok {
			return def
		}
	}
	return ""
}

// RenderSpecFieldDefinition renders a provider-sourced field definition
// with type and description.
func RenderSpecFieldDefinition(
	fieldName string,
	specFieldSchema *provider.ResourceDefinitionsSchema,
) string {
	if specFieldSchema == nil {
		return fmt.Sprintf("`%s`", fieldName)
	}

	description := specFieldSchema.FormattedDescription
	if description == "" {
		description = specFieldSchema.Description
	}

	return fmt.Sprintf(
		"`%s`\n\n**type:** `%s`\n\n%s",
		fieldName,
		specFieldSchema.Type,
		description,
	)
}

// RenderLinkAnnotationDefinition renders a link annotation definition
// with type, description, allowed values, and default.
func RenderLinkAnnotationDefinition(
	annotationKey string,
	def *provider.LinkAnnotationDefinition,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"**Link Annotation** `%s`\n\n",
		annotationKey,
	))

	sb.WriteString(fmt.Sprintf("**type:** `%s`\n\n", string(def.Type)))

	if def.Required {
		sb.WriteString("**required**\n\n")
	}

	if def.Description != "" {
		sb.WriteString(def.Description)
		sb.WriteString("\n\n")
	}

	if def.DefaultValue != nil {
		sb.WriteString(fmt.Sprintf("**default:** `%s`\n\n", def.DefaultValue.ToString()))
	}

	if len(def.AllowedValues) > 0 {
		sb.WriteString("**allowed values:** ")
		values := make([]string, 0, len(def.AllowedValues))
		for _, v := range def.AllowedValues {
			if v != nil {
				values = append(values, fmt.Sprintf("`%s`", v.ToString()))
			}
		}
		sb.WriteString(strings.Join(values, ", "))
	}

	return strings.TrimSpace(sb.String())
}

// RenderAnnotationKeyInfo renders basic annotation key information
// as a fallback when a link annotation definition is not found.
func RenderAnnotationKeyInfo(annotationKey string, annotationValue string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Annotation** `%s`\n\n", annotationKey))

	if annotationValue != "" {
		sb.WriteString(fmt.Sprintf("**value:** `%s`", annotationValue))
	}

	return strings.TrimSpace(sb.String())
}

// RenderMetadataDefinition renders a schema definition for a metadata section.
func RenderMetadataDefinition(parentContext string) string {
	switch parentContext {
	case "resource":
		return "**metadata**\n\nContains annotations, labels, display name, " +
			"and custom metadata for this resource."
	case "datasource":
		return "**metadata**\n\nContains display name, annotations, " +
			"and custom metadata for this data source."
	default:
		return "**metadata**\n\nMetadata containing annotations and other key-value data."
	}
}

// RenderLinkSelectorDefinition renders a schema definition for a link selector.
func RenderLinkSelectorDefinition() string {
	return "**linkSelector**\n\nConfigures which resources this resource links to. " +
		"Resources are matched by label selectors using `byLabel`. " +
		"Matched resources establish link relationships that enable " +
		"inter-resource configuration via annotations."
}

// RenderDataSourceExportFieldDefinition renders a data source export field definition
// without requiring a substitution reference. When specSchema is provided,
// it shows the provider-sourced field definition; otherwise falls back to the
// blueprint-level description.
func RenderDataSourceExportFieldDefinition(
	fieldName string,
	ds *schema.DataSource,
	export *schema.DataSourceFieldExport,
	specSchema *provider.DataSourceSpecSchema,
) string {
	var sb strings.Builder

	dataSourceType := "unknown"
	if ds.Type != nil {
		dataSourceType = string(ds.Type.Value)
	}

	fieldType := "unknown"
	if export.Type != nil {
		fieldType = string(export.Type.Value)
	}

	sb.WriteString(fmt.Sprintf(
		"**Export Field** `%s`\n\n**type:** `%s`\n\n",
		fieldName,
		fieldType,
	))

	if export.AliasFor != nil && export.AliasFor.StringValue != nil {
		sb.WriteString(fmt.Sprintf("**alias for:** `%s`\n\n", *export.AliasFor.StringValue))
	}

	writeExportFieldDescription(&sb, export, specSchema)

	sb.WriteString(fmt.Sprintf("*Data source type:* `%s`", dataSourceType))

	return strings.TrimSpace(sb.String())
}

func writeExportFieldDescription(
	sb *strings.Builder,
	export *schema.DataSourceFieldExport,
	specSchema *provider.DataSourceSpecSchema,
) {
	if specSchema != nil {
		description := specSchema.FormattedDescription
		if description == "" {
			description = specSchema.Description
		}
		if description != "" {
			sb.WriteString(description)
			sb.WriteString("\n\n")
			return
		}
	}

	if export.Description != nil {
		desc, _ := substitutions.SubstitutionsToString("", export.Description)
		if desc != "" {
			sb.WriteString(desc)
			sb.WriteString("\n\n")
		}
	}
}

// RenderLabelsDefinition renders a schema definition for a labels map.
func RenderLabelsDefinition() string {
	return "**labels**\n\nKey-value pairs for organizing and categorizing resources. " +
		"Used by link selectors to match resources for linking."
}

// RenderAnnotationsDefinition renders a schema definition for an annotations map.
func RenderAnnotationsDefinition() string {
	return "**annotations**\n\nKey-value pairs for storing metadata about the resource. " +
		"Link annotations configure link behaviour between linked resources."
}

// RenderByLabelDefinition renders a schema definition for a byLabel selector.
func RenderByLabelDefinition() string {
	return "**byLabel**\n\nLabel selector that matches resources whose labels " +
		"contain the specified key-value pairs."
}

// MatchingResourceInfo holds information about a resource that matches a byLabel selector.
type MatchingResourceInfo struct {
	Name         string
	ResourceType string
	Linked       bool
}

// RenderByLabelHoverContent renders hover content for a byLabel selector,
// showing the base definition, optional focused label key, and matching resources.
func RenderByLabelHoverContent(
	labelKey string,
	labelValue string,
	matchingResources []MatchingResourceInfo,
) string {
	var sb strings.Builder
	sb.WriteString("**byLabel**\n\nLabel selector that matches resources whose labels " +
		"contain the specified key-value pairs.\n\n")

	if labelKey != "" {
		sb.WriteString(fmt.Sprintf("**%s:** `%s`\n\n", labelKey, labelValue))
	}

	if len(matchingResources) == 0 {
		sb.WriteString("*No matching resources found.*")
		return sb.String()
	}

	sb.WriteString("**Matching resources:**\n\n")
	for _, res := range matchingResources {
		linkStatus := ""
		if res.Linked {
			linkStatus = " *(linked)*"
		}
		if res.ResourceType != "" {
			sb.WriteString(fmt.Sprintf("- `%s` — `%s`%s\n", res.Name, res.ResourceType, linkStatus))
		} else {
			sb.WriteString(fmt.Sprintf("- `%s`%s\n", res.Name, linkStatus))
		}
	}

	return strings.TrimSpace(sb.String())
}

// RenderExcludeDefinition renders a schema definition for an exclude list.
func RenderExcludeDefinition() string {
	return "**exclude**\n\nA list of resource names to exclude from link matching."
}

// RenderDataSourceFieldTypeDefinition renders hover info for a data source export field type.
func RenderDataSourceFieldTypeDefinition(fieldType string) string {
	return fmt.Sprintf(
		"**type** `%s`\n\nThe data type of this exported field.",
		fieldType,
	)
}

// RenderDataSourceFilterDefinition renders hover info for a data source filter.
func RenderDataSourceFilterDefinition(filter *schema.DataSourceFilter) string {
	var sb strings.Builder
	sb.WriteString("**filter**\n\nA filter expression matching data source records by field values.\n\n")

	if filter.Field != nil && filter.Field.StringValue != nil {
		sb.WriteString(fmt.Sprintf("**field:** `%s`\n\n", *filter.Field.StringValue))
	}

	if filter.Operator != nil {
		sb.WriteString(fmt.Sprintf("**operator:** `%s`", string(filter.Operator.Value)))
	}

	return strings.TrimSpace(sb.String())
}

// RenderDataSourceFilterOperatorDefinition renders hover info for a filter operator.
func RenderDataSourceFilterOperatorDefinition(operator string) string {
	return fmt.Sprintf(
		"**operator** `%s`\n\n"+
			"The comparison operator for this filter.\n\n"+
			"**valid operators:** `=`, `!=`, `in`, `not in`, `has key`, `not has key`, "+
			"`contains`, `not contains`, `starts with`, `not starts with`, "+
			"`ends with`, `not ends with`, `>`, `<`, `>=`, `<=`",
		operator,
	)
}

// RenderDataSourceFilterSearchDefinition renders hover info for a filter search field.
func RenderDataSourceFilterSearchDefinition() string {
	return "**search**\n\nThe value or values to match against in the filtered data source field."
}

var metadataFieldDefinitions = map[string]string{
	"displayName": "**displayName**\n\nThe human-readable display name for this element.",
	"annotations": "**annotations**\n\nKey-value pairs for storing metadata about the resource. " +
		"Link annotations configure link behaviour between linked resources.",
	"labels": "**labels**\n\nKey-value pairs for organizing and categorizing resources. " +
		"Used by link selectors to match resources for linking.",
	"custom": "**custom**\n\nArbitrary key-value metadata for use by tools and systems " +
		"built around blueprints.",
}

// RenderMetadataFieldDefinition renders a schema definition for a
// specific field within a metadata section.
func RenderMetadataFieldDefinition(fieldName string) string {
	if def, ok := metadataFieldDefinitions[fieldName]; ok {
		return def
	}
	return ""
}

// RenderDataSourceFilterFieldKeyDefinition renders hover info for the
// "field" key within a data source filter, showing the field value and
// optional provider-sourced field definition.
func RenderDataSourceFilterFieldKeyDefinition(
	fieldValue string,
	filterSchema *provider.DataSourceFilterSchema,
) string {
	var sb strings.Builder

	if fieldValue != "" {
		sb.WriteString(fmt.Sprintf("**field** `%s`\n\n", fieldValue))
	} else {
		sb.WriteString("**field**\n\n")
	}

	sb.WriteString("The data source field to filter on.\n\n")

	if filterSchema != nil {
		description := filterSchema.FormattedDescription
		if description == "" {
			description = filterSchema.Description
		}
		if description != "" {
			sb.WriteString(description)
			sb.WriteString("\n\n")
		}
		if filterSchema.Type != "" {
			sb.WriteString(fmt.Sprintf("**type:** `%s`", string(filterSchema.Type)))
		}
	}

	return strings.TrimSpace(sb.String())
}
