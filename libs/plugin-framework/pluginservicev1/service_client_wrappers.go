package pluginservicev1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ResourceServiceFromClient creates a new instance of a ResourceService
// that uses the provided ServiceClient to interact with the deploy engine.
// This allows plugin implementations to interact with the deploy engine
// using the blueprint framework interfaces abstracting away the communication
// protocol from plugin developers.
func ResourceServiceFromClient(
	client ServiceClient,
) provider.ResourceService {
	return &resourceServiceClientWrapper{
		client: client,
	}
}

type resourceServiceClientWrapper struct {
	client ServiceClient
}

func (r *resourceServiceClientWrapper) Deploy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDeployServiceInput,
) (*provider.ResourceDeployOutput, error) {
	deployReq, err := convertv1.ToPBDeployResourceRequest(resourceType, input.DeployInput)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceDeployResource,
		)
	}

	response, err := r.client.DeployResource(
		ctx,
		&DeployResourceServiceRequest{
			DeployRequest:   deployReq,
			WaitUntilStable: input.WaitUntilStable,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceDeployResource,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.DeployResourceResponse_CompleteResponse:
		return r.toResourceDeployOutput(result.CompleteResponse)
	case *sharedtypesv1.DeployResourceResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceDeployResource,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceDeployResource,
		),
		errorsv1.PluginActionServiceDeployResource,
	)
}

func (r *resourceServiceClientWrapper) toResourceDeployOutput(
	response *sharedtypesv1.DeployResourceCompleteResponse,
) (*provider.ResourceDeployOutput, error) {
	computedFieldValues, err := convertv1.FromPBMappingNodeMap(
		response.ComputedFieldValues,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceDeployResource,
		)
	}

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: computedFieldValues,
	}, nil
}

func (r *resourceServiceClientWrapper) Destroy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDestroyInput,
) error {
	destroyReq, err := convertv1.ToPBDestroyResourceRequest(resourceType, input)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceDestroyResource,
		)
	}

	response, err := r.client.DestroyResource(
		ctx,
		destroyReq,
	)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceDestroyResource,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.DestroyResourceResponse_Result:
		if result.Result != nil && result.Result.Destroyed {
			return nil
		}
		return errorsv1.CreateGeneralError(
			errorsv1.ErrResourceNotDestroyed(
				resourceType,
				errorsv1.PluginActionServiceDestroyResource,
			),
			errorsv1.PluginActionServiceDestroyResource,
		)
	case *sharedtypesv1.DestroyResourceResponse_ErrorResponse:
		return errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceDestroyResource,
		)
	}

	return errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceDestroyResource,
		),
		errorsv1.PluginActionServiceDestroyResource,
	)
}

func (r *resourceServiceClientWrapper) LookupResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (*state.ResourceState, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceLookupResourceInState,
		)
	}

	response, err := r.client.LookupResourceInState(
		ctx,
		&LookupResourceInStateRequest{
			InstanceId:   input.InstanceID,
			ResourceType: input.ResourceType,
			ExternalId:   input.ExternalID,
			Context:      providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceDeployResource,
		)
	}

	switch result := response.Response.(type) {
	case *LookupResourceInStateResponse_Resource:
		resourceState, err := convertv1.FromPBResourceState(result.Resource)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionServiceLookupResourceInState,
			)
		}
		return resourceState, nil
	case *LookupResourceInStateResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceLookupResourceInState,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceLookupResourceInState,
		),
		errorsv1.PluginActionServiceLookupResourceInState,
	)
}

func (r *resourceServiceClientWrapper) HasResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (bool, error) {
	resourceState, err := r.LookupResourceInState(ctx, input)
	if err != nil {
		return false, err
	}

	return resourceState != nil, nil
}

func (r *resourceServiceClientWrapper) AcquireResourceLock(
	ctx context.Context,
	input *provider.AcquireResourceLockInput,
) error {
	linkID, linkIDInContext := ctx.Value(utils.ContextKeyLinkID).(string)
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceAcquireResourceLock,
		)
	}

	acquireReq := &AcquireResourceLockRequest{
		InstanceId:   input.InstanceID,
		ResourceName: input.ResourceName,
		AcquiredBy:   input.AcquiredBy,
		Context:      providerCtx,
	}
	if acquireReq.AcquiredBy == "" && linkIDInContext {
		acquireReq.AcquiredBy = linkID
	}

	response, err := r.client.AcquireResourceLock(
		ctx,
		acquireReq,
	)
	if err != nil {
		return errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceAcquireResourceLock,
		)
	}

	switch result := response.Response.(type) {
	case *AcquireResourceLockResponse_Result:
		if result.Result != nil && result.Result.Acquired {
			return nil
		}
		if err != nil {
			return errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionServiceAcquireResourceLock,
			)
		}
		return nil
	case *AcquireResourceLockResponse_ErrorResponse:
		return errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceAcquireResourceLock,
		)
	}

	return errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceAcquireResourceLock,
		),
		errorsv1.PluginActionServiceAcquireResourceLock,
	)
}

