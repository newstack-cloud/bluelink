package testtransformer

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingTransformerServer struct {
	transformerserverv1.UnimplementedTransformerServer
}

func (p *failingTransformerServer) GetTransformName(
	ctx context.Context,
	req *transformerserverv1.TransformerRequest,
) (*transformerserverv1.TransformNameResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving transform name",
	)
}

func (p *failingTransformerServer) GetConfigDefinition(
	ctx context.Context,
	req *transformerserverv1.TransformerRequest,
) (*sharedtypesv1.ConfigDefinitionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving config definition",
	)
}

func (p *failingTransformerServer) Transform(
	ctx context.Context,
	req *transformerserverv1.BlueprintTransformRequest,
) (*transformerserverv1.BlueprintTransformResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred transforming blueprint",
	)
}

func (p *failingTransformerServer) ListAbstractResourceTypes(
	ctx context.Context,
	req *transformerserverv1.TransformerRequest,
) (*transformerserverv1.AbstractResourceTypesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred listing abstract resource types",
	)
}

func (p *failingTransformerServer) CustomValidateAbstractResource(
	ctx context.Context,
	req *transformerserverv1.CustomValidateAbstractResourceRequest,
) (*transformerserverv1.CustomValidateAbstractResourceResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred validating abstract resource",
	)
}

func (p *failingTransformerServer) GetAbstractResourceSpecDefinition(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*transformerserverv1.AbstractResourceSpecDefinitionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving abstract resource spec definition",
	)
}

func (p *failingTransformerServer) CanAbstractResourceLinkTo(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*transformerserverv1.CanAbstractResourceLinkToResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred checking the resource types that the abstract resource can link to",
	)
}

func (p *failingTransformerServer) IsAbstractResourceCommonTerminal(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*transformerserverv1.IsAbstractResourceCommonTerminalResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred checking if abstract resource is a common terminal",
	)
}

func (p *failingTransformerServer) GetAbstractResourceType(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*sharedtypesv1.ResourceTypeResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving abstract resource type",
	)
}

func (p *failingTransformerServer) GetAbstractResourceTypeDescription(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*sharedtypesv1.TypeDescriptionResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving abstract resource type description",
	)
}

func (p *failingTransformerServer) GetAbstractResourceExamples(
	ctx context.Context,
	req *transformerserverv1.AbstractResourceRequest,
) (*sharedtypesv1.ExamplesResponse, error) {
	return nil, status.Error(
		codes.Unknown,
		"internal error occurred retrieving abstract resource examples",
	)
}
