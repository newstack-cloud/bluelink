package container

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A link that failed still has to release the links waiting on a capability it provides.
//
// The deployer used to record a link only after it deployed successfully, which was
// enough while a failure ended the batch that was running. A scheduler outlives the
// failure, so a link that never reports leaves every requirer pending for the rest of the
// deployment with nothing said about why.
func Test_a_settled_link_releases_the_links_waiting_on_its_capability(t *testing.T) {
	vpc, function, queue := linkOrderingChain()
	deployState := NewDefaultDeploymentState()
	deployState.SetLinkCapabilityGraph(placementOrderingGraph(t))

	deployState.UpdateLinkDeploymentState(vpc)
	deployState.UpdateLinkDeploymentState(queue)
	deployState.UpdateLinkDeploymentState(function)

	require.Equal(
		t,
		[]string{"netVPC::netFunction"},
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
		"the access link should wait for the placement link that provides its capability",
	)

	// Whatever the outcome. The scheduler settles a link that failed, one that ran out
	// of retries and one that is not in the change set, all through here.
	deployState.MarkLinkSettled("netVPC::netFunction")

	require.Empty(
		t,
		deployState.AwaitingCapabilityProviders("netFunction::netQueue"),
		"a settled provider should release its requirers rather than leaving them pending",
	)
}

// Links on unrelated resources are deployed at the same time, up to the deployment's
// concurrency bound.
//
// The benchmark measures this across several shapes; this asserts it, so a change that
// quietly reintroduces serial link deployment fails the suite rather than only showing up
// in a benchmark nobody ran.
func Test_links_on_distinct_resources_are_deployed_concurrently(t *testing.T) {
	// A per-phase cost is what makes overlap observable at all. With links that return
	// immediately, one finishes before the next is dispatched however concurrent the
	// scheduler is, and the measurement says nothing.
	shape := linkThroughputShape{
		functions:            20,
		tablesPerFunction:    3,
		updateResourceA:      20 * time.Millisecond,
		updateResourceB:      20 * time.Millisecond,
		updateIntermediaries: 20 * time.Millisecond,
	}

	result, err := runLinkThroughputDeployment(context.Background(), shape, 0)
	require.NoError(t, err)

	// This shape reaches the default bound of 20 in flight with a mean above 19, so the
	// thresholds sit well below what it does rather than at the edge of it. Serial link
	// deployment produces a mean of 1.0 whatever the shape, which is what this is here
	// to catch.
	require.GreaterOrEqual(
		t,
		result.peakInFlight,
		10,
		"links on distinct functions should be in flight together",
	)
	require.Greater(
		t,
		result.meanInFlight,
		5.0,
		"the link phase should be concurrent throughout, not only in a brief burst",
	)
}
