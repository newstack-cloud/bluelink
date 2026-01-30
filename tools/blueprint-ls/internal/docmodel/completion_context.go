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

	// Resource annotation key (inside metadata.annotations, suggests link annotation keys)
	CompletionContextResourceAnnotationKey

	// Resource annotation value (value for a resource annotation with AllowedValues)
	CompletionContextResourceAnnotationValue

	// Resource definition field (type, description, spec, metadata, etc.)
	CompletionContextResourceDefinitionField

	// Resource linkSelector field (byLabel, exclude)
	CompletionContextLinkSelectorField

	// Resource linkSelector exclude value (resource names)
	CompletionContextLinkSelectorExcludeValue

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
	CompletionContextStringSubChildProperty         // After children.{name}., suggest child exports
	CompletionContextStringSubValueProperty          // After values.{name}., suggest value fields/indices
	CompletionContextStringSubPartialPath            // Typing a partial path (no completions)
	CompletionContextStringSubPotentialResourceProp  // Potential standalone resource property (needs blueprint validation)

	// Data source annotation contexts
	CompletionContextDataSourceAnnotationKey   // Inside data source metadata.annotations, suggest annotation keys
	CompletionContextDataSourceAnnotationValue // Value for a data source annotation

	// Resource label key context
	CompletionContextResourceLabelKey // Inside metadata.labels, suggest label keys from linked resources

	// Export field reference contexts (field property in blueprint exports)
	CompletionContextExportFieldTopLevel           // Empty field value, suggest namespaces
	CompletionContextExportFieldResourceRef        // After "resources.", suggest resource names
	CompletionContextExportFieldResourceProperty   // After "resources.{name}.", suggest spec/metadata/fields
	CompletionContextExportFieldVariableRef        // After "variables.", suggest variable names
	CompletionContextExportFieldValueRef           // After "values.", suggest value names
	CompletionContextExportFieldValueProperty      // After "values.{name}.", suggest value fields/indices
	CompletionContextExportFieldChildRef           // After "children.", suggest child names
	CompletionContextExportFieldChildProperty      // After "children.{name}.", suggest exports
	CompletionContextExportFieldDataSourceRef      // After "datasources.", suggest data source names
	CompletionContextExportFieldDataSourceProperty // After "datasources.{name}.", suggest exports
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
	// datasources.{name}.{partial} - typing a partial export name after a data source name
	dataSourcePropertyPartialTextPattern = regexp.MustCompile(`datasources\.([A-Za-z0-9_-]+)\.[A-Za-z0-9_-]+$`)
	// values. or values.{name}. or values.{name}.{partial}
	valueRefTextPattern             = regexp.MustCompile(`values\.$`)
	valuePropertyTextPattern        = regexp.MustCompile(`values\.([A-Za-z0-9_-]+)\.$`)
	valuePropertyPartialTextPattern = regexp.MustCompile(`values\.([A-Za-z0-9_-]+)\.[A-Za-z0-9_-]+$`)
	// children. or children.{name}. or children.{name}.{partial}
	childRefTextPattern              = regexp.MustCompile(`children\.$`)
	childPropertyTextPattern         = regexp.MustCompile(`children\.([A-Za-z0-9_-]+)\.$`)
	childPropertyPartialTextPattern  = regexp.MustCompile(`children\.([A-Za-z0-9_-]+)\.[A-Za-z0-9_-]+$`)
	// elem. or elem.{partial}
	elemRefTextPattern        = regexp.MustCompile(`elem\.$`)
	elemRefPartialTextPattern = regexp.MustCompile(`elem\.[A-Za-z][A-Za-z0-9_-]*$`)
	subOpenTextPattern = regexp.MustCompile(`\${$`)
	inSubTextPattern   = regexp.MustCompile(`\${[^\}]*$`)

	// Partial segment patterns - match when typing a name after a namespace separator
	// e.g., ${resources.ordersT or ${variables.env (no trailing dot)
	variableRefPartialTextPattern   = regexp.MustCompile(`variables\.[A-Za-z][A-Za-z0-9_-]*$`)
	resourceRefPartialTextPattern   = regexp.MustCompile(`resources\.[A-Za-z][A-Za-z0-9_-]*$`)
	dataSourceRefPartialTextPattern = regexp.MustCompile(`datasources\.[A-Za-z][A-Za-z0-9_-]*$`)
	valueRefPartialTextPattern      = regexp.MustCompile(`values\.[A-Za-z][A-Za-z0-9_-]*$`)
	childRefPartialTextPattern      = regexp.MustCompile(`children\.[A-Za-z][A-Za-z0-9_-]*$`)

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

	// Pattern to extract annotation key from TextBefore when at a value position
	// Annotation keys can contain dots (e.g., aws.lambda.dynamodb.accessType)
	// Matches: annotationKey: or "annotationKey": at the end (with optional trailing space)
	// Captures: group 1 = the quoted annotation key, group 2 = the unquoted annotation key
	annotationKeyFieldPattern = regexp.MustCompile(`(?:"([A-Za-z][A-Za-z0-9_.\-<>]*)"|([A-Za-z][A-Za-z0-9_.\-<>]*)):\s*$`)

	// Export field reference patterns (no ${} wrapping, used in blueprint exports field property)
	// These patterns match paths at any depth, capturing the element name for context.
	// Supports both dot notation (.field) and bracket notation ([0], ["key"], ['key'])
	// pathSegment matches: .fieldName or [index] or ["key"] or ['key']
	// Pattern: resources.name followed by any number of path segments, ending with .
	exportFieldResourcesPattern      = regexp.MustCompile(`resources\.$`)
	exportFieldResourcePropPattern   = regexp.MustCompile(`resources\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\.$`)
	exportFieldVariablesPattern      = regexp.MustCompile(`variables\.$`)
	exportFieldValuesPattern         = regexp.MustCompile(`values\.$`)
	exportFieldChildrenPattern       = regexp.MustCompile(`children\.$`)
	exportFieldChildPropPattern      = regexp.MustCompile(`children\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\.$`)
	exportFieldValuePropPattern      = regexp.MustCompile(`values\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\.$`)
	exportFieldDatasourcesPattern    = regexp.MustCompile(`datasources\.$`)
	exportFieldDatasourcePropPattern = regexp.MustCompile(`datasources\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\.$`)

	// Export field bracket patterns - match paths ending with "[" for array/map access
	exportFieldResourcePropBracketPattern   = regexp.MustCompile(`resources\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\[$`)
	exportFieldValuePropBracketPattern      = regexp.MustCompile(`values\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\[$`)
	exportFieldChildPropBracketPattern      = regexp.MustCompile(`children\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\[$`)
	exportFieldDatasourcePropBracketPattern = regexp.MustCompile(`datasources\.([A-Za-z0-9_-]+)((\.[A-Za-z0-9_-]+)|(\[[0-9]+\])|(\["[^"]*"\])|(\['[^']*'\]))*\[$`)
)

