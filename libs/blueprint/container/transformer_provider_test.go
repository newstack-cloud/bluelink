package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type TransformerResourceProviderTestSuite struct {
	suite.Suite
}

func (s *TransformerResourceProviderTestSuite) Test_provides_transformer_context_to_abstract_resources() {
	baseTransformer := &internal.CelerityTransformer{}
	baseAbstractResource, err := baseTransformer.AbstractResource(
		context.TODO(),
		internal.CelerityHandlerResourceType,
	)
	s.Require().NoError(err)

	transformer := &contextRecordingTransformer{
		SpecTransformer: baseTransformer,
		abstractResource: &contextRecordingAbstractResource{
			AbstractResource: baseAbstractResource,
		},
	}
	loader := NewDefaultLoader(
		map[string]provider.Provider{
			"aws": newTestAWSProvider(
				/* alwaysStabilise */ false,
				/* skipRetryFailuresForLinkNames */ []string{},
				/* stateContainer */ nil,
			),
		},
		map[string]transform.SpecTransformer{
			internal.CelerityTransformName: transformer,
		},
		/* stateContainer */ nil,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)

	_, err = loader.Validate(
		context.TODO(),
		"__testdata/loader/valid-celerity-blueprint.yml",
		createParamsWithCelerityConfig(),
	)
	s.Require().NoError(err)

	recorded := transformer.abstractResource.lastTransformerContext
	s.Require().NotNil(recorded)
	configVar, hasConfigVar := recorded.TransformerConfigVariable("deployTarget")
	s.Require().True(hasConfigVar)
	s.Assert().Equal("serverless", core.StringValueFromScalar(configVar))
}

func createParamsWithCelerityConfig() core.BlueprintParams {
	deployTarget := "serverless"
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{
			// The transformer namespace is derived from the abstract resource
			// type prefix ("celerity/handler"), not the transform name.
			"celerity": {
				"deployTarget": {StringValue: &deployTarget},
			},
		},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

type contextRecordingTransformer struct {
	transform.SpecTransformer
	abstractResource *contextRecordingAbstractResource
}

func (t *contextRecordingTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	if resourceType != internal.CelerityHandlerResourceType {
		return nil, nil
	}

	return t.abstractResource, nil
}

type contextRecordingAbstractResource struct {
	transform.AbstractResource
	lastTransformerContext transform.Context
}

func (r *contextRecordingAbstractResource) GetSpecDefinition(
	ctx context.Context,
	input *transform.AbstractResourceGetSpecDefinitionInput,
) (*transform.AbstractResourceGetSpecDefinitionOutput, error) {
	r.lastTransformerContext = input.TransformerContext
	return r.AbstractResource.GetSpecDefinition(ctx, input)
}

func TestTransformerResourceProviderTestSuite(t *testing.T) {
	suite.Run(t, new(TransformerResourceProviderTestSuite))
}
