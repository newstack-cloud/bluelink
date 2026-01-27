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

	// Data source export value contexts (aliasFor references data source fields)
	CompletionContextDataSourceExportAliasForValue // aliasFor: value field referencing data source spec fields
	CompletionContextDataSourceExportName          // Export name key (suggests data source spec fields)

	// Resource spec field (editing directly in YAML/JSONC definition)
	CompletionContextResourceSpecField
	// Resource spec field value (editing a value for a field with AllowedValues)
	CompletionContextResourceSpecFieldValue

	// Top-level blueprint fields with enum values
	CompletionContextVersionField   // version: field with known spec versions
	CompletionContextTransformField // transform: field with available transforms

	// Custom variable type options (default value for custom type variables)
	CompletionContextCustomVariableTypeValue

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
	aliasForFieldPattern = regexp.MustCompile(`("aliasFor"|aliasFor):\s*$`)
	defaultFieldPattern  = regexp.MustCompile(`("default"|default):\s*$`)

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

	// Pattern to extract field name from TextBefore when at a value position
	// Matches: fieldName: or "fieldName": at the end (with optional trailing space)
	// Captures: group 1 = the field name (without quotes)
	valuePositionFieldPattern = regexp.MustCompile(`(?:"([A-Za-z][A-Za-z0-9_-]*)"|([A-Za-z][A-Za-z0-9_-]*)):\s*$`)
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

	// Check data source export contexts (aliasFor, export name)
	if kind := determineDataSourceExportContext(nodeCtx); kind != CompletionContextUnknown {
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
	if len(path) < 3 {
		return false
	}
	if path.At(0).FieldName != "datasources" {
		return false
	}
	// Support both singular "filter" and plural "filters"
	return path.At(2).FieldName == "filter" || path.At(2).FieldName == "filters"
}

func determineDataSourceExportContext(nodeCtx *NodeContext) CompletionContextKind {
	path := nodeCtx.ASTPath
	textBefore := nodeCtx.TextBefore

	// Check for existing aliasFor field position based on path
	if path.IsDataSourceExportAliasFor() {
		return CompletionContextDataSourceExportAliasForValue
	}

	// Check for text-based aliasFor pattern (handles cursor right after "aliasFor:" or "aliasFor: ")
	if isInDataSourceExports(path) {
		if aliasForFieldPattern.MatchString(textBefore) {
			return CompletionContextDataSourceExportAliasForValue
		}
	}

	// Check for export name key position (at exports level, ready to add new export)
	// This is handled by path-based detection in determineSectionDefinitionContext

	return CompletionContextUnknown
}

func isInDataSourceExports(path StructuredPath) bool {
	if len(path) < 3 {
		return false
	}
	if path.At(0).FieldName != "datasources" {
		return false
	}
	return path.At(2).FieldName == "exports"
}

// Checks if the cursor's indent level matches the export name level.
// This is used to distinguish between adding a new export sibling vs. adding a field inside an export.
//
// When the path is at export definition level (4 segments: /datasources/{name}/exports/{exportName}),
// but the cursor indent is at the same level as the export name, the user wants to add a new export.
//
// Example:
//
//	exports:
//	  vpc:           <- export name at indent level 6 (2 spaces per level * 3)
//	    type: string <- field at indent level 8
//	  |              <- cursor at indent 6 = wants new export sibling
//	    |            <- cursor at indent 8 = wants field inside vpc
func isAtExportNameIndentLevel(nodeCtx *NodeContext) bool {
	if nodeCtx == nil || nodeCtx.UnifiedNode == nil {
		return false
	}

	// Get the current line's leading whitespace
	currentLineText := nodeCtx.TextBefore
	if lastNewline := strings.LastIndex(nodeCtx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = nodeCtx.TextBefore[lastNewline+1:]
	}
	trimmed := strings.TrimLeft(currentLineText, " \t")
	cursorIndent := len(currentLineText) - len(trimmed)

	// The UnifiedNode should be the export definition node (e.g., "vpc")
	// Its start column indicates the export name indent level
	node := nodeCtx.UnifiedNode
	if node.Range.Start == nil {
		return false
	}

	// nodeStartCol is 1-based, convert to 0-based for comparison
	exportNameIndent := node.Range.Start.Column - 1

	// If cursor indent matches the export name indent (not deeper),
	// user wants to add a new export sibling
	return cursorIndent <= exportNameIndent
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

	// Check for value positions first (AllowedValues completions for resource spec fields)
	// This must come BEFORE the key position check
	if nodeCtx.IsAtValuePosition() {
		if result := determineValuePositionContext(nodeCtx, path); result.kind != CompletionContextUnknown {
			return result
		}
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
	// However, if we're at the exports sibling level (same indent as existing exports),
	// this is actually for adding a new export name, not a field inside the export.
	if path.IsDataSourceExportDefinition() {
		if isAtExportNameIndentLevel(nodeCtx) {
			return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportName}
		}
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportDefinitionField}
	}

	// Check data source exports section: /datasources/{name}/exports (at key position to add new export)
	// This suggests field names from the data source spec as potential export names
	if path.IsDataSourceExports() && len(path) == 3 {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportName}
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

