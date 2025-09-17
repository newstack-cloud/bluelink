package transformerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
)

// NewTransformerPlugin creates a new instance of a transformer plugin
// from a blueprint framework transform.SpecTransforemr implementation.
// This produces a gRPC server plugin that the deploy engine host
// can interact with.
// The `TransformerPluginDefinition` utility type can be passed in to
// create a transformer plugin server as it implements the `transform.SpecTransformer`
// interface.
//
// The host info container is used to retrieve the ID of the host
// that the plugin was registered with.
//
// The service client is used to communicate with other plugins
// that are registered with the deploy engine host.
func NewTransformerPlugin(
	bpTransformer transform.SpecTransformer,
	hostInfoContainer pluginutils.HostInfoContainer,
	serviceClient pluginservicev1.ServiceClient,
) transformerserverv1.TransformerServer {
	return &blueprintTransformerPluginImpl{
		bpTransformer:     bpTransformer,
		hostInfoContainer: hostInfoContainer,
		serviceClient:     serviceClient,
	}
}

type blueprintTransformerPluginImpl struct {
	transformerserverv1.UnimplementedTransformerServer
	bpTransformer     transform.SpecTransformer
	hostInfoContainer pluginutils.HostInfoContainer
	serviceClient     pluginservicev1.ServiceClient
}

func (p *blueprintTransformerPluginImpl) GetTransformName(
	ctx context.Context,
	req *transformerserverv1.TransformerRequest,
) (*transformerserverv1.TransformNameResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toTransformNameErrorResponse(err), nil
	}

	transformName, err := p.bpTransformer.GetTransformName(ctx)
	if err != nil {
		return toTransformNameErrorResponse(err), nil
	}

	return toPBTransformNameResponse(transformName), nil
}

func (p *blueprintTransformerPluginImpl) GetConfigDefinition(
	ctx context.Context,
	req *transformerserverv1.TransformerRequest,
) (*sharedtypesv1.ConfigDefinitionResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return convertv1.ToPBConfigDefinitionErrorResponse(err), nil
	}

	configDefinition, err := p.bpTransformer.ConfigDefinition(ctx)
	if err != nil {
		return convertv1.ToPBConfigDefinitionErrorResponse(err), nil
	}

	configDefinitionPB, err := convertv1.ToPBConfigDefinitionResponse(
		configDefinition,
	)
	if err != nil {
		return convertv1.ToPBConfigDefinitionErrorResponse(err), nil
	}

	return configDefinitionPB, nil
}

func (p *blueprintTransformerPluginImpl) Transform(
	ctx context.Context,
	req *transformerserverv1.BlueprintTransformRequest,
) (*transformerserverv1.BlueprintTransformResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toBlueprintTransformErrorResponse(err), nil
	}

	inputBlueprint, err := serialisation.FromSchemaPB(
		req.InputBlueprint,
	)
	if err != nil {
		return toBlueprintTransformErrorResponse(err), nil
	}

	transformOutput, err := p.bpTransformer.Transform(
		ctx,
		&transform.SpecTransformerTransformInput{
			InputBlueprint: inputBlueprint,
		},
	)
	if err != nil {
		return toBlueprintTransformErrorResponse(err), nil
	}

	transformedBlueprint, err := serialisation.ToSchemaPB(
		transformOutput.TransformedBlueprint,
	)
	if err != nil {
		return toBlueprintTransformErrorResponse(err), nil
	}

	return &transformerserverv1.BlueprintTransformResponse{
		Response: &transformerserverv1.BlueprintTransformResponse_TransformedBlueprint{
			TransformedBlueprint: transformedBlueprint,
		},
	}, nil
}

func (p *blueprintTransformerPluginImpl) ListAbstractResourceTypes(
	ctx context.Context,
	req *transformerserverv1.TransformerRequest,
) (*transformerserverv1.AbstractResourceTypesResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toListAbstractResourceTypesErrorResponse(err), nil
	}

	abstractResourceTypes, err := p.bpTransformer.ListAbstractResourceTypes(ctx)
	if err != nil {
		return toListAbstractResourceTypesErrorResponse(err), nil
	}

	return toPBAbstractResourceTypesResponse(abstractResourceTypes), nil
}

func (p *blueprintTransformerPluginImpl) CustomValidateAbstractResource(
	ctx context.Context,
	req *transformerserverv1.CustomValidateAbstractResourceRequest,
) (*transformerserverv1.CustomValidateAbstractResourceResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toCustomValidateAbstractResourceErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return toCustomValidateAbstractResourceErrorResponse(err), nil
	}

	validationInput, err := fromPBCustomValidateAbstractResourceRequest(req)
	if err != nil {
		return toCustomValidateAbstractResourceErrorResponse(err), nil
	}

	output, err := abstractResource.CustomValidate(
		ctx,
		validationInput,
	)
	if err != nil {
		return toCustomValidateAbstractResourceErrorResponse(err), nil
	}

	return toPBCustomValidateAbstractResourceResponse(output)
}

