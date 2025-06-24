package providerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

type dataSourceProviderClientWrapper struct {
	client         ProviderClient
	dataSourceType string
	hostID         string
}

func (d *dataSourceProviderClientWrapper) GetType(
	ctx context.Context,
	input *provider.DataSourceGetTypeInput,
) (*provider.DataSourceGetTypeOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceType,
		)
	}

	response, err := d.client.GetDataSourceType(
		ctx,
		&DataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			HostId:  d.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceType,
		)
	}

	switch result := response.Response.(type) {
	case *DataSourceTypeResponse_DataSourceTypeInfo:
		return &provider.DataSourceGetTypeOutput{
			Type:  result.DataSourceTypeInfo.Type.Type,
			Label: result.DataSourceTypeInfo.Label,
		}, nil
	case *DataSourceTypeResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetDataSourceType,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetDataSourceType,
		),
		errorsv1.PluginActionProviderGetDataSourceType,
	)
}

func (d *dataSourceProviderClientWrapper) GetTypeDescription(
	ctx context.Context,
	input *provider.DataSourceGetTypeDescriptionInput,
) (*provider.DataSourceGetTypeDescriptionOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceTypeDescription,
		)
	}

	response, err := d.client.GetDataSourceTypeDescription(
		ctx,
		&DataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			HostId:  d.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceTypeDescription,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.TypeDescriptionResponse_Description:
		return fromPBTypeDescriptionForDataSource(result.Description), nil
	case *sharedtypesv1.TypeDescriptionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetDataSourceTypeDescription,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetDataSourceTypeDescription,
		),
		errorsv1.PluginActionProviderGetDataSourceTypeDescription,
	)
}

func (d *dataSourceProviderClientWrapper) GetExamples(
	ctx context.Context,
	input *provider.DataSourceGetExamplesInput,
) (*provider.DataSourceGetExamplesOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceExamples,
		)
	}

	response, err := d.client.GetDataSourceExamples(
		ctx,
		&DataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			HostId:  d.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceExamples,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ExamplesResponse_Examples:
		return fromPBExamplesForDataSource(result.Examples), nil
	case *sharedtypesv1.ExamplesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetDataSourceExamples,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetDataSourceExamples,
		),
		errorsv1.PluginActionProviderGetDataSourceExamples,
	)
}

func (d *dataSourceProviderClientWrapper) CustomValidate(
	ctx context.Context,
	input *provider.DataSourceValidateInput,
) (*provider.DataSourceValidateOutput, error) {
	schemaDataSourcePB, err := serialisation.ToDataSourcePB(input.SchemaDataSource)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCustomValidateDataSource,
		)
	}

	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCustomValidateDataSource,
		)
	}

	response, err := d.client.CustomValidateDataSource(
		ctx,
		&CustomValidateDataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			HostId:           d.hostID,
			SchemaDataSource: schemaDataSourcePB,
			Context:          providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderCustomValidateDataSource,
		)
	}

	switch result := response.Response.(type) {
	case *CustomValidateDataSourceResponse_CompleteResponse:
		return &provider.DataSourceValidateOutput{
			Diagnostics: sharedtypesv1.ToCoreDiagnostics(
				result.CompleteResponse.GetDiagnostics(),
			),
		}, nil
	case *CustomValidateDataSourceResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderCustomValidateDataSource,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderCustomValidateDataSource,
		),
		errorsv1.PluginActionProviderCustomValidateDataSource,
	)
}

func (d *dataSourceProviderClientWrapper) GetSpecDefinition(
	ctx context.Context,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceSpecDefinition,
		)
	}

	response, err := d.client.GetDataSourceSpecDefinition(
		ctx,
		&DataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			HostId:  d.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceSpecDefinition,
		)
	}

	switch result := response.Response.(type) {
	case *DataSourceSpecDefinitionResponse_SpecDefinition:
		specDefOutput, err := fromPBDataSourceSpecDefinition(result.SpecDefinition)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionProviderGetDataSourceSpecDefinition,
			)
		}

		return specDefOutput, nil
	case *DataSourceSpecDefinitionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetDataSourceSpecDefinition,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetDataSourceSpecDefinition,
		),
		errorsv1.PluginActionProviderGetDataSourceSpecDefinition,
	)
}

func (d *dataSourceProviderClientWrapper) GetFilterFields(
	ctx context.Context,
	input *provider.DataSourceGetFilterFieldsInput,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceFilterFields,
		)
	}

	response, err := d.client.GetDataSourceFilterFields(
		ctx,
		&DataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			HostId:  d.hostID,
			Context: providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderGetDataSourceFilterFields,
		)
	}

	switch result := response.Response.(type) {
	case *DataSourceFilterFieldsResponse_FilterFields:
		filterFieldsOutput, err := fromPBDataSourceFilterFields(result.FilterFields)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionProviderGetDataSourceFilterFields,
			)
		}

		return filterFieldsOutput, nil
	case *DataSourceFilterFieldsResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderGetDataSourceFilterFields,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderGetDataSourceFilterFields,
		),
		errorsv1.PluginActionProviderGetDataSourceFilterFields,
	)
}

func (d *dataSourceProviderClientWrapper) Fetch(
	ctx context.Context,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	providerCtx, err := convertv1.ToPBProviderContext(input.ProviderContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderFetchDataSource,
		)
	}

	dataSourceWithResolvedSubsPB, err := toPBResolvedDataSource(
		input.DataSourceWithResolvedSubs,
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderFetchDataSource,
		)
	}

	response, err := d.client.FetchDataSource(
		ctx,
		&FetchDataSourceRequest{
			DataSourceType: &DataSourceType{
				Type: d.dataSourceType,
			},
			DataSourceWithResolvedSubs: dataSourceWithResolvedSubsPB,
			HostId:                     d.hostID,
			Context:                    providerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionProviderFetchDataSource,
		)
	}

	switch result := response.Response.(type) {
	case *FetchDataSourceResponse_CompleteResponse:
		data, err := convertv1.FromPBMappingNodeMap(result.CompleteResponse.Data)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionProviderFetchDataSource,
			)
		}

		return &provider.DataSourceFetchOutput{
			Data: data,
		}, nil
	case *FetchDataSourceResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionProviderFetchDataSource,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionProviderFetchDataSource,
		),
		errorsv1.PluginActionProviderFetchDataSource,
	)
}
