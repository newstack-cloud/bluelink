package plugintestsuites

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testtransformer"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

const (
	celerityHandlerAbstractResourceType = "celerity/handler"
)

func (s *TransformerPluginV1Suite) Test_custom_validate_abstract_resource() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.CustomValidate(
		context.Background(),
		abstractResourceValidateInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testtransformer.AbstractResourceHandlerValidateOutput(),
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_custom_validate_abstract_resource_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.CustomValidate(
		context.Background(),
		abstractResourceValidateInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerCustomValidateAbstractResource,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_custom_validate_abstract_resource_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.CustomValidate(
		context.Background(),
		abstractResourceValidateInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred validating abstract resource")
}

func abstractResourceValidateInput() *transform.AbstractResourceValidateInput {
	return &transform.AbstractResourceValidateInput{
		SchemaResource: &schema.Resource{
			Type: &schema.ResourceTypeWrapper{
				Value: celerityHandlerAbstractResourceType,
			},
			Metadata: &schema.Metadata{
				Annotations: &schema.StringOrSubstitutionsMap{
					Values: map[string]*substitutions.StringOrSubstitutions{},
				},
				Labels: &schema.StringMap{
					Values: map[string]string{},
				},
			},
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"handlerName": core.MappingNodeFromString("my-handler"),
				},
			},
		},
		TransformerContext: testutils.CreateTestTransformerContext("celerity"),
	}
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_spec_definition() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.GetSpecDefinition(
		context.Background(),
		&transform.AbstractResourceGetSpecDefinitionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&transform.AbstractResourceGetSpecDefinitionOutput{
			SpecDefinition: &provider.ResourceSpecDefinition{
				Schema:  testtransformer.AbstractResourceHandlerSchema(),
				IDField: "id",
			},
		},
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_spec_definition_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetSpecDefinition(
		context.Background(),
		&transform.AbstractResourceGetSpecDefinitionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_spec_definition_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetSpecDefinition(
		context.Background(),
		&transform.AbstractResourceGetSpecDefinitionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred retrieving abstract resource spec definition")
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_can_link_to() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(
		context.Background(),
		&transform.AbstractResourceCanLinkToInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&transform.AbstractResourceCanLinkToOutput{
			CanLinkTo: []string{"celerity/datastore"},
		},
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_can_link_to_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.CanLinkTo(
		context.Background(),
		&transform.AbstractResourceCanLinkToInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerCheckCanAbstractResourceLinkTo,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_can_link_to_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.CanLinkTo(
		context.Background(),
		&transform.AbstractResourceCanLinkToInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred checking the resource types that the abstract resource can link to",
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_check_is_common_terminal() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.IsCommonTerminal(
		context.Background(),
		&transform.AbstractResourceIsCommonTerminalInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&transform.AbstractResourceIsCommonTerminalOutput{
			IsCommonTerminal: true,
		},
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_check_is_common_terminal_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.IsCommonTerminal(
		context.Background(),
		&transform.AbstractResourceIsCommonTerminalInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerCheckIsAbstractResourceCommonTerminal,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_check_is_common_terminal_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.IsCommonTerminal(
		context.Background(),
		&transform.AbstractResourceIsCommonTerminalInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred checking if abstract resource is a common terminal",
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_type() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.GetType(
		context.Background(),
		&transform.AbstractResourceGetTypeInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&transform.AbstractResourceGetTypeOutput{
			Type:  celerityHandlerAbstractResourceType,
			Label: "Celerity Handler",
		},
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_type_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetType(
		context.Background(),
		&transform.AbstractResourceGetTypeInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractResourceType,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_type_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetType(
		context.Background(),
		&transform.AbstractResourceGetTypeInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred retrieving abstract resource type",
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_type_description() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.GetTypeDescription(
		context.Background(),
		&transform.AbstractResourceGetTypeDescriptionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testtransformer.AbstractResourceHandlerTypeDescription(),
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_type_description_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetTypeDescription(
		context.Background(),
		&transform.AbstractResourceGetTypeDescriptionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractResourceTypeDescription,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_type_description_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetTypeDescription(
		context.Background(),
		&transform.AbstractResourceGetTypeDescriptionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred retrieving abstract resource type description",
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_examples() {
	resource, err := s.transformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	output, err := resource.GetExamples(
		context.Background(),
		&transform.AbstractResourceGetExamplesInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testtransformer.AbstractResourceHandlerExamples(),
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_examples_fails_for_unexpected_host() {
	abstractResource, err := s.transformerWrongHost.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetExamples(
		context.Background(),
		&transform.AbstractResourceGetExamplesInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractResourceExamples,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_resource_get_examples_reports_expected_error_for_failure() {
	abstractResource, err := s.failingTransformer.AbstractResource(
		context.Background(),
		celerityHandlerAbstractResourceType,
	)
	s.Require().NoError(err)

	_, err = abstractResource.GetExamples(
		context.Background(),
		&transform.AbstractResourceGetExamplesInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred retrieving abstract resource examples",
	)
}
