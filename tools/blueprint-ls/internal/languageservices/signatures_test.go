package languageservices

import (
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

type SignatureServiceSuite struct {
	suite.Suite
	service *SignatureService
	tree    *schema.TreeNode
}

func (s *SignatureServiceSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	funcRegistry := &testutils.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"replace":    corefunctions.NewReplaceFunction(),
			"replace_g":  corefunctions.NewReplace_G_Function(),
			"jsondecode": corefunctions.NewJSONDecodeFunction(),
			"list":       corefunctions.NewListFunction(),
			"object":     corefunctions.NewObjectFunction(),
			"join":       corefunctions.NewJoinFunction(),
			"map":        corefunctions.NewMapFunction(),
		},
	}
	s.service = NewSignatureService(funcRegistry, logger)
	content, err := loadTestBlueprintContent("blueprint-signature.yaml")
	s.Require().NoError(err)
	blueprint, err := schema.LoadString(content, schema.YAMLSpecFormat)
	s.Require().NoError(err)
	s.tree = schema.SchemaToTree(blueprint)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_scalar_types() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      81,
			Character: 42,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "replace(searchIn: string, searchFor: string, replaceWith: string) -> string",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "Replaces all occurrences of a substring in a string with another substring.\n\n" +
					"**Examples:**\n\n" +
					"```\n${replace(values.cacheClusterConfig.host, \"http://\", \"https://\")}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label: "searchIn: string",
					Documentation: "A valid string literal, reference or function call yielding a return value representing" +
						" an input string that contains a substring that needs replacing.",
				},
				{
					Label:         "searchFor: string",
					Documentation: "The \"search\" substring to replace.",
				},
				{
					Label:         "replaceWith: string",
					Documentation: "The substring to replace the \"search\" substring with.",
				},
			},
		},
	}, signatureInfo)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_function_return_type() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      51,
			Character: 49,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "replace_g(searchFor: string, replaceWith: string) -> function",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "A composable version of the \"replace\" function that takes the search and replace substrings " +
					"as static arguments and returns a function that takes the string to replace the substrings in.\n\n" +
					"**Examples:**\n\n" +
					"```\n${map(values.cacheClusterConfig.hosts, replace_g(\"http://\", \"https://\"))}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label:         "searchFor: string",
					Documentation: "The \"search\" substring to replace.",
				},
				{
					Label:         "replaceWith: string",
					Documentation: "The substring to replace the \"search\" substring with.",
				},
			},
		},
	}, signatureInfo)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_function_parameter() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      51,
			Character: 17,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "map(items: list[any], mapFunc: function) -> list[any]",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "Maps a list of values to a new list of values using a provided function.\n\n" +
					"**Examples:**\n\n" +
					"```\n${map(\n  datasources.network.subnets,\n  compose(to_upper, getattr(\"id\")\n)}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label:         "items: list[any]",
					Documentation: "An array of items where all items are of the same type to map.",
				},
				{
					Label:         "mapFunc: function",
					Documentation: "The function to apply to each element in the list.",
				},
			},
		},
	}, signatureInfo)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_any_return_type() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      46,
			Character: 25,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "jsondecode(jsonString: string) -> any",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "Decodes a serialised json string into a primitive value, array or mapping.\n\n" +
					"**Examples:**\n\n" +
					"```\n${jsondecode(variables.cacheClusterConfig)}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label:         "jsonString: string",
					Documentation: "A valid string literal, reference or function call yielding the json string to decode.",
				},
			},
		},
	}, signatureInfo)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_list_parameter() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      154,
			Character: 18,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "join(strings: list[string], delimiter: string) -> string",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "Joins an array of strings into a single string using a delimiter.\n\n" +
					"**Examples:**\n\n" +
					"```\n${join(values.cacheClusterConfig.hosts, \",\")}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label:         "strings: list[string]",
					Documentation: "A reference or function call yielding a return value representing an array of strings to join together.",
				},
				{
					Label:         "delimiter: string",
					Documentation: "The delimiter to join the strings with.",
				},
			},
		},
	}, signatureInfo)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_variadic_params() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      155,
			Character: 22,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "object(attributes: any...) -> object",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "Creates an object from named arguments.\n\n" +
					"**Examples:**\n\n" +
					"```\n${object(id=\"subnet-1234\", label=\"Subnet 1234\")}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label: "attributes: any...",
					Documentation: "N named arguments that will be used to create an object/mapping. " +
						"When no arguments are passed, an empty object should be returned.",
				},
			},
		},
	}, signatureInfo)
}

func (s *SignatureServiceSuite) Test_produces_function_signature_definition_with_list_return_type() {
	lspCtx := &common.LSPContext{}
	signatureInfo, err := s.service.GetFunctionSignatures(lspCtx, s.tree, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: blueprintURI,
		},
		Position: lsp.Position{
			Line:      156,
			Character: 21,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]*lsp.SignatureInformation{
		{
			Label: "list(values: any...) -> list[any]",
			Documentation: lsp.MarkupContent{
				Kind: lsp.MarkupKindMarkdown,
				Value: "Creates a list of values from arguments of the same type.\n\n" +
					"**Examples:**\n\n" +
					"```\n${list(\"item1\",\"item2\",\"item3\",\"item4\")}\n```",
			},
			Parameters: []*lsp.ParameterInformation{
				{
					Label:         "values: any...",
					Documentation: "N arguments of the same type that will be used to create a list.",
				},
			},
		},
	}, signatureInfo)
}

func TestSignatureServiceSuite(t *testing.T) {
	suite.Run(t, new(SignatureServiceSuite))
}
