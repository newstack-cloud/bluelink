package transformutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/stretchr/testify/suite"
)

const testBoolAnnotationKey = "celerity.handler.public"

type GetBoolAnnotationTestSuite struct {
	suite.Suite
}

func (s *GetBoolAnnotationTestSuite) Test_reads_a_flag_written_as_true() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: pluginutils.StringToSubstitutions("true"),
		},
	)

	value, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", false)

	s.Require().True(found, "the annotation is set, so it has to be reported as found")
	s.Assert().True(value)
}

func (s *GetBoolAnnotationTestSuite) Test_reads_a_flag_written_as_false() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: pluginutils.StringToSubstitutions("false"),
		},
	)

	value, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", false)

	s.Require().True(found)
	s.Assert().False(value)
}

func (s *GetBoolAnnotationTestSuite) Test_the_untyped_helper_cannot_report_a_flag_as_true() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: pluginutils.StringToSubstitutions("true"),
		},
	)

	value, found := GetAnnotation(resource, testBoolAnnotationKey, "")

	s.Require().True(found)
	s.Assert().False(
		core.BoolValue(value),
		"if this ever reports true the typed accessor is no longer the only way to "+
			"read a flag, and the reason it exists has changed",
	)
}

func (s *GetBoolAnnotationTestSuite) Test_reports_a_value_that_is_not_a_flag_as_absent() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: pluginutils.StringToSubstitutions("yes"),
		},
	)

	_, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", false)

	s.Assert().False(found)
}

// An annotation carrying a substitution has no value until it is resolved, which is after
// the transform runs, so it cannot be answered here either.
func (s *GetBoolAnnotationTestSuite) Test_reports_an_unresolved_substitution_as_absent() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: variableSubstitution("enablePublicAccess"),
		},
	)

	_, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", false)

	s.Assert().False(found)
}

func (s *GetBoolAnnotationTestSuite) Test_falls_back_to_the_fallback_key() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testFallbackAnnotationKey: pluginutils.StringToSubstitutions("true"),
		},
	)

	value, found := GetBoolAnnotation(resource, testBoolAnnotationKey, testFallbackAnnotationKey, false)

	s.Require().True(found)
	s.Assert().True(value)
}

func (s *GetBoolAnnotationTestSuite) Test_reports_an_annotation_that_is_not_set_as_absent() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{},
	)

	_, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", false)

	s.Assert().False(found)
}

func (s *GetBoolAnnotationTestSuite) Test_returns_the_given_default_when_no_flag_is_found() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{},
	)

	value, _ := GetBoolAnnotation(resource, testBoolAnnotationKey, "", true)

	s.Assert().True(
		value,
		"discarding the found value gave false for an absent flag whose safe value is true",
	)
}

func (s *GetBoolAnnotationTestSuite) Test_a_flag_set_to_false_overrides_a_true_default() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: pluginutils.StringToSubstitutions("false"),
		},
	)

	value, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", true)

	s.Require().True(found)
	s.Assert().False(value)
}

func (s *GetBoolAnnotationTestSuite) Test_keeps_a_true_default_for_a_value_that_is_not_a_flag() {
	resource := newResourceWithAnnotations(
		map[string]*substitutions.StringOrSubstitutions{
			testBoolAnnotationKey: pluginutils.StringToSubstitutions("yes"),
		},
	)

	value, found := GetBoolAnnotation(resource, testBoolAnnotationKey, "", true)

	s.Require().False(found)
	s.Assert().True(value)
}

func TestGetBoolAnnotationTestSuite(t *testing.T) {
	suite.Run(t, new(GetBoolAnnotationTestSuite))
}
