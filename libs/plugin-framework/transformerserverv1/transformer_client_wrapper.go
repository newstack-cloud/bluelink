package transformerserverv1

import (
	context "context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

// WrapTransformerClient wraps a transformer plugin v1 TransformerClient
// in a blueprint framework SpecTransformer to allow the deploy engine
// to interact with the transformer in a way that is compatible
// with the blueprint framework and is agnostic to the underlying
// communication protocol.
func WrapTransformerClient(client TransformerClient, hostID string) transform.SpecTransformer {
	return &transformerClientWrapper{
		client: client,
		hostID: hostID,
	}
}

type transformerClientWrapper struct {
	client TransformerClient
	hostID string
}

func (p *transformerClientWrapper) GetTransformName(ctx context.Context) (string, error) {
	response, err := p.client.GetTransformName(ctx, &TransformerRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return "", errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetTransformName,
		)
	}

	switch result := response.Response.(type) {
	case *TransformNameResponse_NameInfo:
		return result.NameInfo.GetTransformName(), nil
	case *TransformNameResponse_ErrorResponse:
		return "", errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerGetTransformName,
		)
	}

	return "", errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerGetTransformName,
		),
		errorsv1.PluginActionTransformerGetTransformName,
	)
}

func (p *transformerClientWrapper) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	response, err := p.client.GetConfigDefinition(ctx, &TransformerRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerGetConfigDefinition,
		)
	}

	return convertv1.FromPBConfigDefinitionResponse(
		response,
		errorsv1.PluginActionTransformerGetConfigDefinition,
	)
}

func (p *transformerClientWrapper) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	transformerCtx, err := toPBTransformerContext(input.TransformerContext)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerTransform,
		)
	}

	blueprintPB, err := serialisation.ToSchemaPB(input.InputBlueprint)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerTransform,
		)
	}

	response, err := p.client.Transform(ctx, &BlueprintTransformRequest{
		InputBlueprint: blueprintPB,
		HostId:         p.hostID,
		Context:        transformerCtx,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerTransform,
		)
	}

	switch result := response.Response.(type) {
	case *BlueprintTransformResponse_TransformedBlueprint:
		transformed, err := serialisation.FromSchemaPB(result.TransformedBlueprint)
		if err != nil {
			return nil, errorsv1.CreateGeneralError(
				err,
				errorsv1.PluginActionTransformerTransform,
			)
		}

		return &transform.SpecTransformerTransformOutput{
			TransformedBlueprint: transformed,
		}, nil
	case *BlueprintTransformResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerTransform,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerTransform,
		),
		errorsv1.PluginActionTransformerTransform,
	)
}

func (p *transformerClientWrapper) AbstractResource(
	ctx context.Context,
	abstractResourceType string,
) (transform.AbstractResource, error) {
	return &abstractResourceTransformerClientWrapper{
		client:               p.client,
		abstractResourceType: abstractResourceType,
		hostID:               p.hostID,
	}, nil
}

func (p *transformerClientWrapper) ListAbstractResourceTypes(ctx context.Context) ([]string, error) {
	response, err := p.client.ListAbstractResourceTypes(ctx, &TransformerRequest{
		HostId: p.hostID,
	})
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			errorsv1.PluginActionTransformerListAbstractResourceTypes,
		)
	}

	switch result := response.Response.(type) {
	case *AbstractResourceTypesResponse_AbstractResourceTypes:
		return sharedtypesv1.FromPBResourceTypes(
			result.AbstractResourceTypes.ResourceTypes,
		), nil
	case *AbstractResourceTypesResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			errorsv1.PluginActionTransformerListAbstractResourceTypes,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			errorsv1.PluginActionTransformerListAbstractResourceTypes,
		),
		errorsv1.PluginActionTransformerListAbstractResourceTypes,
	)
}
