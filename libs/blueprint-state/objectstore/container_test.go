package objectstore_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// Compile-time assertion that *StateContainer satisfies the blueprint
// state.Container interface — catches signature drift in per-category
// sub-containers without needing a runtime test.
var _ state.Container = (*objectstore.StateContainer)(nil)

const testPrefix = "bluelink-state/"

// Shared setup for the suites below with one mock bucket per test,
// and a way to open an engine's view of it.
type objectstoreSuite struct {
	suite.Suite
	svc *mockservice.Service
}

func (s *objectstoreSuite) SetupTest() {
	s.svc = mockservice.New()
}

// Opens a state container over the test's bucket.
//
// Every call is a separate engine instance with its own in-memory state, which is what
// lets a test tell what reached storage from what one engine happens to be holding.
func (s *objectstoreSuite) newContainer() *objectstore.StateContainer {
	container, err := objectstore.LoadStateContainer(
		context.Background(),
		s.svc,
		testPrefix,
		core.NewNopLogger(),
	)
	s.Require().NoError(err)

	return container
}

func (s *objectstoreSuite) objectKeys(prefix string) []string {
	objects, err := s.svc.List(context.Background(), prefix)
	s.Require().NoError(err)

	keys := make([]string, len(objects))
	for i, object := range objects {
		keys[i] = object.Key
	}

	return keys
}

// Loading a container, and the claim and name-record paths that only objectstore has.
type ContainerLoadSuite struct {
	objectstoreSuite
}

func (s *ContainerLoadSuite) Test_initialises_cleanly_on_an_empty_bucket() {
	container := s.newContainer()

	_, err := container.Instances().Get(context.Background(), "missing-instance")
	s.Require().Error(err)
	stateErr, ok := err.(*state.Error)
	s.Require().True(ok, "expected *state.Error, got %T", err)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *ContainerLoadSuite) Test_initialise_and_claim_roundtrips_through_the_service() {
	const instanceID = "round-trip-instance"
	const instanceName = "RoundTripInstance"

	version, err := s.newContainer().Instances().InitialiseAndClaim(
		context.Background(),
		state.InstanceState{
			InstanceID:   instanceID,
			InstanceName: instanceName,
		},
		core.InstanceStatusPreparing,
	)
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), version)

	// A fresh container re-loading from the same bucket must find the
	// persisted record — proves the round-trip through Service/Storage.
	retrieved, err := s.newContainer().Instances().Get(context.Background(), instanceID)
	s.Require().NoError(err)
	s.Assert().Equal(instanceName, retrieved.InstanceName)
	s.Assert().Equal(core.InstanceStatusPreparing, retrieved.Status)
	s.Assert().Equal(int64(1), retrieved.Version)
}

func (s *ContainerLoadSuite) Test_lazy_loads_an_instance_by_name_via_its_name_record() {
	const instanceID = "lazy-by-name-instance"
	const instanceName = "LazyByNameInstance"

	_, err := s.newContainer().Instances().InitialiseAndClaim(
		context.Background(),
		state.InstanceState{InstanceID: instanceID, InstanceName: instanceName},
		core.InstanceStatusPreparing,
	)
	s.Require().NoError(err)

	// A fresh container starts with an empty in-memory State. Resolving by
	// name must therefore go through the ServiceLoader — proves both the
	// name record was written by InitialiseAndClaim and the lazy loader
	// can resolve it.
	gotID, err := s.newContainer().Instances().LookupIDByName(context.Background(), instanceName)
	s.Require().NoError(err)
	s.Assert().Equal(instanceID, gotID)
}

