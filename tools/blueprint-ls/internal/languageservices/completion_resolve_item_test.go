package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type CompletionServiceResolveItemSuite struct {
	suite.Suite
	service *CompletionService
}

func (s *CompletionServiceResolveItemSuite) SetupTest() {
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
	s.service = NewCompletionService(resourceRegistry, dataSourceRegistry, customVarTypeRegistry, functionRegistry, nil, state, logger)
}

func (s *CompletionServiceResolveItemSuite) Test_resolves_resource_type_docs() {
	itemKind := lsp.CompletionItemKindEnum
	item := &lsp.CompletionItem{
		Label: "aws/dynamodb/table",
		Kind:  &itemKind,
	}
	lspCtx := &common.LSPContext{}
	resolvedItem, err := s.service.ResolveCompletionItem(lspCtx, item, "resourceType")
	s.Require().NoError(err)
	s.Assert().Equal(lsp.MarkupContent{
		Kind:  lsp.MarkupKindMarkdown,
		Value: "# DynamoDB Table\n\nA table in DynamoDB.",
	}, resolvedItem.Documentation)
}

func (s *CompletionServiceResolveItemSuite) Test_resolves_datasource_type_docs() {
	itemKind := lsp.CompletionItemKindEnum
	item := &lsp.CompletionItem{
		Label: "aws/vpc",
		Kind:  &itemKind,
	}
	lspCtx := &common.LSPContext{}
	resolvedItem, err := s.service.ResolveCompletionItem(lspCtx, item, "dataSourceType")
	s.Require().NoError(err)
	s.Assert().Equal(lsp.MarkupContent{
		Kind:  lsp.MarkupKindMarkdown,
		Value: "# VPC\n\n A Virtual Private Cloud (VPC) in AWS.",
	}, resolvedItem.Documentation)
}

func (s *CompletionServiceResolveItemSuite) Test_resolves_custom_variable_type_docs() {
	itemKind := lsp.CompletionItemKindEnum
	item := &lsp.CompletionItem{
		Label: "aws/ec2/instanceType",
		Kind:  &itemKind,
	}
	lspCtx := &common.LSPContext{}
	resolvedItem, err := s.service.ResolveCompletionItem(lspCtx, item, "variableType")
	s.Require().NoError(err)
	s.Assert().Equal(lsp.MarkupContent{
		Kind:  lsp.MarkupKindMarkdown,
		Value: "# EC2 Instance Type\n\nAn EC2 instance type.",
	}, resolvedItem.Documentation)
}

func (s *CompletionServiceResolveItemSuite) Test_resolves_function_docs() {
	itemKind := lsp.CompletionItemKindEnum
	item := &lsp.CompletionItem{
		Label: "len",
		Kind:  &itemKind,
	}
	lspCtx := &common.LSPContext{}
	resolvedItem, err := s.service.ResolveCompletionItem(lspCtx, item, "function")
	s.Require().NoError(err)
	s.Assert().Equal(lsp.MarkupContent{
		Kind: lsp.MarkupKindMarkdown,
		Value: "Get the length of a string, array, or mapping.\n\n" +
			"**Examples:**\n\n" +
			"```\n${len(values.cacheClusterConfig.endpoints)}\n```",
	}, resolvedItem.Documentation)
}

func TestCompletionServiceResolveItemSuite(t *testing.T) {
	suite.Run(t, new(CompletionServiceResolveItemSuite))
}