// CompletionContext provides rich context for completion suggestions.
type CompletionContext struct {
	Kind CompletionContextKind

	// Cursor context from position resolution (unified structural + syntactic context)
	CursorCtx *CursorContext

	// Extracted names when applicable
	ResourceName          string
	DataSourceName        string
	ChildName             string
	PotentialResourceName string // For standalone resource detection (needs blueprint validation)

	// Text analysis results
	TextBefore string
	IsInSub    bool
}

// DetermineCompletionContext analyzes the cursor context and text to determine
// what kind of completion should be provided.
func DetermineCompletionContext(cursorCtx *CursorContext) *CompletionContext {
	ctx := &CompletionContext{
		Kind:       CompletionContextUnknown,
		CursorCtx:  cursorCtx,
		TextBefore: cursorCtx.TextBefore,
		IsInSub:    cursorCtx.InSubstitution(),
	}

	// Check path-based type fields first (most specific)
	if kind := determineTypeFieldContext(cursorCtx); kind != CompletionContextUnknown {
		ctx.Kind = kind
		return ctx
	}

	// Check data source filter contexts
	if kind := determineDataSourceFilterContext(cursorCtx); kind != CompletionContextUnknown {
		ctx.Kind = kind
		return ctx
	}

	// Check data source export contexts (aliasFor, export name)
	if kind := determineDataSourceExportContext(cursorCtx); kind != CompletionContextUnknown {
		ctx.Kind = kind
		return ctx
	}

	// Check export field contexts (blueprint exports field property, before substitution check)
	if result := determineExportFieldContext(cursorCtx); result.kind != CompletionContextUnknown {
		ctx.Kind = result.kind
		ctx.ResourceName = result.resourceName
		ctx.DataSourceName = result.dataSourceName
		ctx.ChildName = result.childName
		return ctx
	}

	// Check substitution contexts
	result := determineSubstitutionContext(cursorCtx)
	if result.kind != CompletionContextUnknown {
		ctx.Kind = result.kind
		ctx.ResourceName = result.resourceName
		ctx.DataSourceName = result.dataSourceName
		ctx.ChildName = result.childName
		ctx.PotentialResourceName = result.potentialResourceName
		return ctx
	}

	// Check section definition field contexts (resources, variables, values, etc.)
	if result := determineSectionDefinitionContext(cursorCtx); result.kind != CompletionContextUnknown {
		ctx.Kind = result.kind
		ctx.ResourceName = result.resourceName
		ctx.DataSourceName = result.dataSourceName
		return ctx
	}

	return ctx
}

