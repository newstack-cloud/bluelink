package languageservices

import (
	"path/filepath"
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
	state            *State
	logger           *zap.Logger
	blueprintContent string
	docCtx           *docmodel.DocumentContext
}

func (s *GotoDefinitionServiceSuite) SetupTest() {
	var err error
	s.logger, err = zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	s.state = NewState()
	s.state.SetLinkSupportCapability(true)
	s.service = NewGotoDefinitionService(s.state, nil, s.logger)
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
			Line:      173,
			Character: 21,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      173,
					Character: 19,
				},
				End: lsp.Position{
					Line:      173,
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
			Line:      177,
			Character: 11,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      177,
					Character: 9,
				},
				End: lsp.Position{
					Line:      177,
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
			Line:      171,
			Character: 27,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      171,
					Character: 25,
				},
				End: lsp.Position{
					Line:      171,
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
			Line:      172,
			Character: 22,
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal([]lsp.LocationLink{
		{
			OriginSelectionRange: &lsp.Range{
				Start: lsp.Position{
					Line:      172,
					Character: 16,
				},
				End: lsp.Position{
					Line:      172,
					Character: 49,
				},
			},
			TargetURI: "file:///blueprint.yaml",
			TargetRange: lsp.Range{
				Start: lsp.Position{
					Line:      166,
					Character: 2,
				},
				End: lsp.Position{
					Line:      168,
					Character: 60,
				},
			},
			TargetSelectionRange: lsp.Range{
				Start: lsp.Position{
					Line:      166,
					Character: 2,
				},
				End: lsp.Position{
					Line:      168,
					Character: 60,
				},
			},
		},
	}, definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_export_field_resource_ref() {
	// Line 190 (0-indexed: 189): field: resources.getOrderHandler.spec.handlerName
	// Cursor on the field value, character 15 is within "getOrderHandler".
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      189,
			Character: 15,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be the getOrderHandler resource (lines 126-147, 0-indexed: 125-146)
	s.Assert().Equal(lsp.Position{Line: 125, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_export_field_variable_ref() {
	// Line 186 (0-indexed: 185): field: variables.certificateId
	// Cursor on the field value, character 15 is within "certificateId".
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      185,
			Character: 15,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be the certificateId variable (lines 9-11, 0-indexed: 8-10)
	s.Assert().Equal(lsp.Position{Line: 8, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_export_field_child_ref() {
	// Line 194 (0-indexed: 193): field: children.networking.someExport
	// Cursor on the field value, character 15 is within "networking".
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      193,
			Character: 15,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be the networking include (lines 167-169, 0-indexed: 166-168)
	s.Assert().Equal(lsp.Position{Line: 166, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_export_field_datasource_ref() {
	// Line 198 (0-indexed: 197): field: datasources.network.vpc
	// Cursor on the field value, character 18 is within "network".
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      197,
			Character: 18,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be the network datasource (lines 50-67, 0-indexed: 49-66)
	s.Assert().Equal(lsp.Position{Line: 49, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_export_field_value_ref() {
	// Line 202 (0-indexed: 201): field: values.derivitiveCertInfo
	// Cursor on the field value, character 15 is within "derivitiveCertInfo".
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      201,
			Character: 15,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be the derivitiveCertInfo value (lines 39-42, 0-indexed: 38-41)
	s.Assert().Equal(lsp.Position{Line: 38, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_export_name_returns_empty() {
	// Line 184 (0-indexed: 183): certificateId:
	// Hovering on the export name (not the field value) should produce no definitions.
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      183,
			Character: 5,
		},
	})
	s.Require().NoError(err)
	s.Assert().Empty(definitions)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_exclude_item() {
	// Line 155 (0-indexed: 154): - resource53
	// Cursor on "resource53" in the exclude list.
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      154,
			Character: 12,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be resource53 (lines 70-77, 0-indexed: 69-76)
	s.Assert().Equal(lsp.Position{Line: 69, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_depends_on_item() {
	// Line 162 (0-indexed: 161): - getOrderHandler
	// Cursor on "getOrderHandler" in the dependsOn list.
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      161,
			Character: 10,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)
	s.Assert().Equal("file:///blueprint.yaml", string(definitions[0].TargetURI))
	// Target should be getOrderHandler resource (lines 126-147, 0-indexed: 125-146)
	s.Assert().Equal(lsp.Position{Line: 125, Character: 2}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_include_path() {
	childResolver := NewChildBlueprintResolver(s.logger)
	service := NewGotoDefinitionService(s.state, childResolver, s.logger)

	absTestdataDir, err := filepath.Abs("__testdata")
	s.Require().NoError(err)
	realURI := "file://" + filepath.Join(absTestdataDir, "blueprint-definitions.yaml")

	docCtx := docmodel.NewDocumentContextFromSchema(
		realURI, s.docCtx.Blueprint, s.docCtx.SchemaTree,
	)
	docCtx.Content = s.blueprintContent

	// Line 168 (0-indexed: 167): path: networking.blueprint.yaml
	definitions, err := service.GetDefinitionsFromContext(docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: lsp.URI(realURI)},
		Position:     lsp.Position{Line: 167, Character: 12},
	})
	s.Require().NoError(err)
	s.Require().Len(definitions, 1)

	expectedTargetURI := "file://" + filepath.Join(absTestdataDir, "networking.blueprint.yaml")
	s.Assert().Equal(expectedTargetURI, string(definitions[0].TargetURI))
	s.Assert().Equal(lsp.Position{Line: 0, Character: 0}, definitions[0].TargetRange.Start)
}

func (s *GotoDefinitionServiceSuite) Test_get_definitions_for_include_path_returns_empty_for_unresolvable() {
	// With nil childResolver, include path definitions return empty.
	definitions, err := s.service.GetDefinitionsFromContext(s.docCtx, &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file:///blueprint.yaml",
		},
		Position: lsp.Position{
			Line:      167,
			Character: 12,
		},
	})
	s.Require().NoError(err)
	s.Assert().Empty(definitions)
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
