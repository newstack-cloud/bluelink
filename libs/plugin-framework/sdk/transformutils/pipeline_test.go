package transformutils

import (
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	testTransformerID = "celerity-test-2026"
	testTarget        = Target("test-target")
)

type RunTransformPipelineTestSuite struct {
	suite.Suite
	transformCtx transform.Context
}

func (s *RunTransformPipelineTestSuite) SetupTest() {
	s.transformCtx = testutils.CreateTestTransformerContext(testTransformerID)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_input_blueprint_is_nil() {
	registry := NewTransformerRegistry()
	out, err := RunTransformPipeline(nil, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "input blueprint is required")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_registry_is_nil() {
	bp := &schema.Blueprint{}
	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, nil, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "transformer registry is required")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_target_has_no_aggregator() {
	registry := NewTransformerRegistry()
	bp := &schema.Blueprint{}
	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "does not support deploy target")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_resolver_missing_for_resource() {
	registry := newRegistryWithAggregator(passthroughAggregator)

	bp := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"orderHandler": newSchemaResource("celerity/handler", nil),
			},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "no resolver registered for abstract resource type")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_resolver_returns_error() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registry.RegisterResolver("celerity/handler", func(
		_ string,
		_ *schema.Resource,
		_ linktypes.DeclaredLinkGraph,
		_ *schema.Blueprint,
	) (ResolvedResource, error) {
		return nil, errors.New("boom")
	})

	bp := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"orderHandler": newSchemaResource("celerity/handler", nil),
			},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "failed to resolve resource")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_no_rewriter_for_primary() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterEmit(registry, testTarget, emitHandlerOK)

	bp := newBlueprintWithHandler("orderHandler")

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "no rewriter factory registered")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_no_emitter_for_primary() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)

	bp := newBlueprintWithHandler("orderHandler")

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "no emitter registered")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_on_emit_resource_collision() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, func(
		r *resolvedHandler,
		_ ResourcePropertyRewriter,
		_ transform.Context,
	) (*EmitResult, error) {
		return &EmitResult{
			Resources: map[string]*schema.Resource{
				"sharedConcrete": newSchemaResource("aws/lambda/function", nil),
			},
		}, nil
	})

	bp := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"handlerA": newSchemaResource("celerity/handler", nil),
				"handlerB": newSchemaResource("celerity/handler", nil),
			},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "produced by multiple primaries")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_returns_error_when_derived_value_collides_with_user_value() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, func(
		r *resolvedHandler,
		_ ResourcePropertyRewriter,
		_ transform.Context,
	) (*EmitResult, error) {
		return &EmitResult{
			Resources: map[string]*schema.Resource{
				r.Name + "_lambda": newSchemaResource("aws/lambda/function", nil),
			},
			DerivedValues: map[string]*schema.Value{
				"shared_endpoint": {Value: core.MappingNodeFromString("derived")},
			},
		}, nil
	})

	bp := newBlueprintWithHandler("orderHandler")
	bp.Values = &schema.ValueMap{
		Values: map[string]*schema.Value{
			"shared_endpoint": {Value: core.MappingNodeFromString("user")},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "collides with a user-defined value")
	s.Assert().Nil(out)
}

func (s *RunTransformPipelineTestSuite) Test_emits_concrete_resources_and_derived_values_on_happy_path() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, emitHandlerOK)

	bp := newBlueprintWithHandler("orderHandler")

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().NoError(err)
	s.Require().NotNil(out)
	s.Require().NotNil(out.TransformedBlueprint)

	resources := out.TransformedBlueprint.Resources
	s.Require().NotNil(resources)
	s.Assert().Contains(resources.Values, "orderHandler_lambda")

	values := out.TransformedBlueprint.Values
	s.Require().NotNil(values)
	s.Assert().Contains(values.Values, "orderHandler_lambda_arn")
}

func (s *RunTransformPipelineTestSuite) Test_strips_current_transformer_id_from_transform_list() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, emitHandlerOK)

	bp := newBlueprintWithHandler("orderHandler")
	bp.Transform = &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values: []string{"other-transformer", testTransformerID},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().NoError(err)
	s.Require().NotNil(out.TransformedBlueprint.Transform)
	s.Assert().Equal([]string{"other-transformer"}, out.TransformedBlueprint.Transform.Values)
}

func (s *RunTransformPipelineTestSuite) Test_leaves_transform_list_unchanged_when_id_absent() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, emitHandlerOK)

	bp := newBlueprintWithHandler("orderHandler")
	bp.Transform = &schema.TransformValueWrapper{
		StringList: schema.StringList{Values: []string{"other-transformer"}},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().NoError(err)
	s.Assert().Equal([]string{"other-transformer"}, out.TransformedBlueprint.Transform.Values)
}

