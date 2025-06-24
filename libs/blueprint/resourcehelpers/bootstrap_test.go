package resourcehelpers

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

// //////////////////////////////////////
// Provider
// //////////////////////////////////////

type testProvider struct {
	functions      map[string]provider.Function
	links          map[string]provider.Link
	resources      map[string]provider.Resource
	dataSources    map[string]provider.DataSource
	customVarTypes map[string]provider.CustomVariableType
	namespace      string
}

func (p *testProvider) Namespace(ctx context.Context) (string, error) {
	return p.namespace, nil
}

func (p *testProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (p *testProvider) Resource(ctx context.Context, resourceType string) (provider.Resource, error) {
	resource, ok := p.resources[resourceType]
	if !ok {
		return nil, errors.New("resource not found")
	}
	return resource, nil
}

func (p *testProvider) DataSource(ctx context.Context, dataSourceType string) (provider.DataSource, error) {
	dataSource, ok := p.dataSources[dataSourceType]
	if !ok {
		return nil, errors.New("data source not found")
	}
	return dataSource, nil
}

func (p *testProvider) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (provider.Link, error) {
	link, ok := p.links[resourceTypeA+resourceTypeB]
	if !ok {
		return nil, errors.New("link not found")
	}
	return link, nil
}

func (p *testProvider) CustomVariableType(ctx context.Context, customVariableType string) (provider.CustomVariableType, error) {
	return nil, nil
}

func (p *testProvider) ListResourceTypes(ctx context.Context) ([]string, error) {
	resourceTypes := []string{}
	for resourceType := range p.resources {
		resourceTypes = append(resourceTypes, resourceType)
	}
	return resourceTypes, nil
}

func (p *testProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	linkTypes := []string{}
	for linkType := range p.links {
		linkTypes = append(linkTypes, linkType)
	}
	return linkTypes, nil
}

func (p *testProvider) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	dataSourceTypes := []string{}
	for dataSourceType := range p.dataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}
	return dataSourceTypes, nil
}

func (p *testProvider) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	customVarTypes := []string{}
	for customVarType := range p.customVarTypes {
		customVarTypes = append(customVarTypes, customVarType)
	}
	return customVarTypes, nil
}

func (p *testProvider) ListFunctions(ctx context.Context) ([]string, error) {
	functionNames := []string{}
	for name := range p.functions {
		functionNames = append(functionNames, name)
	}
	return functionNames, nil
}

func (p *testProvider) Function(ctx context.Context, functionName string) (provider.Function, error) {
	function, ok := p.functions[functionName]
	if !ok {
		return nil, errors.New("function not found")
	}
	return function, nil
}

func (p *testProvider) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}

type testExampleResource struct {
	definition           *provider.ResourceSpecDefinition
	markdownDescription  string
	plainTextDescription string
	markdownExamples     []string
	plainTextExamples    []string
}

func newTestExampleResource() provider.Resource {
	return &testExampleResource{
		definition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"name": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"ids": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeObject,
							Attributes: map[string]*provider.ResourceDefinitionsSchema{
								"name": {
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
							},
						},
					},
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
				},
			},
		},
		markdownDescription:  "## bluelink/exampleResource\n\nThis is an example resource.",
		plainTextDescription: "bluelink/exampleResource\n\nThis is an example resource.",
		markdownExamples:     []string{},
		plainTextExamples:    []string{},
	}
}

// CanLinkTo is not used for validation!
func (r *testExampleResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{}, nil
}

// StabilisedDependencies is not used for validation!
func (r *testExampleResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{}, nil
}

// IsCommonTerminal is not used for validation!
func (r *testExampleResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *testExampleResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "bluelink/exampleResource",
	}, nil
}

func (r *testExampleResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		MarkdownDescription:  r.markdownDescription,
		PlainTextDescription: r.plainTextDescription,
	}, nil
}

func (r *testExampleResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  r.markdownExamples,
		PlainTextExamples: r.plainTextExamples,
	}, nil
}

func (r *testExampleResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *testExampleResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: r.definition,
	}, nil
}

func (r *testExampleResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString("test-example-resource-item-id-1"),
		},
	}, nil
}

func (r *testExampleResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

// GetExternalState is not used for validation!
func (r *testExampleResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{}, nil
}

// Destroy is not used for validation!
func (r *testExampleResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

// //////////////////////////////////////
// Spec transformer
// //////////////////////////////////////

type testSpecTransformer struct {
	abstractResources map[string]transform.AbstractResource
}

func (t *testSpecTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "celerity-test", nil
}

func (t *testSpecTransformer) ConfigDefinition(
	ctx context.Context,
) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (t *testSpecTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
	}, nil
}

func (t *testSpecTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	abstractResource, ok := t.abstractResources[resourceType]
	if !ok {
		return nil, errors.New("abstract resource not found")
	}
	return abstractResource, nil
}

func (t *testSpecTransformer) ListAbstractResourceTypes(
	ctx context.Context,
) ([]string, error) {
	abstractResourceTypes := []string{}
	for abstractResourceType := range t.abstractResources {
		abstractResourceTypes = append(abstractResourceTypes, abstractResourceType)
	}
	return abstractResourceTypes, nil
}

type testExampleAbstractResource struct{}

func newTestExampleAbstractResource() transform.AbstractResource {
	return &testExampleAbstractResource{}
}

func (r *testExampleAbstractResource) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	return &transform.AbstractResourceGetTypeOutput{
		Type: "test/exampleAbstractResource",
	}, nil
}

func (r *testExampleAbstractResource) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	return &transform.AbstractResourceGetTypeDescriptionOutput{}, nil
}

func (r *testExampleAbstractResource) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	return &transform.AbstractResourceGetExamplesOutput{}, nil
}

func (r *testExampleAbstractResource) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	return &transform.AbstractResourceCanLinkToOutput{}, nil
}

func (r *testExampleAbstractResource) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	return &transform.AbstractResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *testExampleAbstractResource) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return &transform.AbstractResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *testExampleAbstractResource) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	return &transform.AbstractResourceGetSpecDefinitionOutput{}, nil
}
