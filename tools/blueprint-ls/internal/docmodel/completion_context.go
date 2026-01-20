package docmodel

import (
	"regexp"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// CompletionContextKind provides type-safe completion context classification.
// This replaces the string-based element type matching in completion.go.
type CompletionContextKind int

const (
	CompletionContextUnknown CompletionContextKind = iota

	// Type fields
	CompletionContextResourceType
	CompletionContextDataSourceType
	CompletionContextVariableType
	CompletionContextValueType
	CompletionContextExportType
	CompletionContextDataSourceFieldType

	// Data source filter fields
	CompletionContextDataSourceFilterField
	CompletionContextDataSourceFilterOperator

	// Substitution references
	CompletionContextStringSub           // Inside ${...} but no specific reference
	CompletionContextStringSubVariableRef
	CompletionContextStringSubResourceRef
	CompletionContextStringSubResourceProperty
	CompletionContextStringSubDataSourceRef
	CompletionContextStringSubDataSourceProperty
	CompletionContextStringSubValueRef
	CompletionContextStringSubChildRef
	CompletionContextStringSubElemRef
)

var (
	// Patterns for detecting text context before cursor.
	// Support both YAML (type: ) and JSON ("type": ) syntax.
	typeFieldPattern     = regexp.MustCompile(`("type"|type):\s*$`)
	fieldFieldPattern    = regexp.MustCompile(`("field"|field):\s*$`)
	operatorFieldPattern = regexp.MustCompile(`("operator"|operator):\s*$`)

	// Substitution reference patterns.
	variableRefTextPattern             = regexp.MustCompile(`variables\.$`)
	resourceRefTextPattern             = regexp.MustCompile(`resources\.$`)
	resourcePropertyTextPattern        = regexp.MustCompile(`resources\.([A-Za-z0-9_-]|"|\.|\[|\])+\.$`)
	resourceWithoutNamespacePropTextPattern = regexp.MustCompile(`([A-Za-z0-9_-]|"|\.|\[|\])+\.$`)
	dataSourceRefTextPattern           = regexp.MustCompile(`datasources\.$`)
	dataSourcePropertyTextPattern      = regexp.MustCompile(`datasources\.(([A-Za-z0-9_-]|"|\.|\[|\])+)\.$`)
	valueRefTextPattern                = regexp.MustCompile(`values\.$`)
	childRefTextPattern                = regexp.MustCompile(`children\.$`)
	elemRefTextPattern                 = regexp.MustCompile(`elem\.$`)
	subOpenTextPattern                 = regexp.MustCompile(`\${$`)
	inSubTextPattern                   = regexp.MustCompile(`\${[^\}]*$`)
)

// CompletionContext provides rich context for completion suggestions.
type CompletionContext struct {
	Kind CompletionContextKind

	// Node context from position resolution
	NodeCtx *NodeContext

	// Extracted names when applicable
	ResourceName   string
	DataSourceName string

	// Text analysis results
	TextBefore string
	IsInSub    bool
}

// DetermineCompletionContext analyzes the node context and text to determine
// what kind of completion should be provided.
func DetermineCompletionContext(nodeCtx *NodeContext) *CompletionContext {
	ctx := &CompletionContext{
		Kind:       CompletionContextUnknown,
		NodeCtx:    nodeCtx,
		TextBefore: nodeCtx.TextBefore,
		IsInSub:    nodeCtx.InSubstitution(),
	}

	// Check path-based type fields first (most specific)
	if kind := determineTypeFieldContext(nodeCtx); kind != CompletionContextUnknown {
		ctx.Kind = kind
		return ctx
	}

	// Check data source filter contexts
	if kind := determineDataSourceFilterContext(nodeCtx); kind != CompletionContextUnknown {
		ctx.Kind = kind
		return ctx
	}

	// Check substitution contexts
	if kind, resourceName, dsName := determineSubstitutionContext(nodeCtx); kind != CompletionContextUnknown {
		ctx.Kind = kind
		ctx.ResourceName = resourceName
		ctx.DataSourceName = dsName
		return ctx
	}

	return ctx
}

func determineTypeFieldContext(nodeCtx *NodeContext) CompletionContextKind {
	path := nodeCtx.ASTPath
	textBefore := nodeCtx.TextBefore

	// Check for existing type field positions based on path
	if path.IsResourceType() {
		return CompletionContextResourceType
	}
	if path.IsDataSourceType() {
		return CompletionContextDataSourceType
	}
	if path.IsVariableType() {
		return CompletionContextVariableType
	}
	if path.IsValueType() {
		return CompletionContextValueType
	}
	if path.IsExportType() {
		return CompletionContextExportType
	}

	// Check for data source field type (nested path)
	if isDataSourceFieldTypePath(path) {
		return CompletionContextDataSourceFieldType
	}

	// Check for new type field being entered (text-based detection)
	if typeFieldPattern.MatchString(textBefore) {
		return determineNewTypeFieldContext(path)
	}

	return CompletionContextUnknown
}

func isDataSourceFieldTypePath(path StructuredPath) bool {
	// Pattern: /datasources/{name}/exports/{exportName}/type
	return len(path) == 5 &&
		path.At(0).FieldName == "datasources" &&
		path.At(2).FieldName == "exports" &&
		path.At(4).FieldName == "type"
}

var sectionToTypeFieldContext = map[string]CompletionContextKind{
	"resources": CompletionContextResourceType,
	"variables": CompletionContextVariableType,
	"values":    CompletionContextValueType,
	"exports":   CompletionContextExportType,
}

func determineNewTypeFieldContext(path StructuredPath) CompletionContextKind {
	if len(path) < 2 {
		return CompletionContextUnknown
	}

	section := path.At(0).FieldName

	// Data sources require special handling due to nested field types
	if section == "datasources" {
		if len(path) >= 4 {
			return CompletionContextDataSourceFieldType
		}
		return CompletionContextDataSourceType
	}

	if kind, ok := sectionToTypeFieldContext[section]; ok {
		return kind
	}

	return CompletionContextUnknown
}

func determineDataSourceFilterContext(nodeCtx *NodeContext) CompletionContextKind {
	path := nodeCtx.ASTPath
	textBefore := nodeCtx.TextBefore

	// Check for existing filter field/operator positions
	if path.IsDataSourceFilterField() {
		return CompletionContextDataSourceFilterField
	}
	if path.IsDataSourceFilterOperator() {
		return CompletionContextDataSourceFilterOperator
	}

	// Check for new filter field being entered
	if isInDataSourceFilters(path) {
		if fieldFieldPattern.MatchString(textBefore) {
			return CompletionContextDataSourceFilterField
		}
		if operatorFieldPattern.MatchString(textBefore) {
			return CompletionContextDataSourceFilterOperator
		}
	}

	return CompletionContextUnknown
}

func isInDataSourceFilters(path StructuredPath) bool {
	return len(path) >= 4 &&
		path.At(0).FieldName == "datasources" &&
		path.At(2).FieldName == "filters"
}

func determineSubstitutionContext(nodeCtx *NodeContext) (CompletionContextKind, string, string) {
	textBefore := nodeCtx.TextBefore

	// Must be in a substitution context
	if !nodeCtx.InSubstitution() && !subOpenTextPattern.MatchString(textBefore) {
		// Check for opening ${ pattern
		if subOpenTextPattern.MatchString(textBefore) {
			return CompletionContextStringSub, "", ""
		}
		return CompletionContextUnknown, "", ""
	}

	// Check for variable reference: variables.
	if variableRefTextPattern.MatchString(textBefore) {
		return CompletionContextStringSubVariableRef, "", ""
	}

	// Check for data source property reference: datasources.{name}.
	if matches := dataSourcePropertyTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return CompletionContextStringSubDataSourceProperty, "", matches[1]
	}

	// Check for data source reference: datasources.
	if dataSourceRefTextPattern.MatchString(textBefore) {
		return CompletionContextStringSubDataSourceRef, "", ""
	}

	// Check for value reference: values.
	if valueRefTextPattern.MatchString(textBefore) {
		return CompletionContextStringSubValueRef, "", ""
	}

	// Check for child reference: children.
	if childRefTextPattern.MatchString(textBefore) {
		return CompletionContextStringSubChildRef, "", ""
	}

	// Check for elem reference: elem.
	if elemRefTextPattern.MatchString(textBefore) {
		return CompletionContextStringSubElemRef, "", ""
	}

	// Check for resource property reference: resources.{name}.{prop}.
	if resourcePropertyTextPattern.MatchString(textBefore) {
		resourceName := extractResourceNameFromText(textBefore)
		return CompletionContextStringSubResourceProperty, resourceName, ""
	}

	// Check for resource reference: resources.
	if resourceRefTextPattern.MatchString(textBefore) {
		return CompletionContextStringSubResourceRef, "", ""
	}

	// Check for resource property without namespace (when in resource property context)
	if nodeCtx.SchemaElement != nil {
		if _, isResourceProp := nodeCtx.SchemaElement.(*substitutions.SubstitutionResourceProperty); isResourceProp {
			if resourceWithoutNamespacePropTextPattern.MatchString(textBefore) {
				resourceName := extractResourceNameFromSchemaElement(nodeCtx.SchemaElement)
				return CompletionContextStringSubResourceProperty, resourceName, ""
			}
		}
	}

	// General substitution context (inside ${...})
	if inSubTextPattern.MatchString(textBefore) || nodeCtx.InSubstitution() {
		return CompletionContextStringSub, "", ""
	}

	return CompletionContextUnknown, "", ""
}

