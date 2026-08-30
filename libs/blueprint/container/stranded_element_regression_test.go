package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/require"
)

// An element in a later deployment group must still be deployed when the group before it
// produces no completions.
//
// Deployment is driven entirely by completions. When an element completes, the group
// after it is evaluated in full and the groups beyond that were evaluated only for
// elements depending on the one that completed. An element that is ready but is not a
// dependant of anything that completes is then never dispatched, and the deployment waits
// on a completion that cannot arrive until the engine's deployment timeout.
//
// A group produces no completions when every element in it is unchanged, which is
// ordinary on an update where part of the blueprint is already deployed, and is what the
// change set here reproduces: three resources in a chain, with the middle one carrying no
// changes.
func Test_an_element_after_a_change_free_group_is_deployed(t *testing.T) {
	ctx := context.Background()
	stateContainer := memstate.NewMemoryStateContainer()
	loader := newLinkThroughputLoader(
		stateContainer,
		newTestAWSProvider(
			/* alwaysStabilise */ true,
			/* skipRetryFailuresForLinkNames */ []string{},
			stateContainer,
		),
	)
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)

	// dependsOn rather than references, so that the middle resource orders the other two
	// without the last one needing to read anything the deployment has not deployed.
	spec := `version: 2025-11-02
resources:
  firstStore:
    type: aws/dynamodb/table
    spec:
      tableName: "first-store"
      region: "eu-west-2"
  middleStore:
    type: aws/dynamodb/table
    dependsOn:
      - firstStore
    spec:
      tableName: "middle-store"
      region: "eu-west-2"
  lastStore:
    type: aws/dynamodb/table
    dependsOn:
      - middleStore
    spec:
      tableName: "last-store"
      region: "eu-west-2"
`

	blueprintContainer, err := loader.LoadString(ctx, spec, schema.YAMLSpecFormat, params)
	require.NoError(t, err)

	deployChanges, err := stageLinkThroughputChanges(ctx, blueprintContainer, params)
	require.NoError(t, err)

	// The middle resource is already deployed and has nothing to apply, so it is not
	// part of this deployment and never completes.
	require.Contains(t, deployChanges.NewResources, "middleStore")
	delete(deployChanges.NewResources, "middleStore")

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		ctx,
		&DeployInput{
			InstanceName: "StrandedElementInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	require.NoError(t, err)

	err = awaitLinkThroughputDeployment(channels)
	require.NoError(t, err)

	instanceID, err := stateContainer.Instances().LookupIDByName(ctx, "StrandedElementInstance")
	require.NoError(t, err)

	instance, err := stateContainer.Instances().Get(ctx, instanceID)
	require.NoError(t, err)
	require.Contains(
		t,
		instance.ResourceIDs,
		"lastStore",
		"an element behind a group with no changes was never deployed",
	)
}
