package transformerserverv1

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type TransformerCanLinkToWrapperTestSuite struct {
	suite.Suite
}

func TestTransformerCanLinkToWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(TransformerCanLinkToWrapperTestSuite))
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{
			"test/resource/a::test/resource/b",
			"test/resource/a::test/resource/c",
		},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.ElementsMatch([]string{"test/resource/b", "test/resource/c"}, output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to_reverse() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{
			"test/resource/a::test/resource/b",
		},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/b")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.ElementsMatch([]string{"test/resource/a"}, output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to_deduplicates() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{
			"test/resource/a::test/resource/b",
			"test/resource/a::test/resource/b",
		},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.Equal([]string{"test/resource/b"}, output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to_resources_of_the_same_type() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{
			"test/resource/a::test/resource/a",
		},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.ElementsMatch([]string{"test/resource/a"}, output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to_unknown_resource() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{
			"test/resource/a::test/resource/b",
		},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/unknown")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.Empty(output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to_no_link_types() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.Empty(output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_can_link_to_malformed_link_type() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{
			"malformed-link-type",
			"test/resource/a::test/resource/b",
		},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	output, err := resource.CanLinkTo(context.Background(), &transform.AbstractResourceCanLinkToInput{})
	s.Require().NoError(err)

	s.ElementsMatch([]string{"test/resource/b"}, output.CanLinkTo)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_transformer_delegates_other_methods() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{"test/resource/a::test/resource/b"},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	name, err := wrapped.GetTransformName(context.Background())
	s.Require().NoError(err)
	s.Equal("test-transform", name)
}

func (s *TransformerCanLinkToWrapperTestSuite) Test_wrapped_abstract_resource_delegates_get_type() {
	mockTransformer := &mockTransformerForCanLinkToWrapper{
		linkTypes: []string{"test/resource/a::test/resource/b"},
	}

	wrapped := WrapTransformerWithDerivedCanLinkTo(mockTransformer)

	resource, err := wrapped.AbstractResource(context.Background(), "test/resource/a")
	s.Require().NoError(err)

	typeOutput, err := resource.GetType(context.Background(), &transform.AbstractResourceGetTypeInput{})
	s.Require().NoError(err)
	s.Equal("test/resource/a", typeOutput.Type)
	s.Equal("Test Resource A", typeOutput.Label)
}

// mockTransformerForCanLinkToWrapper implements transform.SpecTransformer for testing
type mockTransformerForCanLinkToWrapper struct {
	linkTypes []string
}

func (m *mockTransformerForCanLinkToWrapper) GetTransformName(
	ctx context.Context,
) (string, error) {
	return "test-transform", nil
}

func (m *mockTransformerForCanLinkToWrapper) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (m *mockTransformerForCanLinkToWrapper) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
	}, nil
}

func (m *mockTransformerForCanLinkToWrapper) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{}, nil
}

func (m *mockTransformerForCanLinkToWrapper) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	return &mockAbstractResourceForCanLinkToWrapper{resourceType: resourceType}, nil
}

func (m *mockTransformerForCanLinkToWrapper) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	return &mockAbstractLinkForCanLinkToWrapper{linkType: linkType}, nil
}

func (m *mockTransformerForCanLinkToWrapper) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	return []string{"test/resource/a", "test/resource/b"}, nil
}

func (m *mockTransformerForCanLinkToWrapper) ListAbstractLinkTypes(
	ctx context.Context,
) ([]string, error) {
	return m.linkTypes, nil
}

// mockAbstractResourceForCanLinkToWrapper implements transform.AbstractResource for testing
type mockAbstractResourceForCanLinkToWrapper struct {
	resourceType string
}

func (m *mockAbstractResourceForCanLinkToWrapper) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return &transform.AbstractResourceValidateOutput{}, nil
}

func (m *mockAbstractResourceForCanLinkToWrapper) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{},
	}, nil
}

func (m *mockAbstractResourceForCanLinkToWrapper) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	return &transform.AbstractResourceCanLinkToOutput{CanLinkTo: []string{"original/value"}}, nil
}

func (m *mockAbstractResourceForCanLinkToWrapper) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	return &transform.AbstractResourceIsCommonTerminalOutput{}, nil
}

func (m *mockAbstractResourceForCanLinkToWrapper) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{
		Type:  m.resourceType,
		Label: "Test Resource A",
	}, nil
}

func (m *mockAbstractResourceForCanLinkToWrapper) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{}, nil
}

func (m *mockAbstractResourceForCanLinkToWrapper) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{}, nil
}

// mockAbstractLinkForCanLinkToWrapper implements transform.AbstractLink for testing
type mockAbstractLinkForCanLinkToWrapper struct {
	linkType string
}

func (m *mockAbstractLinkForCanLinkToWrapper) GetType(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeInput,
) (*transform.AbstractLinkGetTypeOutput, error) {
	return &transform.AbstractLinkGetTypeOutput{
		Type:          m.linkType,
		ResourceTypeA: "test/resource/a",
		ResourceTypeB: "test/resource/b",
	}, nil
}

func (m *mockAbstractLinkForCanLinkToWrapper) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeDescriptionInput,
) (*transform.AbstractLinkGetTypeDescriptionOutput, error) {
	return &transform.AbstractLinkGetTypeDescriptionOutput{}, nil
}

func (m *mockAbstractLinkForCanLinkToWrapper) GetAnnotationDefinitions(
	ctx context.Context,
	input *transform.AbstractLinkGetAnnotationDefinitionsInput,
) (*transform.AbstractLinkGetAnnotationDefinitionsOutput, error) {
	return &transform.AbstractLinkGetAnnotationDefinitionsOutput{}, nil
}

func (m *mockAbstractLinkForCanLinkToWrapper) GetCardinality(
	ctx context.Context,
	input *transform.AbstractLinkGetCardinalityInput,
) (*transform.AbstractLinkGetCardinalityOutput, error) {
	return &transform.AbstractLinkGetCardinalityOutput{}, nil
}

// Ensure mocks satisfy their interfaces at compile time.
var (
	_ transform.SpecTransformer  = (*mockTransformerForCanLinkToWrapper)(nil)
	_ transform.AbstractResource = (*mockAbstractResourceForCanLinkToWrapper)(nil)
	_ transform.AbstractLink     = (*mockAbstractLinkForCanLinkToWrapper)(nil)
)
