package transformerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

type abstractResourceTransformerClientWrapper struct {
	client               TransformerClient
	abstractResourceType string
	hostID               string
}

func (t *abstractResourceTransformerClientWrapper) CustomValidate(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	schemaResourcePB, err := serialisation.ToResourcePB(input.SchemaResource)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCustomValidateAbstractResource,
		)
	}

	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCustomValidateAbstractResource,
		)
	}

	response, err := t.client.CustomValidateAbstractResource(
		ctx,
		&CustomValidateAbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:         t.hostID,
			SchemaResource: schemaResourcePB,
			Context:        transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCustomValidateAbstractResource,
		)
	}

	switch result := response.Response.(type) {
	case *CustomValidateAbstractResourceResponse_CompleteResponse:
		return &transform.AbstractResourceValidateOutput{
			Diagnostics: sharedtypesv1.ToCoreDiagnostics(result.CompleteResponse.GetDiagnostics()),
		}, nil
	case *CustomValidateAbstractResourceResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerCustomValidateAbstractResource,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerCustomValidateAbstractResource,
		),
		errorsv1.PluginActionTransformerCustomValidateAbstractResource,
	)
}

func (t *abstractResourceTransformerClientWrapper) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
		)
	}

	response, err := t.client.GetAbstractResourceSpecDefinition(
		ctx,
		&AbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:  t.hostID,
			Context: transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
		)
	}

	switch result := response.Response.(type) {
	case *AbstractResourceSpecDefinitionResponse_SpecDefinition:
		specDefinition, err := convertv1.FromPBResourceSpecDefinition(result.SpecDefinition)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
			)
		}
		return &transform.AbstractResourceGetSpecDefinitionOutput{
			SpecDefinition: specDefinition,
		}, nil
	case *AbstractResourceSpecDefinitionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
		),
		errorsv1.PluginActionTransformerGetAbstractResourceSpecDefinition,
	)
}

func (t *abstractResourceTransformerClientWrapper) CanLinkTo(
	ctx context.Context,
	input *transform.AbstractResourceCanLinkToInput,
) (*transform.AbstractResourceCanLinkToOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCheckCanAbstractResourceLinkTo,
		)
	}

	response, err := t.client.CanAbstractResourceLinkTo(
		ctx,
		&AbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:  t.hostID,
			Context: transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCheckCanAbstractResourceLinkTo,
		)
	}

	switch result := response.Response.(type) {
	case *CanAbstractResourceLinkToResponse_ResourceTypes:
		canLinkTo := sharedtypesv1.FromPBResourceTypes(
			result.ResourceTypes.ResourceTypes,
		)

		return &transform.AbstractResourceCanLinkToOutput{
			CanLinkTo: canLinkTo,
		}, nil
	case *CanAbstractResourceLinkToResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerCheckCanAbstractResourceLinkTo,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerCheckCanAbstractResourceLinkTo,
		),
		errorsv1.PluginActionTransformerCheckCanAbstractResourceLinkTo,
	)
}

func (t *abstractResourceTransformerClientWrapper) IsCommonTerminal(
	ctx context.Context,
	input *transform.AbstractResourceIsCommonTerminalInput,
) (*transform.AbstractResourceIsCommonTerminalOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCheckIsAbstractResourceCommonTerminal,
		)
	}

	response, err := t.client.IsAbstractResourceCommonTerminal(
		ctx,
		&AbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:  t.hostID,
			Context: transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerCheckIsAbstractResourceCommonTerminal,
		)
	}

	switch result := response.Response.(type) {
	case *IsAbstractResourceCommonTerminalResponse_Data:
		return &transform.AbstractResourceIsCommonTerminalOutput{
			IsCommonTerminal: result.Data.IsCommonTerminal,
		}, nil
	case *IsAbstractResourceCommonTerminalResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerCheckIsAbstractResourceCommonTerminal,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerCheckIsAbstractResourceCommonTerminal,
		),
		errorsv1.PluginActionTransformerCheckIsAbstractResourceCommonTerminal,
	)
}

func (t *abstractResourceTransformerClientWrapper) GetType(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeInput,
) (*transform.AbstractResourceGetTypeOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceType,
		)
	}

	// Fetching an abstract resource type from the plugin allows to obtain
	// the human-readable label for the abstract resource type so is worth while
	// for documentation and tooling purposes despite the type ID
	// already being known at this point.
	response, err := t.client.GetAbstractResourceType(
		ctx,
		&AbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:  t.hostID,
			Context: transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceType,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ResourceTypeResponse_ResourceTypeInfo:
		return &transform.AbstractResourceGetTypeOutput{
			Type:  convertv1.ResourceTypeToString(result.ResourceTypeInfo.Type),
			Label: result.ResourceTypeInfo.Label,
		}, nil
	case *sharedtypesv1.ResourceTypeResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractResourceType,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractResourceType,
		),
		errorsv1.PluginActionTransformerGetAbstractResourceType,
	)
}

func (t *abstractResourceTransformerClientWrapper) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractResourceGetTypeDescriptionInput,
) (*transform.AbstractResourceGetTypeDescriptionOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceTypeDescription,
		)
	}

	response, err := t.client.GetAbstractResourceTypeDescription(
		ctx,
		&AbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:  t.hostID,
			Context: transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceTypeDescription,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.TypeDescriptionResponse_Description:
		return fromPBTypeDescription(result.Description), nil
	case *sharedtypesv1.TypeDescriptionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractResourceTypeDescription,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractResourceTypeDescription,
		),
		errorsv1.PluginActionTransformerGetAbstractResourceTypeDescription,
	)
}

func (t *abstractResourceTransformerClientWrapper) GetExamples(
	ctx context.Context,
	input *transform.AbstractResourceGetExamplesInput,
) (*transform.AbstractResourceGetExamplesOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceExamples,
		)
	}

	response, err := t.client.GetAbstractResourceExamples(
		ctx,
		&AbstractResourceRequest{
			AbstractResourceType: &sharedtypesv1.ResourceType{
				Type: t.abstractResourceType,
			},
			HostId:  t.hostID,
			Context: transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractResourceExamples,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.ExamplesResponse_Examples:
		return fromPBExamplesForAbstractResource(result.Examples), nil
	case *sharedtypesv1.ExamplesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractResourceExamples,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractResourceExamples,
		),
		errorsv1.PluginActionTransformerGetAbstractResourceExamples,
	)
}
