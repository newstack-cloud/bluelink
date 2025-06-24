package providerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

type resourceProviderClientWrapper struct {
	client       ProviderClient
	resourceType string
	hostID       string
}

func (r *resourceProviderClientWrapper) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	schemaResourcePB, err := serialisation.ToResourcePB(input.SchemaResource)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCustomValidateResource,
		)
	}

	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCustomValidateResource,
		)
	}

	response, err := r.client.CustomValidateResource(
		ctx,
		&CustomValidateResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:         r.hostID,
			SchemaResource: schemaResourcePB,
			Context:        providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCustomValidateResource,
		)
	}

	switch result := response.Response.(type) {
	case *CustomValidateResourceResponse_CompleteResponse:
		return &provider.ResourceValidateOutput{
			Diagnostics: sharedtypesv1.ToCoreDiagnostics(result.CompleteResponse.GetDiagnostics()),
		}, nil
	case *CustomValidateResourceResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderCustomValidateResource,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(errorsv1.PluginActionProviderCustomValidateResource),
		errorsv1.PluginActionProviderCustomValidateResource,
	)
}

func (r *resourceProviderClientWrapper) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceSpecDefinition,
		)
	}

	response, err := r.client.GetResourceSpecDefinition(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceSpecDefinition,
		)
	}

	switch result := response.Response.(type) {
	case *ResourceSpecDefinitionResponse_SpecDefinition:
		specDefinition, err := convertv1.FromPBResourceSpecDefinition(result.SpecDefinition)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionProviderGetResourceSpecDefinition,
			)
		}
		return &provider.ResourceGetSpecDefinitionOutput{
			SpecDefinition: specDefinition,
		}, nil
	case *ResourceSpecDefinitionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetResourceSpecDefinition,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetResourceSpecDefinition,
		),
		errorsv1.PluginActionProviderGetResourceSpecDefinition,
	)
}

func (r *resourceProviderClientWrapper) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckCanResourceLinkTo,
		)
	}

	response, err := r.client.CanResourceLinkTo(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckCanResourceLinkTo,
		)
	}

	switch result := response.Response.(type) {
	case *CanResourceLinkToResponse_ResourceTypes:
		canLinkTo := sharedtypesv1.FromPBResourceTypes(
			result.ResourceTypes.ResourceTypes,
		)

		return &provider.ResourceCanLinkToOutput{
			CanLinkTo: canLinkTo,
		}, nil
	case *CanResourceLinkToResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderCheckCanResourceLinkTo,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderCheckCanResourceLinkTo,
		),
		errorsv1.PluginActionProviderCheckCanResourceLinkTo,
	)
}

func (r *resourceProviderClientWrapper) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceStabilisedDeps,
		)
	}

	response, err := r.client.GetResourceStabilisedDeps(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceStabilisedDeps,
		)
	}

	switch result := response.Response.(type) {
	case *ResourceStabilisedDepsResponse_StabilisedDependencies:
		stabilisedDeps := sharedtypesv1.FromPBResourceTypes(
			result.StabilisedDependencies.ResourceTypes,
		)
		return &provider.ResourceStabilisedDependenciesOutput{
			StabilisedDependencies: stabilisedDeps,
		}, nil
	case *ResourceStabilisedDepsResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetResourceStabilisedDeps,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetResourceStabilisedDeps,
		),
		errorsv1.PluginActionProviderGetResourceStabilisedDeps,
	)
}

func (r *resourceProviderClientWrapper) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckIsResourceCommonTerminal,
		)
	}

	response, err := r.client.IsResourceCommonTerminal(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckIsResourceCommonTerminal,
		)
	}

	switch result := response.Response.(type) {
	case *IsResourceCommonTerminalResponse_Data:
		return &provider.ResourceIsCommonTerminalOutput{
			IsCommonTerminal: result.Data.IsCommonTerminal,
		}, nil
	case *IsResourceCommonTerminalResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderCheckIsResourceCommonTerminal,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderCheckIsResourceCommonTerminal,
		),
		errorsv1.PluginActionProviderCheckIsResourceCommonTerminal,
	)
}

func (r *resourceProviderClientWrapper) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceType,
		)
	}

	// Fetching a resource type from the plugin allows to obtain
	// the human-readable label for the resource type so is worth while
	// for documentation and tooling purposes despite the resource type ID
	// already being known at this point.
	response, err := r.client.GetResourceType(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceType,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ResourceTypeResponse_ResourceTypeInfo:
		return &provider.ResourceGetTypeOutput{
			Type:  convertv1.ResourceTypeToString(result.ResourceTypeInfo.Type),
			Label: result.ResourceTypeInfo.Label,
		}, nil
	case *sharedtypesv1.ResourceTypeResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetResourceType,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetResourceType,
		),
		errorsv1.PluginActionProviderGetResourceType,
	)
}

func (r *resourceProviderClientWrapper) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceTypeDescription,
		)
	}

	response, err := r.client.GetResourceTypeDescription(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceTypeDescription,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.TypeDescriptionResponse_Description:
		return convertv1.FromPBTypeDescriptionForResource(result.Description), nil
	case *sharedtypesv1.TypeDescriptionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetResourceTypeDescription,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetResourceTypeDescription,
		),
		errorsv1.PluginActionProviderGetResourceTypeDescription,
	)
}

