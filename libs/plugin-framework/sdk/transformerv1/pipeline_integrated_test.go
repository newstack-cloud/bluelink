package transformerv1_test

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformutils"
	"github.com/stretchr/testify/suite"
)

const (
	integratedTransformerID = "celerity-e2e-2026"
	integratedDeployTarget  = "aws-serverless"
)

// PipelineIntegratedTestSuite exercises RunTransformPipeline through the public
// TransformerPluginDefinition.Transform entrypoint, verifying that the
// declarative AbstractResourceDefinition fields wire correctly into the
// driver and that user-written substitutions get rewritten across
// resource specs and the top-level blueprint sections.
type PipelineIntegratedTestSuite struct {
	suite.Suite
}

func (s *PipelineIntegratedTestSuite) Test_full_pipeline_emits_concrete_resources_and_rewrites_refs() {
	plugin := buildHandlerTransformer()
	bp := blueprintWithHandlerAndExport()

	out, err := plugin.Transform(context.Background(), &transform.SpecTransformerTransformInput{
		InputBlueprint:     bp,
		LinkGraph:          &emptyLinkGraph{},
		TransformerContext: integratedTransformerContext(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(out)
	s.Require().NotNil(out.TransformedBlueprint)

	resources := out.TransformedBlueprint.Resources
	s.Require().NotNil(resources)
	concrete := resources.Values["orderHandler_lambda"]
	s.Require().NotNil(concrete, "expected concrete lambda resource")
	s.Assert().Equal("aws/lambda/function", concrete.Type.Value)

	values := out.TransformedBlueprint.Values
	s.Require().NotNil(values)
	s.Assert().Contains(values.Values, "orderHandler_lambda_arn", "derived value should be merged into output")

	exportedRef := exportRef(out.TransformedBlueprint, "handlerArn")
	s.Require().NotNil(exportedRef)
	s.Require().NotNil(exportedRef.ValueReference)
	s.Assert().Equal("orderHandler_lambda_arn", exportedRef.ValueReference.ValueName)

	specRef := specRef(out.TransformedBlueprint, "orderHandler_lambda", "memorySize")
	s.Require().NotNil(specRef, "expected rewritten spec reference under memorySize")
	s.Require().NotNil(specRef.ResourceProperty)
	s.Assert().Equal("orderHandler_lambda", specRef.ResourceProperty.ResourceName)
}

func (s *PipelineIntegratedTestSuite) Test_full_pipeline_strips_current_transformer_id() {
	plugin := buildHandlerTransformer()
	bp := blueprintWithHandlerAndExport()
	bp.Transform = &schema.TransformValueWrapper{
		StringList: schema.StringList{
			Values: []string{"some-other-transformer", integratedTransformerID},
		},
	}

	out, err := plugin.Transform(context.Background(), &transform.SpecTransformerTransformInput{
		InputBlueprint:     bp,
		LinkGraph:          &emptyLinkGraph{},
		TransformerContext: integratedTransformerContext(),
	})
	s.Require().NoError(err)

	s.Require().NotNil(out.TransformedBlueprint.Transform)
	s.Assert().Equal([]string{"some-other-transformer"}, out.TransformedBlueprint.Transform.Values)
}

func (s *PipelineIntegratedTestSuite) Test_full_pipeline_returns_error_when_target_missing() {
	plugin := buildHandlerTransformer()
	bp := blueprintWithHandlerAndExport()

	ctx := transform.NewTransformerContextFromParams(
		integratedTransformerID,
		core.NewDefaultParams(
			map[string]map[string]*core.ScalarValue{},
			map[string]map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{
				"deployTarget": core.ScalarFromString("unsupported-target"),
			},
			map[string]*core.ScalarValue{},
		),
	)

	out, err := plugin.Transform(context.Background(), &transform.SpecTransformerTransformInput{
		InputBlueprint:     bp,
		LinkGraph:          &emptyLinkGraph{},
		TransformerContext: ctx,
	})

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "does not support deploy target")
	s.Assert().Nil(out)
}

func (s *PipelineIntegratedTestSuite) Test_full_pipeline_warns_on_reference_outside_supported_paths() {
	plugin := buildHandlerTransformer()
	bp := blueprintWithHandlerAndExport()
	bp.Exports.Values["unsupported"] = &schema.Export{
		Type: stringExportType(),
		Description: stringSubsLiteral(
			"orderHandler",
			[]*substitutions.SubstitutionPathItem{
				{FieldName: "spec"},
				{FieldName: "unknownField"},
			},
		),
	}

	out, err := plugin.Transform(context.Background(), &transform.SpecTransformerTransformInput{
		InputBlueprint:     bp,
		LinkGraph:          &emptyLinkGraph{},
		TransformerContext: integratedTransformerContext(),
	})
	s.Require().NoError(err)

	warnings := pickWarnings(out.Diagnostics)
	s.Require().NotEmpty(warnings, "expected at least one capability-matrix warning")
	s.Assert().Contains(warnings[0].Message, "unknownField")
}

func TestPipelineIntegratedTestSuite(t *testing.T) {
	suite.Run(t, new(PipelineIntegratedTestSuite))
}

// transformer fixture

type resolvedHandler struct {
	Name string
}

func (r *resolvedHandler) ResourceName() string { return r.Name }
func (r *resolvedHandler) ResourceType() string { return "celerity/handler" }

func buildHandlerTransformer() *transformerv1.TransformerPluginDefinition {
	pm := handlerPropertyMap()
	def := &transformerv1.AbstractResourceDefinition{
		Type: "celerity/handler",
		Resolve: func(
			name string,
			_ *schema.Resource,
			_ linktypes.DeclaredLinkGraph,
			_ *schema.Blueprint,
		) (transformutils.ResolvedResource, error) {
			return &resolvedHandler{Name: name}, nil
		},
		PropertyMaps: map[string]transformutils.PropertyMap{
			integratedDeployTarget: *pm,
		},
		Emitters: map[string]transformutils.EmitterRegistration{
			integratedDeployTarget: transformutils.TypedEmitter(emitHandlerToLambda),
		},
		Rewriters: map[string]transformutils.RewriterRegistration{
			integratedDeployTarget: transformutils.RewriterFromPropertyMap(pm, func(r *resolvedHandler) string {
				return r.Name + "_lambda"
			}),
		},
	}

	return &transformerv1.TransformerPluginDefinition{
		TransformName: integratedTransformerID,
		AbstractResources: map[string]*transformerv1.AbstractResourceDefinition{
			"celerity/handler": def,
		},
		Aggregators: map[string]transformutils.Aggregator{
			integratedDeployTarget: func(resolved []transformutils.ResolvedResource) *transformutils.EmitPlan {
				return &transformutils.EmitPlan{Primaries: resolved}
			},
		},
	}
}

func handlerPropertyMap() *transformutils.PropertyMap {
	return &transformutils.PropertyMap{
		Renames: map[string][]string{
			"spec.memory": {"spec", "memorySize"},
		},
		ValueRefs: map[string]*transformutils.ValueRefSpec{
			"spec.arn": {Suffix: "_arn"},
		},
	}
}

func emitHandlerToLambda(
	r *resolvedHandler,
	chained transformutils.ResourcePropertyRewriter,
	_ transform.Context,
) (*transformutils.EmitResult, error) {
	originalRef := &substitutions.SubstitutionResourceProperty{
		ResourceName: r.Name,
		Path: []*substitutions.SubstitutionPathItem{
			{FieldName: "spec"},
			{FieldName: "memory"},
		},
	}
	rewritten := chained(originalRef)
	memorySpec := &core.MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{{SubstitutionValue: rewritten}},
		},
	}

	return &transformutils.EmitResult{
		Resources: map[string]*schema.Resource{
			r.Name + "_lambda": {
				Type: &schema.ResourceTypeWrapper{Value: "aws/lambda/function"},
				Spec: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"memorySize": memorySpec,
					},
				},
			},
		},
		DerivedValues: map[string]*schema.Value{
			r.Name + "_lambda_arn": {
				Value: core.MappingNodeFromString("arn:aws:lambda::derived"),
			},
		},
	}, nil
}

