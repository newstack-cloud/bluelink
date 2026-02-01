package languageservices

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

// Shared valid blueprint content for export field completion tests.
// This is parsed to obtain the schema + tree used for position mapping.
const exportFieldValidContent = `version: 2024-07-20
variables:
  instanceType:
    type: aws/ec2/instanceType
    description: "The EC2 instance type."
  environment:
    type: string
values:
  tableName:
    type: string
    value: "${variables.environment}-ordersTable"
datasources:
  network:
    type: aws/vpc
    description: "Networking resources."
    filter:
      field: tags
      operator: "not contains"
      search: service
    metadata:
      displayName: Networking
    exports:
      vpc:
        type: string
        aliasFor: vpcId
        description: |
          The ID of the VPC.
      subnetIds:
        type: array
        description: "The IDs of the subnets."
include:
  networking:
    path: networking.blueprint.yaml
    description: "Networking child blueprint."
resources:
  ordersTable:
    type: aws/dynamodb/table
    description: "Orders DynamoDB table."
    spec:
      tableName: ${variables.environment}
  saveOrderHandler:
    type: aws/lambda/function
    description: "Lambda function to save orders."
exports:
  orderTableName:
    type: string
    field: resources.ordersTable.spec.tableName`

// exportFieldEditingTemplate is a template for editing content in export field tests.
// The %s placeholder is replaced with the field value being tested.
const exportFieldEditingTemplate = `version: 2024-07-20
variables:
  instanceType:
    type: aws/ec2/instanceType
    description: "The EC2 instance type."
  environment:
    type: string
values:
  tableName:
    type: string
    value: "${variables.environment}-ordersTable"
datasources:
  network:
    type: aws/vpc
    description: "Networking resources."
    filter:
      field: tags
      operator: "not contains"
      search: service
    metadata:
      displayName: Networking
    exports:
      vpc:
        type: string
        aliasFor: vpcId
        description: |
          The ID of the VPC.
      subnetIds:
        type: array
        description: "The IDs of the subnets."
include:
  networking:
    path: networking.blueprint.yaml
    description: "Networking child blueprint."
resources:
  ordersTable:
    type: aws/dynamodb/table
    description: "Orders DynamoDB table."
    spec:
      tableName: ${variables.environment}
  saveOrderHandler:
    type: aws/lambda/function
    description: "Lambda function to save orders."
exports:
  orderTableName:
    type: string
    field: %s`

// exportFieldLine is the 0-indexed line number of the "field:" property in the template.
const exportFieldLine = 46

// exportFieldValueStart is the character offset where the field value begins ("    field: " = 11 chars).
const exportFieldValueStart = 11

