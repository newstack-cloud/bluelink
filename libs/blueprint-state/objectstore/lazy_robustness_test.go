package objectstore_test

import (
	"context"
	"fmt"
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

// What a lazily loading engine does after a write sequence was cut short, or when another
// engine or goroutine is working on the same instance.
type LazyRobustnessSuite struct {
	objectstoreSuite
}

func (s *LazyRobustnessSuite) Test_an_instance_whose_removal_was_interrupted_cannot_be_claimed() {
	s.interruptRemoval()

	_, err := s.newContainer().Instances().ClaimForDeployment(
		context.Background(), recordsInstanceID, 0, core.InstanceStatusUpdating,
	)
	s.Assert().True(state.IsInstanceNotFound(err), "got %v", err)
}

func (s *LazyRobustnessSuite) Test_the_id_of_an_interrupted_removal_can_be_initialised_again() {
	s.interruptRemoval()

	_, err := s.newContainer().Instances().InitialiseAndClaim(
		context.Background(), recordsInstance(), core.InstanceStatusPreparing,
	)
	s.Require().NoError(err)

	instance, err := s.newContainer().Instances().Get(context.Background(), recordsInstanceID)
	s.Require().NoError(err)
	s.Assert().Equal(core.InstanceStatusPreparing, instance.Status)
}

// The removal stops while the name record still names the instance being removed.
func (s *LazyRobustnessSuite) Test_the_name_of_an_interrupted_removal_can_be_taken_by_a_new_instance() {
	s.interruptRemoval("instances_by_name/" + recordsInstanceName + ".json")
	ctx := context.Background()

	renewed := recordsInstance()
	renewed.InstanceID = "renewed-instance"
	_, err := s.newContainer().Instances().InitialiseAndClaim(ctx, renewed, core.InstanceStatusPreparing)
	s.Require().NoError(err)

	id, err := s.newContainer().Instances().LookupIDByName(ctx, recordsInstanceName)
	s.Require().NoError(err)
	s.Assert().Equal("renewed-instance", id)
}

// Taking the name finishes the old removal, so none of what it left behind remains.
func (s *LazyRobustnessSuite) Test_taking_the_name_of_an_interrupted_removal_finishes_it() {
	s.interruptRemoval(firstRecordKeySuffix)
	ctx := context.Background()

	renewed := eqInstance("renewed-instance", recordsInstanceName, nil, nil)
	_, err := s.newContainer().Instances().InitialiseAndClaim(ctx, *renewed, core.InstanceStatusPreparing)
	s.Require().NoError(err)

	s.assertRemovalFinished()
	s.Assert().Equal(
		[]string{testPrefix + "instances/renewed-instance.json"},
		s.objectKeys(testPrefix+"instances/"),
	)
}

func (s *LazyRobustnessSuite) Test_initialising_the_id_of_an_interrupted_removal_finishes_it() {
	s.interruptRemoval(firstRecordKeySuffix)

	reused := eqInstance(recordsInstanceID, "ReusedInstance", nil, nil)
	_, err := s.newContainer().Instances().InitialiseAndClaim(
		context.Background(), *reused, core.InstanceStatusPreparing,
	)
	s.Require().NoError(err)

	s.assertRemovalFinished()
}

func (s *LazyRobustnessSuite) Test_saving_the_id_of_an_interrupted_removal_finishes_it() {
	s.interruptRemoval(firstRecordKeySuffix)

	reused := eqInstance(recordsInstanceID, "ReusedInstance", nil, nil)
	s.Require().NoError(s.newContainer().Instances().Save(context.Background(), *reused))

	s.assertRemovalFinished()
}

// The removal stops after the name record is deleted, so the name is free for a new instance
// before the old instance's removal is finished.
func (s *LazyRobustnessSuite) Test_finishing_an_interrupted_removal_keeps_the_name_record_of_its_successor() {
	s.interruptRemoval()
	ctx := context.Background()

	renewed := eqInstance("renewed-instance", recordsInstanceName, nil, nil)
	_, err := s.newContainer().Instances().InitialiseAndClaim(ctx, *renewed, core.InstanceStatusPreparing)
	s.Require().NoError(err)

	_, err = s.newContainer().Instances().Remove(ctx, recordsInstanceID)
	s.Require().NoError(err)

	id, err := s.newContainer().Instances().LookupIDByName(ctx, recordsInstanceName)
	s.Require().NoError(err)
	s.Assert().Equal("renewed-instance", id)
}

// A process that died between reserving a name and writing its instance leaves a record
// naming an instance that does not exist. Once old enough, it is taken over.
func (s *LazyRobustnessSuite) Test_an_abandoned_name_reservation_is_taken_over() {
	s.reserveNameForUnwrittenInstance()
	ctx := context.Background()
	later := s.containerAt(reservationTimestamp + int64((11 * time.Minute).Seconds()))

	_, err := later.Instances().InitialiseAndClaim(ctx, recordsInstance(), core.InstanceStatusPreparing)
	s.Require().NoError(err)

	id, err := s.newContainer().Instances().LookupIDByName(ctx, recordsInstanceName)
	s.Require().NoError(err)
	s.Assert().Equal(recordsInstanceID, id)
}

// A recent reservation for an instance not yet written may be another engine between the
// two writes, so it is left alone.
func (s *LazyRobustnessSuite) Test_a_recent_name_reservation_cannot_be_taken_over() {
	s.reserveNameForUnwrittenInstance()
	soon := s.containerAt(reservationTimestamp + int64(time.Minute.Seconds()))

	_, err := soon.Instances().InitialiseAndClaim(
		context.Background(), recordsInstance(), core.InstanceStatusPreparing,
	)
	s.Require().ErrorIs(err, state.ErrInstanceAlreadyExists)

	data, _, err := s.svc.Get(context.Background(), testPrefix+"instances_by_name/"+recordsInstanceName+".json")
	s.Require().NoError(err)
	s.Assert().Contains(string(data), "crashed-instance")
}

// The name is reserved before the instance object is written, so a name another instance
// holds must stop the create before any object exists.
func (s *LazyRobustnessSuite) Test_initialising_under_a_taken_name_does_not_wirte_an_instance_object() {
	ctx := context.Background()
	s.Require().NoError(s.newContainer().Instances().Save(ctx, recordsInstance()))

	other := recordsInstance()
	other.InstanceID = "other-instance"
	_, err := s.newContainer().Instances().InitialiseAndClaim(ctx, other, core.InstanceStatusPreparing)
	s.Require().ErrorIs(err, state.ErrInstanceAlreadyExists)

	s.Assert().Equal(
		[]string{testPrefix + "instances/" + recordsInstanceID + ".json"},
		s.objectKeys(testPrefix+"instances/"),
	)
}

// The failed save leaves the resource in the engine's cache but neither it nor its record
// in storage. A retry from the same engine must still write the record.
func (s *LazyRobustnessSuite) Test_a_retried_save_after_a_failed_write_saves_the_record() {
	ctx := context.Background()
	s.Require().NoError(s.newContainer().Instances().Save(ctx, recordsInstance()))
	container := s.newContainer()
	resource := *eqResource(recordsInstanceID, "retried-resource", "retriedQueue", "aws/sqs/queue")

	s.svc.SetHooks(mockservice.Hooks{
		BeforePut: func(_ context.Context, key string, _ []byte, _ *objectstore.PutOptions) error {
			if strings.HasSuffix(key, "resources_by_id/retried-resource.json") {
				return errInjected
			}
			return nil
		},
	})
	s.Require().ErrorIs(container.Resources().Save(ctx, resource), errInjected)
	s.svc.SetHooks(mockservice.Hooks{})

	s.Require().NoError(container.Resources().Save(ctx, resource))
	_, err := s.newContainer().Resources().Get(ctx, "retried-resource")
	s.Assert().NoError(err)
}

// A load that read an instance before another goroutine cached and changed it must not
// replace the newer cached copy when it finishes. The slow load is held up loading the
// instance's child, after it has read the instance itself.
func (s *LazyRobustnessSuite) Test_a_slow_load_does_not_overwrite_a_newer_cached_instance() {
	ctx := context.Background()
	withChild := recordsInstance()
	withChild.ChildBlueprints = map[string]*state.InstanceState{
		"child": eqInstance("records-child", "RecordsChild", nil, nil),
	}
	s.Require().NoError(s.newContainer().Instances().Save(ctx, withChild))
	container := s.newContainer()

	slowLoadStarted := make(chan struct{})
	releaseSlowLoad := make(chan struct{})
	// Only the first read is held back.
	// This is used instead of sync.Once as it would also block every later
	// read until the first returned.
	var firstRead atomic.Bool
	s.svc.SetHooks(mockservice.Hooks{
		BeforeGet: func(_ context.Context, key string) error {
			isChild := strings.HasSuffix(key, "instances/records-child.json")
			if isChild && firstRead.CompareAndSwap(false, true) {
				close(slowLoadStarted)
				<-releaseSlowLoad
			}
			return nil
		},
	})

	slowLoad := make(chan error)
	go func() {
		_, err := container.Instances().Get(ctx, recordsInstanceID)
		slowLoad <- err
	}()
	<-slowLoadStarted

	added := *eqResource(recordsInstanceID, "added-resource", "addedQueue", "aws/sqs/queue")
	s.Require().NoError(container.Resources().Save(ctx, added))
	close(releaseSlowLoad)
	s.Require().NoError(<-slowLoad)
	s.svc.SetHooks(mockservice.Hooks{})

	_, err := container.Resources().Get(ctx, "added-resource")
	s.Assert().NoError(err)
}

const reservationTimestamp = int64(1760000000)

// The removal deletes ID records first, so stopping at this one leaves every record, the
// name record and the marked instance object behind.
const firstRecordKeySuffix = "resources_by_id/" + recordsResourceID + ".json"

// Nothing the removed records-instance left behind is still in the bucket, no ID record or
// name record naming it, and no object for it unless a new instance reused its ID.
func (s *LazyRobustnessSuite) assertRemovalFinished() {
	ctx := context.Background()
	for _, prefix := range []string{"resources_by_id/", "links_by_id/"} {
		s.Assert().Empty(s.objectKeys(testPrefix+prefix), prefix)
	}
	if data, _, err := s.svc.Get(ctx, testPrefix+"instances_by_name/"+recordsInstanceName+".json"); err == nil {
		s.Assert().NotContains(string(data), `"id":"`+recordsInstanceID+`"`)
	}
	if data, _, err := s.svc.Get(ctx, testPrefix+"instances/"+recordsInstanceID+".json"); err == nil {
		s.Assert().NotContains(string(data), `"removing":true`)
	}
}

// What a create leaves behind when its process dies after reserving the name.
func (s *LazyRobustnessSuite) reserveNameForUnwrittenInstance() {
	record := fmt.Sprintf(
		`{"id":"crashed-instance","name":%q,"reservedAt":%d}`,
		recordsInstanceName, reservationTimestamp,
	)
	_, err := s.svc.Put(
		context.Background(),
		testPrefix+"instances_by_name/"+recordsInstanceName+".json",
		[]byte(record),
		nil,
	)
	s.Require().NoError(err)
}

func (s *LazyRobustnessSuite) containerAt(unixTime int64) *objectstore.StateContainer {
	container, err := objectstore.LoadStateContainer(
		context.Background(),
		s.svc,
		testPrefix,
		core.NewNopLogger(),
		objectstore.WithClock(&internal.MockClock{Timestamp: unixTime}),
	)
	s.Require().NoError(err)
	return container
}

// Stops the removal at the last step, deleting the instance object, unless an earlier key is
// given to fail on.
func (s *LazyRobustnessSuite) interruptRemoval(failingKeySuffix ...string) {
	failOn := "instances/" + recordsInstanceID + ".json"
	if len(failingKeySuffix) > 0 {
		failOn = failingKeySuffix[0]
	}
	ctx := context.Background()
	s.Require().NoError(s.newContainer().Instances().Save(ctx, recordsInstance()))
	s.svc.SetHooks(mockservice.Hooks{
		BeforeDelete: func(_ context.Context, key string) error {
			if strings.HasSuffix(key, failOn) {
				return errInjected
			}
			return nil
		},
	})
	_, err := s.newContainer().Instances().Remove(ctx, recordsInstanceID)
	s.Require().ErrorIs(err, errInjected)
	s.svc.SetHooks(mockservice.Hooks{})
}

func TestLazyRobustnessSuite(t *testing.T) {
	suite.Run(t, new(LazyRobustnessSuite))
}
