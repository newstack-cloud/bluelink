package plugintestsuites

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testprovider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

const (
	testLinkID      = "link-id-1"
	testLinkName    = "processOrderFunction_0::ordersTable"
	testResource2ID = "test-resource-2"
)

func (s *ProviderPluginV1Suite) Test_stage_link_changes() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.StageChanges(
		context.Background(),
		linkStageChangesInput(),
	)
	s.Require().NoError(err)
	expected := testprovider.LinkLambdaDynamoDBChangesOutput()
	testutils.AssertLinkChangesEquals(
		expected.Changes,
		output.Changes,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_stage_link_changes_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.StageChanges(
		context.Background(),
		linkStageChangesInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderStageLinkChanges,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_stage_link_changes_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.StageChanges(
		context.Background(),
		linkStageChangesInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred when staging changes for link")
}

func (s *ProviderPluginV1Suite) Test_link_update_resource_a() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.UpdateResourceA(
		context.Background(),
		linkUpdateResourceAInput(),
	)
	s.Require().NoError(err)
	expected := testprovider.LinkLambdaDynamoDBUpdateResourceAOutput()
	s.Assert().Equal(expected, output)
}

func (s *ProviderPluginV1Suite) Test_link_update_resource_a_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.UpdateResourceA(
		context.Background(),
		linkUpdateResourceAInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderUpdateLinkResourceA,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_update_resource_a_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.UpdateResourceA(
		context.Background(),
		linkUpdateResourceAInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred when updating resource A for link")
}

func (s *ProviderPluginV1Suite) Test_link_update_resource_b() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.UpdateResourceB(
		context.Background(),
		linkUpdateResourceBInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(&provider.LinkUpdateResourceOutput{}, output)
}

func (s *ProviderPluginV1Suite) Test_link_update_resource_b_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.UpdateResourceB(
		context.Background(),
		linkUpdateResourceBInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderUpdateLinkResourceB,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_update_resource_b_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.UpdateResourceB(
		context.Background(),
		linkUpdateResourceBInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred when updating resource B for link")
}

func (s *ProviderPluginV1Suite) Test_link_update_intermediary_resources() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.UpdateIntermediaryResources(
		context.Background(),
		linkUpdateIntermediaryResourcesInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(&provider.LinkUpdateIntermediaryResourcesOutput{}, output)
}

func (s *ProviderPluginV1Suite) Test_link_update_intermediary_resources_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.UpdateIntermediaryResources(
		context.Background(),
		linkUpdateIntermediaryResourcesInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderUpdateLinkIntermediaryResources,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_update_intermediary_resources_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.UpdateIntermediaryResources(
		context.Background(),
		linkUpdateIntermediaryResourcesInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred when updating intermediary resources for link")
}

func (s *ProviderPluginV1Suite) Test_link_get_priority_resource() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.GetPriorityResource(
		context.Background(),
		linkGetPriorityResourceInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.LinkGetPriorityResourceOutput{
			PriorityResource:     provider.LinkPriorityResourceB,
			PriorityResourceType: dynamoDBTableResourceType,
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_priority_resource_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetPriorityResource(
		context.Background(),
		linkGetPriorityResourceInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetLinkPriorityResource,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_priority_resource_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetPriorityResource(
		context.Background(),
		linkGetPriorityResourceInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred when retrieving the priority resource for link")
}

func (s *ProviderPluginV1Suite) Test_link_get_type() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.GetType(
		context.Background(),
		linkGetTypeInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.LinkGetTypeOutput{
			Type: core.LinkType(
				lambdaFunctionResourceType,
				dynamoDBTableResourceType,
			),
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_type_description() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.GetTypeDescription(
		context.Background(),
		linkGetTypeDescriptionInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testprovider.LinkLambdaFunctionDDBTableTypeDescriptionOutput(),
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_type_description_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetTypeDescription(
		context.Background(),
		linkGetTypeDescriptionInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetLinkTypeDescription,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_type_description_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetTypeDescription(
		context.Background(),
		linkGetTypeDescriptionInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred when retrieving type description for link")
}

func (s *ProviderPluginV1Suite) Test_link_get_annotation_definitions() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.GetAnnotationDefinitions(
		context.Background(),
		linkGetAnnotationDefnitionsInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testprovider.LinkLambdaFunctionDDBTableAnnotations(),
		output.AnnotationDefinitions,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_annotation_definitions_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetAnnotationDefinitions(
		context.Background(),
		linkGetAnnotationDefnitionsInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetLinkAnnotationDefinitions,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_annotation_definitions_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetAnnotationDefinitions(
		context.Background(),
		linkGetAnnotationDefnitionsInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred when retrieving annotation definitions for link",
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_kind() {
	link, err := s.provider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	output, err := link.GetKind(
		context.Background(),
		linkGetKindInput(),
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&provider.LinkGetKindOutput{
			Kind: provider.LinkKindHard,
		},
		output,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_kind_fails_for_unexpected_host() {
	link, err := s.providerWrongHost.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetKind(
		context.Background(),
		linkGetKindInput(),
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionProviderGetLinkKind,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *ProviderPluginV1Suite) Test_link_get_kind_reports_expected_error_for_failure() {
	link, err := s.failingProvider.Link(
		context.Background(),
		lambdaFunctionResourceType,
		dynamoDBTableResourceType,
	)
	s.Require().NoError(err)

	_, err = link.GetKind(
		context.Background(),
		linkGetKindInput(),
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred when retrieving link kind",
	)
}
