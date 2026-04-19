package testutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// MockTransformer is a mock implementation of the
// `transform.SpecTransformer` interface
// for plugins in launch testing.
type MockTransformer struct {
	TransformName string
}

func (m *MockTransformer) GetTransformName(ctx context.Context) (string, error) {
	return m.TransformName, nil
}

func (m *MockTransformer) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{}, nil
}

func (m *MockTransformer) Transform(
	ctx context.Context,
	inputs *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return nil, nil
}

func (m *MockTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	return nil, nil
}

func (m *MockTransformer) ListAbstractResourceTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockTransformer) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	return nil, nil
}

func (m *MockTransformer) ListAbstractLinkTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return nil, nil
}
