package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type GotoDefinitionServiceSuite struct {
	suite.Suite
	service          *GotoDefinitionService
	blueprintContent string
	docCtx           *docmodel.DocumentContext
}

func (s *GotoDefinitionServiceSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	state.SetLinkSupportCapability(true)
	s.service = NewGotoDefinitionService(state, logger)
	s.blueprintContent, err = loadTestBlueprintContent("blueprint-definitions.yaml")
	s.Require().NoError(err)

	blueprint, err := schema.LoadString(s.blueprintContent, schema.YAMLSpecFormat)
	s.Require().NoError(err)

	tree := schema.SchemaToTree(blueprint)
	s.Require().NoError(err)

	s.docCtx = docmodel.NewDocumentContextFromSchema(
		"file:///blueprint.yaml",
		blueprint,
		tree,
	)
	s.docCtx.Content = s.blueprintContent
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_resource_ref() {
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      156,
			Character: 21,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      156,
					Character: 19,
				},
				End: lsp.Position{
					Line:      156,
					Character: 68,
				},
			},
			TargetURI: "file:///blueprint.yaml",
			TargetRange: lsp.Range{
				Start: lsp.Position{
					Line:      125,
					Character: 2,
				},
				End: lsp.Position{
					Line:      146,
					Character: 57,
				},
			},
			TargetSelectionRange: lsp.Range{
				Start: lsp.Position{
					Line:      125,
					Character: 2,
				},
				End: lsp.Position{
					Line:      146,
					Character: 57,
				},
			},
		},
	}, definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_datasource_ref() {
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      160,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      160,
					Character: 9,
				},
				End: lsp.Position{
					Line:      160,
					Character: 38,
				},
			},
			TargetURI: "file:///blueprint.yaml",
			TargetRange: lsp.Range{
				Start: lsp.Position{
					Line:      49,
					Character: 2,
				},
				End: lsp.Position{
					Line:      66,
					Character: 46,
				},
			},
			TargetSelectionRange: lsp.Range{
				Start: lsp.Position{
					Line:      49,
					Character: 2,
				},
				End: lsp.Position{
					Line:      66,
					Character: 46,
				},
			},
		},
	}, definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_var_ref() {
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      107,
			Character: 30,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      107,
					Character: 26,
				},
				End: lsp.Position{
					Line:      107,
					Character: 49,
				},
			},
			TargetURI: "file:///blueprint.yaml",
			TargetRange: lsp.Range{
				Start: lsp.Position{
					Line:      8,
					Character: 2,
				},
				End: lsp.Position{
					Line:      10,
					Character: 75,
				},
			},
			TargetSelectionRange: lsp.Range{
				Start: lsp.Position{
					Line:      8,
					Character: 2,
				},
				End: lsp.Position{
					Line:      10,
					Character: 75,
				},
			},
		},
	}, definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_val_ref() {
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      154,
			Character: 27,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      154,
					Character: 25,
				},
				End: lsp.Position{
					Line:      154,
					Character: 50,
				},
			},
			TargetURI: "file:///blueprint.yaml",
			TargetRange: lsp.Range{
				Start: lsp.Position{
					Line:      38,
					Character: 2,
				},
				End: lsp.Position{
					Line:      41,
					Character: 18,
				},
			},
			TargetSelectionRange: lsp.Range{
				Start: lsp.Position{
					Line:      38,
					Character: 2,
				},
				End: lsp.Position{
					Line:      41,
					Character: 18,
				},
			},
		},
	}, definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_child_ref() {
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      155,
			Character: 22,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      155,
					Character: 16,
				},
				End: lsp.Position{
					Line:      155,
					Character: 49,
				},
			},
			TargetURI: "file:///blueprint.yaml",
			TargetRange: lsp.Range{
				Start: lsp.Position{
					Line:      149,
					Character: 2,
				},
				End: lsp.Position{
					Line:      151,
					Character: 60,
				},
			},
			TargetSelectionRange: lsp.Range{
				Start: lsp.Position{
					Line:      149,
					Character: 2,
				},
				End: lsp.Position{
					Line:      151,
					Character: 60,
				},
			},
		},
	}, definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_returns_empty_list_for_a_non_ref_position() {
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      0,
			Character: 0,
		},
	})
	s.Require().NoError(err)
	s.Assert().Empty(definitions)
}

func TestGotoDefinitionServiceSuite(t *testing.T) {
	suite.Run(t, new(GotoDefinitionServiceSuite))
}
