package providerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

// WrapProviderClient wraps a provider plugin v1 ProviderClient
// in a blueprint framework Provider to allow the deploy engine
// to interact with the provider in a way that is compatible
// with the blueprint framework and is agnostic to the underlying
// communication protocol.
func WrapProviderClient(client ProviderClient, hostID string) provider.Provider {
	return &providerClientWrapper{
		client: client,
		hostID: hostID,
	}
}

type providerClientWrapper struct {
	client ProviderClient
	hostID string
}

func (p *providerClientWrapper) Namespace(ctx context.Context) (string, error) {
	response, err := p.client.GetNamespace(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return "", errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetNamespace,
		)
	}

	switch result := response.Response.(type) {
	case *NamespaceResponse_Namespace:
		return result.Namespace.GetNamespace(), nil
	case *NamespaceResponse_ErrorResponse:
		return "", errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetNamespace,
		)
	}

	return "", errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetNamespace,
		),
		errorsv1.PluginActionProviderGetNamespace,
	)
}

func (p *providerClientWrapper) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	response, err := p.client.GetConfigDefinition(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetConfigDefinition,
		)
	}

	return convertv1.FromPBConfigDefinitionResponse(
		response,
		errorsv1.PluginActionProviderGetConfigDefinition,
	)
}

func (p *providerClientWrapper) Resource(ctx context.Context, resourceType string) (provider.Resource, error) {
	return &resourceProviderClientWrapper{
		client:       p.client,
		resourceType: resourceType,
		hostID:       p.hostID,
	}, nil
}

func (p *providerClientWrapper) DataSource(ctx context.Context, dataSourceType string) (provider.DataSource, error) {
	return &dataSourceProviderClientWrapper{
		client:         p.client,
		dataSourceType: dataSourceType,
		hostID:         p.hostID,
	}, nil
}

func (p *providerClientWrapper) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (provider.Link, error) {
	return &linkProviderClientWrapper{
		client:        p.client,
		resourceTypeA: resourceTypeA,
		resourceTypeB: resourceTypeB,
		hostID:        p.hostID,
	}, nil
}

func (p *providerClientWrapper) CustomVariableType(
	ctx context.Context,
	customVariableType string,
) (provider.CustomVariableType, error) {
	return &customVarTypeProviderClientWrapper{
		client:             p.client,
		customVariableType: customVariableType,
		hostID:             p.hostID,
	}, nil
}

func (p *providerClientWrapper) ListFunctions(ctx context.Context) ([]string, error) {
	response, err := p.client.ListFunctions(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderListFunctions,
		)
	}

	switch result := response.Response.(type) {
	case *FunctionListResponse_FunctionList:
		return result.FunctionList.Functions, nil
	case *FunctionListResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderListFunctions,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderListFunctions,
		),
		errorsv1.PluginActionProviderListFunctions,
	)
}

func (p *providerClientWrapper) ListResourceTypes(ctx context.Context) ([]string, error) {
	response, err := p.client.ListResourceTypes(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderListResourceTypes,
		)
	}

	switch result := response.Response.(type) {
	case *ResourceTypesResponse_ResourceTypes:
		return sharedtypesv1.FromPBResourceTypes(result.ResourceTypes.ResourceTypes), nil
	case *ResourceTypesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderListResourceTypes,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderListResourceTypes,
		),
		errorsv1.PluginActionProviderListResourceTypes,
	)
}

func (p *providerClientWrapper) ListLinkTypes(ctx context.Context) ([]string, error) {
	response, err := p.client.ListLinkTypes(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderListLinkTypes,
		)
	}

	switch result := response.Response.(type) {
	case *LinkTypesResponse_LinkTypes:
		return fromPBLinkTypes(result.LinkTypes.LinkTypes), nil
	case *LinkTypesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderListLinkTypes,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderListLinkTypes,
		),
		errorsv1.PluginActionProviderListLinkTypes,
	)
}

func (p *providerClientWrapper) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	response, err := p.client.ListDataSourceTypes(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderListDataSourceTypes,
		)
	}

	switch result := response.Response.(type) {
	case *DataSourceTypesResponse_DataSourceTypes:
		return fromPBDataSourceTypes(result.DataSourceTypes.DataSourceTypes), nil
	case *DataSourceTypesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderListDataSourceTypes,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderListDataSourceTypes,
		),
		errorsv1.PluginActionProviderListDataSourceTypes,
	)
}

func (p *providerClientWrapper) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	response, err := p.client.ListCustomVariableTypes(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderListCustomVariableTypes,
		)
	}

	switch result := response.Response.(type) {
	case *CustomVariableTypesResponse_CustomVariableTypes:
		return fromPBCustomVarTypes(result.CustomVariableTypes.CustomVariableTypes), nil
	case *CustomVariableTypesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderListCustomVariableTypes,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderListCustomVariableTypes,
		),
		errorsv1.PluginActionProviderListCustomVariableTypes,
	)
}

func (p *providerClientWrapper) Function(ctx context.Context, functionName string) (provider.Function, error) {
	return &functionProviderClientWrapper{
		client:       p.client,
		functionName: functionName,
		hostID:       p.hostID,
	}, nil
}

func (p *providerClientWrapper) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	response, err := p.client.GetRetryPolicy(ctx, &ProviderRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetRetryPolicy,
		)
	}

	switch result := response.Response.(type) {
	case *RetryPolicyResponse_RetryPolicy:
		return fromPBRetryPolicy(result.RetryPolicy), nil
	case *RetryPolicyResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetRetryPolicy,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetRetryPolicy,
		),
		errorsv1.PluginActionProviderGetRetryPolicy,
	)
}

func fromPBDataSourceTypes(dataSourceTypes []*DataSourceType) []string {
	types := make([]string, len(dataSourceTypes))
	for i, dataSourceType := range dataSourceTypes {
		types[i] = dataSourceType.Type
	}
	return types
}

func fromPBLinkTypes(linkTypes []*LinkType) []string {
	types := make([]string, len(linkTypes))
	for i, linkType := range linkTypes {
		types[i] = linkType.Type
	}
	return types
}

func fromPBCustomVarTypes(customVarTypes []*CustomVariableType) []string {
	types := make([]string, len(customVarTypes))
	for i, customVarType := range customVarTypes {
		types[i] = customVarType.Type
	}
	return types
}

func fromPBRetryPolicy(retryPolicy *RetryPolicy) *provider.RetryPolicy {
	return &provider.RetryPolicy{
		MaxRetries:      int(retryPolicy.MaxRetries),
		FirstRetryDelay: retryPolicy.FirstRetryDelay,
		MaxDelay:        retryPolicy.MaxDelay,
		BackoffFactor:   retryPolicy.BackoffFactor,
		Jitter:          retryPolicy.Jitter,
	}
}