// blueprint fixture

func blueprintWithHandlerAndExport() *schema.Blueprint {
	return &schema.Blueprint{
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{
				"orderHandler": {
					Type: &schema.ResourceTypeWrapper{Value: "celerity/handler"},
					Spec: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"memory": core.MappingNodeFromInt(512),
						},
					},
				},
			},
		},
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"handlerArn": {
					Type: stringExportType(),
					Description: stringSubsLiteral(
						"orderHandler",
						[]*substitutions.SubstitutionPathItem{
							{FieldName: "spec"},
							{FieldName: "arn"},
						},
					),
				},
			},
		},
	}
}

// helpers

func integratedTransformerContext() transform.Context {
	return transform.NewTransformerContextFromParams(
		integratedTransformerID,
		core.NewDefaultParams(
			map[string]map[string]*core.ScalarValue{},
			map[string]map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{
				"deployTarget": core.ScalarFromString(integratedDeployTarget),
			},
			map[string]*core.ScalarValue{},
		),
	)
}

func stringExportType() *schema.ExportTypeWrapper {
	return &schema.ExportTypeWrapper{Value: schema.ExportType("string")}
}

func stringSubsLiteral(
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

func exportRef(blueprint *schema.Blueprint, exportName string) *substitutions.Substitution {
	if blueprint.Exports == nil {
		return nil
	}
	export := blueprint.Exports.Values[exportName]
	if export == nil || export.Description == nil || len(export.Description.Values) == 0 {
		return nil
	}
	return export.Description.Values[0].SubstitutionValue
}

func specRef(
	blueprint *schema.Blueprint,
	resourceName, field string,
) *substitutions.Substitution {
	if blueprint.Resources == nil {
		return nil
	}
	resource := blueprint.Resources.Values[resourceName]
	if resource == nil || resource.Spec == nil {
		return nil
	}
	node := resource.Spec.Fields[field]
	if node == nil || node.StringWithSubstitutions == nil ||
		len(node.StringWithSubstitutions.Values) == 0 {
		return nil
	}
	return node.StringWithSubstitutions.Values[0].SubstitutionValue
}

func pickWarnings(diagnostics []*core.Diagnostic) []*core.Diagnostic {
	out := []*core.Diagnostic{}
	for _, d := range diagnostics {
		if d.Level == core.DiagnosticLevelWarning {
			out = append(out, d)
		}
	}
	return out
}

// minimal link graph stub

type emptyLinkGraph struct{}

func (g *emptyLinkGraph) Edges() []*linktypes.ResolvedLink             { return nil }
func (g *emptyLinkGraph) EdgesFrom(_ string) []*linktypes.ResolvedLink { return nil }
func (g *emptyLinkGraph) EdgesTo(_ string) []*linktypes.ResolvedLink   { return nil }
func (g *emptyLinkGraph) Resource(_ string) (*schema.Resource, linktypes.ResourceClass, bool) {
	return nil, "", false
}