// FunctionRegistryFromClient creates a new instance of a FunctionRegistry
// that uses the provided ServiceClient to interact with the deploy engine.
// This allows plugin implementations to interact with the deploy engine
// using the blueprint framework interfaces abstracting away the communication
// protocol from plugin developers.
func FunctionRegistryFromClient(
	client ServiceClient,
	hostInfo pluginutils.HostInfoContainer,
) provider.FunctionRegistry {
	return &functionRegistryClientWrapper{
		client:    client,
		callStack: function.NewStack(),
		hostInfo:  hostInfo,
	}
}

type functionRegistryClientWrapper struct {
	client    ServiceClient
	callStack function.Stack
	hostInfo  pluginutils.HostInfoContainer
}

func (f *functionRegistryClientWrapper) ForCallContext(
	stack function.Stack,
) provider.FunctionRegistry {
	return &functionRegistryClientWrapper{
		client:    f.client,
		callStack: stack,
		hostInfo:  f.hostInfo,
	}
}

func (f *functionRegistryClientWrapper) Call(
	ctx context.Context,
	functionName string,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	// On the server-side (plugin host), the internal
	// function registry implementation is expected
	// to handle the call stack and push/pop the
	// function call to/from the stack.
	// On the client-side (plugin), all we need to do is pass
	// the current call context before the function call is made
	// from the host.
	callReq, err := convertv1.ToPBFunctionCallRequest(
		ctx,
		functionName,
		input,
		f.hostInfo.GetID(),
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceCallFunction,
		)
	}

	response, err := f.client.CallFunction(
		ctx,
		callReq,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceCallFunction,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.FunctionCallResponse_FunctionResult:
		return f.toFunctionCallOutput(result.FunctionResult)
	case *sharedtypesv1.FunctionCallResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceCallFunction,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceCallFunction,
		),
		errorsv1.PluginActionServiceCallFunction,
	)
}

func (f *functionRegistryClientWrapper) toFunctionCallOutput(
	response *sharedtypesv1.FunctionCallResult,
) (*provider.FunctionCallOutput, error) {
	funcCallOutput, err := convertv1.FromPBFunctionCallResult(response)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceCallFunction,
		)
	}

	return funcCallOutput, nil
}

func (f *functionRegistryClientWrapper) GetDefinition(
	ctx context.Context,
	functionName string,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	definitionReq, err := convertv1.ToPBFunctionDefinitionRequest(
		functionName,
		input,
		f.hostInfo.GetID(),
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceGetFunctionDefinition,
		)
	}

	response, err := f.client.GetFunctionDefinition(
		ctx,
		definitionReq,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceGetFunctionDefinition,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.FunctionDefinitionResponse_FunctionDefinition:
		return convertv1.FromPBFunctionDefinitionResponse(
			result.FunctionDefinition,
			errorsv1.PluginActionServiceGetFunctionDefinition,
		)
	case *sharedtypesv1.FunctionDefinitionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceGetFunctionDefinition,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceGetFunctionDefinition,
		),
		errorsv1.PluginActionServiceGetFunctionDefinition,
	)
}

func (f *functionRegistryClientWrapper) HasFunction(
	ctx context.Context,
	functionName string,
) (bool, error) {
	hasFunctionReq := &HasFunctionRequest{
		FunctionName: functionName,
	}

	response, err := f.client.HasFunction(
		ctx,
		hasFunctionReq,
	)
	if err != nil {
		return false, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceCheckHasFunction,
		)
	}

	switch result := response.Response.(type) {
	case *HasFunctionResponse_FunctionCheckResult:
		return result.FunctionCheckResult.HasFunction, nil
	case *HasFunctionResponse_ErrorResponse:
		return false, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceCheckHasFunction,
		)
	}

	return false, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceCheckHasFunction,
		),
		errorsv1.PluginActionServiceCheckHasFunction,
	)
}

func (f *functionRegistryClientWrapper) ListFunctions(
	ctx context.Context,
) ([]string, error) {

	response, err := f.client.ListFunctions(
		ctx,
		&emptypb.Empty{},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionServiceListFunctions,
		)
	}
	switch result := response.Response.(type) {
	case *ListFunctionsResponse_FunctionList:
		return result.FunctionList.Functions, nil
	case *ListFunctionsResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionServiceListFunctions,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionServiceListFunctions,
		),
		errorsv1.PluginActionServiceListFunctions,
	)
}
