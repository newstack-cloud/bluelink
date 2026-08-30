package container

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
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

// Counts stabilisation checks per resource so the link settle poll can be told apart from
// the one the resource deployer runs.
type settleCountingResource struct {
	provider.Resource
	mu     *sync.Mutex
	counts map[string]int
}

func (r *settleCountingResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	r.mu.Lock()
	r.counts[input.ResourceID] += 1
	r.mu.Unlock()

	return r.Resource.HasStabilised(ctx, input)
}

// A link waits for the resource it wrote to settle before releasing it.
//
// The deployer used to release the resource lock the moment the plugin call returned. An
// asynchronous cloud API accepts a change and returns while the resource is still applying
// it, so the next link acquired the resource at exactly the point the API would refuse it,
// and providers worked around that individually with their own backoff.
func Test_a_link_waits_for_the_resource_it_wrote_to_settle(t *testing.T) {
	counts := map[string]int{}
	countsMu := &sync.Mutex{}
	stateContainer := memstate.NewMemoryStateContainer()

	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{},
		stateContainer,
	).(*internal.ProviderMock)
	for resourceType, resourceImpl := range awsProvider.Resources {
		awsProvider.Resources[resourceType] = &settleCountingResource{
			Resource: resourceImpl,
			mu:       countsMu,
			counts:   counts,
		}
	}

	shape := linkThroughputShape{functions: 2, tablesPerFunction: 1}
	err := deployGeneratedBlueprint(context.Background(), shape, stateContainer, awsProvider)
	require.NoError(t, err)

	countsMu.Lock()
	defer countsMu.Unlock()

	// Every resource is polled once by its own deployment. A resource a link wrote is
	// polled again before the link gives up its lock, so the functions each see more
	// checks than the tables that no link writes on the A side.
	require.NotEmpty(t, counts, "stabilisation should have been checked at all")

	totalChecks := 0
	for _, count := range counts {
		totalChecks += count
	}
	require.Greater(
		t,
		totalChecks,
		len(counts),
		"a link should check that the resource it wrote has settled, "+
			"over and above the check each resource gets from its own deployment",
	)
}