func determineTypeFieldContext(cursorCtx *CursorContext) CompletionContextKind {
	path := cursorCtx.StructuralPath
	textBefore := cursorCtx.TextBefore

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

func determineDataSourceFilterContext(cursorCtx *CursorContext) CompletionContextKind {
	path := cursorCtx.StructuralPath
	textBefore := cursorCtx.TextBefore

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

func determineDataSourceExportContext(cursorCtx *CursorContext) CompletionContextKind {
	path := cursorCtx.StructuralPath
	textBefore := cursorCtx.TextBefore

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
func isAtExportNameIndentLevel(cursorCtx *CursorContext) bool {
	if cursorCtx == nil || cursorCtx.UnifiedNode == nil {
		return false
	}

	// Get the current line's leading whitespace
	currentLineText := cursorCtx.TextBefore
	if lastNewline := strings.LastIndex(cursorCtx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = cursorCtx.TextBefore[lastNewline+1:]
	}
	trimmed := strings.TrimLeft(currentLineText, " \t")
	cursorIndent := len(currentLineText) - len(trimmed)

	// The UnifiedNode should be the export definition node (e.g., "vpc")
	// Its start column indicates the export name indent level
	node := cursorCtx.UnifiedNode
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
func isAtDocumentRootLevel(cursorCtx *CursorContext) bool {
	// Check if cursor is at column 1 (1-based)
	if cursorCtx.Position.Column == 1 {
		return true
	}

	// Check if the text before cursor on the current line is only whitespace
	// and starts at column 1 (meaning we're typing at root level)
	currentLineText := cursorCtx.TextBefore
	if lastNewline := strings.LastIndex(cursorCtx.TextBefore, "\n"); lastNewline >= 0 {
		currentLineText = cursorCtx.TextBefore[lastNewline+1:]
	}

	// If there's no indentation (text starts immediately), we're at root level
	trimmed := strings.TrimLeft(currentLineText, " \t")
	leadingWhitespace := len(currentLineText) - len(trimmed)

	// Root level: no leading whitespace, or only typing at the very beginning
	return leadingWhitespace == 0
}

type sectionDefinitionContextResult struct {
	kind           CompletionContextKind
	resourceName   string
	dataSourceName string
}

func determineSectionDefinitionContext(cursorCtx *CursorContext) sectionDefinitionContextResult {
	path := cursorCtx.StructuralPath

	// Don't trigger for substitution contexts - those are handled separately
	if cursorCtx.InSubstitution() {
		return sectionDefinitionContextResult{kind: CompletionContextUnknown}
	}

	// Check for array value contexts first - these don't follow the key/value pattern
	// and need to be handled before the key position check.
	// Examples: linkSelector.exclude values, transform list values, etc.
	if result := determineArrayValueContext(cursorCtx, path); result.kind != CompletionContextUnknown {
		return result
	}

	// Check for value positions first (AllowedValues completions for resource spec fields)
	// This must come BEFORE the key position check
	if cursorCtx.IsAtValuePosition() {
		if result := determineValuePositionContext(cursorCtx, path); result.kind != CompletionContextUnknown {
			return result
		}
	}

	// Only suggest field names when at a key position, not when typing values
	if !cursorCtx.IsAtKeyPosition() {
		return sectionDefinitionContextResult{kind: CompletionContextUnknown}
	}

	// Check for document root level: empty path at root indentation
	// This handles typing new top-level fields in an empty or partially filled document
	if path.IsEmpty() && isAtDocumentRootLevel(cursorCtx) {
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

	// Check if we're inside resource metadata annotations: /resources/{name}/metadata/annotations
	// This must come before the general IsResourceMetadata() check (more specific path first)
	if path.IsResourceMetadataAnnotations() {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextResourceAnnotationKey,
			resourceName: resourceName,
		}
	}

	// Check if we're inside resource metadata labels: /resources/{name}/metadata/labels
	// Must come before the general IsResourceMetadata() check.
	if path.IsResourceMetadataLabels() {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextResourceLabelKey,
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

	// Note: linkSelector.exclude is handled by determineArrayValueContext before this function.
	// Check if we're inside resource linkSelector: /resources/{name}/linkSelector
	if path.IsResourceLinkSelector() {
		resourceName, ok := path.GetResourceName()
		if !ok {
			return sectionDefinitionContextResult{kind: CompletionContextUnknown}
		}
		return sectionDefinitionContextResult{
			kind:         CompletionContextLinkSelectorField,
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
		if isAtExportNameIndentLevel(cursorCtx) {
			return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportName}
		}
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportDefinitionField}
	}

	// Check data source exports section: /datasources/{name}/exports (at key position to add new export)
	// This suggests field names from the data source spec as potential export names
	if path.IsDataSourceExports() && len(path) == 3 {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportName}
	}

	// Check data source metadata annotations: /datasources/{name}/metadata/annotations
	// Must come before the general IsDataSourceMetadata() check.
	if path.IsDataSourceMetadataAnnotations() {
		dsName, _ := path.GetDataSourceName()
		return sectionDefinitionContextResult{
			kind:           CompletionContextDataSourceAnnotationKey,
			dataSourceName: dsName,
		}
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

// determineArrayValueContext checks for contexts where we're inside an array/sequence
// and should provide value completions. These contexts don't follow the key/value pattern
// and need to be handled separately from key and value position checks.
// Examples: linkSelector.exclude values, transform list values, etc.
func determineArrayValueContext(_ *CursorContext, path StructuredPath) sectionDefinitionContextResult {
	// Check for linkSelector exclude value: /resources/{name}/linkSelector/exclude
	// This provides completions for resource names to exclude from link matching.
	// Works for both:
	// - YAML: "- " sequence items
	// - JSONC: values inside [""] arrays
	if path.IsResourceLinkSelectorExclude() {
		resourceName, ok := path.GetResourceName()
		if ok {
			return sectionDefinitionContextResult{
				kind:         CompletionContextLinkSelectorExcludeValue,
				resourceName: resourceName,
			}
		}
	}

	return sectionDefinitionContextResult{kind: CompletionContextUnknown}
}

// determineValuePositionContext checks for contexts where we're at a value position
// and should provide enum/lookup completions (e.g., AllowedValues for resource spec fields).
func determineValuePositionContext(cursorCtx *CursorContext, path StructuredPath) sectionDefinitionContextResult {
	// Check for data source export aliasFor value: /datasources/{name}/exports/{exportName}/aliasFor
	// This provides completions for available field names from the data source spec
	if path.IsDataSourceExportAliasFor() {
		return sectionDefinitionContextResult{kind: CompletionContextDataSourceExportAliasForValue}
	}

	// Check for resource annotation value: /resources/{name}/metadata/annotations/{key}
	// When at value position inside an annotation, suggest AllowedValues from link definitions
	if path.IsResourceMetadataAnnotationValue() {
		resourceName, ok := path.GetResourceName()
		if ok {
			return sectionDefinitionContextResult{
				kind:         CompletionContextResourceAnnotationValue,
				resourceName: resourceName,
			}
		}
	}

	// Fallback: If path is at annotations level (len == 4) but we're at a value position,
	// try to extract the annotation key from TextBefore. This handles the case where
	// the cursor is positioned after "annotationKey: " but outside the AST node's range.
	// Use the annotation-specific pattern since annotation keys can contain dots.
	if path.IsResourceMetadataAnnotations() && len(path) == 4 {
		annotationKey := extractAnnotationKeyFromTextBefore(cursorCtx.TextBefore)
		if annotationKey != "" {
			resourceName, ok := path.GetResourceName()
			if ok {
				// Store the extracted field name in the cursor context for later use
				cursorCtx.ExtractedFieldName = annotationKey
				return sectionDefinitionContextResult{
					kind:         CompletionContextResourceAnnotationValue,
					resourceName: resourceName,
				}
			}
		}
	}

	// Check for data source annotation value: /datasources/{name}/metadata/annotations/{key}
	if path.IsDataSourceMetadataAnnotationValue() {
		dsName, _ := path.GetDataSourceName()
		return sectionDefinitionContextResult{
			kind:           CompletionContextDataSourceAnnotationValue,
			dataSourceName: dsName,
		}
	}

	// Fallback: If path is at data source annotations level (len == 4) but at value position,
	// try to extract the annotation key from TextBefore.
	if path.IsDataSourceMetadataAnnotations() && len(path) == 4 {
		annotationKey := extractAnnotationKeyFromTextBefore(cursorCtx.TextBefore)
		if annotationKey != "" {
			cursorCtx.ExtractedFieldName = annotationKey
			dsName, _ := path.GetDataSourceName()
			return sectionDefinitionContextResult{
				kind:           CompletionContextDataSourceAnnotationValue,
				dataSourceName: dsName,
			}
		}
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
		fieldName := extractFieldNameFromTextBefore(cursorCtx.TextBefore)
		if fieldName != "" {
			resourceName, ok := path.GetResourceName()
			if ok {
				// Store the extracted field name in the cursor context for later use
				cursorCtx.ExtractedFieldName = fieldName
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
		defaultFieldPattern.MatchString(cursorCtx.TextBefore) {
		return sectionDefinitionContextResult{kind: CompletionContextCustomVariableTypeValue}
	}

	return sectionDefinitionContextResult{kind: CompletionContextUnknown}
}

// determineExportFieldContext checks if the cursor is in an export field value position
// and returns the appropriate completion context based on what has been typed.
func determineExportFieldContext(cursorCtx *CursorContext) substitutionContextResult {
	if !isInExportFieldPosition(cursorCtx) {
		return substitutionContextResult{kind: CompletionContextUnknown}
	}

	fieldValue := extractExportFieldValue(cursorCtx.TextBefore)
	if fieldValue == "" {
		return substitutionContextResult{kind: CompletionContextExportFieldTopLevel}
	}

	// Try to match a specific pattern (paths ending in dot)
	result := matchExportFieldPattern(fieldValue)
	if result.kind != CompletionContextUnknown {
		return result
	}

	// If no pattern matched and there's no dot in the value, the user is typing
	// a partial top-level namespace (e.g., "res" for "resources")
	if !strings.Contains(fieldValue, ".") {
		return substitutionContextResult{kind: CompletionContextExportFieldTopLevel}
	}

	// Value contains a dot but doesn't match any pattern - user is mid-typing a segment
	// Return unknown to avoid showing irrelevant completions
	return substitutionContextResult{kind: CompletionContextUnknown}
}

// isInExportFieldPosition checks if cursor is at an export field value position.
func isInExportFieldPosition(cursorCtx *CursorContext) bool {
	path := cursorCtx.StructuralPath

	// Check if path explicitly indicates export field position
	if path.IsExportField() {
		return true
	}

	// Check if we're in exports section and text contains field: pattern.
	// We check for containment (not just ending with) because the user may have typed
	// part of the field value (e.g., "field: resources.")
	if path.IsInExports() && containsExportFieldPattern(cursorCtx.TextBefore) {
		return true
	}

	// Check for export definition level with field: pattern in text
	if path.IsExportDefinition() && containsExportFieldPattern(cursorCtx.TextBefore) {
		return true
	}

	return false
}

// containsExportFieldPattern checks if text contains the field: pattern for export fields.
// This matches either YAML (field:) or JSONC ("field":) syntax.
func containsExportFieldPattern(text string) bool {
	return strings.Contains(text, "field:") || strings.Contains(text, "\"field\":")
}

// extractExportFieldValue extracts the value portion after "field:" from TextBefore.
func extractExportFieldValue(textBefore string) string {
	// Try YAML pattern first: field:
	idx := strings.LastIndex(textBefore, "field:")
	if idx != -1 {
		return strings.TrimSpace(textBefore[idx+len("field:"):])
	}

	// Try JSONC pattern: "field":
	idx = strings.LastIndex(textBefore, "\"field\":")
	if idx != -1 {
		value := strings.TrimSpace(textBefore[idx+len("\"field\":"):])
		// Remove leading quote if present
		value = strings.TrimPrefix(value, "\"")
		return value
	}

	return ""
}

// matchExportFieldPattern determines the context kind based on the typed field value.
// Handles both complete paths ending with "." or "[" and partial segments being typed.
func matchExportFieldPattern(fieldValue string) substitutionContextResult {
	// First try matching exact patterns (paths ending with . or [)
	if result := matchExactExportFieldPattern(fieldValue); result.kind != CompletionContextUnknown {
		return result
	}

	// If no exact match and value contains a dot, try matching with a trailing dot appended
	// This handles partial segment typing (e.g., "resources.myResource.sp" → match as "resources.myResource.sp.")
	if strings.Contains(fieldValue, ".") && !strings.HasSuffix(fieldValue, ".") {
		return matchExactExportFieldPattern(fieldValue + ".")
	}

	return substitutionContextResult{kind: CompletionContextUnknown}
}

// matchExactExportFieldPattern matches field values ending with "." or "[".
func matchExactExportFieldPattern(fieldValue string) substitutionContextResult {
	// Check for bracket patterns first (more specific)
	if strings.HasSuffix(fieldValue, "[") {
		return matchExportFieldBracketPattern(fieldValue)
	}

	// Check for data source property: datasources.{name}.
	if matches := exportFieldDatasourcePropPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{
			kind:           CompletionContextExportFieldDataSourceProperty,
			dataSourceName: matches[1],
		}
	}

	// Check for data source reference: datasources.
	if exportFieldDatasourcesPattern.MatchString(fieldValue) {
		return substitutionContextResult{kind: CompletionContextExportFieldDataSourceRef}
	}

	// Check for resource property: resources.{name}.
	if matches := exportFieldResourcePropPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{
			kind:         CompletionContextExportFieldResourceProperty,
			resourceName: matches[1],
		}
	}

	// Check for resource reference: resources.
	if exportFieldResourcesPattern.MatchString(fieldValue) {
		return substitutionContextResult{kind: CompletionContextExportFieldResourceRef}
	}

	// Check for child property: children.{name}.
	if matches := exportFieldChildPropPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextExportFieldChildProperty, childName: matches[1]}
	}

	// Check for child reference: children.
	if exportFieldChildrenPattern.MatchString(fieldValue) {
		return substitutionContextResult{kind: CompletionContextExportFieldChildRef}
	}

	// Check for value property: values.{name}.
	if matches := exportFieldValuePropPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextExportFieldValueProperty}
	}

	// Check for variable reference: variables.
	if exportFieldVariablesPattern.MatchString(fieldValue) {
		return substitutionContextResult{kind: CompletionContextExportFieldVariableRef}
	}

	// Check for value reference: values.
	if exportFieldValuesPattern.MatchString(fieldValue) {
		return substitutionContextResult{kind: CompletionContextExportFieldValueRef}
	}

	return substitutionContextResult{kind: CompletionContextUnknown}
}

// matchExportFieldBracketPattern matches field values ending with "[" for array/map access.
func matchExportFieldBracketPattern(fieldValue string) substitutionContextResult {
	// Check for resource property bracket: resources.{name}...[
	if matches := exportFieldResourcePropBracketPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{
			kind:         CompletionContextExportFieldResourceProperty,
			resourceName: matches[1],
		}
	}

	// Check for value property bracket: values.{name}...[
	if matches := exportFieldValuePropBracketPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextExportFieldValueProperty}
	}

	// Check for child property bracket: children.{name}...[
	if matches := exportFieldChildPropBracketPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextExportFieldChildProperty, childName: matches[1]}
	}

	// Check for data source property bracket: datasources.{name}...[
	if matches := exportFieldDatasourcePropBracketPattern.FindStringSubmatch(fieldValue); len(matches) >= 2 {
		return substitutionContextResult{
			kind:           CompletionContextExportFieldDataSourceProperty,
			dataSourceName: matches[1],
		}
	}

	return substitutionContextResult{kind: CompletionContextUnknown}
}

