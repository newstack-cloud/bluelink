package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// ProviderPluginDefinition is a template to be used when creating provider plugins.
// It provides a structure that allows you to define the resources, data sources,
// links and custom variable types that are supported by the provider plugin.
// This doesn't have to be used but is a useful way to define the plugin's capabilities,
// there are multiple convenience functions to create new plugins.
// This implements the `provider.Provider` interface and can be used in the same way
// as any other provider implementation to create a provider plugin.
type ProviderPluginDefinition struct {
	ProviderNamespace        string
	ProviderConfigDefinition *core.ConfigDefinition
	Resources                map[string]provider.Resource
	DataSources              map[string]provider.DataSource
	Links                    map[string]provider.Link
	CustomVariableTypes      map[string]provider.CustomVariableType
	Functions                map[string]provider.Function
	ProviderRetryPolicy      *provider.RetryPolicy
}

func (p *ProviderPluginDefinition) Namespace(ctx context.Context) (string, error) {
	return p.ProviderNamespace, nil
}

func (p *ProviderPluginDefinition) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return p.ProviderConfigDefinition, nil
}

func (p *ProviderPluginDefinition) Resource(
	ctx context.Context,
	resourceType string,
) (provider.Resource, error) {
	resource, ok := p.Resources[resourceType]
	if !ok {
		return nil, errResourceTypeNotFound(resourceType)
	}
	return resource, nil
}

func (p *ProviderPluginDefinition) DataSource(
	ctx context.Context,
	dataSourceType string,
) (provider.DataSource, error) {
	dataSource, ok := p.DataSources[dataSourceType]
	if !ok {
		return nil, errDataSourceTypeNotFound(dataSourceType)
	}
	return dataSource, nil
}

func (p *ProviderPluginDefinition) Link(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (provider.Link, error) {
	linkType := core.LinkType(resourceTypeA, resourceTypeB)
	link, ok := p.Links[linkType]
	if !ok {
		return nil, errLinkTypeNotFound(linkType)
	}
	return link, nil
}

func (p *ProviderPluginDefinition) CustomVariableType(
	ctx context.Context,
	customVariableType string,
) (provider.CustomVariableType, error) {
	customVariable, ok := p.CustomVariableTypes[customVariableType]
	if !ok {
		return nil, errCustomVariableTypeNotFound(customVariableType)
	}
	return customVariable, nil
}

func (p *ProviderPluginDefinition) Function(
	ctx context.Context,
	functionName string,
) (provider.Function, error) {
	function, ok := p.Functions[functionName]
	if !ok {
		return nil, errFunctionNotFound(functionName)
	}
	return function, nil
}

func (p *ProviderPluginDefinition) ListResourceTypes(ctx context.Context) ([]string, error) {
	return utils.GetKeys(p.Resources), nil
}

func (p *ProviderPluginDefinition) ListLinkTypes(ctx context.Context) ([]string, error) {
	return utils.GetKeys(p.Links), nil
}

func (p *ProviderPluginDefinition) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	return utils.GetKeys(p.DataSources), nil
}

func (p *ProviderPluginDefinition) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	return utils.GetKeys(p.CustomVariableTypes), nil
}

func (p *ProviderPluginDefinition) ListFunctions(ctx context.Context) ([]string, error) {
	return utils.GetKeys(p.Functions), nil
}

func (p *ProviderPluginDefinition) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return p.ProviderRetryPolicy, nil
}
