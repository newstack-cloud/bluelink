package providerserverv1

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type CanLinkToWrapperTestSuite struct {
	suite.Suite
}

func TestCanLinkToWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(CanLinkToWrapperTestSuite))
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_finds_direct_links() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
		"aws/lambda/function::aws/sqs/queue",
		"aws/ec2/instance::aws/ebs/volume",
	}

	result := deriveLinkableTypes("aws/lambda/function", allLinkTypes)

	s.ElementsMatch([]string{"aws/dynamodb/table", "aws/sqs/queue"}, result)
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_finds_reverse_links() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
	}

	result := deriveLinkableTypes("aws/dynamodb/table", allLinkTypes)

	s.Equal([]string{"aws/lambda/function"}, result)
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_deduplicates() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
		"aws/lambda/function::aws/dynamodb/table",
	}

	result := deriveLinkableTypes("aws/lambda/function", allLinkTypes)

	s.Equal([]string{"aws/dynamodb/table"}, result)
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_empty_for_unknown_resource() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
	}

	result := deriveLinkableTypes("gcp/compute/instance", allLinkTypes)

	s.Empty(result)
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_handles_malformed_link_types() {
	allLinkTypes := []string{
		"aws/lambda/function::aws/dynamodb/table",
		"malformed-link-type",
		"",
	}

	result := deriveLinkableTypes("aws/lambda/function", allLinkTypes)

	s.Equal([]string{"aws/dynamodb/table"}, result)
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_handles_empty_input() {
	result := deriveLinkableTypes("aws/lambda/function", []string{})

	s.Empty(result)
}

func (s *CanLinkToWrapperTestSuite) Test_deriveLinkableTypes_handles_nil_input() {
	result := deriveLinkableTypes("aws/lambda/function", nil)

	s.Empty(result)
}

func (s *CanLinkToWrapperTestSuite) Test_wrapped_provider_resource_can_link_to() {
	mockProvider := &mockProviderForCanLinkToWrapper{}
	allLinkTypes := []string{
		"test/resource/a::test/resource/b",
		"test/resource/a::test/resource/c",
	}

	wrapped := WrapProviderWithDerivedCanLinkTo(mockProvider, allLinkTypes)

	resource, err := wrapped.Resource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &provider.ResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.ElementsMatch([]string{"test/resource/b", "test/resource/c"}, output.CanLinkTo)
}

func (s *CanLinkToWrapperTestSuite) Test_wrapped_provider_resource_can_link_to_resources_of_the_same_type() {
	mockProvider := &mockProviderForCanLinkToWrapper{}
	allLinkTypes := []string{
		"test/resource/a::test/resource/a",
	}

	wrapped := WrapProviderWithDerivedCanLinkTo(mockProvider, allLinkTypes)

	resource, err := wrapped.Resource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &provider.ResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.ElementsMatch([]string{"test/resource/a"}, output.CanLinkTo)
}

func (s *CanLinkToWrapperTestSuite) Test_wrapped_provider_delegates_other_methods() {
	mockProvider := &mockProviderForCanLinkToWrapper{}
	allLinkTypes := []string{"test/resource/a::test/resource/b"}

	wrapped := WrapProviderWithDerivedCanLinkTo(mockProvider, allLinkTypes)

	namespace, err := wrapped.Namespace(context.Background())
	s.Require().NoError(err)
	s.Equal("test", namespace)
}

func (s *CanLinkToWrapperTestSuite) Test_wrapped_resource_delegates_get_type() {
	mockProvider := &mockProviderForCanLinkToWrapper{}
	allLinkTypes := []string{"test/resource/a::test/resource/b"}

	wrapped := WrapProviderWithDerivedCanLinkTo(mockProvider, allLinkTypes)

	resource, err := wrapped.Resource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	typeOutput, err := resource.GetType(context.Background(), &provider.ResourceGetTypeInput{})
	s.Require().NoError(err)
	s.Equal("test/resource/a", typeOutput.Type)
	s.Equal("Test Resource A", typeOutput.Label)
}

// mockProviderForCanLinkToWrapper implements provider.Provider for testing
type mockProviderForCanLinkToWrapper struct{}

func (m *mockProviderForCanLinkToWrapper) Namespace(ctx context.Context) (string, error) {
	return "test", nil
}

func (m *mockProviderForCanLinkToWrapper) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) Resource(
	ctx context.Context,
	resourceType string,
) (provider.Resource, error) {
	return &mockResourceForCanLinkToWrapper{resourceType: resourceType}, nil
}

func (m *mockProviderForCanLinkToWrapper) DataSource(
	ctx context.Context,
	dataSourceType string,
) (provider.DataSource, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) Link(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (provider.Link, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) CustomVariableType(
	ctx context.Context,
	customVariableType string,
) (provider.CustomVariableType, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) ListFunctions(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) ListResourceTypes(ctx context.Context) ([]string, error) {
	return []string{"test/resource/a", "test/resource/b"}, nil
}

func (m *mockProviderForCanLinkToWrapper) ListLinkTypes(ctx context.Context) ([]string, error) {
	return []string{"test/resource/a::test/resource/b"}, nil
}

func (m *mockProviderForCanLinkToWrapper) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) Function(
	ctx context.Context,
	functionName string,
) (provider.Function, error) {
	return nil, nil
}

func (m *mockProviderForCanLinkToWrapper) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}

// mockResourceForCanLinkToWrapper implements provider.Resource for testing
type mockResourceForCanLinkToWrapper struct {
	resourceType string
}

func (m *mockResourceForCanLinkToWrapper) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{CanLinkTo: []string{"original/value"}}, nil
}

func (m *mockResourceForCanLinkToWrapper) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type:  m.resourceType,
		Label: "Test Resource A",
	}, nil
}

func (m *mockResourceForCanLinkToWrapper) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{Stabilised: true}, nil
}

func (m *mockResourceForCanLinkToWrapper) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{}, nil
}

func (m *mockResourceForCanLinkToWrapper) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}