func (p *blueprintTransformerPluginImpl) GetAbstractResourceSpecDefinition(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*transformerserverv1.AbstractResourceSpecDefinitionResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toAbstractResourceSpecDefinitionErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return toAbstractResourceSpecDefinitionErrorResponse(err), nil
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return toAbstractResourceSpecDefinitionErrorResponse(err), nil
	}

	output, err := abstractResource.GetSpecDefinition(
		ctx,
		&transform.AbstractResourceGetSpecDefinitionInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return toAbstractResourceSpecDefinitionErrorResponse(err), nil
	}

	response, err := toPBAbstractResourceSpecDefinitionResponse(output)
	if err != nil {
		return toAbstractResourceSpecDefinitionErrorResponse(err), nil
	}

	return response, nil
}

func (p *blueprintTransformerPluginImpl) CanAbstractResourceLinkTo(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*transformerserverv1.CanAbstractResourceLinkToResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toCanAbstractResourceLinkToErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return toCanAbstractResourceLinkToErrorResponse(err), nil
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return toCanAbstractResourceLinkToErrorResponse(err), nil
	}

	output, err := abstractResource.CanLinkTo(
		ctx,
		&transform.AbstractResourceCanLinkToInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return toCanAbstractResourceLinkToErrorResponse(err), nil
	}

	return toPBCanAbstractResourceLinkToResponse(output), nil
}

func (p *blueprintTransformerPluginImpl) IsAbstractResourceCommonTerminal(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*transformerserverv1.IsAbstractResourceCommonTerminalResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return toIsAbstractResourceCommonTerminalErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return toIsAbstractResourceCommonTerminalErrorResponse(err), nil
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return toIsAbstractResourceCommonTerminalErrorResponse(err), nil
	}

	output, err := abstractResource.IsCommonTerminal(
		ctx,
		&transform.AbstractResourceIsCommonTerminalInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return toIsAbstractResourceCommonTerminalErrorResponse(err), nil
	}

	return toPBAbstractResourceCommonTerminalResponse(output), nil
}

func (p *blueprintTransformerPluginImpl) GetAbstractResourceType(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*sharedtypesv1.ResourceTypeResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return convertv1.ToPBResourceTypeErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return convertv1.ToPBResourceTypeErrorResponse(err), nil
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return convertv1.ToPBResourceTypeErrorResponse(err), nil
	}

	output, err := abstractResource.GetType(
		ctx,
		&transform.AbstractResourceGetTypeInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return convertv1.ToPBResourceTypeErrorResponse(err), nil
	}

	return toPBAbstractResourceTypeResponse(output), nil
}

func (p *blueprintTransformerPluginImpl) GetAbstractResourceTypeDescription(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*sharedtypesv1.TypeDescriptionResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return convertv1.ToPBTypeDescriptionErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return convertv1.ToPBTypeDescriptionErrorResponse(err), nil
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return convertv1.ToPBTypeDescriptionErrorResponse(err), nil
	}

	output, err := abstractResource.GetTypeDescription(
		ctx,
		&transform.AbstractResourceGetTypeDescriptionInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return convertv1.ToPBTypeDescriptionErrorResponse(err), nil
	}

	return toPBAbstractResourceTypeDescriptionResponse(output), nil
}

func (p *blueprintTransformerPluginImpl) GetAbstractResourceExamples(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*sharedtypesv1.ExamplesResponse, error) {
	err := p.checkHostID(req.HostId)
	if err != nil {
		return convertv1.ToPBExamplesErrorResponse(err), nil
	}

	abstractResource, err := p.bpTransformer.AbstractResource(
		ctx,
		convertv1.ResourceTypeToString(req.AbstractResourceType),
	)
	if err != nil {
		return convertv1.ToPBExamplesErrorResponse(err), nil
	}

	transformerCtx, err := fromPBTransformerContext(req.Context)
	if err != nil {
		return convertv1.ToPBExamplesErrorResponse(err), nil
	}

	output, err := abstractResource.GetExamples(
		ctx,
		&transform.AbstractResourceGetExamplesInput{
			TransformerContext: transformerCtx,
		},
	)
	if err != nil {
		return convertv1.ToPBExamplesErrorResponse(err), nil
	}

	return toPBAbstractResourceExamplesResponse(output), nil
}

func (p *blueprintTransformerPluginImpl) checkHostID(hostID string) error {
	if hostID != p.hostInfoContainer.GetID() {
		return errorsv1.ErrInvalidHostID(hostID)
	}

	return nil
}
