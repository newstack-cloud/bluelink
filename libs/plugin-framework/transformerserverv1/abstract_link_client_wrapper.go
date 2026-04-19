package transformerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

type abstractLinkTransformerClientWrapper struct {
	client           TransformerClient
	abstractLinkType string
	hostID           string
}

func (t *abstractLinkTransformerClientWrapper) GetType(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeInput,
) (*transform.AbstractLinkGetTypeOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkType,
		)
	}

	response, err := t.client.GetAbstractLinkType(
		ctx,
		&GetAbstractLinkTypeRequest{
			LinkType: t.abstractLinkType,
			HostId:   t.hostID,
			Context:  transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkType,
		)
	}

	switch result := response.Response.(type) {
	case *GetAbstractLinkTypeResponse_LinkType:
		return fromPBAbstractLinkType(result.LinkType), nil
	case *GetAbstractLinkTypeResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractLinkType,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractLinkType,
		),
		errorsv1.PluginActionTransformerGetAbstractLinkType,
	)
}

func (t *abstractLinkTransformerClientWrapper) GetTypeDescription(
	ctx context.Context,
	input *transform.AbstractLinkGetTypeDescriptionInput,
) (*transform.AbstractLinkGetTypeDescriptionOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkTypeDescription,
		)
	}

	response, err := t.client.GetAbstractLinkTypeDescription(
		ctx,
		&GetAbstractLinkTypeRequest{
			LinkType: t.abstractLinkType,
			HostId:   t.hostID,
			Context:  transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkTypeDescription,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.TypeDescriptionResponse_Description:
		return fromPBAbstractLinkTypeDescription(result.Description), nil
	case *sharedtypesv1.TypeDescriptionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractLinkTypeDescription,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractLinkTypeDescription,
		),
		errorsv1.PluginActionTransformerGetAbstractLinkTypeDescription,
	)
}

func (t *abstractLinkTransformerClientWrapper) GetAnnotationDefinitions(
	ctx context.Context,
	input *transform.AbstractLinkGetAnnotationDefinitionsInput,
) (*transform.AbstractLinkGetAnnotationDefinitionsOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
		)
	}

	response, err := t.client.GetAbstractLinkAnnotationDefinitions(
		ctx,
		&GetAbstractLinkTypeRequest{
			LinkType: t.abstractLinkType,
			HostId:   t.hostID,
			Context:  transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.LinkAnnotationDefinitionsResponse_AnnotationDefinitions:
		output, err := fromPBAbstractLinkAnnotationDefinitions(
			result.AnnotationDefinitions,
		)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
			)
		}

		return output, nil
	case *sharedtypesv1.LinkAnnotationDefinitionsResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
		),
		errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
	)
}

func (t *abstractLinkTransformerClientWrapper) GetCardinality(
	ctx context.Context,
	input *transform.AbstractLinkGetCardinalityInput,
) (*transform.AbstractLinkGetCardinalityOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkCardinality,
		)
	}

	response, err := t.client.GetAbstractLinkCardinality(
		ctx,
		&GetAbstractLinkTypeRequest{
			LinkType: t.abstractLinkType,
			HostId:   t.hostID,
			Context:  transformerCtx,
		},
	)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetAbstractLinkCardinality,
		)
	}

	switch result := response.Response.(type) {
	case *sharedtypesv1.LinkCardinalityResponse_CardinalityInfo:
		return convertv1.FromPBLinkCardinalityResponseForAbstract(result), nil
	case *sharedtypesv1.LinkCardinalityResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetAbstractLinkCardinality,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetAbstractLinkCardinality,
		),
		errorsv1.PluginActionTransformerGetAbstractLinkCardinality,
	)
}
