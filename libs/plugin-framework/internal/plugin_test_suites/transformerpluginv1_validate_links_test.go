package plugintestsuites

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linktypes"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	transformerv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func (s *TransformerPluginV1Suite) Test_validate_links_with_valid_links() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "api1",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					"celerity/handler::celerity.handler.http.method": "GET",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"api1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	output, err := s.transformer.ValidateLinks(context.Background(), input)
	s.Require().NoError(err)
	s.Assert().Empty(output.Diagnostics)
}

func (s *TransformerPluginV1Suite) Test_validate_links_cross_boundary_link() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "lambdaFunc",
				SourceType: "celerity/handler",
				TargetType: "aws/lambda/function",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					"celerity/handler::celerity.handler.http.method": "GET",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"lambdaFunc": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassConcrete,
			},
		},
	)
	s.Require().NoError(err)

	output, err := s.transformer.ValidateLinks(context.Background(), input)
	s.Require().NoError(err)
	s.Require().Len(output.Diagnostics, 1)
	s.Assert().Equal(core.DiagnosticLevelError, output.Diagnostics[0].Level)
	s.Assert().Contains(output.Diagnostics[0].Message, "can not link to")
	s.Assert().Equal(
		transformerv1.ErrorReasonCodeCrossAbstractConcreteBoundaryLink,
		output.Diagnostics[0].Context.ReasonCode,
	)
}

func (s *TransformerPluginV1Suite) Test_validate_links_unknown_link_type() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "scheduler1",
				SourceType: "celerity/handler",
				TargetType: "celerity/scheduler",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					"celerity/handler::celerity.handler.http.method": "GET",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"scheduler1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	output, err := s.transformer.ValidateLinks(context.Background(), input)
	s.Require().NoError(err)
	s.Require().Len(output.Diagnostics, 1)
	s.Assert().Equal(core.DiagnosticLevelError, output.Diagnostics[0].Level)
	s.Assert().Contains(output.Diagnostics[0].Message, "No abstract link definition found")
	s.Assert().Equal(
		transformerv1.ErrorReasonCodeNoSuchAbstractLinkDefinition,
		output.Diagnostics[0].Context.ReasonCode,
	)
}

func (s *TransformerPluginV1Suite) Test_validate_links_missing_required_annotation() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "api1",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				// No annotations provided — required annotation is missing.
				schema:         createAbstractHandlerResource(map[string]string{}),
				classification: linktypes.ResourceClassAbstract,
			},
			"api1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	output, err := s.transformer.ValidateLinks(context.Background(), input)
	s.Require().NoError(err)
	s.Require().NotEmpty(output.Diagnostics)

	hasMissingAnnotationDiag := false
	for _, diag := range output.Diagnostics {
		if diag.Level == core.DiagnosticLevelError &&
			containsAll(diag.Message, "annotation is required", "celerity.handler.http.method") {
			hasMissingAnnotationDiag = true
		}
	}
	s.Assert().True(hasMissingAnnotationDiag, "expected a diagnostic about the missing required annotation")
}

func (s *TransformerPluginV1Suite) Test_validate_links_invalid_annotation_value() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "api1",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					// PATCH is not in AllowedValues (GET, POST, PUT, DELETE).
					"celerity/handler::celerity.handler.http.method": "PATCH",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"api1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	output, err := s.transformer.ValidateLinks(context.Background(), input)
	s.Require().NoError(err)
	s.Require().NotEmpty(output.Diagnostics)

	hasInvalidValueDiag := false
	for _, diag := range output.Diagnostics {
		if diag.Level == core.DiagnosticLevelError &&
			containsAll(diag.Message, "not one of the allowed values") {
			hasInvalidValueDiag = true
		}
	}
	s.Assert().True(hasInvalidValueDiag, "expected a diagnostic about the invalid annotation value")
}

func (s *TransformerPluginV1Suite) Test_validate_links_cardinality_violation() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "api1",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
			{
				Source:     "handler1",
				Target:     "api2",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					"celerity/handler::celerity.handler.http.method": "GET",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"api1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
			"api2": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	output, err := s.transformer.ValidateLinks(context.Background(), input)
	s.Require().NoError(err)
	s.Require().NotEmpty(output.Diagnostics)

	hasCardinalityDiag := false
	for _, diag := range output.Diagnostics {
		if diag.Level == core.DiagnosticLevelError &&
			containsAll(diag.Message, "exceeding the maximum") {
			hasCardinalityDiag = true
		}
	}
	s.Assert().True(hasCardinalityDiag, "expected a diagnostic about cardinality violation")
}

func (s *TransformerPluginV1Suite) Test_validate_links_fails_for_unexpected_host() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "api1",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					"celerity/handler::celerity.handler.http.method": "GET",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"api1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	_, err = s.transformerWrongHost.ValidateLinks(context.Background(), input)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerValidateLinks,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_validate_links_reports_expected_error_for_failure() {
	input, err := createValidateLinksInput(
		[]*linktypes.ResolvedLink{
			{
				Source:     "handler1",
				Target:     "api1",
				SourceType: "celerity/handler",
				TargetType: "celerity/api",
			},
		},
		map[string]testResourceEntry{
			"handler1": {
				schema: createAbstractHandlerResource(map[string]string{
					"celerity/handler::celerity.handler.http.method": "GET",
				}),
				classification: linktypes.ResourceClassAbstract,
			},
			"api1": {
				schema:         createAbstractAPIResource(),
				classification: linktypes.ResourceClassAbstract,
			},
		},
	)
	s.Require().NoError(err)

	_, err = s.failingTransformer.ValidateLinks(context.Background(), input)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred validating links")
}

func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