func extractResourceNameFromText(textBefore string) string {
	// Extract resource name from pattern like "resources.myResource."
	idx := strings.Index(textBefore, "resources.")
	if idx == -1 {
		return ""
	}

	after := textBefore[idx+len("resources."):]
	// Find the resource name (up to the next .)
	parts := strings.SplitN(after, ".", 2)
	if len(parts) == 0 {
		return ""
	}

	// Clean up the name (remove quotes if present)
	name := strings.Trim(parts[0], "\"")
	return name
}

func extractResourceNameFromSchemaElement(elem any) string {
	if resourceProp, ok := elem.(*substitutions.SubstitutionResourceProperty); ok {
		return resourceProp.ResourceName
	}
	return ""
}

var completionContextKindNames = map[CompletionContextKind]string{
	CompletionContextUnknown:                    "unknown",
	CompletionContextResourceType:               "resourceType",
	CompletionContextDataSourceType:             "dataSourceType",
	CompletionContextVariableType:               "variableType",
	CompletionContextValueType:                  "valueType",
	CompletionContextExportType:                 "exportType",
	CompletionContextDataSourceFieldType:        "dataSourceFieldType",
	CompletionContextDataSourceFilterField:      "dataSourceFilterField",
	CompletionContextDataSourceFilterOperator:   "dataSourceFilterOperator",
	CompletionContextStringSub:                  "stringSub",
	CompletionContextStringSubVariableRef:       "stringSubVariableRef",
	CompletionContextStringSubResourceRef:       "stringSubResourceRef",
	CompletionContextStringSubResourceProperty:  "stringSubResourceProperty",
	CompletionContextStringSubDataSourceRef:     "stringSubDataSourceRef",
	CompletionContextStringSubDataSourceProperty: "stringSubDataSourceProperty",
	CompletionContextStringSubValueRef:          "stringSubValueRef",
	CompletionContextStringSubChildRef:          "stringSubChildRef",
	CompletionContextStringSubElemRef:           "stringSubElemRef",
}

