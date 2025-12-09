package validation

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type LinkAnnotationValidationTestSuite struct {
	suite.Suite
}

func (s *LinkAnnotationValidationTestSuite) Test_successfully_validates_link_chains_with_no_issues() {
	linkChains := createTestLinkChain(&fixtureConfig{})
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Empty(diagnostics)
}

func (s *LinkAnnotationValidationTestSuite) Test_reports_warnings_when_annotations_include_substitutions() {
	linkChains := createTestLinkChain(
		&fixtureConfig{
			includeSubstitutions: true,
		},
	)
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level: core.DiagnosticLevelWarning,
				Message: "The value of the \"test.int.annotation\" annotation in the \"resourceA\" " +
					"resource contains substitutions and can not be validated against a type. " +
					"When substitutions are resolved, this value must be a valid integer.",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
				},
			},
		},
		diagnostics,
	)
}

func (s *LinkAnnotationValidationTestSuite) Test_reports_errors_for_missing_required_annotations() {
	linkChains := createTestLinkChain(
		&fixtureConfig{
			omitRequiredAnnotations: true,
		},
	)
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level: core.DiagnosticLevelError,
				Message: "The \"test.string.annotation\" annotation is required for the \"resourceA\" " +
					"resource in relation to the \"resourceB\" resource, but is missing or null.",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
				},
			},
		},
		diagnostics,
	)
}

func (s *LinkAnnotationValidationTestSuite) Test_reports_errors_for_annotations_with_invalid_types() {
	linkChains := createTestLinkChain(
		&fixtureConfig{
			invalidTypes: true,
		},
	)
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level: core.DiagnosticLevelError,
				Message: "The value of the \"test.int.annotation\" annotation in the \"resourceA\" " +
					"resource is not a valid integer. Expected a value of type integer, but got string.",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
				},
			},
		},
		diagnostics,
	)
}

func (s *LinkAnnotationValidationTestSuite) Test_reports_errors_for_annotations_with_values_not_in_allowed_list() {
	linkChains := createTestLinkChain(
		&fixtureConfig{
			invalidAllowedValues: true,
		},
	)
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level: core.DiagnosticLevelError,
				Message: "The value of the \"test.string.annotation\" annotation in the \"resourceA\" " +
					"resource is not one of the allowed values. invalid-value was provided " +
					"but expected one of test-value, targeted-test-value",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
				},
			},
		},
		diagnostics,
	)
}

func (s *LinkAnnotationValidationTestSuite) Test_reports_errors_for_annotation_that_fails_custom_validation() {
	linkChains := createTestLinkChain(
		&fixtureConfig{
			failsCustomValidation: true,
		},
	)
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Equal(
		[]*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "test.int.annotation value exceeds maximum allowed value of 800000.",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
				},
			},
		},
		diagnostics,
	)
}

func (s *LinkAnnotationValidationTestSuite) Test_does_not_panic_for_missing_optional_annotations() {
	// This test verifies that validation doesn't panic when optional annotations
	// are not defined. Previously this would cause a nil pointer dereference
	// when accessing annotation values for non-required annotations that were missing.
	linkChains := createTestLinkChain(
		&fixtureConfig{
			omitOptionalAnnotations: true,
		},
	)
	diagnostics, err := ValidateLinkAnnotations(
		context.Background(),
		linkChains,
		createParams(),
	)
	s.Assert().NoError(err)
	s.Assert().Empty(diagnostics)
}

type fixtureConfig struct {
	includeSubstitutions    bool
	omitRequiredAnnotations bool
	omitOptionalAnnotations bool
	invalidTypes            bool
	invalidAllowedValues    bool
	failsCustomValidation   bool
}

