package links

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

const testCelerityTransformName = "celerity-2026-04-01"

type testCelerityTransformer struct {
	abstractResources map[string]transform.AbstractResource
}

func newTestCelerityTransformer() transform.SpecTransformer {
	return &testCelerityTransformer{
		abstractResources: map[string]transform.AbstractResource{
			"celerity/handler":   &testCelerityHandlerResource{},
			"celerity/datastore": &testCelerityDatastoreResource{},
			"celerity/queue":     &testCelerityQueueResource{},
		},
	}
}

func (t *testCelerityTransformer) GetTransformName(ctx context.Context) (string, error) {
	return testCelerityTransformName, nil
}

func (t *testCelerityTransformer) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{},
	}, nil
}

func (t *testCelerityTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
	}, nil
}

func (t *testCelerityTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (t *testCelerityTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	return t.abstractResources[resourceType], nil
}

func (t *testCelerityTransformer) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	types := make([]string, 0, len(t.abstractResources))
	for resourceType := range t.abstractResources {
		types = append(types, resourceType)
	}
	return types, nil
}

func (t *testCelerityTransformer) ListAbstractLinkTypes(
	ctx context.Context,
) ([]string, error) {
	return []string{}, nil
}

func (t *testCelerityTransformer) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	return nil, errors.New("no abstract links defined for testCelerityTransformer")
}

type testCelerityHandlerResource struct{}

func (r *testCelerityHandlerResource) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return &transform.AbstractResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *testCelerityHandlerResource) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"handler": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *testCelerityHandlerResource) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	return &transform.AbstractResourceCanLinkToOutput{
		CanLinkTo: []string{"celerity/datastore", "celerity/queue"},
	}, nil
}

func (r *testCelerityHandlerResource) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	return &transform.AbstractResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *testCelerityHandlerResource) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{
		Type:  "celerity/handler",
		Label: "Celerity Handler",
	}, nil
}

func (r *testCelerityHandlerResource) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{}, nil
}

func (r *testCelerityHandlerResource) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{}, nil
}

type testCelerityDatastoreResource struct{}

func (r *testCelerityDatastoreResource) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return &transform.AbstractResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *testCelerityDatastoreResource) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"name": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *testCelerityDatastoreResource) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	return &transform.AbstractResourceCanLinkToOutput{
		CanLinkTo: []string{},
	}, nil
}

func (r *testCelerityDatastoreResource) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	return &transform.AbstractResourceIsCommonTerminalOutput{
		IsCommonTerminal: true,
	}, nil
}

func (r *testCelerityDatastoreResource) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{
		Type:  "celerity/datastore",
		Label: "Celerity Datastore",
	}, nil
}

func (r *testCelerityDatastoreResource) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{}, nil
}

func (r *testCelerityDatastoreResource) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{}, nil
}

type testCelerityQueueResource struct{}

func (r *testCelerityQueueResource) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return &transform.AbstractResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *testCelerityQueueResource) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"name": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
				},
			},
		},
	}, nil
}

func (r *testCelerityQueueResource) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	return &transform.AbstractResourceCanLinkToOutput{
		CanLinkTo: []string{"celerity/handler"},
	}, nil
}

func (r *testCelerityQueueResource) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	return &transform.AbstractResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *testCelerityQueueResource) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{
		Type:  "celerity/queue",
		Label: "Celerity Queue",
	}, nil
}

func (r *testCelerityQueueResource) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{}, nil
}

func (r *testCelerityQueueResource) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{}, nil
}
