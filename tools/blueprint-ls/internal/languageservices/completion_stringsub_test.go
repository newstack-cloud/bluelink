package languageservices

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_standalone_resource_spec_fields() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Editing content has a substitution referencing the resource's spec path
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${ordersTable.spec."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	// Cursor at the end of "${ordersTable.spec."
	// Line 5: "      tableName: \"${ordersTable.spec."
	//         0123456789...
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 37},
		},
	)
	s.Require().NoError(err)

	// Should get spec fields from the DynamoDB table resource
	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "billingMode")
	s.Assert().Contains(labels, "id")
	s.Assert().Contains(labels, "tableName")
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_resource_top_level_props() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Editing content with "${ordersTable." to trigger resource prop completion
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${ordersTable."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	// Cursor at end of "${ordersTable."
	// Line 5: "      tableName: \"${ordersTable."
	// 6 + 10 + 2 + 1 + 2 + 11 + 1 = 33
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 32},
		},
	)
	s.Require().NoError(err)

	// Should get resource top-level properties
	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "metadata")
	s.Assert().Contains(labels, "spec")
	s.Assert().Contains(labels, "state")
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_nonexistent_resource_falls_through() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Editing with a non-existent resource name
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${nonExistent."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 32},
		},
	)
	s.Require().NoError(err)

	// Should fall through to all substitution refs since "nonExistent" is not a valid resource
	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "resources.ordersTable")
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_value_property_mapping_node() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders\nvalues:\n  config:\n    type: object\n    value:\n      database: test\n      cacheHost: redis"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Editing with "${values.config." to trigger value property completion
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${values.config."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 34},
		},
	)
	s.Require().NoError(err)

	// Should get MappingNode field keys from the config value
	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "cacheHost")
	s.Assert().Contains(labels, "database")
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_value_property_nonexistent() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders\nvalues:\n  config:\n    type: object\n    value:\n      database: test"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Editing with a non-existent value name
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${values.unknown."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 35},
		},
	)
	s.Require().NoError(err)
	s.Assert().Empty(completionItems.Items)
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_datasource_prop_no_exports() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders\ndatasources:\n  network:\n    type: aws/vpc\n    filter:\n      field: vpcId"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Editing with "${datasources.network." but the DS has no exports defined
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${datasources.network."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 40},
		},
	)
	s.Require().NoError(err)
	s.Assert().Empty(completionItems.Items)
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_datasource_prop_with_exports() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders\ndatasources:\n  network:\n    type: aws/vpc\n    filter:\n      field: vpcId\n    exports:\n      vpcId:\n        type: string\n      subnetId:\n        type: string"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${datasources.network."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 40},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "subnetId")
	s.Assert().Contains(labels, "vpcId")
}

func (s *CompletionServiceGetItemsSuite) Test_get_stringsub_schema_nonexistent_attribute() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders"
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	// Navigate to a non-existent attribute in the spec path
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: \"${ordersTable.spec.nonExistent."
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 50},
		},
	)
	s.Require().NoError(err)
	s.Assert().Empty(completionItems.Items)
}
