package providerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

type customVarTypeProviderClientWrapper struct {
	client             ProviderClient
	customVariableType string
	hostID             string
}

func (v *customVarTypeProviderClientWrapper) GetType(
	ctx context.Context,
	input *provider.CustomVariableTypeGetTypeInput,
) (*provider.CustomVariableTypeGetTypeOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableType,
		)
	}

	response, err := v.client.GetCustomVariableType(
		ctx,
		&CustomVariableTypeRequest{
			CustomVariableType: &CustomVariableType{
				Type: v.customVariableType,
			},
			HostId:  v.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableType,
		)
	}

	switch result := response.Response.(type) {
	case *CustomVariableTypeResponse_CustomVarTypeInfo:
		return &provider.CustomVariableTypeGetTypeOutput{
			Type:  result.CustomVarTypeInfo.Type.Type,
			Label: result.CustomVarTypeInfo.Label,
		}, nil
	case *CustomVariableTypeResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetCustomVariableType,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetCustomVariableType,
		),
		errorsv1.PluginActionProviderGetCustomVariableType,
	)
}

func (v *customVarTypeProviderClientWrapper) GetDescription(
	ctx context.Context,
	input *provider.CustomVariableTypeGetDescriptionInput,
) (*provider.CustomVariableTypeGetDescriptionOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableTypeDescription,
		)
	}

	response, err := v.client.GetCustomVariableTypeDescription(
		ctx,
		&CustomVariableTypeRequest{
			CustomVariableType: &CustomVariableType{
				Type: v.customVariableType,
			},
			HostId:  v.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableTypeDescription,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.TypeDescriptionResponse_Description:
		return fromPBTypeDescriptionForCustomVariableType(result.Description), nil
	case *sharedtypesv1.TypeDescriptionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetCustomVariableTypeDescription,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetCustomVariableTypeDescription,
		),
		errorsv1.PluginActionProviderGetCustomVariableTypeDescription,
	)
}

func (v *customVarTypeProviderClientWrapper) Options(
	ctx context.Context,
	input *provider.CustomVariableTypeOptionsInput,
) (*provider.CustomVariableTypeOptionsOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableTypeOptions,
		)
	}

	response, err := v.client.GetCustomVariableTypeOptions(
		ctx,
		&CustomVariableTypeRequest{
			CustomVariableType: &CustomVariableType{
				Type: v.customVariableType,
			},
			HostId:  v.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableTypeOptions,
		)
	}

	switch result := response.Response.(type) {
	case *CustomVariableTypeOptionsResponse_Options:
		optionsOutput, err := fromPBCustomVarTypeOptions(result.Options)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionProviderGetCustomVariableTypeOptions,
			)
		}

		return optionsOutput, nil
	case *CustomVariableTypeOptionsResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetCustomVariableTypeOptions,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetCustomVariableTypeOptions,
		),
		errorsv1.PluginActionProviderGetCustomVariableTypeOptions,
	)
}

func (v *customVarTypeProviderClientWrapper) GetExamples(
	ctx context.Context,
	input *provider.CustomVariableTypeGetExamplesInput,
) (*provider.CustomVariableTypeGetExamplesOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableTypeExamples,
		)
	}

	response, err := v.client.GetCustomVariableTypeExamples(
		ctx,
		&CustomVariableTypeRequest{
			CustomVariableType: &CustomVariableType{
				Type: v.customVariableType,
			},
			HostId:  v.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetCustomVariableTypeExamples,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ExamplesResponse_Examples:
		return fromPBExamplesForCustomVarType(result.Examples), nil
	case *sharedtypesv1.ExamplesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetCustomVariableTypeExamples,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetCustomVariableTypeExamples,
		),
		errorsv1.PluginActionProviderGetCustomVariableTypeExamples,
	)
}
