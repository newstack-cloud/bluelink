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

	// Data source nested field contexts
	CompletionContextDataSourceFilterDefinitionField // Fields inside a filter definition (field, operator, search)
	CompletionContextDataSourceExportDefinitionField // Fields inside an export definition (type, aliasFor, description)
	CompletionContextDataSourceMetadataField         // Fields inside data source metadata (displayName, annotations, custom)

	// Resource spec field (editing directly in YAML/JSONC definition)
	CompletionContextResourceSpecField

	// Resource metadata field (displayName, labels, annotations, custom)
	CompletionContextResourceMetadataField

	// Resource definition field (type, description, spec, metadata, etc.)
	CompletionContextResourceDefinitionField

	// Definition field contexts for other sections
	CompletionContextVariableDefinitionField
	CompletionContextValueDefinitionField
	CompletionContextDataSourceDefinitionField
	CompletionContextIncludeDefinitionField
	CompletionContextExportDefinitionField
	CompletionContextBlueprintTopLevelField

	// Substitution references
	CompletionContextStringSub // Inside ${...} but no specific reference
	CompletionContextStringSubVariableRef
	CompletionContextStringSubResourceRef
	CompletionContextStringSubResourceProperty
	CompletionContextStringSubDataSourceRef
	CompletionContextStringSubDataSourceProperty
	CompletionContextStringSubValueRef
	CompletionContextStringSubChildRef
	CompletionContextStringSubElemRef
	CompletionContextStringSubPartialPath           // Typing a partial path (no completions)
	CompletionContextStringSubPotentialResourceProp // Potential standalone resource property (needs blueprint validation)
)

var (
	// Patterns for detecting text context before cursor.
	// Support both YAML (type: ) and JSON ("type": ) syntax.
	typeFieldPattern     = regexp.MustCompile(`("type"|type):\s*$`)
	fieldFieldPattern    = regexp.MustCompile(`("field"|field):\s*$`)
	operatorFieldPattern = regexp.MustCompile(`("operator"|operator):\s*$`)

	// Substitution reference patterns.
	variableRefTextPattern = regexp.MustCompile(`variables\.$`)
	// resources. or resources[
	resourceRefTextPattern = regexp.MustCompile(`resources(\.$|\[$)`)
	// resources.{name}. or resources.{name}[ or resources.{name}.{prop}. or resources.{name}.{prop}[
	resourcePropertyTextPattern                    = regexp.MustCompile(`resources\.([A-Za-z0-9_-]|"|'|\.|\[|\])+\.$`)
	resourcePropertyBracketTextPattern             = regexp.MustCompile(`resources\.([A-Za-z0-9_-]|"|'|\.|\[|\])*\[$`)
	resourcePropertyPartialTextPattern             = regexp.MustCompile(`resources\.[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)+$`)
	resourceWithoutNamespacePropTextPattern        = regexp.MustCompile(`([A-Za-z0-9_-]|"|'|\.|\[|\])+\.$`)
	resourceWithoutNamespacePropBracketTextPattern = regexp.MustCompile(`([A-Za-z0-9_-]|"|'|\.|\[|\])+\[$`)
	// datasources. or datasources[
	dataSourceRefTextPattern = regexp.MustCompile(`datasources(\.$|\[$)`)
	// datasources.{name}. or datasources.{name}.{prop}. or datasources.{name}[
	// Capture group 1 is just the name (up to first . or [)
	dataSourcePropertyTextPattern        = regexp.MustCompile(`datasources\.([A-Za-z0-9_-]+)([A-Za-z0-9_-]|"|'|\.|\[|\])*\.$`)
	dataSourcePropertyBracketTextPattern = regexp.MustCompile(`datasources\.([A-Za-z0-9_-]+)([A-Za-z0-9_-]|"|'|\.|\[|\])*\[$`)
	valueRefTextPattern                  = regexp.MustCompile(`values\.$`)
	childRefTextPattern                  = regexp.MustCompile(`children\.$`)
	elemRefTextPattern                   = regexp.MustCompile(`elem\.$`)
	subOpenTextPattern                   = regexp.MustCompile(`\${$`)
	inSubTextPattern                     = regexp.MustCompile(`\${[^\}]*$`)

	// Pattern for potential standalone resource property (e.g., ${myResource. or ${myResource[)
	// Matches: word characters followed by property access (. or [) that's NOT a reserved namespace
	// Captures: group 1 = the potential resource name
	potentialStandaloneResourcePropPattern = regexp.MustCompile(`\${([A-Za-z][A-Za-z0-9_-]*)([A-Za-z0-9_\-\.\[\]"']+)?[\.\[]$`)

	// Reserved namespace prefixes that are NOT standalone resources
	reservedNamespaces = []string{"resources", "datasources", "variables", "values", "children", "elem"}
)