func (s *RunTransformPipelineTestSuite) Test_assembles_shared_parent_with_merged_contributions() {
	registry := NewTransformerRegistry()
	registry.RegisterAggregator(testTarget, func(resolved []ResolvedResource) *EmitPlan {
		return &EmitPlan{
			Primaries: resolved,
			SharedParents: []SharedParent{{
				Key:          "fnApp",
				ResourceName: "celerity_function_app",
				ResourceType: "azure/web/site",
				SeedSpec: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"siteName": core.MappingNodeFromString("default"),
					},
				},
			}},
		}
	})
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, func(
		r *resolvedHandler,
		_ ResourcePropertyRewriter,
		_ transform.Context,
	) (*EmitResult, error) {
		return &EmitResult{
			Resources: map[string]*schema.Resource{
				r.Name + "_function": newSchemaResource("azure/web/functions", nil),
			},
			SharedParentContributions: map[string]*core.MappingNode{
				"fnApp": {
					Fields: map[string]*core.MappingNode{
						"appSettings": {
							Fields: map[string]*core.MappingNode{
								r.Name: core.MappingNodeFromString(r.Name + "_setting"),
							},
						},
					},
				},
			},
		}, nil
	})

	bp := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"a": newSchemaResource("celerity/handler", nil),
				"b": newSchemaResource("celerity/handler", nil),
			},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)
	s.Require().NoError(err)

	parent := out.TransformedBlueprint.Resources.Values["celerity_function_app"]
	s.Require().NotNil(parent)
	s.Assert().Equal("azure/web/site", parent.Type.Value)

	settings := parent.Spec.Fields["appSettings"]
	s.Require().NotNil(settings)
	s.Assert().Contains(settings.Fields, "a")
	s.Assert().Contains(settings.Fields, "b")
}

func (s *RunTransformPipelineTestSuite) Test_emits_diagnostic_on_shared_parent_conflict() {
	registry := NewTransformerRegistry()
	registry.RegisterAggregator(testTarget, func(resolved []ResolvedResource) *EmitPlan {
		return &EmitPlan{
			Primaries: resolved,
			SharedParents: []SharedParent{{
				Key:          "fnApp",
				ResourceName: "celerity_function_app",
				ResourceType: "azure/web/site",
				SeedSpec:     &core.MappingNode{Fields: map[string]*core.MappingNode{}},
			}},
		}
	})
	registerHandlerResolver(registry)
	RegisterRewriter(registry, testTarget, rewriterFactoryNoop)
	RegisterEmit(registry, testTarget, func(
		r *resolvedHandler,
		_ ResourcePropertyRewriter,
		_ transform.Context,
	) (*EmitResult, error) {
		return &EmitResult{
			Resources: map[string]*schema.Resource{
				r.Name + "_function": newSchemaResource("azure/web/functions", nil),
			},
			SharedParentContributions: map[string]*core.MappingNode{
				"fnApp": {
					Fields: map[string]*core.MappingNode{
						"runtime": core.MappingNodeFromString(r.Name + "_runtime"),
					},
				},
			},
		}, nil
	})

	bp := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"a": newSchemaResource("celerity/handler", nil),
				"b": newSchemaResource("celerity/handler", nil),
			},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().NoError(err)
	s.Assert().NotEmpty(out.Diagnostics)
	s.Assert().Equal(core.DiagnosticLevelError, out.Diagnostics[0].Level)
	s.Assert().Contains(out.Diagnostics[0].Message, "fnApp")
}

func (s *RunTransformPipelineTestSuite) Test_warns_when_reference_targets_unsupported_path() {
	registry := newRegistryWithAggregator(passthroughAggregator)
	registerHandlerResolver(registry)
	pm := &PropertyMap{
		Renames: map[string][]string{
			"spec.memory": {"spec", "memorySize"},
		},
	}
	RewriterFromPropertyMap(pm, func(r *resolvedHandler) string {
		return r.Name + "_lambda"
	})(registry, testTarget)
	RegisterEmit(registry, testTarget, emitHandlerOK)

	bp := &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"orderHandler": newSchemaResource("celerity/handler", nil),
			},
		},
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"unsupportedExport": {
					Type: stringExportType(),
					Description: stringSubsLiteralWithRef(
						"orderHandler",
						[]*substitutions.SubstitutionPathItem{{FieldName: "spec"}, {FieldName: "unknownField"}},
					),
				},
				"supportedExport": {
					Type: stringExportType(),
					Description: stringSubsLiteralWithRef(
						"orderHandler",
						[]*substitutions.SubstitutionPathItem{{FieldName: "spec"}, {FieldName: "memory"}},
					),
				},
			},
		},
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)
	s.Require().NoError(err)

	warnings := filterWarnings(out.Diagnostics)
	s.Require().Len(warnings, 1)
	s.Assert().Contains(warnings[0].Message, "spec.unknownField")
}

