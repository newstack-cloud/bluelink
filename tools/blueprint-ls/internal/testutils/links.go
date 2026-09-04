package testutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// MockLink provides a mock implementation of provider.Link for testing.
// Only GetAnnotationDefinitions and GetCardinality honour configured values;
// other methods return empty/nil outputs.
type MockLink struct {
	provider.LinkNoCapabilities

	AnnotationDefs map[string]*provider.LinkAnnotationDefinition
	CardinalityA   provider.LinkCardinality
	CardinalityB   provider.LinkCardinality
}

func (l *MockLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *MockLink) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	return &provider.LinkProduceResourceContributionsOutput{}, nil
}

func (l *MockLink) UpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	return &provider.LinkUpdateLinkedResourcesOutput{}, nil
}

func (l *MockLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *MockLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{}, nil
}

func (l *MockLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{}, nil
}

func (l *MockLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *MockLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: l.AnnotationDefs,
	}, nil
}

func (l *MockLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{}, nil
}

func (l *MockLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *MockLink) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{
		CardinalityA: l.CardinalityA,
		CardinalityB: l.CardinalityB,
	}, nil
}

func (l *MockLink) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}
