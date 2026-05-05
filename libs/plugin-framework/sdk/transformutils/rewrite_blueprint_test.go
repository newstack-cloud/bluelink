package transformutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subwalk"
	"github.com/stretchr/testify/suite"
)

type RewriteBlueprintRefsTestSuite struct {
	suite.Suite
}

func (s *RewriteBlueprintRefsTestSuite) Test_returns_nil_for_nil_blueprint() {
	s.Assert().Nil(RewriteBlueprintRefs(nil, identityVisitor()))
}

func (s *RewriteBlueprintRefsTestSuite) Test_returns_a_new_blueprint_value() {
	bp := &schema.Blueprint{Version: core.ScalarFromString("2025-02-01")}

	got := RewriteBlueprintRefs(bp, identityVisitor())

	s.Require().NotNil(got)
	s.Assert().NotSame(bp, got)
	s.Assert().Same(bp.Version, got.Version)
}

func (s *RewriteBlueprintRefsTestSuite) Test_passes_through_unwalked_sections_pointer_shared() {
	variables := &schema.VariableMap{Values: map[string]*schema.Variable{}}
	resources := &schema.ResourceMap{Values: map[string]*schema.Resource{}}
	transform := &schema.TransformValueWrapper{}

	bp := &schema.Blueprint{
		Variables: variables,
		Resources: resources,
		Transform: transform,
	}

	got := RewriteBlueprintRefs(bp, identityVisitor())

	s.Assert().Same(variables, got.Variables)
	s.Assert().Same(resources, got.Resources)
	s.Assert().Same(transform, got.Transform)
}

func (s *RewriteBlueprintRefsTestSuite) Test_passes_through_nil_exports() {
	bp := &schema.Blueprint{}

	got := RewriteBlueprintRefs(bp, identityVisitor())

	s.Require().NotNil(got)
	s.Assert().Nil(got.Exports)
}

func (s *RewriteBlueprintRefsTestSuite) Test_preserves_empty_exports_map() {
	bp := &schema.Blueprint{
		Exports: &schema.ExportMap{Values: map[string]*schema.Export{}},
	}

	got := RewriteBlueprintRefs(bp, identityVisitor())

	s.Require().NotNil(got.Exports)
	s.Assert().Empty(got.Exports.Values)
}

func (s *RewriteBlueprintRefsTestSuite) Test_passes_through_export_with_nil_description_and_field() {
	bp := &schema.Blueprint{
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"ex": {Type: stringExportType()},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, identityVisitor())

	out := got.Exports.Values["ex"]
	s.Require().NotNil(out)
	s.Assert().Nil(out.Description)
	s.Assert().Nil(out.Field)
}