func (s *RunTransformPipelineTestSuite) Test_passes_through_blueprint_when_no_resources() {
	registry := newRegistryWithAggregator(passthroughAggregator)

	bp := &schema.Blueprint{
		Version: core.ScalarFromString("2025-02-01"),
	}

	out, err := RunTransformPipeline(bp, &fakeLinkGraph{}, testTarget, testTransformerID, registry, s.transformCtx)

	s.Require().NoError(err)
	s.Require().NotNil(out.TransformedBlueprint)
	s.Assert().Same(bp.Version, out.TransformedBlueprint.Version)
	s.Require().NotNil(out.TransformedBlueprint.Resources)
	s.Assert().Empty(out.TransformedBlueprint.Resources.Values)
}

func TestRunTransformPipelineTestSuite(t *testing.T) {
	suite.Run(t, new(RunTransformPipelineTestSuite))
}

// --- helpers ---

type resolvedHandler struct {
	Name string
}

func (r *resolvedHandler) ResourceName() string { return r.Name }
func (r *resolvedHandler) ResourceType() string { return "celerity/handler" }

type fakeLinkGraph struct{}

func (g *fakeLinkGraph) Edges() []*linktypes.ResolvedLink                        { return nil }
func (g *fakeLinkGraph) EdgesFrom(_ string) []*linktypes.ResolvedLink            { return nil }
func (g *fakeLinkGraph) EdgesTo(_ string) []*linktypes.ResolvedLink              { return nil }
func (g *fakeLinkGraph) Resource(_ string) (*schema.Resource, linktypes.ResourceClass, bool) {
	return nil, "", false
}

func passthroughAggregator(resolved []ResolvedResource) *EmitPlan {
	return &EmitPlan{Primaries: resolved}
}

func newRegistryWithAggregator(aggregator Aggregator) *TransformerRegistry {
	registry := NewTransformerRegistry()
	registry.RegisterAggregator(testTarget, aggregator)
	return registry
}

func registerHandlerResolver(registry *TransformerRegistry) {
	registry.RegisterResolver("celerity/handler", func(
		name string,
		_ *schema.Resource,
		_ linktypes.DeclaredLinkGraph,
		_ *schema.Blueprint,
	) (ResolvedResource, error) {
		return &resolvedHandler{Name: name}, nil
	})
}

func rewriterFactoryNoop(_ *resolvedHandler) []ResourcePropertyRewriter {
	return []ResourcePropertyRewriter{
		func(_ *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
			return nil
		},
	}
}

func emitHandlerOK(
	r *resolvedHandler,
	_ ResourcePropertyRewriter,
	_ transform.Context,
) (*EmitResult, error) {
	return &EmitResult{
		Resources: map[string]*schema.Resource{
			r.Name + "_lambda": newSchemaResource("aws/lambda/function", nil),
		},
		DerivedValues: map[string]*schema.Value{
			r.Name + "_lambda_arn": {Value: core.MappingNodeFromString("arn:aws:lambda::derived")},
		},
	}, nil
}

func newSchemaResource(resourceType string, spec *core.MappingNode) *schema.Resource {
	return &schema.Resource{
		Type: &schema.ResourceTypeWrapper{Value: resourceType},
		Spec: spec,
	}
}

func newBlueprintWithHandler(name string) *schema.Blueprint {
	return &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				name: newSchemaResource("celerity/handler", nil),
			},
		},
	}
}

func stringSubsLiteralWithRef(
	resourceName string,
	path []*substitutions.SubstitutionPathItem,
) *substitutions.StringOrSubstitutions {
	return &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{{
			SubstitutionValue: &substitutions.Substitution{
				ResourceProperty: &substitutions.SubstitutionResourceProperty{
					ResourceName: resourceName,
					Path:         path,
				},
			},
		}},
	}
}

func filterWarnings(diagnostics []*core.Diagnostic) []*core.Diagnostic {
	warnings := []*core.Diagnostic{}
	for _, d := range diagnostics {
		if d.Level == core.DiagnosticLevelWarning {
			warnings = append(warnings, d)
		}
	}
	return warnings
}
