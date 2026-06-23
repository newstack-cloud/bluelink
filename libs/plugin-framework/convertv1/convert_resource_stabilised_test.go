package convertv1

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verifies that the computed
// fields finalised when a resource stabilises survive the gRPC boundary: the provider
// output is converted to its protobuf form and back without loss.
func Test_resource_has_stabilised_computed_fields_round_trip(t *testing.T) {
	output := &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.endpoint.address": core.MappingNodeFromString("db.example.com"),
			"spec.arn":              core.MappingNodeFromString("arn:aws:rds:::db/x"),
		},
	}

	pbResponse, err := ToPBResourceHasStabilisedResponse(output)
	require.NoError(t, err)

	info, ok := pbResponse.Response.(*sharedtypesv1.ResourceHasStabilisedResponse_ResourceStabilisationInfo)
	require.True(t, ok, "expected a ResourceStabilisationInfo response")
	assert.True(t, info.ResourceStabilisationInfo.Stabilised)

	computedFieldValues, err := FromPBMappingNodeMap(info.ResourceStabilisationInfo.ComputedFieldValues)
	require.NoError(t, err)

	assert.Equal(t, output.ComputedFieldValues, computedFieldValues)
}

// Verifies the common case where a
// resource stabilises without finalising any computed fields still round-trips.
func Test_resource_has_stabilised_no_computed_fields(t *testing.T) {
	output := &provider.ResourceHasStabilisedOutput{Stabilised: true}

	pbResponse, err := ToPBResourceHasStabilisedResponse(output)
	require.NoError(t, err)

	info, ok := pbResponse.Response.(*sharedtypesv1.ResourceHasStabilisedResponse_ResourceStabilisationInfo)
	require.True(t, ok, "expected a ResourceStabilisationInfo response")
	assert.True(t, info.ResourceStabilisationInfo.Stabilised)

	computedFieldValues, err := FromPBMappingNodeMap(info.ResourceStabilisationInfo.ComputedFieldValues)
	require.NoError(t, err)
	assert.Empty(t, computedFieldValues)
}
