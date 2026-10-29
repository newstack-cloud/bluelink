package objectstore_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// Housekeeping only judges records older than an hour, so tests move the clock past it.
const sweepGracePeriod = time.Hour

// Leftovers are made the way they arise, by cutting a write short,
// and are only judged once the bucket's clock has moved past the grace period.
type LeftoverRecordsSuite struct {
	objectstoreSuite
	clock *internal.MockClock
}

func (s *LeftoverRecordsSuite) SetupTest() {
	s.objectstoreSuite.SetupTest()
	s.clock = &internal.MockClock{Timestamp: reservationTimestamp}
	s.svc.SetClock(s.clock)
}

func (s *LeftoverRecordsSuite) Test_a_live_instance_does_not_lose_anything() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, recordsInstance()))
	s.saveDrift(ctx)
	before := s.objectKeys(testPrefix)

	deleted := s.sweepAfterGracePeriod()

	s.Assert().Zero(deleted)
	s.Assert().Equal(before, s.objectKeys(testPrefix))
}

func (s *LeftoverRecordsSuite) Test_the_record_of_a_resource_removed_before_its_record_was_deleted_is_swept() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, recordsInstance()))
	s.failDeletesOf("resources_by_id/" + recordsResourceID + ".json")
	_, err := s.engine().Resources().Remove(ctx, recordsResourceID)
	s.Require().ErrorIs(err, errInjected)
	s.svc.SetHooks(mockservice.Hooks{})

	deleted := s.sweepAfterGracePeriod()

	s.Assert().Equal(int64(1), deleted)
	s.Assert().Empty(s.objectKeys(testPrefix + "resources_by_id/"))
	s.Assert().NotEmpty(s.objectKeys(testPrefix + "links_by_id/"))
}

// A create writes the instance without resources or links, then their records, then the whole
// instance. Cut short before the last write, the records name an instance that never got them.
func (s *LeftoverRecordsSuite) Test_the_records_of_a_create_cut_short_before_its_instance_held_them_are_swept() {
	ctx := context.Background()
	instanceKey := testPrefix + "instances/" + recordsInstanceID + ".json"
	writes := 0
	s.svc.SetHooks(mockservice.Hooks{
		BeforePut: func(_ context.Context, key string, _ []byte, _ *objectstore.PutOptions) error {
			if key != instanceKey {
				return nil
			}
			writes += 1
			if writes == 2 {
				return errInjected
			}
			return nil
		},
	})
	s.Require().ErrorIs(s.engine().Instances().Save(ctx, recordsInstance()), errInjected)
	s.svc.SetHooks(mockservice.Hooks{})
	s.Require().NotEmpty(s.objectKeys(testPrefix + "resources_by_id/"))

	deleted := s.sweepAfterGracePeriod()

	s.Assert().Equal(int64(2), deleted)
	s.Assert().Empty(s.objectKeys(testPrefix + "resources_by_id/"))
	s.Assert().Empty(s.objectKeys(testPrefix + "links_by_id/"))
	s.Assert().NotEmpty(s.objectKeys(testPrefix+"instances_by_name/"), "the name of a live instance is kept")
}

func (s *LeftoverRecordsSuite) Test_an_instance_whose_removal_was_cut_short_is_finished() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, recordsInstance()))
	s.saveDrift(ctx)
	s.failDeletesOf("instances/" + recordsInstanceID + ".json")
	_, err := s.engine().Instances().Remove(ctx, recordsInstanceID)
	s.Require().ErrorIs(err, errInjected)
	s.svc.SetHooks(mockservice.Hooks{})

	deleted := s.sweepAfterGracePeriod()

	s.Assert().Positive(deleted)
	s.Assert().Empty(s.objectKeys(testPrefix))
}

func (s *LeftoverRecordsSuite) Test_drift_of_a_resource_that_no_longer_exists_is_swept() {
	ctx := context.Background()
	s.Require().NoError(s.engine().Instances().Save(ctx, recordsInstance()))
	s.saveDrift(ctx)
	s.putObject("resource_drift/gone-resource.json", state.ResourceDriftState{ResourceID: "gone-resource"})

	deleted := s.sweepAfterGracePeriod()

	s.Assert().Equal(int64(1), deleted)
	s.Assert().Equal(
		[]string{testPrefix + "resource_drift/" + recordsResourceID + ".json"},
		s.objectKeys(testPrefix+"resource_drift/"),
	)
}

func (s *LeftoverRecordsSuite) Test_an_abandoned_name_reservation_is_swept() {
	s.putObject("instances_by_name/Abandoned.json", statestore.InstanceNameRecord{
		ID:         "never-written",
		Name:       "Abandoned",
		ReservedAt: s.clock.Timestamp,
	})

	deleted := s.sweepAfterGracePeriod()

	s.Assert().Equal(int64(1), deleted)
	s.Assert().Empty(s.objectKeys(testPrefix))
}

// The bucket's write time can pass the grace period while the reservation itself is still in
// its abandonment window, which the name record has its own timestamp for.
func (s *LeftoverRecordsSuite) Test_a_reservation_still_inside_its_abandonment_window_is_kept() {
	s.putObject("instances_by_name/Reserving.json", statestore.InstanceNameRecord{
		ID:         "being-written",
		Name:       "Reserving",
		ReservedAt: s.clock.Timestamp + int64(sweepGracePeriod.Seconds()),
	})

	s.Assert().Zero(s.sweepAfterGracePeriod())
	s.Assert().NotEmpty(s.objectKeys(testPrefix + "instances_by_name/"))
}

