package blueprint

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type bluelinkProvider struct {
	resources       map[string]provider.Resource
	dataSources     map[string]provider.DataSource
	customVariables map[string]provider.CustomVariableType
}

func NewBluelinkProvider() provider.Provider {
	return &bluelinkProvider{
		resources: map[string]provider.Resource{
			"bluelink/handler": &bluelinkHandlerResource{},
		},
		dataSources: map[string]provider.DataSource{
			"bluelink/vpc": &bluelinkVPCDataSource{},
		},
		customVariables: map[string]provider.CustomVariableType{
			"bluelink/customVariable": &bluelinkCustomVariableType{},
		},
	}
}

func (p *bluelinkProvider) Namespace(ctx context.Context) (string, error) {
	return "bluelink", nil
}

func (p *bluelinkProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (p *bluelinkProvider) Resource(ctx context.Context, resourceType string) (provider.Resource, error) {
	resource, hasResource := p.resources[resourceType]
	if !hasResource {
		return nil, errors.New("resource not found")
	}

	return resource, nil
}

func (p *bluelinkProvider) DataSource(ctx context.Context, dataSourceType string) (provider.DataSource, error) {
	dataSource, hasDataSource := p.dataSources[dataSourceType]
	if !hasDataSource {
		return nil, errors.New("data source not found")
	}

	return dataSource, nil
}

func (p *bluelinkProvider) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (provider.Link, error) {
	return nil, errors.New("links not implemented")
}

func (p *bluelinkProvider) CustomVariableType(ctx context.Context, customVariableType string) (provider.CustomVariableType, error) {
	customVarType, hasCustomVarType := p.customVariables[customVariableType]
	if !hasCustomVarType {
		return nil, errors.New("custom variable type not found")
	}

	return customVarType, nil
}

func (p *bluelinkProvider) ListResourceTypes(ctx context.Context) ([]string, error) {
	resourceTypes := []string{}
	for resourceType := range p.resources {
		resourceTypes = append(resourceTypes, resourceType)
	}

	return resourceTypes, nil
}

func (p *bluelinkProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *bluelinkProvider) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	dataSourceTypes := []string{}
	for dataSourceType := range p.dataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}

	return dataSourceTypes, nil
}

func (p *bluelinkProvider) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	customVariableTypes := []string{}
	for customVariableType := range p.customVariables {
		customVariableTypes = append(customVariableTypes, customVariableType)
	}

	return customVariableTypes, nil
}

func (p *bluelinkProvider) ListFunctions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *bluelinkProvider) Function(ctx context.Context, functionName string) (provider.Function, error) {
	return nil, errors.New("functions not implemented")
}

func (p *bluelinkProvider) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}

type bluelinkHandlerResource struct{}

func (r *bluelinkHandlerResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{},
	}, nil
}

func (r *bluelinkHandlerResource) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: false,
	}, nil
}

func (r *bluelinkHandlerResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type: "bluelink/handler",
	}, nil
}

func (d *bluelinkHandlerResource) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		MarkdownDescription: "A resource that represents a handler for a Bluelink application.\n\n" +
			"[`bluelink/handler` resource docs](https://www.bluelinkframework.com/docs/resources/bluelink-handler)",
		PlainTextDescription: "A resource that represents a handler for a Bluelink application.",
	}, nil
}

func (r *bluelinkHandlerResource) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *bluelinkHandlerResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Description: "A resource that represents a handler for a Bluelink application.",
				Type:        provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Description: "The ID of the handler in the deployed environment.",
						Type:        provider.ResourceDefinitionsSchemaTypeString,
						Computed:    true,
					},
					"handlerName": {
						Description: "The name of the handler.",
						Type:        provider.ResourceDefinitionsSchemaTypeString,
					},
					"runtime": {
						Description: "The runtime that the handler uses.",
						Type:        provider.ResourceDefinitionsSchemaTypeString,
					},
					"info": {
						Description: "Additional information about the handler.",
						Type:        provider.ResourceDefinitionsSchemaTypeObject,
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"applicationId": {
								Description: "The ID of the application that the handler is part of.",
								Type:        provider.ResourceDefinitionsSchemaTypeString,
							},
						},
					},
				},
			},
		},
	}, nil
}

func (r *bluelinkHandlerResource) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{
		StabilisedDependencies: []string{},
	}, nil
}

func (r *bluelinkHandlerResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{}, nil
}

func (r *bluelinkHandlerResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{}, nil
}

func (r *bluelinkHandlerResource) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

func (r *bluelinkHandlerResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *bluelinkHandlerResource) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}

type bluelinkVPCDataSource struct{}