func (s *ContainerLoadSuite) Test_a_stale_claim_for_deployment_loses() {
	const instanceID = "claim-via-container"
	container := s.newContainer()
	ctx := context.Background()

	// Bootstrap at V1 via InitialiseAndClaim.
	_, err := container.Instances().InitialiseAndClaim(
		ctx,
		state.InstanceState{InstanceID: instanceID, InstanceName: instanceID},
		core.InstanceStatusPreparing,
	)
	s.Require().NoError(err)

	// Follow-up ClaimForDeployment at expected=1 should succeed and bump to V2.
	newVersion, err := container.Instances().ClaimForDeployment(
		ctx,
		instanceID,
		1,
		core.InstanceStatusDeploying,
	)
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), newVersion)

	_, err = container.Instances().ClaimForDeployment(
		ctx,
		instanceID,
		1,
		core.InstanceStatusDeploying,
	)
	s.Require().Error(err)
	s.Assert().True(errors.Is(err, state.ErrVersionConflict))
}

func TestContainerLoadSuite(t *testing.T) {
	suite.Run(t, new(ContainerLoadSuite))
}

// The audit trail is served from a per-entity object layout, so Query enumerates
// under the key prefix. That path is not shared with the chunked backends.
type AuditSuite struct {
	objectstoreSuite
}