type substitutionContextResult struct {
	kind                  CompletionContextKind
	resourceName          string
	dataSourceName        string
	childName             string
	potentialResourceName string
}

func determineSubstitutionContext(cursorCtx *CursorContext) substitutionContextResult {
	textBefore := cursorCtx.TextBefore

	// Check for opening ${ pattern first (immediate trigger for completion)
	if subOpenTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSub}
	}

	// Must be in a substitution context for other checks
	if !cursorCtx.InSubstitution() {
		return substitutionContextResult{kind: CompletionContextUnknown}
	}

	// Check for variable reference: variables. or variables.partialName
	if variableRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubVariableRef}
	}
	if variableRefPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubVariableRef}
	}

	// Check for data source property reference: datasources.{name}. or datasources.{name}[
	if matches := dataSourcePropertyTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceProperty, dataSourceName: matches[1]}
	}
	if matches := dataSourcePropertyBracketTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceProperty, dataSourceName: matches[1]}
	}
	// Check for partial data source property: datasources.{name}.{partial}
	if matches := dataSourcePropertyPartialTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceProperty, dataSourceName: matches[1]}
	}

	// Check for data source reference: datasources. or datasources.partialName
	if dataSourceRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceRef}
	}
	if dataSourceRefPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubDataSourceRef}
	}

	// Check for value property reference: values.{name}. or values.{name}.{partial}
	// Must be checked before value ref to avoid false matches on deeper paths.
	if valuePropertyTextPattern.MatchString(textBefore) ||
		valuePropertyPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubValueProperty}
	}

	// Check for value reference: values. or values.partialName
	if valueRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubValueRef}
	}
	if valueRefPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubValueRef}
	}

	// Check for child property reference: children.{name}. or children.{name}.{partial}
	// Must be checked before child ref to avoid false matches on deeper paths.
	if matches := childPropertyTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubChildProperty, childName: matches[1]}
	}
	if matches := childPropertyPartialTextPattern.FindStringSubmatch(textBefore); len(matches) >= 2 {
		return substitutionContextResult{kind: CompletionContextStringSubChildProperty, childName: matches[1]}
	}

	// Check for child reference: children. or children.partialName
	if childRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubChildRef}
	}
	if childRefPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubChildRef}
	}

	// Check for elem reference: elem. or elem.partialProperty
	if elemRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubElemRef}
	}
	if elemRefPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubElemRef}
	}

	// Check for resource property reference: resources.{name}.{prop}. or resources.{name}.{prop}[
	if resourcePropertyTextPattern.MatchString(textBefore) ||
		resourcePropertyBracketTextPattern.MatchString(textBefore) {
		resourceName := extractResourceNameFromText(textBefore)
		return substitutionContextResult{kind: CompletionContextStringSubResourceProperty, resourceName: resourceName}
	}

	// Check for partial resource property path (e.g., "resources.myResource.sp")
	// This is when the user is typing a property name without trailing dot.
	// Return ResourceProperty context and let prefix filtering handle showing matching completions.
	if resourcePropertyPartialTextPattern.MatchString(textBefore) {
		resourceName := extractResourceNameFromText(textBefore)
		return substitutionContextResult{kind: CompletionContextStringSubResourceProperty, resourceName: resourceName}
	}

	// Check for resource reference: resources. or resources.partialName
	if resourceRefTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubResourceRef}
	}
	if resourceRefPartialTextPattern.MatchString(textBefore) {
		return substitutionContextResult{kind: CompletionContextStringSubResourceRef}
	}

	// Check for resource property without namespace (when in resource property context)
	if cursorCtx.SchemaElement != nil {
		if _, isResourceProp := cursorCtx.SchemaElement.(*substitutions.SubstitutionResourceProperty); isResourceProp {
			if resourceWithoutNamespacePropTextPattern.MatchString(textBefore) ||
				resourceWithoutNamespacePropBracketTextPattern.MatchString(textBefore) {
				resourceName := extractResourceNameFromSchemaElement(cursorCtx.SchemaElement)
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
	if inSubTextPattern.MatchString(textBefore) || cursorCtx.InSubstitution() {
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
	CompletionContextResourceMetadataField:            "resourceMetadataField",
	CompletionContextResourceAnnotationKey:            "resourceAnnotationKey",
	CompletionContextResourceAnnotationValue:          "resourceAnnotationValue",
	CompletionContextResourceDefinitionField:          "resourceDefinitionField",
	CompletionContextLinkSelectorField:               "linkSelectorField",
	CompletionContextLinkSelectorExcludeValue:        "linkSelectorExcludeValue",
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
	CompletionContextStringSubChildProperty:          "stringSubChildProperty",
	CompletionContextStringSubValueProperty:          "stringSubValueProperty",
	CompletionContextStringSubPartialPath:             "stringSubPartialPath",
	CompletionContextStringSubPotentialResourceProp:  "stringSubPotentialResourceProp",
	CompletionContextDataSourceAnnotationKey:         "dataSourceAnnotationKey",
	CompletionContextDataSourceAnnotationValue:       "dataSourceAnnotationValue",
	CompletionContextResourceLabelKey:                "resourceLabelKey",
	CompletionContextExportFieldTopLevel:             "exportFieldTopLevel",
	CompletionContextExportFieldResourceRef:          "exportFieldResourceRef",
	CompletionContextExportFieldResourceProperty:     "exportFieldResourceProperty",
	CompletionContextExportFieldVariableRef:          "exportFieldVariableRef",
	CompletionContextExportFieldValueRef:             "exportFieldValueRef",
	CompletionContextExportFieldChildRef:             "exportFieldChildRef",
	CompletionContextExportFieldChildProperty:        "exportFieldChildProperty",
	CompletionContextExportFieldValueProperty:        "exportFieldValueProperty",
	CompletionContextExportFieldDataSourceRef:        "exportFieldDataSourceRef",
	CompletionContextExportFieldDataSourceProperty:   "exportFieldDataSourceProperty",
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
		CompletionContextStringSubValueProperty,
		CompletionContextStringSubChildRef,
		CompletionContextStringSubChildProperty,
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

// IsExportFieldContext returns true if this context is for an export field reference.
func (k CompletionContextKind) IsExportFieldContext() bool {
	switch k {
	case CompletionContextExportFieldTopLevel,
		CompletionContextExportFieldResourceRef,
		CompletionContextExportFieldResourceProperty,
		CompletionContextExportFieldVariableRef,
		CompletionContextExportFieldValueRef,
		CompletionContextExportFieldValueProperty,
		CompletionContextExportFieldChildRef,
		CompletionContextExportFieldChildProperty,
		CompletionContextExportFieldDataSourceRef,
		CompletionContextExportFieldDataSourceProperty:
		return true
	}
	return false
}

// CompletionCapability represents the type of completion being provided.
// This is used to determine format compatibility.
type CompletionCapability int

const (
	// CapabilityKeyCompletion suggests field/key names (e.g., spec field names).
	// These are typically disabled for JSONC because editors handle JSON property completion.
	CapabilityKeyCompletion CompletionCapability = iota

	// CapabilityValueCompletion suggests values for a field (e.g., types, operators, references).
	// These work for both YAML and JSONC formats.
	CapabilityValueCompletion
)

// GetCompletionCapability returns the capability type for this completion context.
// Key completions suggest field names, value completions suggest field values.
func (k CompletionContextKind) GetCompletionCapability() CompletionCapability {
	switch k {
	// Key completions - suggest field/property names
	case CompletionContextResourceSpecField,
		CompletionContextResourceMetadataField,
		CompletionContextResourceAnnotationKey,
		CompletionContextResourceLabelKey,
		CompletionContextResourceDefinitionField,
		CompletionContextVariableDefinitionField,
		CompletionContextValueDefinitionField,
		CompletionContextDataSourceDefinitionField,
		CompletionContextDataSourceFilterDefinitionField,
		CompletionContextDataSourceExportDefinitionField,
		CompletionContextDataSourceMetadataField,
		CompletionContextDataSourceAnnotationKey,
		CompletionContextIncludeDefinitionField,
		CompletionContextExportDefinitionField,
		CompletionContextLinkSelectorField,
		CompletionContextBlueprintTopLevelField,
		CompletionContextDataSourceExportName:
		return CapabilityKeyCompletion

	// Value completions - suggest values for fields
	default:
		return CapabilityValueCompletion
	}
}

// IsEnabledForFormat returns true if this completion context is enabled for the given format.
// Key completions are disabled for JSONC because JSON editors typically handle property completion.
// Value completions work for both formats.
func (k CompletionContextKind) IsEnabledForFormat(format DocumentFormat) bool {
	capability := k.GetCompletionCapability()

	switch format {
	case FormatJSONC:
		// JSONC: Disable key completions (editor handles JSON properties)
		// Enable value completions
		return capability == CapabilityValueCompletion
	case FormatYAML:
		// YAML: Enable all completions
		return true
	default:
		return true
	}
}

// IsSubstitutionContext returns true if this context is for completions inside ${...} substitutions.
func (k CompletionContextKind) IsSubstitutionContext() bool {
	switch k {
	case CompletionContextStringSub,
		CompletionContextStringSubVariableRef,
		CompletionContextStringSubResourceRef,
		CompletionContextStringSubResourceProperty,
		CompletionContextStringSubDataSourceRef,
		CompletionContextStringSubDataSourceProperty,
		CompletionContextStringSubValueRef,
		CompletionContextStringSubValueProperty,
		CompletionContextStringSubChildRef,
		CompletionContextStringSubChildProperty,
		CompletionContextStringSubElemRef,
		CompletionContextStringSubPartialPath,
		CompletionContextStringSubPotentialResourceProp:
		return true
	default:
		return false
	}
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

// extractAnnotationKeyFromTextBefore extracts an annotation key from TextBefore when at a value position.
// Annotation keys can contain dots (e.g., aws.lambda.dynamodb.accessType).
// Returns the annotation key without quotes, or empty string if no match.
func extractAnnotationKeyFromTextBefore(textBefore string) string {
	matches := annotationKeyFieldPattern.FindStringSubmatch(textBefore)
	if matches == nil {
		return ""
	}
	// matches[1] is the quoted annotation key (group 1), matches[2] is the unquoted annotation key (group 2)
	if matches[1] != "" {
		return matches[1]
	}
	return matches[2]
}