// CompletionContext provides rich context for completion suggestions.
type CompletionContext struct {
	Kind CompletionContextKind

	// Node context from position resolution
	NodeCtx *NodeContext

	// Extracted names when applicable
	ResourceName          string
	DataSourceName        string
	PotentialResourceName string // For standalone resource detection (needs blueprint validation)

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
	result := determineSubstitutionContext(nodeCtx)
	if result.kind != CompletionContextUnknown {
		ctx.Kind = result.kind
		ctx.ResourceName = result.resourceName
		ctx.DataSourceName = result.dataSourceName
		ctx.PotentialResourceName = result.potentialResourceName
		return ctx
	}

	// Check section definition field contexts (resources, variables, values, etc.)
	if result := determineSectionDefinitionContext(nodeCtx); result.kind != CompletionContextUnknown {
		ctx.Kind = result.kind
		ctx.ResourceName = result.resourceName
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
	// Pattern: /datasources/{name}/{exportName}/type
	// Note: The schema tree doesn't include "exports" in the path
	return len(path) == 4 &&
		path.At(0).FieldName == "datasources" &&
		path.At(3).FieldName == "type"
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
	// Path /datasources/{name}/{exportName} (3 segments) means we're in an export
	// Path /datasources/{name} (2 segments) means we're at the data source level
	if section == "datasources" {
		if len(path) >= 3 {
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
	return len(path) >= 3 &&
		path.At(0).FieldName == "datasources" &&
		path.At(2).FieldName == "filters"
}

// isAtDocumentRootLevel checks if the cursor is at the document root level.
// This is true when the cursor is at column 1 or at minimal indentation (no leading whitespace).
func isAtDocumentRootLevel(nodeCtx *NodeContext) bool {
	// Check if cursor is at column 1 (1-based)
	if nodeCtx.Position.Column == 1 {
		return true
	}

	// Check if the text before cursor on the current line is only whitespace
	// and starts at column 1 (meaning we're typing at root level)
	currentLineText := nodeCtx.TextBefore
	if lastNewline := strings.LastIndex(nodeCtx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = nodeCtx.TextBefore[lastNewline+1:]
	}

	// If there's no indentation (text starts immediately), we're at root level
	trimmed := strings.TrimLeft(currentLineText, " \t")
	leadingWhitespace := len(currentLineText) - len(trimmed)

	// Root level: no leading whitespace, or only typing at the very beginning
	return leadingWhitespace == 0
}

type sectionDefinitionContextResult struct {
	kind         CompletionContextKind
	resourceName string
}

func determineSectionDefinitionContext(nodeCtx *NodeContext) sectionDefinitionContextResult {
	path := nodeCtx.ASTPath

	// Don't trigger for substitution contexts - those are handled separately
	if nodeCtx.InSubstitution() {
		return sectionDefinitionContextResult{kind: CompletionContextUnknown}
	}

	// Only suggest field names when at a key position, not when typing values
	if !nodeCtx.IsAtKeyPosition() {
		return sectionDefinitionContextResult{kind: CompletionContextUnknown}
	}

	// Check for document root level: empty path at root indentation
	// This handles typing new top-level fields in an empty or partially filled document
	if path.IsEmpty() && isAtDocumentRootLevel(nodeCtx) {
		return sectionDefinitionContextResult{kind: CompletionContextBlueprintTopLevelField}
	}

	// Check blueprint top-level: single segment that is a known top-level field
	if path.IsBlueprintTopLevel() {
		return sectionDefinitionContextResult{kind: CompletionContextBlueprintTopLevelField}
	}

	// Note: Blueprint-level metadata (/metadata/...) is a free-form MappingNode
	// in the schema, not a structured type, so we don't suggest specific fields.

	// Check if we're at resource definition level: /resources/{name}
	// This should show core blueprint fields (type, description, spec, metadata, etc.)
	if path.IsResourceDefinition() {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextResourceDefinitionField,
			resourceName: resourceName,
		}
	}

	// Check if we're inside a resource spec path: /resources/{name}/spec/...
	if path.IsResourceSpec() {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextResourceSpecField,
			resourceName: resourceName,
		}
	}

	// Check if we're inside a resource metadata path: /resources/{name}/metadata/...
	if path.IsResourceMetadata() {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextResourceMetadataField,
			resourceName: resourceName,
		}
	}

	// Check variable definition: /variables/{name}
	if path.IsVariableDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextVariableDefinitionField}
	}

	// Check value definition: /values/{name}
	if path.IsValueDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextValueDefinitionField}
	}

	// Check data source definition: /datasources/{name}
	if path.IsDataSourceDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceDefinitionField}
	}

	// Check data source export definition: /datasources/{name}/exports/{exportName}
	if path.IsDataSourceExportDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportDefinitionField}
	}

	// Check data source metadata: /datasources/{name}/metadata/...
	if path.IsDataSourceMetadata() {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceMetadataField}
	}

	// Check data source filter definition: /datasources/{name}/filter/...
	if path.IsDataSourceFilterDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceFilterDefinitionField}
	}

	// Check include definition: /include/{name}
	if path.IsIncludeDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextIncludeDefinitionField}
	}

	// Check export definition: /exports/{name}
	if path.IsExportDefinition() {
		return sectionDefinitionContextResult{kind: CompletionContextExportDefinitionField}
	}

	return sectionDefinitionContextResult{kind: CompletionContextUnknown}
}

