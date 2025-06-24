package languageservices

import (
	"path"
	"slices"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type CompletionServiceGetItemsSuite struct {
	suite.Suite
	service *CompletionService
}

func (s *CompletionServiceGetItemsSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	state.SetLinkSupportCapability(true)
	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/dynamodb/table": &testutils.DynamoDBTableResource{},
		},
	}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{
			"aws/vpc": &testutils.VPCDataSource{},
		},
	}
	customVarTypeRegistry := &testutils.CustomVarTypeRegistryMock{
		CustomVarTypes: map[string]provider.CustomVariableType{
			"aws/ec2/instanceType": &testutils.InstanceTypeCustomVariableType{},
		},
	}
	functionRegistry := &testutils.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"len": corefunctions.NewLenFunction(),
		},
	}
	s.service = NewCompletionService(resourceRegistry, dataSourceRegistry, customVarTypeRegistry, functionRegistry, state, logger)
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_variable_ref() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-variable-ref")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      31,
			Character: 24,
		},
	})
	s.Require().NoError(err)
	detail := "Variable"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "instanceType",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      31,
						Character: 24,
					},
					End: lsp.Position{
						Line:      31,
						Character: 24,
					},
				},
				NewText: "instanceType",
			},
			Data: map[string]any{
				"completionType": "variable",
			},
		},
	}, completionItems)
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_value_ref() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-value-ref")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      38,
			Character: 21,
		},
	})
	s.Require().NoError(err)
	detail := "Value"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "tableName",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      38,
						Character: 21,
					},
					End: lsp.Position{
						Line:      38,
						Character: 21,
					},
				},
				NewText: "tableName",
			},
			Data: map[string]any{
				"completionType": "value",
			},
		},
	}, completionItems)
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_ref() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-datasource-ref")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      38,
			Character: 26,
		},
	})
	s.Require().NoError(err)
	detail := "Data source"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "network",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      38,
						Character: 26,
					},
					End: lsp.Position{
						Line:      38,
						Character: 26,
					},
				},
				NewText: "network",
			},
			Data: map[string]any{
				"completionType": "dataSource",
			},
		},
	}, completionItems)
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_property_ref() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-datasource-property-ref")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      38,
			Character: 34,
		},
	})
	s.Require().NoError(err)
	detail := "Data source exported field"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "vpc",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      38,
						Character: 34,
					},
					End: lsp.Position{
						Line:      38,
						Character: 34,
					},
				},
				NewText: "vpc",
			},
			Data: map[string]any{
				"completionType": "dataSourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "subnetIds",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      38,
						Character: 34,
					},
					End: lsp.Position{
						Line:      38,
						Character: 34,
					},
				},
				NewText: "subnetIds",
			},
			Data: map[string]any{
				"completionType": "dataSourceProperty",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_child_ref() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-child-ref")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      43,
			Character: 23,
		},
	})
	s.Require().NoError(err)
	detail := "Child blueprint"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "networking",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      43,
						Character: 23,
					},
					End: lsp.Position{
						Line:      43,
						Character: 23,
					},
				},
				NewText: "networking",
			},
			Data: map[string]any{
				"completionType": "child",
			},
		},
	}, completionItems)
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_ref_1() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-resource-ref-1")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      46,
			Character: 29,
		},
	})
	s.Require().NoError(err)
	// Should present all possible reference completion items
	// as this test is for a global identifier used to reference a resource.
	// The LSP client should filter the results based on the context.
	expectedLabels := []string{
		"resources.ordersTable",
		"resources.saveOrderHandler",
		"variables.instanceType",
		"variables.environment",
		"len",
		"datasources.network",
		"values.tableName",
	}
	slices.Sort(expectedLabels)
	s.Assert().Equal(expectedLabels, completionItemLabels(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_ref_2() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-resource-ref-2")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      46,
			Character: 36,
		},
	})
	s.Require().NoError(err)
	// Completion is for "resources." namespaced reference which should only
	// yield resource reference completion items.
	detail := "Resource"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "ordersTable",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 36,
					},
					End: lsp.Position{
						Line:      46,
						Character: 36,
					},
				},
				NewText: "ordersTable",
			},
			Data: map[string]any{
				"completionType": "resource",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "saveOrderHandler",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 36,
					},
					End: lsp.Position{
						Line:      46,
						Character: 36,
					},
				},
				NewText: "saveOrderHandler",
			},
			Data: map[string]any{
				"completionType": "resource",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_property_ref_1() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-resource-property-ref-1")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      46,
			Character: 48,
		},
	})
	s.Require().NoError(err)
	detail := "Resource property"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "spec",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 48,
					},
					End: lsp.Position{
						Line:      46,
						Character: 48,
					},
				},
				NewText: "spec",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "state",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 48,
					},
					End: lsp.Position{
						Line:      46,
						Character: 48,
					},
				},
				NewText: "state",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "metadata",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 48,
					},
					End: lsp.Position{
						Line:      46,
						Character: 48,
					},
				},
				NewText: "metadata",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_property_ref_2() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-resource-property-ref-2")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      46,
			Character: 53,
		},
	})
	s.Require().NoError(err)
	detail := "Resource spec property"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "tableName",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 53,
					},
					End: lsp.Position{
						Line:      46,
						Character: 53,
					},
				},
				NewText: "tableName",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "id",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 53,
					},
					End: lsp.Position{
						Line:      46,
						Character: 53,
					},
				},
				NewText: "id",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_property_ref_4() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-resource-property-ref-4")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      46,
			Character: 57,
		},
	})
	s.Require().NoError(err)
	detail := "Resource metadata property"
	itemKind := lsp.CompletionItemKindField
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "annotations",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 57,
					},
					End: lsp.Position{
						Line:      46,
						Character: 57,
					},
				},
				NewText: "annotations",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "custom",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 57,
					},
					End: lsp.Position{
						Line:      46,
						Character: 57,
					},
				},
				NewText: "custom",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "displayName",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 57,
					},
					End: lsp.Position{
						Line:      46,
						Character: 57,
					},
				},
				NewText: "displayName",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "labels",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      46,
						Character: 57,
					},
					End: lsp.Position{
						Line:      46,
						Character: 57,
					},
				},
				NewText: "labels",
			},
			Data: map[string]any{
				"completionType": "resourceProperty",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_resource_type() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-resource-type")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      35,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	detail := "Resource type"
	itemKind := lsp.CompletionItemKindEnum
	insertText := "aws/dynamodb/table"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "aws/dynamodb/table",
			Detail:     &detail,
			InsertText: &insertText,
			Data: map[string]any{
				"completionType": "resourceType",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_type() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-datasource-type")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      15,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	detail := "Data source type"
	itemKind := lsp.CompletionItemKindEnum
	insertText := "aws/vpc"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "aws/vpc",
			Detail:     &detail,
			InsertText: &insertText,
			Data: map[string]any{
				"completionType": "dataSourceType",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_variable_type() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-variable-type")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      3,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	detail := "Variable type"
	itemKind := lsp.CompletionItemKindEnum
	insertTextInstanceType := "aws/ec2/instanceType"
	insertTextBool := "boolean"
	insertTextFloat := "float"
	insertTextInteger := "integer"
	insertTextString := "string"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "aws/ec2/instanceType",
			Detail:     &detail,
			InsertText: &insertTextInstanceType,
			Data: map[string]any{
				"completionType": "variableType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "boolean",
			Detail:     &detail,
			InsertText: &insertTextBool,
			Data: map[string]any{
				"completionType": "variableType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "float",
			Detail:     &detail,
			InsertText: &insertTextFloat,
			Data: map[string]any{
				"completionType": "variableType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "integer",
			Detail:     &detail,
			InsertText: &insertTextInteger,
			Data: map[string]any{
				"completionType": "variableType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "string",
			Detail:     &detail,
			InsertText: &insertTextString,
			Data: map[string]any{
				"completionType": "variableType",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_value_type() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-value-type")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      10,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	detail := "Value type"
	itemKind := lsp.CompletionItemKindEnum
	insertTextBool := "boolean"
	insertTextFloat := "float"
	insertTextInteger := "integer"
	insertTextString := "string"
	insertTextArray := "array"
	insertTextObject := "object"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "boolean",
			Detail:     &detail,
			InsertText: &insertTextBool,
			Data: map[string]any{
				"completionType": "valueType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "float",
			Detail:     &detail,
			InsertText: &insertTextFloat,
			Data: map[string]any{
				"completionType": "valueType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "integer",
			Detail:     &detail,
			InsertText: &insertTextInteger,
			Data: map[string]any{
				"completionType": "valueType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "string",
			Detail:     &detail,
			InsertText: &insertTextString,
			Data: map[string]any{
				"completionType": "valueType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "array",
			Detail:     &detail,
			InsertText: &insertTextArray,
			Data: map[string]any{
				"completionType": "valueType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "object",
			Detail:     &detail,
			InsertText: &insertTextObject,
			Data: map[string]any{
				"completionType": "valueType",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_field_type() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-datasource-field-type")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      25,
			Character: 15,
		},
	})
	s.Require().NoError(err)
	detail := "Data source field type"
	itemKind := lsp.CompletionItemKindEnum
	insertTextBool := "boolean"
	insertTextFloat := "float"
	insertTextInteger := "integer"
	insertTextString := "string"
	insertTextArray := "array"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "boolean",
			Detail:     &detail,
			InsertText: &insertTextBool,
			Data: map[string]any{
				"completionType": "dataSourceFieldType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "float",
			Detail:     &detail,
			InsertText: &insertTextFloat,
			Data: map[string]any{
				"completionType": "dataSourceFieldType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "integer",
			Detail:     &detail,
			InsertText: &insertTextInteger,
			Data: map[string]any{
				"completionType": "dataSourceFieldType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "string",
			Detail:     &detail,
			InsertText: &insertTextString,
			Data: map[string]any{
				"completionType": "dataSourceFieldType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "array",
			Detail:     &detail,
			InsertText: &insertTextArray,
			Data: map[string]any{
				"completionType": "dataSourceFieldType",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_filter_field() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-datasource-filter-field")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      18,
			Character: 14,
		},
	})
	s.Require().NoError(err)
	detail := "Data source filter field"
	itemKind := lsp.CompletionItemKindEnum
	insertTextInstanceConfigId := "instanceConfigId"
	insertTextTags := "tags"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "instanceConfigId",
			Detail:     &detail,
			InsertText: &insertTextInstanceConfigId,
			Data: map[string]any{
				"completionType": "dataSourceFilterField",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "tags",
			Detail:     &detail,
			InsertText: &insertTextTags,
			Data: map[string]any{
				"completionType": "dataSourceFilterField",
			},
		},
	}), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_datasource_filter_operator() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-datasource-filter-operator")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      19,
			Character: 17,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(sortCompletionItems(expectedDataSourceFilterOperatorItems()), sortCompletionItems(completionItems))
}

func (s *CompletionServiceGetItemsSuite) Test_get_completion_items_for_export_type() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-export-type")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(lspCtx, blueprintInfo.content, blueprintInfo.tree, blueprintInfo.blueprint, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      47,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	detail := "Export type"
	itemKind := lsp.CompletionItemKindEnum
	insertTextBool := "boolean"
	insertTextFloat := "float"
	insertTextInteger := "integer"
	insertTextString := "string"
	insertTextArray := "array"
	insertTextObject := "object"
	s.Assert().Equal(sortCompletionItems([]*lsp.CompletionItem{
		{
			Kind:       &itemKind,
			Label:      "boolean",
			Detail:     &detail,
			InsertText: &insertTextBool,
			Data: map[string]any{
				"completionType": "exportType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "float",
			Detail:     &detail,
			InsertText: &insertTextFloat,
			Data: map[string]any{
				"completionType": "exportType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "integer",
			Detail:     &detail,
			InsertText: &insertTextInteger,
			Data: map[string]any{
				"completionType": "exportType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "string",
			Detail:     &detail,
			InsertText: &insertTextString,
			Data: map[string]any{
				"completionType": "exportType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "array",
			Detail:     &detail,
			InsertText: &insertTextArray,
			Data: map[string]any{
				"completionType": "exportType",
			},
		},
		{
			Kind:       &itemKind,
			Label:      "object",
			Detail:     &detail,
			InsertText: &insertTextObject,
			Data: map[string]any{
				"completionType": "exportType",
			},
		},
	}), sortCompletionItems(completionItems))
}

type testBlueprintInfo struct {
	blueprint *schema.Blueprint
	tree      *schema.TreeNode
	content   string
}

func loadCompletionBlueprintAndTree(name string) (*testBlueprintInfo, error) {
	// Load and parse the blueprint content before the completion trigger character.
	// This is required as when a completion trigger character is entered, the current
	// state of the document will not be successfully parsed and the completion service
	// will be working with the parsed version of the document before the trigger character
	// was entered.
	// There will often be multiple characters between the last valid state of the document
	// and the completion trigger. For example, in the case of a variable reference completion,
	// the last valid state would be "${variable}" as it will match an identifier token in the
	// substitution language.
	// The sequence before the completion trigger in this case would be:
	// "${variable}" (valid) -> "${variables}" (invalid) -> "${variables.}"
	// "variables" is a keyword that must be followed by ".<variableName>" to be valid.
	contentBefore, err := loadTestBlueprintContent(path.Join(name, "before-completion-trigger.yaml"))
	if err != nil {
		return nil, err
	}

	blueprint, err := schema.LoadString(contentBefore, schema.YAMLSpecFormat)
	if err != nil {
		return nil, err
	}

	tree := schema.SchemaToTree(blueprint)

	// Load the content after the completion trigger character.
	afterTriggerContent, err := loadTestBlueprintContent(path.Join(name, "after-completion-trigger.yaml"))
	if err != nil {
		return nil, err
	}

	return &testBlueprintInfo{
		blueprint: blueprint,
		tree:      tree,
		content:   afterTriggerContent,
	}, nil
}

func completionItemLabels(completionItems []*lsp.CompletionItem) []string {
	labels := make([]string, len(completionItems))
	for i, item := range completionItems {
		labels[i] = item.Label
	}
	slices.Sort(labels)
	return labels
}

func sortCompletionItems(completionItems []*lsp.CompletionItem) []*lsp.CompletionItem {
	items := make([]*lsp.CompletionItem, len(completionItems))
	copy(items, completionItems)
	slices.SortFunc(items, func(a, b *lsp.CompletionItem) int {
		if a.Label < b.Label {
			return -1
		} else if a.Label > b.Label {
			return 1
		} else {
			return 0
		}
	})
	return items
}

func expectedDataSourceFilterOperatorItems() []*lsp.CompletionItem {
	detail := "Data source filter operator"
	itemKind := lsp.CompletionItemKindEnum
	return []*lsp.CompletionItem{
		{
			Kind:   &itemKind,
			Label:  "\"!=\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 20,
					},
				},
				NewText: "\"!=\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"=\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 19,
					},
				},
				NewText: "\"=\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"contains\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 26,
					},
				},
				NewText: "\"contains\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"ends with\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 27,
					},
				},
				NewText: "\"ends with\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"has key\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 25,
					},
				},
				NewText: "\"has key\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"in\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 20,
					},
				},
				NewText: "\"in\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"not contains\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 30,
					},
				},
				NewText: "\"not contains\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"not ends with\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 31,
					},
				},
				NewText: "\"not ends with\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"not has key\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 29,
					},
				},
				NewText: "\"not has key\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"not in\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 24,
					},
				},
				NewText: "\"not in\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"not starts with\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 33,
					},
				},
				NewText: "\"not starts with\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"starts with\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 29,
					},
				},
				NewText: "\"starts with\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\">\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 19,
					},
				},
				NewText: "\">\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"<\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 19,
					},
				},
				NewText: "\"<\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\">=\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 20,
					},
				},
				NewText: "\">=\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
		{
			Kind:   &itemKind,
			Label:  "\"<=\"",
			Detail: &detail,
			TextEdit: lsp.TextEdit{
				Range: &lsp.Range{
					Start: lsp.Position{
						Line:      19,
						Character: 16,
					},
					End: lsp.Position{
						Line:      19,
						Character: 20,
					},
				},
				NewText: "\"<=\"",
			},
			Data: map[string]any{
				"completionType": "dataSourceFilterOperator",
			},
		},
	}
}

func TestCompletionServiceGetItemsSuite(t *testing.T) {
	suite.Run(t, new(CompletionServiceGetItemsSuite))
}
