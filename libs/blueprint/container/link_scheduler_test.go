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
// Shared by every wrapped resource type so the counts describe the deployment rather than
// one resource type's share of it.
type settleRecorder struct {
	mu              sync.Mutex
	checks          map[string]int
	afterLinkChecks int
}

func (r *settleRecorder) record(resourceID string, afterLinkUpdate bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checks[resourceID] += 1
	if afterLinkUpdate {
		r.afterLinkChecks += 1
	}
}

type settleCountingResource struct {
	provider.Resource
	recorder *settleRecorder
}

func (r *settleCountingResource) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	r.recorder.record(input.ResourceID, input.AfterLinkUpdate)

	return r.Resource.HasStabilised(ctx, input)
}

// A link waits for the resource it wrote to settle before releasing it.
//
// The deployer used to release the resource lock the moment the plugin call returned. An
// asynchronous cloud API accepts a change and returns while the resource is still applying
// it, so the next link acquired the resource at exactly the point the API would refuse it,
// and providers worked around that individually with their own backoff.
func Test_a_link_waits_for_the_resource_it_wrote_to_settle(t *testing.T) {
	recorder := &settleRecorder{checks: map[string]int{}}
	stateContainer := memstate.NewMemoryStateContainer()

	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{},
		stateContainer,
	).(*internal.ProviderMock)
	for resourceType, resourceImpl := range awsProvider.Resources {
		awsProvider.Resources[resourceType] = &settleCountingResource{
			Resource: resourceImpl,
			recorder: recorder,
		}
	}

	shape := linkThroughputShape{functions: 2, tablesPerFunction: 1}
	err := deployGeneratedBlueprint(context.Background(), shape, stateContainer, awsProvider)
	require.NoError(t, err)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	require.NotEmpty(t, recorder.checks, "stabilisation should have been checked at all")

	// The flag is what lets an implementation tell the two questions apart. A Cloud
	// Control resource, for one, answers a deployment check by polling the request it
	// started and has no such request for a link's write, so without the flag it reports
	// settled immediately and waits for nothing.
	require.Greater(
		t,
		recorder.afterLinkChecks,
		0,
		"a link should check that the resource it wrote has settled, and say that is why",
	)
}

// A resource that links only read does not serialise them against one another.
//
// Twelve functions linking to one table have twelve links that all touch it, and the
// deployer used to take an exclusive lock on both sides of every link whether or not it
// wrote them. That put every link reading the table in one queue that led to a mean concurrency of
// around one, on a blueprint where nothing actually contends. Declaring which side a link
// writes lets the read side go unlocked.
func Test_links_reading_a_shared_resource_are_not_serialised_by_it(t *testing.T) {
	shape := linkThroughputShape{
		functions:            12,
		tablesPerFunction:    1,
		sharedTable:          true,
		updateResourceA:      20 * time.Millisecond,
		updateResourceB:      20 * time.Millisecond,
		updateIntermediaries: 20 * time.Millisecond,
	}

	result, err := runLinkThroughputDeployment(context.Background(), shape, 0)
	require.NoError(t, err)

	// The link in this fixture declares it writes resource A, the function, and reads the
	// table. Treating both sides as written gives a mean below 1.1 on this shape.
	require.Greater(
		t,
		result.meanInFlight,
		6.0,
		"links that only read the shared table should run alongside one another",
	)
}
