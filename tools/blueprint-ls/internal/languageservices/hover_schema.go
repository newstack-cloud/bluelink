package languageservices

import (
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/helpinfo"
	common "github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// safeRangeToLSPRange converts a source.Range to an LSP range, returning nil
// if the range or its start/end positions are nil.
func safeRangeToLSPRange(bpRange *source.Range) *lsp.Range {
	if bpRange == nil || bpRange.Start == nil || bpRange.End == nil {
		return nil
	}
	return rangeToLSPRange(bpRange)
}

func getResourceNameHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	name := extractElementName(node.Path)
	resource := getResource(blueprint, name)
	if resource == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderResourceHoverInfo(name, resource),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getVariableNameHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	name := extractElementName(node.Path)
	variable := getVariable(blueprint, name)
	if variable == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderVariableHoverInfo(name, variable),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getValueNameHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	name := extractElementName(node.Path)
	value := getValue(blueprint, name)
	if value == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderValueHoverInfo(name, value),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getDataSourceNameHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	name := extractElementName(node.Path)
	ds := getDataSource(blueprint, name)
	if ds == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderDataSourceHoverInfo(name, ds),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getIncludeNameHoverContent(
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	name := extractElementName(node.Path)
	include := getChild(blueprint, name)
	if include == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderIncludeHoverInfo(name, include),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getSectionHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	return &HoverContent{
		Value: helpinfo.RenderSectionDefinition(node.Label),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getMetadataHoverContent(hoverCtx *docmodel.HoverContext) (*HoverContent, error) {
	node := hoverCtx.TreeNode
	parentContext := extractParentContext(node.Path)

	metadata, ok := node.SchemaElement.(*schema.Metadata)
	if ok && metadata != nil {
		fieldKey := findFieldKeyAtPosition(metadata.FieldsSourceMeta, hoverCtx.CursorPosition)
		if fieldKey != "" {
			content := helpinfo.RenderMetadataFieldDefinition(fieldKey)
			if content != "" {
				return &HoverContent{
					Value: content,
					Range: safeRangeToLSPRange(node.Range),
				}, nil
			}
		}
	}

	return &HoverContent{
		Value: helpinfo.RenderMetadataDefinition(parentContext),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getDataSourceMetadataHoverContent(hoverCtx *docmodel.HoverContext) (*HoverContent, error) {
	node := hoverCtx.TreeNode

	dsMetadata, ok := node.SchemaElement.(*schema.DataSourceMetadata)
	if ok && dsMetadata != nil {
		fieldKey := findFieldKeyAtPosition(dsMetadata.FieldsSourceMeta, hoverCtx.CursorPosition)
		if fieldKey != "" {
			content := helpinfo.RenderMetadataFieldDefinition(fieldKey)
			if content != "" {
				return &HoverContent{
					Value: content,
					Range: safeRangeToLSPRange(node.Range),
				}, nil
			}
		}
	}

	return &HoverContent{
		Value: helpinfo.RenderMetadataDefinition("datasource"),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

// findFieldKeyAtPosition checks a FieldsSourceMeta map to find
// which field key the cursor is on based on the key's source position.
func findFieldKeyAtPosition(
	fieldsSourceMeta map[string]*source.Meta,
	pos source.Position,
) string {
	if fieldsSourceMeta == nil {
		return ""
	}

	for key, meta := range fieldsSourceMeta {
		if meta == nil {
			continue
		}
		if pos.Line == meta.Position.Line &&
			pos.Column >= meta.Position.Column &&
			pos.Column < meta.Position.Column+len(key) {
			return key
		}
	}
	return ""
}

func getLinkSelectorHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	return &HoverContent{
		Value: helpinfo.RenderLinkSelectorDefinition(),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getStringMapHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	content := renderStringMapDefinition(node.Path, node.Label)
	return &HoverContent{
		Value: content,
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getStringOrSubsMapHoverContent(
	ctx *common.LSPContext,
	hoverCtx *docmodel.HoverContext,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	node := hoverCtx.TreeNode

	// Check if this is an annotations map that could have link annotation definitions
	if isAnnotationsNode(node.Path) && s.linkRegistry != nil {
		return s.getAnnotationHoverContent(ctx, hoverCtx, blueprint)
	}

	content := renderStringOrSubsMapDefinition(node.Path, node.Label)
	return &HoverContent{
		Value: content,
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getMappingNodeHoverContent(
	ctx *common.LSPContext,
	hoverCtx *docmodel.HoverContext,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	node := hoverCtx.TreeNode
	parts := strings.Split(node.Path, "/")

	if isResourceSpecFieldPath(parts) {
		return s.getSpecFieldHoverContent(ctx, node, parts, blueprint)
	}

	parentContext, fieldName := extractFieldContext(parts)
	content := helpinfo.RenderFieldDefinition(fieldName, parentContext)
	return &HoverContent{
		Value: content,
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getSpecFieldHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
	parts []string,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	// Path: /resources/{name}/spec/{field...}
	if len(parts) < 4 {
		return &HoverContent{}, nil
	}

	resourceName := parts[2]
	resource := getResource(blueprint, resourceName)
	if resource == nil || resource.Type == nil {
		return &HoverContent{}, nil
	}

	// If hovering on "spec" itself (path is /resources/{name}/spec)
	if len(parts) == 4 && parts[3] == "spec" {
		return &HoverContent{
			Value: helpinfo.RenderFieldDefinition("spec", "resource"),
			Range: safeRangeToLSPRange(node.Range),
		}, nil
	}

	s.logger.Debug(
		"Fetching spec definition for schema field hover",
		zap.String("resourceType", resource.Type.Value),
		zap.String("path", node.Path),
	)
	specDefOutput, err := s.resourceRegistry.GetSpecDefinition(
		ctx.Context,
		resource.Type.Value,
		&provider.ResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return &HoverContent{}, nil
	}

	specPath := buildSpecSubstitutionPath(parts[4:])
	specFieldSchema, err := findResourceFieldSchema(specDefOutput.SpecDefinition.Schema, specPath)
	if err != nil || specFieldSchema == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderSpecFieldDefinition(node.Label, specFieldSchema),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getDataSourceFieldExportHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	parts := strings.Split(node.Path, "/")
	// Path: /datasources/{name}/{fieldName}
	if len(parts) < 4 {
		return &HoverContent{}, nil
	}

	dsName := parts[2]
	fieldName := parts[3]
	ds := getDataSource(blueprint, dsName)
	if ds == nil {
		return &HoverContent{}, nil
	}

	field := getDataSourceField(ds, fieldName)
	if field == nil {
		return &HoverContent{}, nil
	}

	specSchema := s.lookupDataSourceFieldSpecSchema(ctx, ds, field, fieldName)

	return &HoverContent{
		Value: helpinfo.RenderDataSourceExportFieldDefinition(fieldName, ds, field, specSchema),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) lookupDataSourceFieldSpecSchema(
	ctx *common.LSPContext,
	ds *schema.DataSource,
	field *schema.DataSourceFieldExport,
	fieldName string,
) *provider.DataSourceSpecSchema {
	if ds.Type == nil || s.dataSourceRegistry == nil {
		return nil
	}

	specDefOutput, err := s.dataSourceRegistry.GetSpecDefinition(
		ctx.Context,
		string(ds.Type.Value),
		&provider.DataSourceGetSpecDefinitionInput{},
	)
	if err != nil || specDefOutput == nil || specDefOutput.SpecDefinition == nil {
		return nil
	}

	// If the field has an alias, look up the aliased field's definition
	lookupName := fieldName
	if field.AliasFor != nil && field.AliasFor.StringValue != nil {
		lookupName = *field.AliasFor.StringValue
	}

	if specSchema, ok := specDefOutput.SpecDefinition.Fields[lookupName]; ok {
		return specSchema
	}

	return nil
}

// getAnnotationHoverContent provides hover for annotation keys with link annotation definitions.
// It uses the cursor position and the annotations map's SourceMeta to determine
// which specific annotation key the cursor is on.
func (s *HoverService) getAnnotationHoverContent(
	ctx *common.LSPContext,
	hoverCtx *docmodel.HoverContext,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	node := hoverCtx.TreeNode

	annotationsMap, ok := node.SchemaElement.(*schema.StringOrSubstitutionsMap)
	if !ok || annotationsMap == nil {
		return &HoverContent{
			Value: helpinfo.RenderAnnotationsDefinition(),
			Range: safeRangeToLSPRange(node.Range),
		}, nil
	}

	annotationKey := findAnnotationKeyAtPosition(annotationsMap, hoverCtx.CursorPosition)
	if annotationKey == "" {
		// On the annotations key itself, show generic definition.
		return &HoverContent{
			Value: helpinfo.RenderAnnotationsDefinition(),
			Range: safeRangeToLSPRange(node.Range),
		}, nil
	}

	parts := strings.Split(node.Path, "/")
	if len(parts) < 5 || parts[1] != "resources" {
		return &HoverContent{}, nil
	}

	resourceName := parts[2]
	resource := getResource(blueprint, resourceName)
	if resource == nil || resource.Type == nil {
		return &HoverContent{}, nil
	}

	def := s.findAnnotationDefinition(ctx, blueprint, resourceName, resource, annotationKey)
	if def != nil {
		return &HoverContent{
			Value: helpinfo.RenderLinkAnnotationDefinition(annotationKey, def),
			Range: safeRangeToLSPRange(node.Range),
		}, nil
	}

	annotationValue := getAnnotationValue(annotationsMap, annotationKey)
	return &HoverContent{
		Value: helpinfo.RenderAnnotationKeyInfo(annotationKey, annotationValue),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

// getAnnotationValue extracts the display string of an annotation value.
// For plain strings, it returns the string value directly.
// For values with substitutions, it returns a placeholder representation.
func getAnnotationValue(
	annotationsMap *schema.StringOrSubstitutionsMap,
	annotationKey string,
) string {
	if annotationsMap.Values == nil {
		return ""
	}

	val, ok := annotationsMap.Values[annotationKey]
	if !ok || val == nil || len(val.Values) == 0 {
		return ""
	}

	// For a simple string value, return it directly.
	if len(val.Values) == 1 && val.Values[0].StringValue != nil {
		return *val.Values[0].StringValue
	}

	return ""
}

// findAnnotationKeyAtPosition checks the annotations map's SourceMeta
// to find which annotation key the cursor is on.
func findAnnotationKeyAtPosition(
	annotationsMap *schema.StringOrSubstitutionsMap,
	pos source.Position,
) string {
	if annotationsMap.SourceMeta == nil {
		return ""
	}

	for key, meta := range annotationsMap.SourceMeta {
		if meta == nil {
			continue
		}
		if pos.Line == meta.Position.Line &&
			pos.Column >= meta.Position.Column &&
			pos.Column < meta.Position.Column+len(key) {
			return key
		}
	}
	return ""
}

func (s *HoverService) findAnnotationDefinition(
	ctx *common.LSPContext,
	blueprint *schema.Blueprint,
	resourceName string,
	resource *schema.Resource,
	annotationKey string,
) *provider.LinkAnnotationDefinition {
	if blueprint.Resources == nil {
		return nil
	}

	currentType := resource.Type.Value
	linkedResources := s.findLinkedResourcesForHover(ctx, blueprint, resourceName, currentType)

	for _, linked := range linkedResources {
		// Try the selector-determined direction first.
		var typeA, typeB string
		if linked.currentIsA {
			typeA = currentType
			typeB = linked.resourceType
		} else {
			typeA = linked.resourceType
			typeB = currentType
		}

		def := s.findAnnotationDefFromLink(ctx, typeA, typeB, linked, annotationKey)
		if def != nil {
			return def
		}

		// Try the reverse direction; real plugins may register bidirectional links
		// with different annotations on each direction.
		def = s.findAnnotationDefFromLink(ctx, typeB, typeA, linked, annotationKey)
		if def != nil {
			return def
		}
	}

	return nil
}

func (s *HoverService) findAnnotationDefFromLink(
	ctx *common.LSPContext,
	typeA, typeB string,
	linked linkedResourceInfoForHover,
	annotationKey string,
) *provider.LinkAnnotationDefinition {
	link, err := s.linkRegistry.Link(ctx.Context, typeA, typeB)
	if err != nil || link == nil {
		return nil
	}

	emptyParams := core.NewDefaultParams(nil, nil, nil, nil)
	linkCtx := provider.NewLinkContextFromParams(emptyParams)
	output, err := link.GetAnnotationDefinitions(
		ctx.Context,
		&provider.LinkGetAnnotationDefinitionsInput{
			LinkContext: linkCtx,
		},
	)
	if err != nil || output == nil || output.AnnotationDefinitions == nil {
		return nil
	}

	for _, def := range output.AnnotationDefinitions {
		expandedNames := expandAnnotationNameForHover(def.Name, linked.name)
		if slices.Contains(expandedNames, annotationKey) {
			return def
		}
	}

	return nil
}

type linkedResourceInfoForHover struct {
	name         string
	resourceType string
	currentIsA   bool
}

func (s *HoverService) findLinkedResourcesForHover(
	ctx *common.LSPContext,
	blueprint *schema.Blueprint,
	resourceName string,
	resourceType string,
) []linkedResourceInfoForHover {
	if blueprint.Resources == nil {
		return nil
	}

	currentResource := blueprint.Resources.Values[resourceName]
	if currentResource == nil {
		return nil
	}

	var result []linkedResourceInfoForHover
	for otherName, otherResource := range blueprint.Resources.Values {
		if otherName == resourceName || otherResource.Type == nil {
			continue
		}

		otherType := otherResource.Type.Value
		currentIsA, linked := s.getLinkDirectionForHover(ctx, resourceType, otherType)
		if !linked {
			continue
		}

		currentSelectsOther := hasMatchingSelector(currentResource, otherResource, otherName)
		otherSelectsCurrent := hasMatchingSelector(otherResource, currentResource, resourceName)
		if !currentSelectsOther && !otherSelectsCurrent {
			continue
		}

		// Correct the link direction based on selector relationships.
		// The resource with the linkSelector is A (the selecting side).
		currentIsA = determineCurrentIsA(
			currentResource, otherResource,
			resourceName, otherName,
			currentIsA,
		)

		result = append(result, linkedResourceInfoForHover{
			name:         otherName,
			resourceType: otherType,
			currentIsA:   currentIsA,
		})
	}

	return result
}

func (s *HoverService) getLinkDirectionForHover(
	ctx *common.LSPContext,
	currentType, otherType string,
) (bool, bool) {
	link, err := s.linkRegistry.Link(ctx.Context, currentType, otherType)
	if err == nil && link != nil {
		return true, true
	}

	link, err = s.linkRegistry.Link(ctx.Context, otherType, currentType)
	if err == nil && link != nil {
		return false, true
	}

	return false, false
}

// expandAnnotationNameForHover expands <resourceName> placeholders with a specific name.
func expandAnnotationNameForHover(name string, linkedResourceName string) []string {
	openIdx := strings.Index(name, "<")
	closeIdx := strings.Index(name, ">")
	if openIdx == -1 || closeIdx == -1 || closeIdx < openIdx {
		return []string{name}
	}

	if linkedResourceName == "" {
		return nil
	}

	return []string{name[:openIdx] + linkedResourceName + name[closeIdx+1:]}
}

// extractElementName extracts the element name from a tree node path.
// Path format: /section/{name} (e.g., /resources/ordersTable)
func extractElementName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// extractParentContext determines the parent context from a tree node path.
// Returns "resource", "datasource", "include", etc.
func extractParentContext(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}

	switch parts[1] {
	case "resources":
		return "resource"
	case "datasources":
		return "datasource"
	case "includes":
		return "include"
	case "exports":
		return "export"
	case "variables":
		return "variable"
	case "values":
		return "value"
	default:
		return ""
	}
}

// extractFieldContext extracts the parent context and field name from path parts.
func extractFieldContext(parts []string) (string, string) {
	if len(parts) < 2 {
		return "", ""
	}

	parentContext := ""
	switch parts[1] {
	case "resources":
		parentContext = "resource"
	case "datasources":
		parentContext = "datasource"
	case "includes":
		parentContext = "include"
	case "exports":
		parentContext = "export"
	case "variables":
		parentContext = "variable"
	case "values":
		parentContext = "value"
	}

	fieldName := parts[len(parts)-1]
	return parentContext, fieldName
}

// isResourceSpecFieldPath checks if path parts represent a resource spec field.
// Matches /resources/{name}/spec or /resources/{name}/spec/{field...}
func isResourceSpecFieldPath(parts []string) bool {
	return len(parts) >= 4 && parts[1] == "resources" && parts[3] == "spec"
}

// buildSpecSubstitutionPath converts path segments after "spec" into
// SubstitutionPathItem slice for use with findResourceFieldSchema.
func buildSpecSubstitutionPath(
	fieldParts []string,
) []*substitutions.SubstitutionPathItem {
	items := make([]*substitutions.SubstitutionPathItem, 0, len(fieldParts))
	for _, part := range fieldParts {
		items = append(items, &substitutions.SubstitutionPathItem{
			FieldName: part,
		})
	}
	return items
}

func isAnnotationsNode(path string) bool {
	return strings.Contains(path, "/metadata/annotations")
}

func renderStringMapDefinition(path string, label string) string {
	if strings.HasSuffix(path, "/labels") || label == "labels" {
		return helpinfo.RenderLabelsDefinition()
	}
	if strings.HasSuffix(path, "/byLabel") || label == "byLabel" {
		return helpinfo.RenderByLabelDefinition()
	}
	return helpinfo.RenderFieldDefinition(label, "")
}

func renderStringOrSubsMapDefinition(path string, label string) string {
	if strings.Contains(path, "/annotations") || label == "annotations" {
		return helpinfo.RenderAnnotationsDefinition()
	}
	return helpinfo.RenderFieldDefinition(label, "")
}

func getDataSourceFieldExportMapHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	return &HoverContent{
		Value: helpinfo.RenderFieldDefinition("exports", "datasource"),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getDataSourceFieldTypeHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	fieldType, ok := node.SchemaElement.(*schema.DataSourceFieldTypeWrapper)
	if !ok || fieldType == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderDataSourceFieldTypeDefinition(string(fieldType.Value)),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getDataSourceFiltersHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	return &HoverContent{
		Value: helpinfo.RenderFieldDefinition("filter", "datasource"),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) getDataSourceFilterHoverContent(
	ctx *common.LSPContext,
	hoverCtx *docmodel.HoverContext,
	blueprint *schema.Blueprint,
) (*HoverContent, error) {
	node := hoverCtx.TreeNode
	filter, ok := node.SchemaElement.(*schema.DataSourceFilter)
	if !ok || filter == nil {
		return &HoverContent{}, nil
	}

	// Check if cursor is on a specific sub-field key
	childKey := findChildKeyOnLine(node.Children, hoverCtx.CursorPosition.Line)
	switch childKey {
	case "field":
		return s.getFilterFieldKeyHoverContent(ctx, node, blueprint, filter)
	case "operator":
		return getFilterOperatorKeyHoverContent(node, filter)
	case "search":
		return &HoverContent{
			Value: helpinfo.RenderDataSourceFilterSearchDefinition(),
			Range: safeRangeToLSPRange(node.Range),
		}, nil
	default:
		return &HoverContent{
			Value: helpinfo.RenderDataSourceFilterDefinition(filter),
			Range: safeRangeToLSPRange(node.Range),
		}, nil
	}
}

func (s *HoverService) getFilterFieldKeyHoverContent(
	ctx *common.LSPContext,
	node *schema.TreeNode,
	blueprint *schema.Blueprint,
	filter *schema.DataSourceFilter,
) (*HoverContent, error) {
	fieldValue := ""
	if filter.Field != nil && filter.Field.StringValue != nil {
		fieldValue = *filter.Field.StringValue
	}

	// Extract data source name from path (/datasources/{name}/filter)
	var filterSchema *provider.DataSourceFilterSchema
	parts := strings.Split(node.Path, "/")
	if len(parts) >= 3 && fieldValue != "" {
		dsName := parts[2]
		ds := getDataSource(blueprint, dsName)
		if ds != nil && ds.Type != nil {
			filterSchema = s.lookupFilterFieldSchema(ctx, string(ds.Type.Value), fieldValue)
		}
	}

	return &HoverContent{
		Value: helpinfo.RenderDataSourceFilterFieldKeyDefinition(fieldValue, filterSchema),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getFilterOperatorKeyHoverContent(
	node *schema.TreeNode,
	filter *schema.DataSourceFilter,
) (*HoverContent, error) {
	operatorStr := ""
	if filter.Operator != nil {
		operatorStr = string(filter.Operator.Value)
	}

	return &HoverContent{
		Value: helpinfo.RenderDataSourceFilterOperatorDefinition(operatorStr),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func (s *HoverService) lookupFilterFieldSchema(
	ctx *common.LSPContext,
	dsType string,
	fieldValue string,
) *provider.DataSourceFilterSchema {
	if s.dataSourceRegistry == nil || fieldValue == "" {
		return nil
	}

	output, err := s.dataSourceRegistry.GetFilterFields(
		ctx.Context,
		dsType,
		&provider.DataSourceGetFilterFieldsInput{},
	)
	if err != nil || output == nil || output.FilterFields == nil {
		return nil
	}

	if filterSchema, ok := output.FilterFields[fieldValue]; ok {
		return filterSchema
	}

	return nil
}

// findChildKeyOnLine checks tree node children to find which child's
// value is on the same line as the cursor position. When the cursor is on
// the YAML key text (before the value), the child's value starts later
// on the same line.
func findChildKeyOnLine(children []*schema.TreeNode, line int) string {
	for _, child := range children {
		if child == nil || child.Range == nil || child.Range.Start == nil {
			continue
		}
		if child.Range.Start.Line == line {
			return child.Label
		}
	}
	return ""
}

func getDataSourceFilterOperatorHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	operator, ok := node.SchemaElement.(*schema.DataSourceFilterOperatorWrapper)
	if !ok || operator == nil {
		return &HoverContent{}, nil
	}

	return &HoverContent{
		Value: helpinfo.RenderDataSourceFilterOperatorDefinition(string(operator.Value)),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getDataSourceFilterSearchHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	return &HoverContent{
		Value: helpinfo.RenderDataSourceFilterSearchDefinition(),
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func getStringListHoverContent(node *schema.TreeNode) (*HoverContent, error) {
	content := renderStringListDefinition(node.Path, node.Label)
	return &HoverContent{
		Value: content,
		Range: safeRangeToLSPRange(node.Range),
	}, nil
}

func renderStringListDefinition(path string, label string) string {
	if strings.HasSuffix(path, "/exclude") || label == "exclude" {
		return helpinfo.RenderExcludeDefinition()
	}
	return helpinfo.RenderFieldDefinition(label, "")
}