// createExportFieldDocContext creates a DocumentContext for export field completion tests.
// The editing content has the cursor on the field: value line with the given fieldValue.
func (s *CompletionServiceGetItemsSuite) createExportFieldDocContext(
	fieldValue string,
) (*docmodel.DocumentContext, lsp.Position) {
	blueprint, err := schema.LoadString(exportFieldValidContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)

	editingContent := fmt.Sprintf(exportFieldEditingTemplate, fieldValue)
	docCtx := docmodel.NewDocumentContextFromSchema("file:///test.yaml", blueprint, tree)
	docCtx.Content = editingContent

	position := lsp.Position{
		Line:      exportFieldLine,
		Character: lsp.UInteger(exportFieldValueStart + len(fieldValue)),
	}

	return docCtx, position
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_top_level_completions() {
	docCtx, position := s.createExportFieldDocContext("")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "resources")
	s.Assert().Contains(labels, "variables")
	s.Assert().Contains(labels, "values")
	s.Assert().Contains(labels, "children")
	s.Assert().Contains(labels, "datasources")
	s.Assert().Len(completionItems.Items, 5)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Kind)
		s.Assert().Equal(lsp.CompletionItemKindField, *item.Kind)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
		s.Assert().NotNil(item.Command, "progressive items should have retrigger command")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_top_level_with_prefix() {
	docCtx, position := s.createExportFieldDocContext("res")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "resources")
	s.Assert().Len(completionItems.Items, 1)
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_resource_ref_completions() {
	docCtx, position := s.createExportFieldDocContext("resources.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "ordersTable")
	s.Assert().Contains(labels, "saveOrderHandler")
	s.Assert().Len(completionItems.Items, 2)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Resource", *item.Detail)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
		s.Assert().NotNil(item.Command, "resource ref items should retrigger for property completion")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_resource_property_top_level() {
	docCtx, position := s.createExportFieldDocContext("resources.ordersTable.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "spec")
	s.Assert().Contains(labels, "metadata")
	s.Assert().Len(completionItems.Items, 2)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Property", *item.Detail)
		s.Assert().NotNil(item.Command, "property items should retrigger for deeper completion")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_resource_spec_fields() {
	docCtx, position := s.createExportFieldDocContext("resources.ordersTable.spec.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "tableName")
	s.Assert().Contains(labels, "id")
	s.Assert().Contains(labels, "billingMode")
	s.Assert().Len(completionItems.Items, 3)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Contains(*item.Detail, "Spec field")
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_resource_metadata_fields() {
	docCtx, position := s.createExportFieldDocContext("resources.ordersTable.metadata.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "displayName")
	s.Assert().Contains(labels, "labels")
	s.Assert().Contains(labels, "annotations")
	s.Assert().Contains(labels, "custom")
	s.Assert().Len(completionItems.Items, 4)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Metadata field", *item.Detail)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_variable_ref_completions() {
	docCtx, position := s.createExportFieldDocContext("variables.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "instanceType")
	s.Assert().Contains(labels, "environment")
	s.Assert().Len(completionItems.Items, 2)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Variable", *item.Detail)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
		// Variable refs are terminal (no deeper path), so no retrigger command
		s.Assert().Nil(item.Command, "variable ref items should be terminal (no retrigger)")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_value_ref_completions() {
	docCtx, position := s.createExportFieldDocContext("values.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "tableName")
	s.Assert().Len(completionItems.Items, 1)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Value", *item.Detail)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_child_ref_completions() {
	docCtx, position := s.createExportFieldDocContext("children.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "networking")
	s.Assert().Len(completionItems.Items, 1)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Child blueprint", *item.Detail)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
		s.Assert().NotNil(item.Command, "child ref items should retrigger for export name completion")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_child_property_nil_resolver() {
	docCtx, position := s.createExportFieldDocContext("children.networking.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	// ChildBlueprintResolver is nil in test setup, so no child export info is available
	s.Assert().Empty(completionItems.Items)
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_datasource_ref_completions() {
	docCtx, position := s.createExportFieldDocContext("datasources.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "network")
	s.Assert().Len(completionItems.Items, 1)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Data source", *item.Detail)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
		s.Assert().NotNil(item.Command, "datasource ref items should retrigger for export name completion")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_datasource_property_completions() {
	docCtx, position := s.createExportFieldDocContext("datasources.network.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems.Items)
	s.Assert().Contains(labels, "vpc")
	s.Assert().Contains(labels, "subnetIds")
	s.Assert().Len(completionItems.Items, 2)

	for _, item := range completionItems.Items {
		s.Assert().NotNil(item.Detail)
		s.Assert().Equal("Data source export", *item.Detail)
		s.Assert().Equal(map[string]any{"completionType": "exportField"}, item.Data)
		// Datasource exports are terminal, no retrigger
		s.Assert().Nil(item.Command, "datasource export items should be terminal (no retrigger)")
	}
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_resource_property_unknown_resource() {
	docCtx, position := s.createExportFieldDocContext("resources.nonExistent.")

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     position,
	})
	s.Require().NoError(err)
	s.Assert().Empty(completionItems.Items)
}

func (s *CompletionServiceGetItemsSuite) Test_get_export_field_resource_spec_nil_type() {
	// Blueprint with resource that has no type - spec field lookup should return empty
	blueprint := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"noTypeResource": {},
			},
		},
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"testExport": {
					Type: &schema.ExportTypeWrapper{Value: schema.ExportTypeString},
				},
			},
		},
	}

	editingContent := `version: 2024-07-20
resources:
  noTypeResource:
    spec:
      name: test
exports:
  testExport:
    type: string
    field: resources.noTypeResource.spec.`

	// Pass nil tree since SchemaToTree can't handle resources without Type
	docCtx := docmodel.NewDocumentContextFromSchema("file:///test.yaml", blueprint, nil)
	docCtx.Content = editingContent

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.yaml"},
		Position:     lsp.Position{Line: 8, Character: 39},
	})
	s.Require().NoError(err)
	s.Assert().Empty(completionItems.Items)
}