func (s *RewriteBlueprintRefsTestSuite) Test_preserves_export_type_and_source_meta() {
	typeWrapper := stringExportType()
	srcMeta := sourceMeta(12, 4)
	bp := &schema.Blueprint{
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"ex": {Type: typeWrapper, SourceMeta: srcMeta},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, identityVisitor())

	out := got.Exports.Values["ex"]
	s.Assert().Same(typeWrapper, out.Type)
	s.Assert().Same(srcMeta, out.SourceMeta)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_export_description() {
	bp := &schema.Blueprint{
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"handlerArn": {
					Type:        stringExportType(),
					Description: descriptionWithResourceRef("myHandler", "spec", "arn"),
				},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	out := got.Exports.Values["handlerArn"].Description
	s.Require().NotNil(out)
	s.Require().Len(out.Values, 1)
	s.Require().NotNil(out.Values[0].SubstitutionValue)
	s.Require().NotNil(out.Values[0].SubstitutionValue.ValueReference)
	s.Assert().Equal("myHandler_arn", out.Values[0].SubstitutionValue.ValueReference.ValueName)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_bare_reference_in_export_field() {
	bp := &schema.Blueprint{
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"handlerArn": {
					Type:  stringExportType(),
					Field: core.ScalarFromString("resources.myHandler.spec.arn"),
				},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	field := got.Exports.Values["handlerArn"].Field
	s.Require().NotNil(field)
	s.Require().NotNil(field.StringValue)
	s.Assert().Equal("values.myHandler_arn", *field.StringValue)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_value_description() {
	bp := &schema.Blueprint{
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"v1": {
					Description: descriptionWithResourceRef("myHandler", "spec", "arn"),
				},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	desc := got.Values.Values["v1"].Description
	s.Require().NotNil(desc)
	s.Require().NotNil(desc.Values[0].SubstitutionValue.ValueReference)
	s.Assert().Equal("myHandler_arn", desc.Values[0].SubstitutionValue.ValueReference.ValueName)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_value_value_mapping_node() {
	value := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"arn": {
				StringWithSubstitutions: descriptionWithResourceRef("myHandler", "spec", "arn"),
			},
		},
	}
	bp := &schema.Blueprint{
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"v1": {Value: value},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	field := got.Values.Values["v1"].Value.Fields["arn"]
	s.Require().NotNil(field)
	s.Require().NotNil(field.StringWithSubstitutions)
	rewritten := field.StringWithSubstitutions.Values[0].SubstitutionValue.ValueReference
	s.Require().NotNil(rewritten)
	s.Assert().Equal("myHandler_arn", rewritten.ValueName)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_include_path() {
	bp := &schema.Blueprint{
		Include: &schema.IncludeMap{
			Values: map[string]*schema.Include{
				"child": {
					Path: descriptionWithResourceRef("myHandler", "spec", "arn"),
				},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	path := got.Include.Values["child"].Path
	s.Require().NotNil(path)
	s.Require().NotNil(path.Values[0].SubstitutionValue.ValueReference)
	s.Assert().Equal("myHandler_arn", path.Values[0].SubstitutionValue.ValueReference.ValueName)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_datasource_filter_search() {
	bp := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"ds": {
					Filter: &schema.DataSourceFilters{
						Filters: []*schema.DataSourceFilter{
							{
								Search: &schema.DataSourceFilterSearch{
									Values: []*substitutions.StringOrSubstitutions{
										descriptionWithResourceRef("myHandler", "spec", "arn"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	search := got.DataSources.Values["ds"].Filter.Filters[0].Search.Values[0]
	s.Require().NotNil(search)
	s.Require().NotNil(search.Values[0].SubstitutionValue.ValueReference)
	s.Assert().Equal("myHandler_arn", search.Values[0].SubstitutionValue.ValueReference.ValueName)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_datasource_metadata_displayname() {
	bp := &schema.Blueprint{
		DataSources: &schema.DataSourceMap{
			Values: map[string]*schema.DataSource{
				"ds": {
					DataSourceMetadata: &schema.DataSourceMetadata{
						DisplayName: descriptionWithResourceRef("myHandler", "spec", "arn"),
					},
				},
			},
		},
	}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	displayName := got.DataSources.Values["ds"].DataSourceMetadata.DisplayName
	s.Require().NotNil(displayName)
	s.Require().NotNil(displayName.Values[0].SubstitutionValue.ValueReference)
	s.Assert().Equal(
		"myHandler_arn",
		displayName.Values[0].SubstitutionValue.ValueReference.ValueName,
	)
}

func (s *RewriteBlueprintRefsTestSuite) Test_rewrites_substitutions_in_top_level_metadata() {
	metadata := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"owner": {
				StringWithSubstitutions: descriptionWithResourceRef("myHandler", "spec", "arn"),
			},
		},
	}
	bp := &schema.Blueprint{Metadata: metadata}

	got := RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	field := got.Metadata.Fields["owner"]
	s.Require().NotNil(field)
	s.Require().NotNil(field.StringWithSubstitutions)
	rewritten := field.StringWithSubstitutions.Values[0].SubstitutionValue.ValueReference
	s.Require().NotNil(rewritten)
	s.Assert().Equal("myHandler_arn", rewritten.ValueName)
}

func (s *RewriteBlueprintRefsTestSuite) Test_does_not_mutate_input_blueprint_on_rewrite() {
	original := descriptionWithResourceRef("myHandler", "spec", "arn")
	bp := &schema.Blueprint{
		Exports: &schema.ExportMap{
			Values: map[string]*schema.Export{
				"ex": {
					Type:        stringExportType(),
					Description: original,
				},
			},
		},
	}

	_ = RewriteBlueprintRefs(bp, resourceToValueVisitor("myHandler", "myHandler_arn"))

	stillOriginal := bp.Exports.Values["ex"].Description
	s.Require().Same(original, stillOriginal)
	s.Assert().NotNil(
		stillOriginal.Values[0].SubstitutionValue.ResourceProperty,
		"original description should still hold the resource property reference",
	)
}

func descriptionWithResourceRef(
	resourceName string,
	fields ...string,
) *substitutions.StringOrSubstitutions {
	path := make([]*substitutions.SubstitutionPathItem, 0, len(fields))
	for _, f := range fields {
		path = append(path, Field(f))
	}
	return &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					ResourceProperty: &substitutions.SubstitutionResourceProperty{
						ResourceName: resourceName,
						Path:         path,
					},
				},
			},
		},
	}
}

func resourceToValueVisitor(resourceName, valueName string) subwalk.SubstitutionVisitor {
	return RewriteResourcePropertyRefs(
		func(ref *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
			if ref.ResourceName == resourceName {
				return ValueRef(valueName)
			}
			return nil
		},
	)
}

func stringExportType() *schema.ExportTypeWrapper {
	return &schema.ExportTypeWrapper{Value: schema.ExportType("string")}
}

func sourceMeta(line, col int) *source.Meta {
	return &source.Meta{Position: source.Position{Line: line, Column: col}}
}

func identityVisitor() subwalk.SubstitutionVisitor {
	return func(sub *substitutions.Substitution) *substitutions.Substitution {
		return sub
	}
}

func TestRewriteBlueprintRefsTestSuite(t *testing.T) {
	suite.Run(t, new(RewriteBlueprintRefsTestSuite))
}