func createTestLinkChain(
	fixtureConf *fixtureConfig,
) []*links.ChainLinkNode {
	stringAnnotationValue := "test-value"
	targetedStringAnnotationValue := "targeted-test-value"
	intAnnotationValue := "509332"

	resourceANode := &links.ChainLinkNode{
		ResourceName: "resourceA",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{
				Value: "test/resourceTypeA",
			},
			Metadata: &schema.Metadata{
				Annotations: &schema.StringOrSubstitutionsMap{
					Values: map[string]*substitutions.StringOrSubstitutions{
						"test.string.annotation": {
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &stringAnnotationValue,
								},
							},
						},
						"test.string.resourceB.annotation": {
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &targetedStringAnnotationValue,
								},
							},
						},
						"test.int.annotation": {
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &intAnnotationValue,
								},
							},
						},
					},
				},
			},
		},
		LinkImplementations: map[string]provider.Link{
			"resourceB": &testResourceTypeAResourceTypeBLink{},
		},
		LinksTo:    []*links.ChainLinkNode{},
		LinkedFrom: []*links.ChainLinkNode{},
	}

	boolAnnotationValue := "true"
	floatAnnotationValue := "504.201983"

	resourceBNode := &links.ChainLinkNode{
		ResourceName: "resourceB",
		Resource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{
				Value: "test/resourceTypeB",
			},
			Metadata: &schema.Metadata{
				Annotations: &schema.StringOrSubstitutionsMap{
					Values: map[string]*substitutions.StringOrSubstitutions{
						"test.bool.annotation": {
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &boolAnnotationValue,
								},
							},
						},
						"test.float.annotation": {
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &floatAnnotationValue,
								},
							},
						},
					},
				},
			},
		},
		LinkImplementations: map[string]provider.Link{},
		LinksTo:             []*links.ChainLinkNode{},
		LinkedFrom: []*links.ChainLinkNode{
			resourceANode,
		},
	}

	resourceANode.LinksTo = append(resourceANode.LinksTo, resourceBNode)

	if fixtureConf.includeSubstitutions {
		resourceANode.Resource.Metadata.Annotations.Values["test.int.annotation"] =
			&substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							Variable: &substitutions.SubstitutionVariable{
								VariableName: "testVariable",
							},
						},
					},
				},
			}
	}

	if fixtureConf.omitRequiredAnnotations {
		delete(resourceANode.Resource.Metadata.Annotations.Values, "test.string.annotation")
	}

	if fixtureConf.invalidTypes {
		resourceANode.Resource.Metadata.Annotations.Values["test.int.annotation"] =
			&substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &stringAnnotationValue, // Invalid type, should be an int
					},
				},
			}
	}

	if fixtureConf.invalidAllowedValues {
		invalidStringAnnotationValue := "invalid-value" // Not in the allowed list
		resourceANode.Resource.Metadata.Annotations.Values["test.string.annotation"] =
			&substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &invalidStringAnnotationValue,
					},
				},
			}
	}

	if fixtureConf.failsCustomValidation {
		intValueTooLarge := "1000000000"
		resourceANode.Resource.Metadata.Annotations.Values["test.int.annotation"] =
			&substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{
						StringValue: &intValueTooLarge,
					},
				},
			}
	}

	if fixtureConf.omitOptionalAnnotations {
		// Remove optional annotations (test.int.annotation is not required)
		// and optional annotations on resourceB (test.bool.annotation and test.float.annotation)
		// to test that validation doesn't panic when optional annotations are missing.
		delete(resourceANode.Resource.Metadata.Annotations.Values, "test.int.annotation")
		delete(resourceANode.Resource.Metadata.Annotations.Values, "test.string.resourceB.annotation")
		delete(resourceBNode.Resource.Metadata.Annotations.Values, "test.bool.annotation")
		delete(resourceBNode.Resource.Metadata.Annotations.Values, "test.float.annotation")
	}

	return []*links.ChainLinkNode{
		resourceANode,
	}
}

func TestLinkAnnotationValidationTestSuite(t *testing.T) {
	suite.Run(t, new(LinkAnnotationValidationTestSuite))
}
