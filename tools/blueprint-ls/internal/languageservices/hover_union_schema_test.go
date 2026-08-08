package languageservices

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

const unionSpecBlueprint = `version "2025-11-02"

resource testApi: test/api {
    spec {
        protocols = [
            "http",
            {
                websocketConfig = {
                    routeKey = "action"
                }
            }
        ]
        cors = {
            allowOrigins = ["http://localhost:3000"]
        }
        auth = {
            guards = {
                jwt = {
                    type = "jwt"
                }
            }
        }
    }
}
`

func (s *HoverExtendedSuite) unionSpecHoverService() *HoverService {
	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)
	funcRegistry := &testutils.FunctionRegistryMock{Functions: map[string]provider.Function{}}
	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"test/api": &unionSpecTestResource{},
		},
	}
	return NewHoverService(
		funcRegistry,
		resourceRegistry,
		&testutils.DataSourceRegistryMock{},
		nil,
		NewSignatureService(funcRegistry, logger),
		nil,
		logger,
	)
}

func (s *HoverExtendedSuite) hoverOnUnionSpecBlueprint(line, character int) *HoverContent {
	docCtx := s.loadBlueprintLang(unionSpecBlueprint)
	content, err := s.unionSpecHoverService().GetHoverContent(
		&common.LSPContext{},
		docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position: lsp.Position{
				Line:      lsp.UInteger(line),
				Character: lsp.UInteger(character),
			},
		},
	)
	s.Require().NoError(err)
	return content
}

func (s *HoverExtendedSuite) Test_hover_spec_field_in_union_array_item() {
	// "websocketConfig" only exists in one branch of the union that the
	// "protocols" array items are defined as.
	content := s.hoverOnUnionSpecBlueprint(7, 20)
	s.Assert().Contains(content.Value, "websocketConfig")
	s.Assert().Contains(content.Value, "WebSocket configuration for the API")
}

func (s *HoverExtendedSuite) Test_hover_nested_spec_field_in_union_array_item() {
	content := s.hoverOnUnionSpecBlueprint(8, 22)
	s.Assert().Contains(content.Value, "routeKey")
	s.Assert().Contains(content.Value, "route key for WebSocket messages")
}

func (s *HoverExtendedSuite) Test_hover_spec_field_in_union_object() {
	// "cors" is a union of a string shorthand and a configuration object.
	content := s.hoverOnUnionSpecBlueprint(13, 14)
	s.Assert().Contains(content.Value, "allowOrigins")
	s.Assert().Contains(content.Value, "origins allowed to access the API")
}

func (s *HoverExtendedSuite) Test_hover_spec_field_in_map_value_object() {
	content := s.hoverOnUnionSpecBlueprint(18, 22)
	s.Assert().Contains(content.Value, "type")
	s.Assert().Contains(content.Value, "The type of guard")
}

// unionSpecTestResource mirrors the shape of a transformer abstract resource
// that defines union spec fields, such as "celerity/api".
type unionSpecTestResource struct {
	provider.Resource
}

func (r *unionSpecTestResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"protocols": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeUnion,
							OneOf: []*provider.ResourceDefinitionsSchema{
								{
									Type: provider.ResourceDefinitionsSchemaTypeString,
									AllowedValues: []*core.MappingNode{
										core.MappingNodeFromString("http"),
										core.MappingNodeFromString("websocket"),
									},
								},
								{
									Type: provider.ResourceDefinitionsSchemaTypeObject,
									Attributes: map[string]*provider.ResourceDefinitionsSchema{
										"websocketConfig": {
											Type:        provider.ResourceDefinitionsSchemaTypeObject,
											Description: "WebSocket configuration for the API.",
											Attributes: map[string]*provider.ResourceDefinitionsSchema{
												"routeKey": {
													Type:        provider.ResourceDefinitionsSchemaTypeString,
													Description: "The route key for WebSocket messages.",
												},
											},
										},
									},
								},
							},
						},
					},
					"auth": {
						Type: provider.ResourceDefinitionsSchemaTypeObject,
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"guards": {
								Type: provider.ResourceDefinitionsSchemaTypeMap,
								MapValues: &provider.ResourceDefinitionsSchema{
									Type: provider.ResourceDefinitionsSchemaTypeObject,
									Attributes: map[string]*provider.ResourceDefinitionsSchema{
										"type": {
											Type:        provider.ResourceDefinitionsSchemaTypeString,
											Description: "The type of guard.",
										},
									},
								},
							},
						},
					},
					"cors": {
						Type: provider.ResourceDefinitionsSchemaTypeUnion,
						OneOf: []*provider.ResourceDefinitionsSchema{
							{
								Type: provider.ResourceDefinitionsSchemaTypeString,
							},
							{
								Type: provider.ResourceDefinitionsSchemaTypeObject,
								Attributes: map[string]*provider.ResourceDefinitionsSchema{
									"allowOrigins": {
										Type:        provider.ResourceDefinitionsSchemaTypeArray,
										Description: "The origins allowed to access the API.",
										Items: &provider.ResourceDefinitionsSchema{
											Type: provider.ResourceDefinitionsSchemaTypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}