func (r *resourceProviderClientWrapper) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExamples,
		)
	}

	response, err := r.client.GetResourceExamples(
		ctx,
		&ResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:  r.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExamples,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ExamplesResponse_Examples:
		return convertv1.FromPBExamplesForResource(result.Examples), nil
	case *sharedtypesv1.ExamplesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetResourceExamples,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetResourceExamples,
		),
		errorsv1.PluginActionProviderGetResourceExamples,
	)
}

func (r *resourceProviderClientWrapper) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderDeployResource,
		)
	}

	resourceChangesPB, err := convertv1.ToPBChanges(input.Changes)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderDeployResource,
		)
	}

	response, err := r.client.DeployResource(
		ctx,
		&sharedtypesv1.DeployResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:       r.hostID,
			InstanceId:   input.InstanceID,
			InstanceName: input.InstanceName,
			ResourceId:   input.ResourceID,
			Changes:      resourceChangesPB,
			Context:      providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExamples,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.DeployResourceResponse_CompleteResponse:
		return convertv1.FromPBDeployResourceCompleteResponse(
			result.CompleteResponse,
		)
	case *sharedtypesv1.DeployResourceResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderDeployResource,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderDeployResource,
		),
		errorsv1.PluginActionProviderDeployResource,
	)
}

func (r *resourceProviderClientWrapper) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckResourceHasStabilised,
		)
	}

	pbResourceSpec, err := serialisation.ToMappingNodePB(
		input.ResourceSpec,
		/* optional */ true,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckResourceHasStabilised,
		)
	}

	pbResourceMetadata, err := convertv1.ToPBResourceMetadataState(
		input.ResourceMetadata,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckResourceHasStabilised,
		)
	}

	response, err := r.client.ResourceHasStabilised(
		ctx,
		&sharedtypesv1.ResourceHasStabilisedRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:           r.hostID,
			InstanceId:       input.InstanceID,
			InstanceName:     input.InstanceName,
			ResourceId:       input.ResourceID,
			ResourceSpec:     pbResourceSpec,
			ResourceMetadata: pbResourceMetadata,
			Context:          providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCheckResourceHasStabilised,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ResourceHasStabilisedResponse_ResourceStabilisationInfo:
		return &provider.ResourceHasStabilisedOutput{
			Stabilised: result.ResourceStabilisationInfo.Stabilised,
		}, nil
	case *sharedtypesv1.ResourceHasStabilisedResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderCheckResourceHasStabilised,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderCheckResourceHasStabilised,
		),
		errorsv1.PluginActionProviderCheckResourceHasStabilised,
	)
}

func (r *resourceProviderClientWrapper) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExternalState,
		)
	}

	pbCurrentResourceSpec, err := serialisation.ToMappingNodePB(
		input.CurrentResourceSpec,
		/* optional */ true,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExternalState,
		)
	}

	pbCurrentResourceMetadata, err := convertv1.ToPBResourceMetadataState(
		input.CurrentResourceMetadata,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExternalState,
		)
	}

	response, err := r.client.GetResourceExternalState(
		ctx,
		&GetResourceExternalStateRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:                  r.hostID,
			InstanceId:              input.InstanceID,
			InstanceName:            input.InstanceName,
			ResourceId:              input.ResourceID,
			CurrentResourceSpec:     pbCurrentResourceSpec,
			CurrentResourceMetadata: pbCurrentResourceMetadata,
			Context:                 providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetResourceExternalState,
		)
	}

	switch result := response.Response.(type) {
	case *GetResourceExternalStateResponse_ResourceSpecState:
		specState, err := serialisation.FromMappingNodePB(
			result.ResourceSpecState,
			/* optional */ false,
		)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionProviderGetResourceExternalState,
			)
		}

		return &provider.ResourceGetExternalStateOutput{
			ResourceSpecState: specState,
		}, nil
	case *GetResourceExternalStateResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetResourceExternalState,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetResourceExternalState,
		),
		errorsv1.PluginActionProviderGetResourceExternalState,
	)
}

func (r *resourceProviderClientWrapper) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderDestroyResource,
		)
	}

	pbResourceState, err := convertv1.ToPBResourceState(
		input.ResourceState,
	)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderDestroyResource,
		)
	}

	response, err := r.client.DestroyResource(
		ctx,
		&sharedtypesv1.DestroyResourceRequest{
			ResourceType: &sharedtypesv1.ResourceType{
				Type: r.resourceType,
			},
			HostId:        r.hostID,
			InstanceId:    input.InstanceID,
			InstanceName:  input.InstanceName,
			ResourceId:    input.ResourceID,
			ResourceState: pbResourceState,
			Context:       providerCtx,
		},
	)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderDestroyResource,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.DestroyResourceResponse_Result:
		return nil
	case *sharedtypesv1.DestroyResourceResponse_ErrorResponse:
		return errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderDestroyResource,
		)
	}

	return errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderDestroyResource,
		),
		errorsv1.PluginActionProviderDestroyResource,
	)
}
