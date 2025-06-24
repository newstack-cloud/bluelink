package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingProviderServer struct {
	providerserverv1.UnimplementedProviderServer
}

func (p *failingProviderServer) GetNamespace(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.NamespaceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving namespace",
	)
}

func (p *failingProviderServer) GetConfigDefinition(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*sharedtypesv1.ConfigDefinitionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving config definition",
	)
}

func (p *failingProviderServer) ListResourceTypes(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.ResourceTypesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred listing resource types",
	)
}

func (p *failingProviderServer) ListLinkTypes(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.LinkTypesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred listing link types",
	)
}

func (p *failingProviderServer) ListDataSourceTypes(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.DataSourceTypesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred listing data source types",
	)
}

func (p *failingProviderServer) ListCustomVariableTypes(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.CustomVariableTypesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred listing custom variable types",
	)
}

func (p *failingProviderServer) ListFunctions(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.FunctionListResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred listing functions",
	)
}

func (p *failingProviderServer) GetRetryPolicy(
	ctx context.Context,
	req *providerserverv1.ProviderRequest,
) (*providerserverv1.RetryPolicyResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving retry policy",
	)
}

func (p *failingProviderServer) CustomValidateResource(
	ctx context.Context,
	req *providerserverv1.CustomValidateResourceRequest,
) (*providerserverv1.CustomValidateResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred applying custom validation for resource",
	)
}

func (p *failingProviderServer) GetResourceSpecDefinition(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*providerserverv1.ResourceSpecDefinitionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving resource spec definition",
	)
}

func (p *failingProviderServer) CanResourceLinkTo(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*providerserverv1.CanResourceLinkToResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving the resources that can be linked to",
	)
}

func (p *failingProviderServer) GetResourceStabilisedDeps(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*providerserverv1.ResourceStabilisedDepsResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving the stabilised dependencies for a resource",
	)
}

func (p *failingProviderServer) IsResourceCommonTerminal(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*providerserverv1.IsResourceCommonTerminalResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving whether or "+
			"not the resource is a common terminal",
	)
}

func (p *failingProviderServer) GetResourceType(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*sharedtypesv1.ResourceTypeResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving resource type information",
	)
}

func (p *failingProviderServer) GetResourceTypeDescription(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*sharedtypesv1.TypeDescriptionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving resource type description",
	)
}

func (p *failingProviderServer) GetResourceExamples(
	ctx context.Context,
	req *providerserverv1.ResourceRequest,
) (*sharedtypesv1.ExamplesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving resource examples",
	)
}

func (p *failingProviderServer) DeployResource(
	ctx context.Context,
	req *sharedtypesv1.DeployResourceRequest,
) (*sharedtypesv1.DeployResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when deploying resource",
	)
}

func (p *failingProviderServer) ResourceHasStabilised(
	ctx context.Context,
	req *sharedtypesv1.ResourceHasStabilisedRequest,
) (*sharedtypesv1.ResourceHasStabilisedResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when checking if resource has stabilised",
	)
}

func (p *failingProviderServer) GetResourceExternalState(
	ctx context.Context,
	req *providerserverv1.GetResourceExternalStateRequest,
) (*providerserverv1.GetResourceExternalStateResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when getting external state for resource",
	)
}

func (p *failingProviderServer) DestroyResource(
	ctx context.Context,
	req *sharedtypesv1.DestroyResourceRequest,
) (*sharedtypesv1.DestroyResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when destroying resource",
	)
}

func (p *failingProviderServer) StageLinkChanges(
	ctx context.Context,
	req *providerserverv1.StageLinkChangesRequest,
) (*providerserverv1.StageLinkChangesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when staging changes for link",
	)
}

func (p *failingProviderServer) UpdateLinkResourceA(
	ctx context.Context,
	req *providerserverv1.UpdateLinkResourceRequest,
) (*providerserverv1.UpdateLinkResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when updating resource A for link",
	)
}

func (p *failingProviderServer) UpdateLinkResourceB(
	ctx context.Context,
	req *providerserverv1.UpdateLinkResourceRequest,
) (*providerserverv1.UpdateLinkResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when updating resource B for link",
	)
}

