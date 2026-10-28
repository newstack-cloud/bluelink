package objectstore_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

const (
	multiEngineCacheAge = time.Minute
	secondResourceID    = "records-second-resource"
)

// Engines sharing one bucket. Each engine is its own container with its own cache, and all of
// them read the same clock, which a test advances to age the caches.
type MultiEngineSuite struct {
	objectstoreSuite
	clock *internal.MockClock
}

func (s *MultiEngineSuite) SetupTest() {
	s.objectstoreSuite.SetupTest()
	s.clock = &internal.MockClock{Timestamp: reservationTimestamp}
}

func (s *MultiEngineSuite) Test_a_change_by_another_engine_is_seen_once_the_cache_age_has_passed() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	engineA, engineB := s.engine(), s.engine()
	s.requireResourceStatus(engineA, recordsResourceID, core.ResourceStatusCreated)

	s.Require().NoError(engineB.Resources().UpdateStatus(ctx, recordsResourceID, updateFailed()))

	s.requireResourceStatus(engineA, recordsResourceID, core.ResourceStatusCreated)
	s.advance(multiEngineCacheAge)
	s.requireResourceStatus(engineA, recordsResourceID, core.ResourceStatusUpdateFailed)
}

// Drift is its own object, but saving it rewrites the instance, which is how engine A notices.
func (s *MultiEngineSuite) Test_drift_saved_by_another_engine_is_seen_once_the_cache_age_has_passed() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	s.Require().NoError(s.engine().Resources().SaveDrift(ctx, state.ResourceDriftState{
		ResourceID:   recordsResourceID,
		ResourceName: recordsResourceName,
	}))
	engineA := s.engine()
	drift, err := engineA.Resources().GetDrift(ctx, recordsResourceID)
	s.Require().NoError(err)
	s.Require().Equal(recordsResourceName, drift.ResourceName)

	s.Require().NoError(s.engine().Resources().SaveDrift(ctx, state.ResourceDriftState{
		ResourceID:   recordsResourceID,
		ResourceName: "renamedInDrift",
	}))
	drift, err = engineA.Resources().GetDrift(ctx, recordsResourceID)
	s.Require().NoError(err)
	s.Require().Equal(recordsResourceName, drift.ResourceName)
	s.advance(multiEngineCacheAge)

	drift, err = engineA.Resources().GetDrift(ctx, recordsResourceID)
	s.Require().NoError(err)
	s.Assert().Equal("renamedInDrift", drift.ResourceName)
}

func (s *MultiEngineSuite) Test_an_instance_removed_by_another_engine_reads_as_not_found_and_leaves_the_list() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	other := eqInstance("other-instance", "OtherInstance", nil, nil)
	s.Require().NoError(s.engine().Instances().Save(ctx, *other))
	engineA := s.engine()
	listed, err := engineA.Instances().List(ctx, state.ListInstancesParams{})
	s.Require().NoError(err)
	s.Require().Len(listed.Instances, 2)

	_, err = s.engine().Instances().Remove(ctx, recordsInstanceID)
	s.Require().NoError(err)
	s.advance(multiEngineCacheAge)

	// Listed before the removed instance is read by ID, which would drop it from the cache.
	listed, err = engineA.Instances().List(ctx, state.ListInstancesParams{})
	s.Require().NoError(err)
	s.Require().Len(listed.Instances, 1)
	s.Assert().Equal("other-instance", listed.Instances[0].InstanceID)
	_, err = engineA.Instances().Get(ctx, recordsInstanceID)
	s.Assert().True(state.IsInstanceNotFound(err), "got %v", err)
}

// Engine B's cached instance predates engine A's change, so B's write is rejected, and B
// applies its change again to the instance as A left it.
func (s *MultiEngineSuite) Test_changes_by_two_engines_to_different_resources_of_one_instance_both_survive() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	engineA, engineB := s.engine(), s.engine()
	s.requireResourceStatus(engineA, recordsResourceID, core.ResourceStatusCreated)
	s.requireResourceStatus(engineB, secondResourceID, core.ResourceStatusCreated)

	s.Require().NoError(engineA.Resources().UpdateStatus(ctx, recordsResourceID, updateFailed()))
	s.Require().NoError(engineB.Resources().UpdateStatus(ctx, secondResourceID, updateFailed()))

	fresh := s.engine()
	s.requireResourceStatus(fresh, recordsResourceID, core.ResourceStatusUpdateFailed)
	s.requireResourceStatus(fresh, secondResourceID, core.ResourceStatusUpdateFailed)
}

// Another engine changes the instance just before each of engine B's writes reaches the
// bucket, so every attempt B makes is rejected.
func (s *MultiEngineSuite) Test_a_write_rejected_on_every_attempt_reports_a_write_conflict() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	engineB := s.engine()
	s.requireResourceStatus(engineB, recordsResourceID, core.ResourceStatusCreated)

	interferer := s.engine()
	var attempts atomic.Int32
	var interfering atomic.Bool
	s.svc.SetHooks(mockservice.Hooks{
		BeforePut: func(ctx context.Context, key string, _ []byte, _ *objectstore.PutOptions) error {
			isInstance := strings.HasSuffix(key, "instances/"+recordsInstanceID+".json")
			if !isInstance || !interfering.CompareAndSwap(false, true) {
				return nil
			}
			defer interfering.Store(false)
			attempts.Add(1)
			s.advance(multiEngineCacheAge)
			return interferer.Instances().UpdateStatus(ctx, recordsInstanceID, state.InstanceStatusInfo{
				Status: core.InstanceStatusUpdating,
			})
		},
	})

	err := engineB.Resources().UpdateStatus(ctx, recordsResourceID, updateFailed())
	s.svc.SetHooks(mockservice.Hooks{})

	s.Require().ErrorIs(err, state.ErrInstanceWriteConflict)
	s.Assert().Equal(int32(3), attempts.Load())
	// The change that was never stored is not left in engine B's cache.
	s.requireResourceStatus(engineB, recordsResourceID, core.ResourceStatusCreated)
}

