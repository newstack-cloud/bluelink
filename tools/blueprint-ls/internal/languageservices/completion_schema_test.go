package languageservices

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

func (s *CompletionServiceGetItemsSuite) createSchemaDocContext(
	validContent string, editingContent string,
) *docmodel.DocumentContext {
	blueprint, err := schema.LoadString(validContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatYAML, nil)
	docCtx.UpdateSchema(blueprint, tree)
	return docCtx
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_definition_fields() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders"
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 4, Character: 4},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"condition", "dependsOn", "description", "each", "linkSelector", "metadata", "spec", "type"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Resource field", *item.Detail)
		s.Assert().NotNil(item.Documentation)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_variable_definition_fields() {
	validContent := "version: 2025-11-02\nvariables:\n  instanceType:\n    type: aws/ec2/instanceType\n    description: The EC2 instance type"
	editingContent := "version: 2025-11-02\nvariables:\n  instanceType:\n    type: aws/ec2/instanceType\n    "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 4, Character: 4},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"allowedValues", "default", "description", "secret", "type"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Variable field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_value_definition_fields() {
	validContent := "version: 2025-11-02\nvalues:\n  tableName:\n    type: string\n    value: orders"
	editingContent := "version: 2025-11-02\nvalues:\n  tableName:\n    type: string\n    "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 4, Character: 4},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"description", "secret", "type", "value"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Value field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_definition_fields() {
	validContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    filter:\n      field: vpcId"
	editingContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 4, Character: 4},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"description", "exports", "filter", "metadata", "type"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Data source field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_filter_fields() {
	validContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    filter:\n      field: vpcId\n      operator: \"=\""
	editingContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    filter:\n      field: vpcId\n      "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 6, Character: 6},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"field", "operator", "search"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Filter field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_export_fields() {
	validContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    exports:\n      vpcId:\n        type: string\n        aliasFor: vpc_id"
	editingContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    exports:\n      vpcId:\n        type: string\n        "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 7, Character: 8},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"aliasFor", "description", "type"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Export field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_metadata_fields() {
	validContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    metadata:\n      displayName: Network VPC"
	editingContent := "version: 2025-11-02\ndatasources:\n  network:\n    type: aws/vpc\n    metadata:\n      displayName: Network VPC\n      "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 6, Character: 6},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"annotations", "custom", "displayName"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Metadata field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_include_definition_fields() {
	validContent := "version: 2025-11-02\ninclude:\n  auth:\n    path: ./auth.yaml\n    description: Auth module"
	editingContent := "version: 2025-11-02\ninclude:\n  auth:\n    path: ./auth.yaml\n    "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 4, Character: 4},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"description", "metadata", "path", "variables"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Include field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_export_definition_fields() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\nexports:\n  tableId:\n    type: string\n    field: resources.ordersTable.state.id"
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\nexports:\n  tableId:\n    type: string\n    "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 7, Character: 4},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"description", "field", "type"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Export field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_link_selector_fields() {
	validContent := "version: 2025-11-02\nresources:\n  saveOrderFunction:\n    type: aws/lambda/function\n    linkSelector:\n      byLabel:\n        app: orders"
	editingContent := "version: 2025-11-02\nresources:\n  saveOrderFunction:\n    type: aws/lambda/function\n    linkSelector:\n      "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 6},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"byLabel", "exclude"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("LinkSelector field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_blueprint_top_level_fields() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table"
	// Position cursor on a new line right after the version line to trigger root-level completion.
	editingContent := "version: 2025-11-02\n"
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 1, Character: 0},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	expectedLabels := []string{"datasources", "exports", "include", "metadata", "resources", "transform", "values", "variables", "version"}
	s.Assert().Equal(expectedLabels, labels)

	for _, item := range completionItems.Items {
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal("Blueprint field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_spec_field_allowed_values() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      billingMode: PAY_PER_REQUEST"
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      billingMode: "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 19},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "PAY_PER_REQUEST")
	s.Assert().Contains(labels, "PROVISIONED")
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_spec_field_no_allowed_values() {
	validContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: orders"
	editingContent := "version: 2025-11-02\nresources:\n  ordersTable:\n    type: aws/dynamodb/table\n    spec:\n      tableName: "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 17},
		},
	)
	s.Require().NoError(err)
	s.Assert().Empty(completionItems.Items)
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_spec_field_unknown_resource_type() {
	validContent := "version: 2025-11-02\nresources:\n  bucket:\n    type: aws/s3/bucket\n    spec:\n      bucketName: my-bucket"
	editingContent := "version: 2025-11-02\nresources:\n  bucket:\n    type: aws/s3/bucket\n    spec:\n      "
	docCtx := s.createSchemaDocContext(validContent, editingContent)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 5, Character: 6},
		},
	)
	s.Require().NoError(err)

	// Should return a hint that the resource type is not found
	s.Require().Len(completionItems.Items, 1)
	s.Assert().Contains(completionItems.Items[0].Label, "not found")
}