type substitutionContextResult struct {
	kind                  CompletionContextKind
	resourceName          string
	dataSourceName        string
	potentialResourceName string
}

func determineSubstitutionContext(nodeCtx *NodeContext) substitutionContextResult {
	textBefore := nodeCtx.TextBefore

	// Check for opening ${ pattern first (immediate trigger for completion)
	if subOpenTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSub}
	}

	// Must be in a substitution context for other checks
	if !nodeCtx.InSubstitution() {
		return substitutionContextResult{kind: CompletionContextUnknown}
	}

	// Check for variable reference: variables.
	if variableRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubVariableRef}
	}

	// Check for data source property reference: datasources.{name}. or datasources.{name}[
	if matches := dataSourcePropertyTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceProperty, dataSourceName: matches[1]}
	}
	if matches := dataSourcePropertyBracketTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceProperty, dataSourceName: matches[1]}
	}

	// Check for data source reference: datasources.
	if dataSourceRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceRef}
	}

	// Check for value reference: values.
	if valueRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubValueRef}
	}

	// Check for child reference: children.
	if childRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubChildRef}
	}

	// Check for elem reference: elem.
	if elemRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubElemRef}
	}

	// Check for resource property reference: resources.{name}.{prop}. or resources.{name}.{prop}[
	if resourcePropertyTextPattern.MatchString(textBefore) ||
		resourcePropertyBracketTextPattern.MatchString(textBefore) {
		resourceName := extractResourceNameFromText(textBefore)
		return substitutionContextResult{kind: CompletionContextStringSubResourceProperty, resourceName: resourceName}
	}

	// Check for partial resource property path (e.g., "resources.myResource.meta")
	// This is when the user is typing a property name without trailing dot - no completions.
	if resourcePropertyPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubPartialPath}
	}

	// Check for resource reference: resources.
	if resourceRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubResourceRef}
	}

	// Check for resource property without namespace (when in resource property context)
	if nodeCtx.SchemaElement != nil {
		if _, isResourceProp := nodeCtx.SchemaElement.(*substitutions.SubstitutionResourceProperty); isResourceProp {
			if resourceWithoutNamespacePropTextPattern.MatchString(textBefore) ||
				resourceWithoutNamespacePropBracketTextPattern.MatchString(textBefore) {
				resourceName := extractResourceNameFromSchemaElement(nodeCtx.SchemaElement)
				return substitutionContextResult{kind: CompletionContextStringSubResourceProperty, resourceName: resourceName}
			}
		}
	}

	// Check for potential standalone resource property (e.g., ${myResource. or ${myResource[)
	// This needs blueprint validation in the completion service
	if potentialName := extractPotentialStandaloneResourceName(textBefore); potentialName != "" {
		return substitutionContextResult{
			kind:                  CompletionContextStringSubPotentialResourceProp,
			potentialResourceName: potentialName,
		}
	}

	// General substitution context (inside ${...})
	if inSubTextPattern.MatchString(textBefore) || nodeCtx.InSubstitution() {
		return substitutionContextResult{kind: CompletionContextStringSub}
	}

	return substitutionContextResult{kind: CompletionContextUnknown}
}