func (s *LeftoverRecordsSuite) Test_a_leftover_written_within_the_grace_period_is_kept() {
	s.putObject("resource_drift/gone-resource.json", state.ResourceDriftState{ResourceID: "gone-resource"})
	s.putObject("resources_by_id/gone-resource.json", statestore.EntityRecord{InstanceID: "never-written"})
	s.clock.Timestamp += int64(sweepGracePeriod.Seconds()) / 2

	deleted, err := s.engine().Housekeeping().Run(context.Background())
	s.Require().NoError(err)

	s.Assert().Zero(deleted)
	s.Assert().Len(s.objectKeys(testPrefix), 2)
}

// A record rewritten after it was listed is too recent to judge, whatever its listed age.
func (s *LeftoverRecordsSuite) Test_a_record_rewritten_since_it_was_listed_is_kept() {
	recordKey := testPrefix + "resources_by_id/gone-resource.json"
	s.putObject("resources_by_id/gone-resource.json", statestore.EntityRecord{InstanceID: "never-written"})
	s.clock.Timestamp += int64(sweepGracePeriod.Seconds()) + 1
	rewritten := false
	s.svc.SetHooks(mockservice.Hooks{
		BeforeGet: func(ctx context.Context, key string) error {
			if key == recordKey && !rewritten {
				rewritten = true
				_, err := s.svc.Put(ctx, key, []byte(`{"instanceId":"never-written"}`), nil)
				return err
			}
			return nil
		},
	})

	deleted, err := s.engine().Housekeeping().Run(context.Background())
	s.svc.SetHooks(mockservice.Hooks{})
	s.Require().NoError(err)

	s.Assert().True(rewritten)
	s.Assert().Zero(deleted)
	s.Assert().Equal([]string{recordKey}, s.objectKeys(testPrefix))
}

// The delete is conditional on the version judged, so a rewrite that lands between judging and
// deleting survives.
func (s *LeftoverRecordsSuite) Test_a_record_rewritten_after_it_was_judged_is_kept() {
	recordKey := testPrefix + "resources_by_id/gone-resource.json"
	s.putObject("resources_by_id/gone-resource.json", statestore.EntityRecord{InstanceID: "never-written"})
	s.clock.Timestamp += int64(sweepGracePeriod.Seconds()) + 1
	rewritten := false
	s.svc.SetHooks(mockservice.Hooks{
		BeforeDelete: func(ctx context.Context, key string) error {
			if key == recordKey && !rewritten {
				rewritten = true
				_, err := s.svc.Put(ctx, key, []byte(`{"instanceId":"never-written"}`), nil)
				return err
			}
			return nil
		},
	})

	deleted, err := s.engine().Housekeeping().Run(context.Background())
	s.svc.SetHooks(mockservice.Hooks{})
	s.Require().NoError(err)

	s.Assert().True(rewritten)
	s.Assert().Zero(deleted)
	s.Assert().Equal([]string{recordKey}, s.objectKeys(testPrefix))
}

func (s *LeftoverRecordsSuite) Test_engines_sweeping_at_once_delete_each_leftover_once() {
	for i := range 20 {
		key := "resources_by_id/gone-" + string(rune('a'+i)) + ".json"
		s.putObject(key, statestore.EntityRecord{InstanceID: "never-written"})
	}
	s.clock.Timestamp += int64(sweepGracePeriod.Seconds()) + 1

	var wg sync.WaitGroup
	results := make([]int64, 3)
	for i := range results {
		wg.Go(func() {
			deleted, err := s.engine().Housekeeping().Run(context.Background())
			s.Assert().NoError(err)
			results[i] = deleted
		})
	}
	wg.Wait()

	s.Assert().Equal(int64(20), results[0]+results[1]+results[2])
	s.Assert().Empty(s.objectKeys(testPrefix))
}

func (s *LeftoverRecordsSuite) engine() *objectstore.StateContainer {
	container, err := objectstore.LoadStateContainer(
		context.Background(), s.svc, testPrefix, core.NewNopLogger(), objectstore.WithClock(s.clock),
	)
	s.Require().NoError(err)
	return container
}

func (s *LeftoverRecordsSuite) sweepAfterGracePeriod() int64 {
	s.clock.Timestamp += int64(sweepGracePeriod.Seconds()) + 1
	deleted, err := s.engine().Housekeeping().Run(context.Background())
	s.Require().NoError(err)
	return deleted
}

func (s *LeftoverRecordsSuite) saveDrift(ctx context.Context) {
	timestamp := 1733145728
	s.Require().NoError(s.engine().Resources().SaveDrift(ctx, state.ResourceDriftState{
		ResourceID:   recordsResourceID,
		ResourceName: recordsResourceName,
		Timestamp:    &timestamp,
	}))
}

func (s *LeftoverRecordsSuite) failDeletesOf(keySuffix string) {
	s.svc.SetHooks(mockservice.Hooks{
		BeforeDelete: func(_ context.Context, key string) error {
			if strings.HasSuffix(key, keySuffix) {
				return errInjected
			}
			return nil
		},
	})
}

func (s *LeftoverRecordsSuite) putObject(keyWithoutPrefix string, value any) {
	data, err := json.Marshal(value)
	s.Require().NoError(err)
	_, err = s.svc.Put(context.Background(), testPrefix+keyWithoutPrefix, data, nil)
	s.Require().NoError(err)
}

func TestLeftoverRecordsSuite(t *testing.T) {
	suite.Run(t, new(LeftoverRecordsSuite))
}