// String returns a string representation of CompletionContextKind.
func (k CompletionContextKind) String() string {
	if name, ok := completionContextKindNames[k]; ok {
		return name
	}
	return "unknown"
}

// IsTypeField returns true if this context is for a type field.
func (k CompletionContextKind) IsTypeField() bool {
	switch k {
	case CompletionContextResourceType,
		CompletionContextDataSourceType,
		CompletionContextVariableType,
		CompletionContextValueType,
		CompletionContextExportType,
		CompletionContextDataSourceFieldType:
		return true
	}
	return false
}

// IsSubstitution returns true if this context is inside a substitution.
func (k CompletionContextKind) IsSubstitution() bool {
	switch k {
	case CompletionContextStringSub,
		CompletionContextStringSubVariableRef,
		CompletionContextStringSubResourceRef,
		CompletionContextStringSubResourceProperty,
		CompletionContextStringSubDataSourceRef,
		CompletionContextStringSubDataSourceProperty,
		CompletionContextStringSubValueRef,
		CompletionContextStringSubChildRef,
		CompletionContextStringSubElemRef:
		return true
	}
	return false
}

// IsDataSourceFilter returns true if this context is for a data source filter.
func (k CompletionContextKind) IsDataSourceFilter() bool {
	return k == CompletionContextDataSourceFilterField ||
		k == CompletionContextDataSourceFilterOperator
}