// Engine A resolved the name to the first instance. That instance is removed and its name
// taken by a new instance, so the name must resolve to the new one.
func (s *MultiEngineSuite) Test_a_cached_name_resolves_to_the_instance_that_took_it_over() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	engineA := s.engine()
	id, err := engineA.Instances().LookupIDByName(ctx, recordsInstanceName)
	s.Require().NoError(err)
	s.Require().Equal(recordsInstanceID, id)

	_, err = s.engine().Instances().Remove(ctx, recordsInstanceID)
	s.Require().NoError(err)
	renewed := eqInstance("renewed-instance", recordsInstanceName, nil, nil)
	_, err = s.engine().Instances().InitialiseAndClaim(ctx, *renewed, core.InstanceStatusPreparing)
	s.Require().NoError(err)
	s.advance(multiEngineCacheAge)

	id, err = engineA.Instances().LookupIDByName(ctx, recordsInstanceName)
	s.Require().NoError(err)
	s.Assert().Equal("renewed-instance", id)
}

func (s *MultiEngineSuite) Test_an_engine_without_revalidation_never_checks_the_bucket() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	var heads atomic.Int32
	s.svc.SetHooks(mockservice.Hooks{
		BeforeHead: func(context.Context, string) error {
			heads.Add(1)
			return nil
		},
	})
	alone := s.engine(objectstore.WithoutCacheRevalidation())
	shared := s.engine()
	s.requireResourceStatus(alone, recordsResourceID, core.ResourceStatusCreated)
	s.requireResourceStatus(shared, recordsResourceID, core.ResourceStatusCreated)

	s.advance(multiEngineCacheAge)
	s.requireResourceStatus(alone, recordsResourceID, core.ResourceStatusCreated)
	s.Assert().Zero(heads.Load())

	s.requireResourceStatus(shared, recordsResourceID, core.ResourceStatusCreated)
	s.svc.SetHooks(mockservice.Hooks{})
	s.Assert().NotZero(heads.Load())
}

// Engine A's removal is held up just before it deletes the instance object. Meanwhile another
// engine finishes that removal and creates a new instance with the same ID, which A's delete
// must leave alone.
func (s *MultiEngineSuite) Test_a_removal_does_not_delete_an_instance_created_under_its_id_meanwhile() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, twoResourceInstance()))
	engineA, engineB := s.engine(), s.engine()

	var replaced atomic.Bool
	s.svc.SetHooks(mockservice.Hooks{
		BeforeDelete: func(ctx context.Context, key string) error {
			isInstance := strings.HasSuffix(key, "instances/"+recordsInstanceID+".json")
			if !isInstance || !replaced.CompareAndSwap(false, true) {
				return nil
			}
			if _, err := engineB.Instances().Remove(ctx, recordsInstanceID); err != nil {
				return err
			}
			recreated := eqInstance(recordsInstanceID, "RecreatedInstance", nil, nil)
			_, err := engineB.Instances().InitialiseAndClaim(ctx, *recreated, core.InstanceStatusPreparing)
			return err
		},
	})

	_, err := engineA.Instances().Remove(ctx, recordsInstanceID)
	s.svc.SetHooks(mockservice.Hooks{})
	s.Require().NoError(err)

	instance, err := s.engine().Instances().Get(ctx, recordsInstanceID)
	s.Require().NoError(err)
	s.Assert().Equal("RecreatedInstance", instance.InstanceName)
}

func (s *MultiEngineSuite) engine(opts ...objectstore.Option) *objectstore.StateContainer {
	opts = append(
		[]objectstore.Option{
			objectstore.WithClock(s.clock),
			objectstore.WithCacheRevalidation(multiEngineCacheAge),
		},
		opts...,
	)
	container, err := objectstore.LoadStateContainer(
		context.Background(), s.svc, testPrefix, core.NewNopLogger(), opts...,
	)
	s.Require().NoError(err)
	return container
}

func (s *MultiEngineSuite) advance(by time.Duration) {
	s.clock.Timestamp += int64(by.Seconds())
}

func (s *MultiEngineSuite) requireResourceStatus(
	engine *objectstore.StateContainer,
	resourceID string,
	expected core.ResourceStatus,
) {
	resource, err := engine.Resources().Get(context.Background(), resourceID)
	s.Require().NoError(err)
	s.Require().Equal(expected, resource.Status, "status of %s", resourceID)
}

func updateFailed() state.ResourceStatusInfo {
	return state.ResourceStatusInfo{
		Status:        core.ResourceStatusUpdateFailed,
		PreciseStatus: core.PreciseResourceStatusUpdateFailed,
	}
}

func twoResourceInstance() state.InstanceState {
	instance := eqInstance(
		recordsInstanceID, recordsInstanceName,
		[]*state.ResourceState{
			eqResource(recordsInstanceID, recordsResourceID, recordsResourceName, "aws/dynamodb/table"),
			eqResource(recordsInstanceID, secondResourceID, "secondQueue", "aws/sqs/queue"),
		},
		map[string]*state.InstanceState{},
	)
	return *instance
}

func TestMultiEngineSuite(t *testing.T) {
	suite.Run(t, new(MultiEngineSuite))
}
