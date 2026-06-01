package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// Describes a blueprint-language completion item for a declaration
// keyword, block field or inline statement. insert is the snippet body inserted
// when the item is accepted; when empty it defaults to a scalar assignment
// ("label = "). All blueprint field items are inserted as LSP snippets.
type bpFieldInfo struct {
	label       string
	description string
	insert      string
}

// Top-level declaration keywords (what is typed at the document root in .bp).
var bpDeclarations = []bpFieldInfo{
	{label: "version", description: "The blueprint specification version directive.", insert: "version \"${1:2025-11-02}\""},
	{label: "transform", description: "A transform to apply to the blueprint.", insert: "transform \"${1:transform}\""},
	{label: "variable", description: "An input variable declaration.", insert: "variable ${1:name}: ${2:string} {\n\t$0\n}"},
	{label: "value", description: "A computed value declaration.", insert: "value ${1:name}: ${2:string} {\n\tvalue = $0\n}"},
	{label: "data", description: "A data source declaration.", insert: "data ${1:name}: ${2:type} {\n\t$0\n}"},
	{label: "resource", description: "A resource declaration.", insert: "resource ${1:name}: ${2:type} {\n\tspec {\n\t\t$0\n\t}\n}"},
	{label: "include", description: "A child blueprint include declaration.", insert: "include ${1:name} \"${2:./path}\" {\n\t$0\n}"},
	{label: "export", description: "A blueprint output export declaration.", insert: "export ${1:name}: ${2:string} {\n\tfield = $0\n}"},
	{label: "metadata", description: "Blueprint-level metadata block.", insert: "metadata {\n\t$0\n}"},
}

// Resource body fields. `type` lives in the declaration header, not the body.
var bpResourceFields = []bpFieldInfo{
	{label: "description", description: "A human-readable description of the resource."},
	{label: "spec", description: "Provider-specific resource configuration.", insert: "spec {\n\t$0\n}"},
	{label: "metadata", description: "displayName, labels, annotations and custom metadata.", insert: "metadata {\n\t$0\n}"},
	{label: "condition", description: "A condition controlling whether the resource is deployed."},
	{label: "foreach", description: "Templates the resource over an array; use elem and i.", insert: "foreach $0"},
	{label: "select by label", description: "Selects resources to link by label.", insert: "select by label {\n\t$0\n}"},
	{label: "dependsOn", description: "Explicit dependencies on other resources."},
	{label: "removalPolicy", description: "What happens to the resource on removal (\"delete\" or \"retain\")."},
}

var bpVariableFields = []bpFieldInfo{
	{label: "description", description: "A human-readable description of the variable."},
	{label: "default", description: "The default value used when none is supplied."},
	{label: "allowedValues", description: "An array of permitted values.", insert: "allowedValues = [$0]"},
	{label: "secret", description: "Mask the value in logs when true."},
}

var bpValueFields = []bpFieldInfo{
	{label: "value", description: "The computed value expression."},
	{label: "description", description: "A human-readable description of the value."},
	{label: "secret", description: "Mask the value in logs when true."},
}

var bpDataSourceFields = []bpFieldInfo{
	{label: "filter", description: "A filter selecting the external resource.", insert: "filter \"${1:field}\" ${2:==} ${3:value}"},
	{label: "export", description: "Exposes a field from the external resource.", insert: "export ${1:field}: ${2:string}"},
	{label: "metadata", description: "displayName, annotations and custom metadata.", insert: "metadata {\n\t$0\n}"},
	{label: "description", description: "A human-readable description of the data source."},
}

// Inside a data source `export ... { }` block; type/aliasFor live in the header.
var bpDataSourceExportFields = []bpFieldInfo{
	{label: "description", description: "A human-readable description of the exported field."},
}

var bpDataSourceMetadataFields = []bpFieldInfo{
	{label: "displayName", description: "A human-readable name for the data source."},
	{label: "annotations", description: "Key-value pairs configuring data source behaviour.", insert: "annotations = {\n\t$0\n}"},
	{label: "custom", description: "Custom provider-specific metadata.", insert: "custom = {\n\t$0\n}"},
}

var bpResourceMetadataFields = []bpFieldInfo{
	{label: "displayName", description: "A human-readable name for the resource."},
	{label: "labels", description: "Key-value labels used by select by label.", insert: "labels = {\n\t$0\n}"},
	{label: "annotations", description: "Key-value pairs for provider/link context.", insert: "annotations = {\n\t$0\n}"},
	{label: "custom", description: "Custom metadata fields.", insert: "custom = {\n\t$0\n}"},
}

// `path` is the positional string in the include header, not a body field.
var bpIncludeFields = []bpFieldInfo{
	{label: "variables", description: "Variables passed to the child blueprint.", insert: "variables {\n\t$0\n}"},
	{label: "metadata", description: "Source-location metadata for include resolvers.", insert: "metadata {\n\t$0\n}"},
	{label: "description", description: "A human-readable description of the included blueprint."},
}

// Top-level export body; the type lives in the declaration header.
var bpExportFields = []bpFieldInfo{
	{label: "field", description: "A reference to the value being exported."},
	{label: "description", description: "A human-readable description of the export."},
}

