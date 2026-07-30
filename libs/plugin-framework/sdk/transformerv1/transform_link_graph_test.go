package transformerv1_test

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/transformerserverv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Records the input it receives so a test can
// assert the gRPC Transform handler reconstructed the link graph from the
// request rather than dropping it.
type linkGraphCapturingTransformer struct {
	transform.SpecTransformer
	captured *transform.SpecTransformerTransformInput
}

func (t *linkGraphCapturingTransformer) Transform(
	_ context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	t.captured = input
	return &transform.SpecTransformerTransformOutput{
		TransformedBlueprint: input.InputBlueprint,
	}, nil
}

// The Transform gRPC handler must reconstruct the declared link graph the host
// sends and pass it to the transformer. Previously it dropped the link graph,
// leaving input.LinkGraph nil, which crashed transformers that iterate it.
func TestTransformHandler_reconstructsLinkGraph(t *testing.T) {
	hostContainer := pluginutils.NewHostInfoContainer()
	hostContainer.SetID("test-host")

	capturing := &linkGraphCapturingTransformer{}
	server := transformerv1.NewTransformerPlugin(capturing, hostContainer, nil)

	bpPB, err := serialisation.ToSchemaPB(&schema.Blueprint{
		Version: core.ScalarFromString("2025-05-12"),
		Resources: &schema.ResourceMap{
			Values: map[string]*schema.Resource{},
		},
	})
	require.NoError(t, err)

	resp, err := server.Transform(context.Background(), &transformerserverv1.BlueprintTransformRequest{
		HostId:         "test-host",
		InputBlueprint: bpPB,
		LinkGraph: &transformerserverv1.DeclaredLinkGraph{
			Edges: []*transformerserverv1.ResolvedLink{
				{
					Source:     "myApi",
					Target:     "myHandler",
					SourceType: "celerity/api",
					TargetType: "celerity/handler",
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.NotNil(t, capturing.captured, "transformer was not invoked")
	require.NotNil(t, capturing.captured.LinkGraph, "link graph was dropped by the handler")
	edges := capturing.captured.LinkGraph.EdgesFrom("myApi")
	require.Len(t, edges, 1)
	assert.Equal(t, "myHandler", edges[0].Target)
}