// determineValuePositionContext checks for contexts where we're at a value position
// and should provide enum/lookup completions (e.g., AllowedValues for resource spec fields).
func determineValuePositionContext(nodeCtx *NodeContext, path StructuredPath) sectionDefinitionContextResult {
	// Check for data source export aliasFor value: /datasources/{name}/exports/{exportName}/aliasFor
	// This provides completions for available field names from the data source spec
	if path.IsDataSourceExportAliasFor() {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportAliasForValue}
	}

	// Check for resource spec field value: /resources/{name}/spec/...
	// When at value position inside a resource spec, check for AllowedValues
	if path.IsResourceSpec() && len(path) > 3 {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextResourceSpecFieldValue,
			resourceName: resourceName,
		}
	}

	// Fallback: If path is at spec level (len == 3) but we're at a value position,
	// try to extract the field name from TextBefore. This handles the case where
	// the cursor is positioned after "fieldName: " but outside the AST node's range.
	if path.IsResourceSpec() && len(path) == 3 {
		fieldName := extractFieldNameFromTextBefore(nodeCtx.TextBefore)
		if fieldName != "" {
			resourceName, ok := path.GetResourceName()
			if ok {
				// Store the extracted field name in the node context for later use
				nodeCtx.ExtractedFieldName = fieldName
				return sectionDefinitionContextResult{
					kind:         CompletionContextResourceSpecFieldValue,
					resourceName: resourceName,
				}
			}
		}
	}

	// Check for version field value: /version
	if len(path) == 1 && path.At(0).FieldName == "version" {
		return sectionDefinitionContextResult{kind: CompletionContextVersionField}
	}

	// Check for transform field value: /transform
	if len(path) == 1 && path.At(0).FieldName == "transform" {
		return sectionDefinitionContextResult{kind: CompletionContextTransformField}
	}

	// Check for custom variable type default value: /variables/{name}/default
	if len(path) == 3 &&
		path.At(0).FieldName == "variables" &&
		path.At(2).FieldName == "default" {
		return sectionDefinitionContextResult{kind: CompletionContextCustomVariableTypeValue}
	}

	// Fallback: If path is at variable definition level (len == 2) but we're at a value position
	// and TextBefore ends with "default:", this is the custom variable type default value.
	// This handles the case where cursor is after "default:" but no value node exists yet.
	if len(path) == 2 &&
		path.At(0).FieldName == "variables" &&
		defaultFieldPattern.MatchString(nodeCtx.TextBefore) {
		return sectionDefinitionContextResult{kind: CompletionContextCustomVariableTypeValue}
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
	CompletionContextUnknown:                         "unknown",
	CompletionContextResourceType:                    "resourceType",
	CompletionContextDataSourceType:                  "dataSourceType",
	CompletionContextVariableType:                    "variableType",
	CompletionContextValueType:                       "valueType",
	CompletionContextExportType:                      "exportType",
	CompletionContextDataSourceFieldType:             "dataSourceFieldType",
	CompletionContextDataSourceFilterField:           "dataSourceFilterField",
	CompletionContextDataSourceFilterOperator:        "dataSourceFilterOperator",
	CompletionContextDataSourceFilterDefinitionField: "dataSourceFilterDefinitionField",
	CompletionContextDataSourceExportDefinitionField: "dataSourceExportDefinitionField",
	CompletionContextDataSourceMetadataField:         "dataSourceMetadataField",
	CompletionContextDataSourceExportAliasForValue:   "dataSourceExportAliasForValue",
	CompletionContextDataSourceExportName:            "dataSourceExportName",
	CompletionContextResourceSpecField:               "resourceSpecField",
	CompletionContextResourceSpecFieldValue:          "resourceSpecFieldValue",
	CompletionContextVersionField:                    "versionField",
	CompletionContextTransformField:                  "transformField",
	CompletionContextCustomVariableTypeValue:         "customVariableTypeValue",
	CompletionContextResourceMetadataField:           "resourceMetadataField",
	CompletionContextResourceDefinitionField:         "resourceDefinitionField",
	CompletionContextVariableDefinitionField:         "variableDefinitionField",
	CompletionContextValueDefinitionField:            "valueDefinitionField",
	CompletionContextDataSourceDefinitionField:       "dataSourceDefinitionField",
	CompletionContextIncludeDefinitionField:          "includeDefinitionField",
	CompletionContextExportDefinitionField:           "exportDefinitionField",
	CompletionContextBlueprintTopLevelField:          "blueprintTopLevelField",
	CompletionContextStringSub:                       "stringSub",
	CompletionContextStringSubVariableRef:            "stringSubVariableRef",
	CompletionContextStringSubResourceRef:            "stringSubResourceRef",
	CompletionContextStringSubResourceProperty:       "stringSubResourceProperty",
	CompletionContextStringSubDataSourceRef:          "stringSubDataSourceRef",
	CompletionContextStringSubDataSourceProperty:     "stringSubDataSourceProperty",
	CompletionContextStringSubValueRef:               "stringSubValueRef",
	CompletionContextStringSubChildRef:               "stringSubChildRef",
	CompletionContextStringSubElemRef:                "stringSubElemRef",
	CompletionContextStringSubPartialPath:            "stringSubPartialPath",
	CompletionContextStringSubPotentialResourceProp:  "stringSubPotentialResourceProp",
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

// Extracts a field name from TextBefore when at a value position.
// This handles cases like "architecture: " or "\"architecture\": " where the cursor is after the colon.
// Returns the field name without quotes, or empty string if no match.
func extractFieldNameFromTextBefore(textBefore string) string {
	matches := valuePositionFieldPattern.FindStringSubmatch(textBefore)
	if matches == nil {
		return ""
	}
	// matches[1] is the quoted field name (group 1), matches[2] is the unquoted field name (group 2)
	if matches[1] != "" {
		return matches[1]
	}
	return matches[2]
}