// Inside `select by label { }` the body holds free-form label keys; `exclude`
// is the only fixed field (byLabel is implicit in the block, never typed).
var bpLinkSelectorFields = []bpFieldInfo{
	{label: "exclude", description: "Resource names to exclude from link matching.", insert: "exclude = [$0]"},
}

func (s *CompletionService) getBlueprintFieldNameCompletions(
	kind docmodel.CompletionContextKind,
	position *lsp.Position,
	completionCtx *docmodel.CompletionContext,
) ([]*lsp.CompletionItem, bool) {
	fields, detail, completionType, ok := blueprintFieldTableForKind(kind)
	if !ok {
		return nil, false
	}

	typedPrefix := ""
	if completionCtx.CursorCtx != nil {
		typedPrefix = completionCtx.CursorCtx.GetTypedPrefix()
	}

	return createBlueprintFieldCompletionItems(fields, position, typedPrefix, detail, completionType), true
}

func blueprintFieldTableForKind(
	kind docmodel.CompletionContextKind,
) ([]bpFieldInfo, string, string, bool) {
	switch kind {
	case docmodel.CompletionContextBlueprintTopLevelField:
		return bpDeclarations, "Blueprint declaration", "bpDeclaration", true
	case docmodel.CompletionContextResourceDefinitionField:
		return bpResourceFields, "Resource field", "bpResourceField", true
	case docmodel.CompletionContextVariableDefinitionField:
		return bpVariableFields, "Variable field", "bpVariableField", true
	case docmodel.CompletionContextValueDefinitionField:
		return bpValueFields, "Value field", "bpValueField", true
	case docmodel.CompletionContextDataSourceDefinitionField:
		return bpDataSourceFields, "Data source field", "bpDataSourceField", true
	case docmodel.CompletionContextDataSourceExportDefinitionField:
		return bpDataSourceExportFields, "Export field", "bpDataSourceExportField", true
	case docmodel.CompletionContextDataSourceMetadataField:
		return bpDataSourceMetadataFields, "Metadata field", "bpDataSourceMetadataField", true
	case docmodel.CompletionContextResourceMetadataField:
		return bpResourceMetadataFields, "Metadata field", "bpResourceMetadataField", true
	case docmodel.CompletionContextIncludeDefinitionField:
		return bpIncludeFields, "Include field", "bpIncludeField", true
	case docmodel.CompletionContextExportDefinitionField:
		return bpExportFields, "Export field", "bpExportField", true
	case docmodel.CompletionContextLinkSelectorField:
		return bpLinkSelectorFields, "Link selector field", "bpLinkSelectorField", true
	default:
		return nil, "", "", false
	}
}

// Bare literal values offered at a blueprint value
// position when no more specific (reference or schema) completion applies.
var bpValueLiterals = []bpFieldInfo{
	{label: "true", description: "Boolean true."},
	{label: "false", description: "Boolean false."},
	{label: "none", description: "Explicit absence of a value (omits the field on materialisation)."},
}

// Offers the bare true/false/none literals at a
// blueprint value position, filtered by the typed prefix.
func blueprintValueLiteralCompletions(position *lsp.Position, typedPrefix string) []*lsp.CompletionItem {
	prefixLen := len(typedPrefix)
	prefixLower := strings.ToLower(typedPrefix)
	valueKind := lsp.CompletionItemKindValue

	items := make([]*lsp.CompletionItem, 0, len(bpValueLiterals))
	for _, literal := range bpValueLiterals {
		if prefixLen > 0 && !strings.HasPrefix(literal.label, prefixLower) {
			continue
		}

		label := literal.label
		item := &lsp.CompletionItem{
			Label: label,
			Kind:  &valueKind,
			TextEdit: lsp.TextEdit{
				NewText: label,
				Range:   getItemInsertRangeWithPrefix(position, prefixLen),
			},
			FilterText: &label,
		}
		if literal.description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: literal.description,
			}
		}
		items = append(items, item)
	}

	return items
}

// Builds snippet completion items for a
// blueprint field table, filtered by the typed prefix.
func createBlueprintFieldCompletionItems(
	fields []bpFieldInfo,
	position *lsp.Position,
	typedPrefix string,
	detailText string,
	completionType string,
) []*lsp.CompletionItem {
	prefixLower := strings.ToLower(typedPrefix)
	prefixLen := len(typedPrefix)
	fieldKind := lsp.CompletionItemKindField
	snippetFormat := lsp.InsertTextFormatSnippet

	items := make([]*lsp.CompletionItem, 0, len(fields))
	for _, field := range fields {
		if prefixLen > 0 && !strings.HasPrefix(strings.ToLower(field.label), prefixLower) {
			continue
		}

		insertText := field.insert
		if insertText == "" {
			insertText = field.label + " = "
		}

		label := field.label
		detail := detailText
		completionTypeValue := completionType
		item := &lsp.CompletionItem{
			Label:            label,
			Detail:           &detail,
			Kind:             &fieldKind,
			InsertTextFormat: &snippetFormat,
			TextEdit: lsp.TextEdit{
				NewText: insertText,
				Range:   getItemInsertRangeWithPrefix(position, prefixLen),
			},
			FilterText: &label,
			Data:       map[string]any{"completionType": completionTypeValue},
		}

		if field.description != "" {
			item.Documentation = lsp.MarkupContent{
				Kind:  lsp.MarkupKindMarkdown,
				Value: field.description,
			}
		}

		items = append(items, item)
	}

	return items
}
