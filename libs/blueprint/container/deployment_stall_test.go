package container

import (
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/mockclock"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/require"
)

func stallTestChanges() *changes.BlueprintChanges {
	return &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"ordersTable":       {},
			"saveOrderFunction": {},
		},
	}
}

func stallTestDeployContext(deployState DeploymentState) *DeployContext {
	return &DeployContext{
		State:        deployState,
		InputChanges: stallTestChanges(),
		LinkScheduler: newLinkScheduler(
			NewLinkSlots(DefaultMaxConcurrentLinks),
			func(string) provider.LinkModifies { return provider.LinkModifiesBoth },
			nil,
			nil,
			func(string) {},
			func(string) []string { return nil },
			func(error) bool { return true },
		),
	}
}

// A deployment waiting on an element that will never be deployed reports it by name.
//
// Nothing else bounds that wait. Every other stall in a deployment has a timeout, and an
// element that is never dispatched has none, so without this the deployment sits until the
// engine's own deployment timeout saying nothing about what it is waiting for.
func Test_a_deployment_waiting_on_an_element_that_will_never_deploy_reports_it(t *testing.T) {
	clock := mockclock.NewAdvanceableClock(time.Unix(mockclock.CurrentTimeUnixMock, 0))
	deployState := NewDefaultDeploymentState()
	deployCtx := stallTestDeployContext(deployState)
	detector := newDeploymentStallDetector(2, clock)

	finished := map[string]*deployUpdateMessageWrapper{
		"resources.ordersTable": {},
	}

	// The first observation only records that it looks stalled: a deployment between
	// elements looks the same as one that will never move again.
	require.Empty(t, detector.check(deployCtx, finished /* draining */, false))

	// Still stalled, but not for long enough to be sure of it.
	clock.Advance(stalledDeploymentGracePeriod / 2)
	require.Empty(t, detector.check(deployCtx, finished /* draining */, false))

	clock.Advance(stalledDeploymentGracePeriod)

	require.Equal(
		t,
		[]string{"resources.saveOrderFunction"},
		detector.check(deployCtx, finished /* draining */, false),
	)
}

// Work in flight is not a stall however long it takes, which matters because a resource
// waiting on a slow provider looks identical from here.
func Test_a_deployment_with_work_in_flight_is_not_reported_as_stalled(t *testing.T) {
	clock := mockclock.NewAdvanceableClock(time.Unix(mockclock.CurrentTimeUnixMock, 0))
	deployState := NewDefaultDeploymentState()
	deployState.SetElementInProgress(&ResourceIDInfo{
		ResourceID:   "test-order-function-id",
		ResourceName: "saveOrderFunction",
	})
	deployCtx := stallTestDeployContext(deployState)
	detector := newDeploymentStallDetector(2, clock)

	finished := map[string]*deployUpdateMessageWrapper{
		"resources.ordersTable": {},
	}

	require.Empty(t, detector.check(deployCtx, finished /* draining */, false))
	clock.Advance(stalledDeploymentGracePeriod + time.Second)
	require.Empty(t, detector.check(deployCtx, finished /* draining */, false))
}

// A deployment that recovers before the grace period elapses is not reported, so a gap
// between elements cannot fail a deployment that was going to succeed.
func Test_a_deployment_that_resumes_within_the_grace_period_is_not_reported(t *testing.T) {
	clock := mockclock.NewAdvanceableClock(time.Unix(mockclock.CurrentTimeUnixMock, 0))
	deployState := NewDefaultDeploymentState()
	deployCtx := stallTestDeployContext(deployState)
	detector := newDeploymentStallDetector(2, clock)

	stalled := map[string]*deployUpdateMessageWrapper{
		"resources.ordersTable": {},
	}
	require.Empty(t, detector.check(deployCtx, stalled /* draining */, false))

	clock.Advance(stalledDeploymentGracePeriod / 2)
	progressed := map[string]*deployUpdateMessageWrapper{
		"resources.ordersTable":       {},
		"resources.saveOrderFunction": {},
	}
	require.Empty(t, detector.check(deployCtx, progressed /* draining */, false))

	// The clock passing the grace period must not report a deployment that moved on.
	clock.Advance(stalledDeploymentGracePeriod + time.Second)
	require.Empty(t, detector.check(deployCtx, progressed /* draining */, false))
}
