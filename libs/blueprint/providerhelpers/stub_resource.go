package providerhelpers

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

const (
	stubResourceType = "core/stub"
)

type stubResource struct{}

func newStubResource() provider.Resource {
	return &stubResource{}
}

func (r *stubResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{}, nil
}

func (r *stubResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Description: "A stub resource that does nothing. " +
					"This is primarily useful for placeholder blueprints that can be used to load " +
					"a blueprint to be able to access functionality to destroy blueprint instances.",
				Required: []string{"value"},
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:        provider.ResourceDefinitionsSchemaTypeString,
						Description: "The ID of the resource.",
					},
					"value": {
						Type:        provider.ResourceDefinitionsSchemaTypeString,
						Description: "A placeholder value for the resource.",
					},
				},
			},
			IDField: "id",
		},
	}, nil
}

func (r *stubResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{},
	}, nil
}

func (r *stubResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{
		StabilisedDependencies: []string{},
	}, nil
}

func (r *stubResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *stubResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type:  stubResourceType,
		Label: "Stub Resource",
	}, nil
}

func (r *stubResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextSummary: "A stub resource that does nothing.",
		PlainTextDescription: "A stub resource that does nothing. " +
			"This is primarily useful for placeholder blueprints that can be used to load " +
			"a blueprint to be able to access functionality to destroy blueprint instances.",
	}, nil
}

func (r *stubResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  []string{},
		PlainTextExamples: []string{},
	}, nil
}

func (r *stubResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{}, nil
}

func (r *stubResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *stubResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{}, nil
}

func (r *stubResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}
