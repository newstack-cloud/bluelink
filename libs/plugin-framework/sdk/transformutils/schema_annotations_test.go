package transformutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/stretchr/testify/suite"
)

const (
	testPrimaryAnnotationKey  = "aws.lambda.function.orderProcessor.runtime"
	testFallbackAnnotationKey = "aws.lambda.function.runtime"
)

type GetAnnotationTestSuite struct {
	suite.Suite
}

func (s *GetAnnotationTestSuite) Test_returns_scalar_mapping_node_for_single_string_annotation() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testPrimaryAnnotationKey: pluginutils.StringToSubstitutions("nodejs20.x"),
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, "")

	s.Require().True(found)
	s.Require().NotNil(value)
	s.Assert().Equal(core.MappingNodeFromString("nodejs20.x"), value)
}

func (s *GetAnnotationTestSuite) Test_preserves_unresolved_substitution_for_substitution_annotation() {
	annotation := variableSubstitution("region")
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testPrimaryAnnotationKey: annotation,
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, "")

	s.Require().True(found)
	s.Require().NotNil(value)
	s.Assert().Nil(value.Scalar)
	// The whole point of this helper is to retain unresolved substitutions,
	// so the original value should be threaded through untouched.
	s.Assert().Same(annotation, value.StringWithSubstitutions)
}

func (s *GetAnnotationTestSuite) Test_preserves_mixed_string_and_substitution_values() {
	annotation := stringWithVariableSubstitution("orders-", "environment")
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testPrimaryAnnotationKey: annotation,
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, "")

	s.Require().True(found)
	s.Require().NotNil(value)
	s.Assert().Nil(value.Scalar)
	s.Assert().Same(annotation, value.StringWithSubstitutions)
}

func (s *GetAnnotationTestSuite) Test_returns_nil_mapping_node_but_found_when_annotation_has_no_values() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testPrimaryAnnotationKey: {},
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, "")

	s.Assert().True(found)
	s.Assert().Nil(value)
}

func (s *GetAnnotationTestSuite) Test_falls_back_to_fallback_key_when_primary_key_absent() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testFallbackAnnotationKey: pluginutils.StringToSubstitutions("python3.12"),
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, testFallbackAnnotationKey)

	s.Require().True(found)
	s.Require().NotNil(value)
	s.Assert().Equal(core.MappingNodeFromString("python3.12"), value)
}

func (s *GetAnnotationTestSuite) Test_prefers_primary_key_over_fallback_when_both_present() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testPrimaryAnnotationKey:  pluginutils.StringToSubstitutions("nodejs20.x"),
			testFallbackAnnotationKey: pluginutils.StringToSubstitutions("python3.12"),
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, testFallbackAnnotationKey)

	s.Require().True(found)
	s.Require().NotNil(value)
	s.Assert().Equal(core.MappingNodeFromString("nodejs20.x"), value)
}

func (s *GetAnnotationTestSuite) Test_returns_nil_and_false_when_primary_and_fallback_absent() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			"aws.lambda.function.unrelated": pluginutils.StringToSubstitutions("nodejs20.x"),
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, testFallbackAnnotationKey)

	s.Assert().False(found)
	s.Assert().Nil(value)
}

func (s *GetAnnotationTestSuite) Test_does_not_use_fallback_when_fallback_key_is_empty() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testFallbackAnnotationKey: pluginutils.StringToSubstitutions("python3.12"),
		},
	)

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, "")

	s.Assert().False(found)
	s.Assert().Nil(value)
}

func (s *GetAnnotationTestSuite) Test_returns_nil_and_false_when_metadata_is_nil() {
	resource := &schema.Resource{}

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, testFallbackAnnotationKey)

	s.Assert().False(found)
	s.Assert().Nil(value)
}

func (s *GetAnnotationTestSuite) Test_returns_nil_and_false_when_annotations_are_nil() {
	resource := &schema.Resource{
		Metadata: &schema.Metadata{},
	}

	value, found := GetAnnotation(resource, testPrimaryAnnotationKey, testFallbackAnnotationKey)

	s.Assert().False(found)
	s.Assert().Nil(value)
}

func TestGetAnnotationTestSuite(t *testing.T) {
	suite.Run(t, new(GetAnnotationTestSuite))
}

func newResourceWithAnnotations(
	annotations map[string]*substitutions.StringOrSubstitutions,
) *schema.Resource {
	return &schema.Resource{
		Metadata: &schema.Metadata{
			Annotations: &schema.StringOrSubstitutionsMap{
				Values: annotations,
			},
		},
	}
}

func variableSubstitution(variableName string) *substitutions.StringOrSubstitutions {
	return &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					Variable: &substitutions.SubstitutionVariable{
						VariableName: variableName,
					},
				},
			},
		},
	}
}

func stringWithVariableSubstitution(
	prefix string,
	variableName string,
) *substitutions.StringOrSubstitutions {
	return &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &prefix,
			},
			{
				SubstitutionValue: &substitutions.Substitution{
					Variable: &substitutions.SubstitutionVariable{
						VariableName: variableName,
					},
				},
			},
		},
	}
}