func (d *bluelinkVPCDataSource) GetSpecDefinition(
	ctx context.Context,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	return &provider.DataSourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.DataSourceSpecDefinition{
			Fields: map[string]*provider.DataSourceSpecSchema{
				"vpcId": {
					Description: "The ID of the VPC.",
					Type:        provider.DataSourceSpecTypeString,
				},
				"subnetIds": {
					Description: "The IDs of subnets in the VPC.",
					Type:        provider.DataSourceSpecTypeArray,
					Items: &provider.DataSourceSpecSchema{
						Description: "The ID of a subnet.",
						Type:        provider.DataSourceSpecTypeString,
					},
				},
			},
		},
	}, nil
}

func (d *bluelinkVPCDataSource) Fetch(
	ctx context.Context,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	return &provider.DataSourceFetchOutput{
		Data: map[string]*core.MappingNode{},
	}, nil
}

func (d *bluelinkVPCDataSource) GetType(
	ctx context.Context,
	input *provider.DataSourceGetTypeInput,
) (*provider.DataSourceGetTypeOutput, error) {
	return &provider.DataSourceGetTypeOutput{
		Type: "test/exampleDataSource",
	}, nil
}

func (d *bluelinkVPCDataSource) GetTypeDescription(
	ctx context.Context,
	input *provider.DataSourceGetTypeDescriptionInput,
) (*provider.DataSourceGetTypeDescriptionOutput, error) {
	return &provider.DataSourceGetTypeDescriptionOutput{
		MarkdownDescription:  "A data source that pulls in bluelink network information.",
		PlainTextDescription: "A data source that pulls in bluelink network information.",
	}, nil
}

func (d *bluelinkVPCDataSource) GetFilterFields(
	ctx context.Context,
	input *provider.DataSourceGetFilterFieldsInput,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	return &provider.DataSourceGetFilterFieldsOutput{
		FilterFields: map[string]*provider.DataSourceFilterSchema{
			"tags": {
				Type: provider.DataSourceFilterSearchValueTypeString,
			},
			"vpcId": {
				Type: provider.DataSourceFilterSearchValueTypeString,
			},
		},
	}, nil
}

func (d *bluelinkVPCDataSource) CustomValidate(
	ctx context.Context,
	input *provider.DataSourceValidateInput,
) (*provider.DataSourceValidateOutput, error) {
	return &provider.DataSourceValidateOutput{
		Diagnostics: []*core.Diagnostic{},
	}, nil
}

func (r *bluelinkVPCDataSource) GetExamples(
	ctx context.Context,
	input *provider.DataSourceGetExamplesInput,
) (*provider.DataSourceGetExamplesOutput, error) {
	return &provider.DataSourceGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}

type bluelinkCustomVariableType struct{}

func (t *bluelinkCustomVariableType) Options(
	ctx context.Context,
	input *provider.CustomVariableTypeOptionsInput,
) (*provider.CustomVariableTypeOptionsOutput, error) {
	t2nano := "t2.nano"
	t2micro := "t2.micro"
	t2small := "t2.small"
	t2medium := "t2.medium"
	t2large := "t2.large"
	t2xlarge := "t2.xlarge"
	t22xlarge := "t2.2xlarge"
	return &provider.CustomVariableTypeOptionsOutput{
		Options: map[string]*provider.CustomVariableTypeOption{
			t2nano: {
				Value: &core.ScalarValue{
					StringValue: &t2nano,
				},
			},
			t2micro: {
				Value: &core.ScalarValue{
					StringValue: &t2micro,
				},
			},
			t2small: {
				Value: &core.ScalarValue{
					StringValue: &t2small,
				},
			},
			t2medium: {
				Value: &core.ScalarValue{
					StringValue: &t2medium,
				},
			},
			t2large: {
				Value: &core.ScalarValue{
					StringValue: &t2large,
				},
			},
			t2xlarge: {
				Value: &core.ScalarValue{
					StringValue: &t2xlarge,
				},
			},
			t22xlarge: {
				Value: &core.ScalarValue{
					StringValue: &t22xlarge,
				},
			},
		},
	}, nil
}

func (t *bluelinkCustomVariableType) GetType(
	ctx context.Context,
	input *provider.CustomVariableTypeGetTypeInput,
) (*provider.CustomVariableTypeGetTypeOutput, error) {
	return &provider.CustomVariableTypeGetTypeOutput{
		Type: "bluelink/customVariable",
	}, nil
}

func (t *bluelinkCustomVariableType) GetDescription(
	ctx context.Context,
	input *provider.CustomVariableTypeGetDescriptionInput,
) (*provider.CustomVariableTypeGetDescriptionOutput, error) {
	return &provider.CustomVariableTypeGetDescriptionOutput{
		MarkdownDescription:  "### Bluelink Custom Variable\n\nA custom variable type for Bluelink.",
		PlainTextDescription: "Bluelink Custom Variable\n\nA custom variable type for Bluelink.",
	}, nil
}

func (t *bluelinkCustomVariableType) GetExamples(
	ctx context.Context,
	input *provider.CustomVariableTypeGetExamplesInput,
) (*provider.CustomVariableTypeGetExamplesOutput, error) {
	return &provider.CustomVariableTypeGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}
