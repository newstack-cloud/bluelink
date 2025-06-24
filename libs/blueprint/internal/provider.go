package internal

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type ProviderMock struct {
	NamespaceValue      string
	Resources           map[string]provider.Resource
	DataSources         map[string]provider.DataSource
	Links               map[string]provider.Link
	CustomVariableTypes map[string]provider.CustomVariableType
	ProviderRetryPolicy *provider.RetryPolicy
}

func (p *ProviderMock) Namespace(ctx context.Context) (string, error) {
	return p.NamespaceValue, nil
}

func (p *ProviderMock) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (p *ProviderMock) Resource(ctx context.Context, resourceType string) (provider.Resource, error) {
	return p.Resources[resourceType], nil
}

func (p *ProviderMock) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (provider.Link, error) {
	linkKey := fmt.Sprintf("%s::%s", resourceTypeA, resourceTypeB)
	return p.Links[linkKey], nil
}

func (p *ProviderMock) DataSource(ctx context.Context, dataSourceType string) (provider.DataSource, error) {
	return p.DataSources[dataSourceType], nil
}

func (p *ProviderMock) CustomVariableType(ctx context.Context, customVariableType string) (provider.CustomVariableType, error) {
	return p.CustomVariableTypes[customVariableType], nil
}

func (p *ProviderMock) ListFunctions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *ProviderMock) ListResourceTypes(ctx context.Context) ([]string, error) {
	resourceTypes := make([]string, 0, len(p.Resources))
	for resourceType := range p.Resources {
		resourceTypes = append(resourceTypes, resourceType)
	}
	return resourceTypes, nil
}

func (p *ProviderMock) ListLinkTypes(ctx context.Context) ([]string, error) {
	linkTypes := make([]string, 0, len(p.Links))
	for linkType := range p.Links {
		linkTypes = append(linkTypes, linkType)
	}
	return linkTypes, nil
}

func (p *ProviderMock) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	dataSourceTypes := make([]string, 0, len(p.DataSources))
	for dataSourceType := range p.DataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}
	return dataSourceTypes, nil
}

func (p *ProviderMock) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	customVariableTypes := make([]string, 0, len(p.CustomVariableTypes))
	for customVariableType := range p.CustomVariableTypes {
		customVariableTypes = append(customVariableTypes, customVariableType)
	}
	return customVariableTypes, nil
}

func (p *ProviderMock) Function(ctx context.Context, functionName string) (provider.Function, error) {
	return nil, nil
}

func (p *ProviderMock) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return p.ProviderRetryPolicy, nil
}
