package providerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

type functionProviderClientWrapper struct {
	client       ProviderClient
	functionName string
	hostID       string
}

func (f *functionProviderClientWrapper) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	definitionReq, err := convertv1.ToPBFunctionDefinitionRequest(
		f.functionName,
		input,
		f.hostID,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetFunctionDefinition,
		)
	}

	response, err := f.client.GetFunctionDefinition(
		ctx,
		definitionReq,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetFunctionDefinition,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.FunctionDefinitionResponse_FunctionDefinition:
		return convertv1.FromPBFunctionDefinitionResponse(
			result.FunctionDefinition,
			errorsv1.PluginActionProviderGetFunctionDefinition,
		)
	case *sharedtypesv1.FunctionDefinitionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetFunctionDefinition,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetFunctionDefinition,
		),
		errorsv1.PluginActionProviderGetFunctionDefinition,
	)
}

func (f *functionProviderClientWrapper) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	callReq, err := convertv1.ToPBFunctionCallRequest(
		ctx,
		f.functionName,
		input,
		f.hostID,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCallFunction,
		)
	}

	response, err := f.client.CallFunction(
		ctx,
		callReq,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCallFunction,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.FunctionCallResponse_FunctionResult:
		return f.toFunctionCallOutput(result.FunctionResult)
	case *sharedtypesv1.FunctionCallResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderCallFunction,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderCallFunction,
		),
		errorsv1.PluginActionProviderCallFunction,
	)
}

func (f *functionProviderClientWrapper) toFunctionCallOutput(
	response *sharedtypesv1.FunctionCallResult,
) (*provider.FunctionCallOutput, error) {
	funcCallOutput, err := convertv1.FromPBFunctionCallResult(response)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCallFunction,
		)
	}

	return funcCallOutput, nil
}
