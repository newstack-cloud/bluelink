package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/stretchr/testify/require"
)

// A resource that nothing links to and nothing references still has to be deployed.
//
// Deployment nodes are built from link chains, so a resource with no links reaches the
// deployment groups only through the standalone pass. A resource that falls out of that is
// counted in the change set but never dispatched, and the deployment waits for a
// completion that cannot arrive.
func Test_a_resource_with_no_links_or_references_is_deployed(t *testing.T) {
	stateContainer := memstate.NewMemoryStateContainer()
	spec := generateLinkThroughputBlueprint(
		linkThroughputShape{functions: 2, tablesPerFunction: 1},
	) + `  isolatedStore:
    type: aws/dynamodb/table
    spec:
      tableName: "isolated-store"
      region: "eu-west-2"
`

	err := deployGeneratedBlueprintSpec(
		context.Background(),
		spec,
		stateContainer,
		newTestAWSProvider(
			/* alwaysStabilise */ true,
			/* skipRetryFailuresForLinkNames */ []string{},
			stateContainer,
		),
	)
	require.NoError(t, err)

	instanceIDs, err := stateContainer.Instances().LookupIDByName(
		context.Background(),
		"SettleObservedInstance",
	)
	require.NoError(t, err)

	instance, err := stateContainer.Instances().Get(context.Background(), instanceIDs)
	require.NoError(t, err)
	require.Contains(
		t,
		instance.ResourceIDs,
		"isolatedStore",
		"a resource with no links was in the change set but never deployed",
	)
}