func extractResourceNameFromText(textBefore string) string {
	// Extract resource name from pattern like "resources.myResource." or "resources.myResource["
	idx := strings.Index(textBefore, "resources.")
	if idx == -1 {
		return ""
	}

	after := textBefore[idx+len("resources."):]

	// Find the resource name (up to the next . or [)
	name := after
	if dotIdx := strings.Index(after, "."); dotIdx != -1 {
		name = after[:dotIdx]
	}
	if bracketIdx := strings.Index(name, "["); bracketIdx != -1 {
		name = name[:bracketIdx]
	}

	// Clean up the name (remove quotes if present)
	name = strings.Trim(name, "\"'")
	return name
}

func extractResourceNameFromSchemaElement(elem any) string {
	if resourceProp, ok := elem.(*substitutions.SubstitutionResourceProperty); ok {
		return resourceProp.ResourceName
	}
	return ""
}

// extractPotentialStandaloneResourceName extracts a potential resource name from patterns
// like ${myResource. or ${myResource[ that are NOT reserved namespaces.
// Returns empty string if not a valid pattern or if it's a reserved namespace.
func extractPotentialStandaloneResourceName(textBefore string) string {
	matches := potentialStandaloneResourcePropPattern.FindStringSubmatch(textBefore)
	if len(matches) < 2 {
		return ""
	}

	potentialName := matches[1]

	// Check if this is a reserved namespace
	for _, reserved := range reservedNamespaces {
		if potentialName == reserved {
			return ""
		}
	}

	return potentialName
}

var completionContextKindNames = map[CompletionContextKind]string{
	CompletionContextUnknown:                        "unknown",
	CompletionContextResourceType:                   "resourceType",
	CompletionContextDataSourceType:                 "dataSourceType",
	CompletionContextVariableType:                   "variableType",
	CompletionContextValueType:                      "valueType",
	CompletionContextExportType:                     "exportType",
	CompletionContextDataSourceFieldType:            "dataSourceFieldType",
	CompletionContextDataSourceFilterField:           "dataSourceFilterField",
	CompletionContextDataSourceFilterOperator:        "dataSourceFilterOperator",
	CompletionContextDataSourceFilterDefinitionField: "dataSourceFilterDefinitionField",
	CompletionContextDataSourceExportDefinitionField: "dataSourceExportDefinitionField",
	CompletionContextDataSourceMetadataField:         "dataSourceMetadataField",
	CompletionContextResourceSpecField:               "resourceSpecField",
	CompletionContextResourceMetadataField:          "resourceMetadataField",
	CompletionContextResourceDefinitionField:        "resourceDefinitionField",
	CompletionContextVariableDefinitionField:        "variableDefinitionField",
	CompletionContextValueDefinitionField:           "valueDefinitionField",
	CompletionContextDataSourceDefinitionField:      "dataSourceDefinitionField",
	CompletionContextIncludeDefinitionField:         "includeDefinitionField",
	CompletionContextExportDefinitionField:          "exportDefinitionField",
	CompletionContextBlueprintTopLevelField:         "blueprintTopLevelField",
	CompletionContextStringSub:                      "stringSub",
	CompletionContextStringSubVariableRef:           "stringSubVariableRef",
	CompletionContextStringSubResourceRef:           "stringSubResourceRef",
	CompletionContextStringSubResourceProperty:      "stringSubResourceProperty",
	CompletionContextStringSubDataSourceRef:         "stringSubDataSourceRef",
	CompletionContextStringSubDataSourceProperty:    "stringSubDataSourceProperty",
	CompletionContextStringSubValueRef:              "stringSubValueRef",
	CompletionContextStringSubChildRef:              "stringSubChildRef",
	CompletionContextStringSubElemRef:               "stringSubElemRef",
	CompletionContextStringSubPartialPath:           "stringSubPartialPath",
	CompletionContextStringSubPotentialResourceProp: "stringSubPotentialResourceProp",
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
