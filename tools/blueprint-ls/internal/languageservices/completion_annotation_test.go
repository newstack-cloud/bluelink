package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// AnnotationKeyCompletionSuite tests annotation key completion functionality.
type AnnotationKeyCompletionSuite struct {
	suite.Suite
	service      *CompletionService
	linkRegistry *testutils.LinkRegistryMock
}

func (s *AnnotationKeyCompletionSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	state.SetLinkSupportCapability(true)

	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/dynamodb/table":  &testutils.DynamoDBTableResource{},
			"aws/lambda/function": &testutils.LambdaFunctionResource{},
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

	s.linkRegistry = &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/dynamodb/table": &testutils.MockLink{
				AnnotationDefs: map[string]*provider.LinkAnnotationDefinition{
					"aws/lambda/function::aws.lambda.dynamodb.accessType": {
						Name:        "aws.lambda.dynamodb.accessType",
						Label:       "DynamoDB Access Type",
						Type:        core.ScalarTypeString,
						Description: "The type of access the Lambda function has to the DynamoDB table.",
						AllowedValues: []*core.ScalarValue{
							core.ScalarFromString("read"),
							core.ScalarFromString("write"),
							core.ScalarFromString("readwrite"),
						},
					},
					"aws/lambda/function::aws.lambda.dynamodb.<tableName>.accessType": {
						Name:        "aws.lambda.dynamodb.<tableName>.accessType",
						Label:       "Table-specific Access Type",
						Type:        core.ScalarTypeString,
						Description: "Access type for a specific DynamoDB table.",
					},
				},
			},
		},
	}

	s.service = NewCompletionService(
		resourceRegistry,
		dataSourceRegistry,
		customVarTypeRegistry,
		functionRegistry,
		state,
		logger,
	)
	s.service.linkRegistry = s.linkRegistry
}

func TestAnnotationKeyCompletionSuite(t *testing.T) {
	suite.Run(t, new(AnnotationKeyCompletionSuite))
}

func (s *AnnotationKeyCompletionSuite) Test_get_completion_items_for_annotation_key_yaml() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-annotation-key")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx,
		blueprintInfo.toDocumentContextWithTreeSitter(),
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 7, Character: 8},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems)
	s.Assert().Contains(labels, "aws.lambda.dynamodb.accessType")
	s.Assert().Contains(labels, "aws.lambda.dynamodb.ordersTable.accessType")
}

func (s *AnnotationKeyCompletionSuite) Test_get_completion_items_for_annotation_key_jsonc_disabled() {
	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-annotation-key")
	s.Require().NoError(err)

	docCtx := blueprintInfo.toDocumentContextWithTreeSitter()
	docCtx.Format = docmodel.FormatJSONC

	lspCtx := &common.LSPContext{}
	completionItems, err := s.service.GetCompletionItems(
		lspCtx,
		docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 7, Character: 8},
		},
	)
	s.Require().NoError(err)
	s.Assert().Empty(completionItems, "JSONC annotation key completions should be disabled")
}

func (s *AnnotationKeyCompletionSuite) Test_get_completion_items_for_annotation_key_no_links() {
	// Create service without any link registry
	logger, _ := zap.NewDevelopment()
	state := NewState()
	serviceNoLinks := NewCompletionService(
		&testutils.ResourceRegistryMock{
			Resources: map[string]provider.Resource{
				"aws/lambda/function": &testutils.LambdaFunctionResource{},
			},
		},
		&testutils.DataSourceRegistryMock{},
		&testutils.CustomVarTypeRegistryMock{},
		&testutils.FunctionRegistryMock{},
		state,
		logger,
	)
	// linkRegistry is nil by default

	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-annotation-key")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := serviceNoLinks.GetCompletionItems(
		lspCtx,
		blueprintInfo.toDocumentContextWithTreeSitter(),
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 7, Character: 8},
		},
	)
	s.Require().NoError(err)
	s.Assert().Empty(completionItems, "Should return empty when no link registry")
}

func (s *AnnotationKeyCompletionSuite) Test_expand_annotation_name_with_placeholder() {
	linkedNames := []string{"ordersTable", "usersTable"}

	// Test static annotation name (no placeholder)
	result := expandAnnotationName("aws.lambda.dynamodb.accessType", linkedNames)
	s.Assert().Equal([]string{"aws.lambda.dynamodb.accessType"}, result)

	// Test dynamic annotation name with placeholder
	result = expandAnnotationName("aws.lambda.dynamodb.<tableName>.accessType", linkedNames)
	s.Assert().Len(result, 2)
	s.Assert().Contains(result, "aws.lambda.dynamodb.ordersTable.accessType")
	s.Assert().Contains(result, "aws.lambda.dynamodb.usersTable.accessType")

	// Test with empty linked names
	result = expandAnnotationName("aws.lambda.dynamodb.<tableName>.accessType", []string{})
	s.Assert().Empty(result)
}

func (s *AnnotationKeyCompletionSuite) Test_get_completion_items_filters_by_applies_to() {
	// Create a link registry with annotations that have different AppliesTo values
	logger, _ := zap.NewDevelopment()
	state := NewState()
	state.SetLinkSupportCapability(true)

	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/lambda/function": &testutils.LambdaFunctionResource{},
		},
	}

	// Create a link between two functions (same type) with AppliesTo filtering
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/lambda/function": &testutils.MockLink{
				AnnotationDefs: map[string]*provider.LinkAnnotationDefinition{
					"aws/lambda/function::aws.lambda.caller.timeout": {
						Name:        "aws.lambda.caller.timeout",
						Label:       "Caller Timeout",
						Type:        core.ScalarTypeInteger,
						Description: "Timeout for the calling function.",
						AppliesTo:   provider.LinkAnnotationResourceA,
					},
					"aws/lambda/function::aws.lambda.callee.maxConcurrency": {
						Name:        "aws.lambda.callee.maxConcurrency",
						Label:       "Callee Max Concurrency",
						Type:        core.ScalarTypeInteger,
						Description: "Max concurrency for the called function.",
						AppliesTo:   provider.LinkAnnotationResourceB,
					},
					"aws/lambda/function::aws.lambda.shared.retries": {
						Name:        "aws.lambda.shared.retries",
						Label:       "Shared Retries",
						Type:        core.ScalarTypeInteger,
						Description: "Retry count applicable to both functions.",
						AppliesTo:   provider.LinkAnnotationResourceAny,
					},
				},
			},
		},
	}

	service := NewCompletionService(
		resourceRegistry,
		&testutils.DataSourceRegistryMock{},
		&testutils.CustomVarTypeRegistryMock{},
		&testutils.FunctionRegistryMock{},
		state,
		logger,
	)
	service.linkRegistry = linkRegistry

	blueprintInfo, err := loadCompletionBlueprintAndTree("blueprint-completion-annotation-key-applies-to")
	s.Require().NoError(err)

	lspCtx := &common.LSPContext{}
	completionItems, err := service.GetCompletionItems(
		lspCtx,
		blueprintInfo.toDocumentContextWithTreeSitter(),
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 7, Character: 8},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(completionItems)
	// callerFunction is A in the link (it references targetFunction)
	// So it should get AppliesTo=A annotations and AppliesTo=Any with matching key type
	s.Assert().Contains(labels, "aws.lambda.caller.timeout", "Should include AppliesTo=A annotation")
	s.Assert().Contains(labels, "aws.lambda.shared.retries", "Should include AppliesTo=Any annotation")
	s.Assert().NotContains(labels, "aws.lambda.callee.maxConcurrency", "Should not include AppliesTo=B annotation")
}