func (p *failingProviderServer) UpdateLinkIntermediaryResources(
	ctx context.Context,
	req *providerserverv1.UpdateLinkIntermediaryResourcesRequest,
) (*providerserverv1.UpdateLinkIntermediaryResourcesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when updating intermediary resources for link",
	)
}

func (p *failingProviderServer) GetLinkPriorityResource(
	ctx context.Context,
	req *providerserverv1.LinkRequest,
) (*providerserverv1.LinkPriorityResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when retrieving the priority resource for link",
	)
}

func (p *failingProviderServer) GetLinkTypeDescription(
	ctx context.Context,
	req *providerserverv1.LinkRequest,
) (*sharedtypesv1.TypeDescriptionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when retrieving type description for link",
	)
}

func (p *failingProviderServer) GetLinkAnnotationDefinitions(
	ctx context.Context,
	req *providerserverv1.LinkRequest,
) (*providerserverv1.LinkAnnotationDefinitionsResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when retrieving annotation definitions for link",
	)
}

func (p *failingProviderServer) GetLinkKind(
	ctx context.Context,
	req *providerserverv1.LinkRequest,
) (*providerserverv1.LinkKindResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when retrieving link kind",
	)
}

func (p *failingProviderServer) CustomValidateDataSource(
	ctx context.Context,
	req *providerserverv1.CustomValidateDataSourceRequest,
) (*providerserverv1.CustomValidateDataSourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred applying custom validation for data source",
	)
}

func (p *failingProviderServer) GetDataSourceType(
	ctx context.Context,
	req *providerserverv1.DataSourceRequest,
) (*providerserverv1.DataSourceTypeResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving data source type information",
	)
}

func (p *failingProviderServer) GetDataSourceTypeDescription(
	ctx context.Context,
	req *providerserverv1.DataSourceRequest,
) (*sharedtypesv1.TypeDescriptionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving data source type description",
	)
}

func (p *failingProviderServer) GetDataSourceExamples(
	ctx context.Context,
	req *providerserverv1.DataSourceRequest,
) (*sharedtypesv1.ExamplesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving data source examples",
	)
}

func (p *failingProviderServer) GetDataSourceSpecDefinition(
	ctx context.Context,
	req *providerserverv1.DataSourceRequest,
) (*providerserverv1.DataSourceSpecDefinitionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving data source spec definition",
	)
}

func (p *failingProviderServer) GetDataSourceFilterFields(
	ctx context.Context,
	req *providerserverv1.DataSourceRequest,
) (*providerserverv1.DataSourceFilterFieldsResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving data source filter fields",
	)
}

func (p *failingProviderServer) FetchDataSource(
	ctx context.Context,
	req *providerserverv1.FetchDataSourceRequest,
) (*providerserverv1.FetchDataSourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred when fetching data source",
	)
}

func (p *failingProviderServer) GetCustomVariableType(
	ctx context.Context,
	req *providerserverv1.CustomVariableTypeRequest,
) (*providerserverv1.CustomVariableTypeResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving custom variable type information",
	)
}

func (p *failingProviderServer) GetCustomVariableTypeDescription(
	ctx context.Context,
	req *providerserverv1.CustomVariableTypeRequest,
) (*sharedtypesv1.TypeDescriptionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving custom variable type description",
	)
}

func (p *failingProviderServer) GetCustomVariableTypeOptions(
	ctx context.Context,
	req *providerserverv1.CustomVariableTypeRequest,
) (*providerserverv1.CustomVariableTypeOptionsResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving custom variable type options",
	)
}

func (p *failingProviderServer) GetCustomVariableTypeExamples(
	ctx context.Context,
	req *providerserverv1.CustomVariableTypeRequest,
) (*sharedtypesv1.ExamplesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving custom variable type examples",
	)
}

func (p *failingProviderServer) GetFunctionDefinition(
	ctx context.Context,
	req *sharedtypesv1.FunctionDefinitionRequest,
) (*sharedtypesv1.FunctionDefinitionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving function definition",
	)
}

func (p *failingProviderServer) CallFunction(
	ctx context.Context,
	req *sharedtypesv1.FunctionCallRequest,
) (*sharedtypesv1.FunctionCallResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred calling function",
	)
}