func (s *AuditSuite) Test_round_trips_the_trail_through_the_service() {
	auditStore := s.newContainer().Audit()
	ctx := context.Background()
	s.saveRecords(
		auditStore,
		auditRecordFixture("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeFailure),
		auditRecordFixture("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		auditRecordFixture("record-2", 2000, "instance.update", "user-b", manage.OutcomeSuccess),
	)

	retrieved, err := auditStore.Get(ctx, "record-2")
	s.Require().NoError(err)
	s.Assert().Equal("instance.update", retrieved.Action)

	records, err := auditStore.Query(ctx, &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-1", "record-2", "record-3"}, auditRecordIDs(records))

	records, err = auditStore.Query(ctx, &manage.AuditRecordQuery{PrincipalSubject: "user-b"})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-2"}, auditRecordIDs(records))
}

// A record whose timestamp equals the threshold is removed, matching the other
// backends, and the object behind it is gone rather than merely dropped from the
// in-memory lookup.
func (s *AuditSuite) Test_cleans_up_records_at_or_below_the_threshold() {
	auditStore := s.newContainer().Audit()
	ctx := context.Background()
	s.saveRecords(
		auditStore,
		auditRecordFixture("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		auditRecordFixture("record-2", 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		auditRecordFixture("record-3", 2001, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	removed, err := auditStore.Cleanup(ctx, time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), removed)

	records, err := auditStore.Query(ctx, &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-3"}, auditRecordIDs(records))

	_, err = auditStore.Get(ctx, "record-1")
	s.Require().Error(err)
	notFoundErr := &manage.AuditRecordNotFound{}
	s.Assert().ErrorAs(err, &notFoundErr)
}

func (s *AuditSuite) saveRecords(auditStore manage.Audit, records ...*manage.AuditRecord) {
	for _, record := range records {
		s.Require().NoError(auditStore.Save(context.Background(), record))
	}
}

func TestAuditSuite(t *testing.T) {
	suite.Run(t, new(AuditSuite))
}

// Cleanup has to remove the objects, not report that it did.
//
// These categories are laid out one object per record here, and their cleanup used to
// run the chunked rewrite regardless: it found no chunk files to filter, reset a chunk
// set that was never written, and returned a count taken from the in-memory lookup. It
// reported records removed while leaving every object in the bucket.
type PerEntityCleanupSuite struct {
	objectstoreSuite
}

func (s *PerEntityCleanupSuite) Test_removes_the_objects_of_every_per_entity_category() {
	container := s.newContainer()
	ctx := context.Background()

	const past = int64(1000)
	threshold := time.Unix(500000, 0)

	s.Require().NoError(container.Validation().Save(ctx, &manage.BlueprintValidation{
		ID: "11111111-1111-1111-1111-111111111111", Status: "COMPLETE", Created: past,
	}))
	s.Require().NoError(container.Changesets().Save(ctx, &manage.Changeset{
		ID: "22222222-2222-2222-2222-222222222222", Status: "COMPLETE", Created: past,
	}))
	s.Require().NoError(container.ReconciliationResults().Save(ctx, &manage.ReconciliationResult{
		ID: "33333333-3333-3333-3333-333333333333", Created: past,
	}))
	s.Require().NoError(container.Audit().Save(ctx, &manage.AuditRecord{
		ID:        "44444444-4444-4444-4444-444444444444",
		Timestamp: past,
		Action:    "instance.create",
	}))

	for _, category := range []struct {
		name   string
		clean  func(context.Context, time.Time) (int64, error)
		prefix string
	}{
		{"validations", container.Validation().Cleanup, testPrefix + "validations/"},
		{"changesets", container.Changesets().Cleanup, testPrefix + "changesets/"},
		{
			"reconciliation results",
			container.ReconciliationResults().Cleanup,
			testPrefix + "reconciliation_results/",
		},
		{"audit records", container.Audit().Cleanup, testPrefix + "audit_records/"},
	} {
		removed, err := category.clean(ctx, threshold)
		s.Require().NoError(err, category.name)
		s.Assert().Equal(
			int64(1),
			removed,
			"%s: one record should have been removed",
			category.name,
		)
		s.Assert().Empty(
			s.objectKeys(category.prefix),
			"%s: the objects should be gone from the bucket",
			category.name,
		)
	}
}

func (s *PerEntityCleanupSuite) Test_reports_nothing_removed_when_all_records_are_retained() {
	container := s.newContainer()
	ctx := context.Background()

	s.Require().NoError(container.Validation().Save(ctx, &manage.BlueprintValidation{
		ID: "11111111-1111-1111-1111-111111111111", Status: "COMPLETE", Created: 900000,
	}))

	removed, err := container.Validation().Cleanup(ctx, time.Unix(500000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), removed)

	retrieved, err := container.Validation().Get(ctx, "11111111-1111-1111-1111-111111111111")
	s.Require().NoError(err, "a record within its retention period must survive")
	s.Assert().Equal(int64(900000), retrieved.Created)
}

func TestPerEntityCleanupSuite(t *testing.T) {
	suite.Run(t, new(PerEntityCleanupSuite))
}

// Events are one object per event, keyed by channel and a zero-padded timestamp.
//
// They used to be one object per channel, rewritten whole on every save, so an engine
// that had not read a channel wrote that object back containing only its own event.
// These cover what that cost: both engines' events surviving, and a channel coming back
// in the order it was produced.
type EventsSuite struct {
	objectstoreSuite
}

func (s *EventsSuite) Test_two_instances_writing_one_channel_keep_both_sets() {
	ctx := context.Background()

	first := s.newContainer()
	s.Require().NoError(
		first.Events().Save(ctx, eventFixture("aaaaaaaa-0000-0000-0000-000000000001", 1000)),
	)
	s.Require().NoError(
		first.Events().Save(ctx, eventFixture("aaaaaaaa-0000-0000-0000-000000000002", 2000)),
	)

	// A second engine that has never read this channel.
	second := s.newContainer()
	s.Require().NoError(
		second.Events().Save(ctx, eventFixture("bbbbbbbb-0000-0000-0000-000000000003", 3000)),
	)

	// Read back through a third, so neither writer's in-memory view can answer.
	third := s.newContainer()
	for _, id := range []string{
		"aaaaaaaa-0000-0000-0000-000000000001",
		"aaaaaaaa-0000-0000-0000-000000000002",
		"bbbbbbbb-0000-0000-0000-000000000003",
	} {
		retrieved, err := third.Events().Get(ctx, id)
		s.Require().NoError(err, "event %s should have survived", id)
		s.Assert().Equal(id, retrieved.ID)
	}

	// And the channel's last event is the newest across both engines, which is what a
	// caller resumes a stream from.
	lastEventID, err := third.Events().GetLastEventID(ctx, "changesets", "ch1")
	s.Require().NoError(err)
	s.Assert().Equal("bbbbbbbb-0000-0000-0000-000000000003", lastEventID)
}

// A channel has to come back in the order its events were produced, whichever engine
// wrote them, since that ordering is what a stream replays.
func (s *EventsSuite) Test_stream_replays_a_channel_in_order_across_instances() {
	ctx := context.Background()

	// Within the recently-queued window, since a stream with no starting event ID
	// replays only events from inside it. Saved out of order, by alternating engines.
	now := time.Now().Unix()
	for _, write := range []struct {
		id        string
		timestamp int64
	}{
		{"cccccccc-0000-0000-0000-000000000003", now - 1},
		{"aaaaaaaa-0000-0000-0000-000000000001", now - 3},
		{"bbbbbbbb-0000-0000-0000-000000000002", now - 2},
	} {
		event := eventFixture(write.id, write.timestamp)
		event.ChannelID = "ch2"
		s.Require().NoError(s.newContainer().Events().Save(ctx, event))
	}

	streamed := make(chan manage.Event, 10)
	errChan := make(chan error, 1)
	endChan, err := s.newContainer().Events().Stream(
		ctx,
		&manage.EventStreamParams{ChannelType: "changesets", ChannelID: "ch2"},
		streamed,
		errChan,
	)
	s.Require().NoError(err)
	defer func() { endChan <- struct{}{} }()

	received := []string{}
	for range 3 {
		select {
		case event := <-streamed:
			received = append(received, event.ID)
		case streamErr := <-errChan:
			s.Require().NoError(streamErr)
		case <-time.After(2 * time.Second):
			s.Require().Fail(
				"timed out waiting for streamed events",
				"received so far: %v",
				received,
			)
		}
	}

	s.Assert().Equal([]string{
		"aaaaaaaa-0000-0000-0000-000000000001",
		"bbbbbbbb-0000-0000-0000-000000000002",
		"cccccccc-0000-0000-0000-000000000003",
	}, received)
}

// A channel's retention is applied from the key listing, so cleanup reads no events and
// leaves the ones inside the window where a stream can still replay them.
func (s *EventsSuite) Test_cleanup_removes_only_events_past_their_retention() {
	events := s.newContainer().Events()
	ctx := context.Background()

	for _, timestamp := range []int64{1000, 2000, 3000} {
		event := eventFixture(fmt.Sprintf("eeeeeeee-0000-0000-0000-%012d", timestamp), timestamp)
		event.ChannelID = "ch3"
		s.Require().NoError(events.Save(ctx, event))
	}

	removed, err := events.Cleanup(ctx, time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), removed, "events at or below the threshold should be removed")
	s.Require().Len(s.objectKeys(testPrefix+"events/"), 1)

	survivor, err := events.Get(ctx, "eeeeeeee-0000-0000-0000-000000003000")
	s.Require().NoError(err)
	s.Assert().Equal(int64(3000), survivor.Timestamp)

	lastEventID, err := events.GetLastEventID(ctx, "changesets", "ch3")
	s.Require().NoError(err)
	s.Assert().Equal("eeeeeeee-0000-0000-0000-000000003000", lastEventID)
}

func TestEventsSuite(t *testing.T) {
	suite.Run(t, new(EventsSuite))
}

func auditRecordFixture(
	id string,
	timestamp int64,
	action string,
	principalSubject string,
	outcome string,
) *manage.AuditRecord {
	return &manage.AuditRecord{
		ID:               id,
		Timestamp:        timestamp,
		Action:           action,
		Outcome:          outcome,
		StatusCode:       200,
		Method:           "POST",
		Path:             "/deployments/instances",
		PrincipalMethod:  "apikey",
		PrincipalSubject: principalSubject,
	}
}

func auditRecordIDs(records []*manage.AuditRecord) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = record.ID
	}

	return ids
}

func eventFixture(id string, timestamp int64) *manage.Event {
	return &manage.Event{
		ID:          id,
		Type:        "resource.deploy",
		ChannelType: "changesets",
		ChannelID:   "ch1",
		Timestamp:   timestamp,
	}
}
