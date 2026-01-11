package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

const (
	nonExistentEventID        = "0196677d-d816-740c-8d99-457fee08eab1"
	cleanupThresholdTimestamp = 1743415200 // 31st Match 2025 10:00 UTC
)

type PostgresEventsTestSuite struct {
	container            *StateContainer
	connPool             *pgxpool.Pool
	saveEventFixtures    map[int]internal.SaveEventFixture
	streamEventFixtures  []internal.SaveEventFixture
	streamEventFixtures2 []internal.SaveEventFixture
	suite.Suite
}

func (s *PostgresEventsTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(
		ctx,
		connPool,
		core.NewNopLogger(),
		WithClock(
			&internal.MockClock{
				// Wednesday, 23 April 2025 13:27:36 UTC
				// Within 5 minutes of the 3 queued events in the seed
				// data for the event partition used in the stream test case.
				// See the __testdata/seed/events.json.
				Timestamp: 1745414856,
			},
		),
	)
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "events")
	saveFixtures, err := internal.SetupSaveEventFixtures(
		dirPath,
	)
	s.Require().NoError(err)
	s.saveEventFixtures = saveFixtures

	streamFixtures, err := internal.CreateEventStreamSaveFixtures(
		"changesets",
		"db58eda8-36c6-4180-a9cb-557f3392361c",
		internal.StreamFixtureEventIDs1,
	)
	s.Require().NoError(err)
	s.streamEventFixtures = streamFixtures

	streamFixtures2, err := internal.CreateEventStreamSaveFixtures(
		"changesets",
		"eabba2f8-5c74-4c51-a068-b340f718314a",
		internal.StreamFixtureEventIDs2,
	)
	s.Require().NoError(err)
	s.streamEventFixtures2 = streamFixtures2
}

func (s *PostgresEventsTestSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresEventsTestSuite) Test_saves_event_and_sends_notification() {
	fixture := s.saveEventFixtures[1]

	eventIDListener := make(chan string)
	go s.listenForEventNotification(eventIDListener, fixture.Event)

	// Sleep to ensure the listener is ready before saving the event
	// as the notification will not be received if the listener is not ready.
	time.Sleep(100 * time.Millisecond)

	events := s.container.Events()
	err := events.Save(
		context.Background(),
		fixture.Event,
	)
	s.Require().NoError(err)

	savedEvent, err := events.Get(
		context.Background(),
		fixture.Event.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedEvent)
	s.assertEventsEqual(fixture.Event, savedEvent)

	select {
	case eventID := <-eventIDListener:
		s.Assert().Equal(fixture.Event.ID, eventID)
	case <-time.After(10 * time.Second):
		s.Fail("Timeout waiting for event notification")
	}
}

func (s *PostgresEventsTestSuite) Test_stream_events() {
	expectedInitialEventIDs := []string{
		// Initial seed event 1 for the channel.
		"01966439-6832-74ba-94e3-9d8d47d98b60",
		// Initial seed event 2 for the channel.
		"0196643a-69f6-7d6d-a4c1-c6ee239851a9",
		// Initial seed event 3 for the channel.
		"0196643c-69b2-7900-bcf7-2ff34d80565e",
	}

	events := s.container.Events()
	internal.TestStreamEvents(
		s.streamEventFixtures,
		events,
		/* channelType */ "changesets",
		/* channelID */ "db58eda8-36c6-4180-a9cb-557f3392361c",
		expectedInitialEventIDs,
		&s.Suite,
	)
}

func (s *PostgresEventsTestSuite) Test_excludes_queued_events_outside_recently_queued_time_window() {
	events := s.container.Events()
	internal.TestStreamEvents(
		s.streamEventFixtures2,
		events,
		/* channelType */ "changesets",
		/* channelID */ "eabba2f8-5c74-4c51-a068-b340f718314a",
		/* expectedInitialEventIDs */ []string{},
		&s.Suite,
	)
}

func (s *PostgresEventsTestSuite) Test_streams_event_when_stream_has_ended_with_recently_queued_events() {
	// If the stream has ended and there are recently queued events,
	// these should be streamed to the client, otherwise, in cases where the caller
	// listens to the stream shortly after the stream has ended, it will miss all the events
	// but the last one used to mark the end of the stream.

	expectedInitialEventIDs := []string{
		// Initial seed event 1 for the channel.
		"01967377-9157-7b2b-ab00-03fd1cb6dc58",
		// Initial seed event 2 for the channel.
		"01967378-0257-7eed-9420-956d21abb0ce",
		// Initial seed event 3 for the channel.
		"01967378-1af3-77fc-9d23-1d791eb96b85",
	}

	events := s.container.Events()
	internal.TestStreamEvents(
		[]internal.SaveEventFixture{},
		events,
		/* channelType */ "changesets",
		/* channelID */ "57ea9d45-9f27-4af5-af29-a9e7099b7333",
		expectedInitialEventIDs,
		&s.Suite,
	)
}

func (s *PostgresEventsTestSuite) Test_ends_stream_when_last_saved_event_is_marked_as_end_of_stream() {
	// This assertion is for the case where there have been no recently
	// queued events.
	events := s.container.Events()
	internal.TestEndOfEventStream(
		events,
		/* channelType */ "changesets",
		/* channelID */ "6900db60-775b-4779-8064-d3accb092b85",
		&s.Suite,
	)
}

func (s *PostgresEventsTestSuite) Test_returns_event_not_found_error_for_missing_event() {
	_, err := s.container.Events().Get(
		context.Background(),
		nonExistentEventID,
	)
	s.Require().Error(err)

	notFoundErr, isNotFoundErr := err.(*manage.EventNotFound)
	s.Require().True(isNotFoundErr)
	s.Assert().Equal(
		"event with ID 0196677d-d816-740c-8d99-457fee08eab1 not found",
		notFoundErr.Error(),
	)
}

func (s *PostgresEventsTestSuite) Test_cleans_up_old_events() {
	_, err := s.container.Events().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	for _, id := range eventsShouldBeCleanedUp {
		_, err := s.container.Events().Get(
			context.Background(),
			id,
		)
		s.Require().Error(err)

		notFoundErr, isNotFoundErr := err.(*manage.EventNotFound)
		s.Require().True(isNotFoundErr)
		s.Assert().Equal(
			fmt.Sprintf("event with ID %s not found", id),
			notFoundErr.Error(),
		)
	}

	for _, id := range eventsShouldNotBeCleanedUp {
		event, err := s.container.Events().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, event.ID)
	}
}

func (s *PostgresEventsTestSuite) Test_get_last_event_id_returns_last_event_in_channel() {
	events := s.container.Events()
	lastEventID, err := events.GetLastEventID(
		context.Background(),
		"changesets",
		"db58eda8-36c6-4180-a9cb-557f3392361c",
	)
	s.Require().NoError(err)
	// The last event ID in the seed data for this channel
	s.Assert().Equal("0196643c-69b2-7900-bcf7-2ff34d80565e", lastEventID)
}

func (s *PostgresEventsTestSuite) Test_get_last_event_id_returns_empty_for_non_existent_channel() {
	events := s.container.Events()
	lastEventID, err := events.GetLastEventID(
		context.Background(),
		"changesets",
		"non-existent-channel-id",
	)
	s.Require().NoError(err)
	s.Assert().Equal("", lastEventID)
}

func (s *PostgresEventsTestSuite) listenForEventNotification(
	eventIDListener chan string,
	event *manage.Event,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		12*time.Second,
	)
	defer cancel()
	conn, err := s.connPool.Acquire(ctx)
	s.Require().NoError(err)
	channel := eventsChannel(event.ChannelType, event.ChannelID)
	_, err = conn.Conn().Exec(
		ctx,
		fmt.Sprintf("LISTEN %q", channel),
	)
	s.Require().NoError(err)
	defer s.unlistenForEventNotification(ctx, conn, channel)

	receivedEvent := false
	for !receivedEvent {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			s.Fail("Failed to wait for notification", err)
			return
		}
		if notification == nil {
			continue
		}

		receivedEvent = true
		eventID := notification.Payload
		eventIDListener <- eventID
	}
}

func (s *PostgresEventsTestSuite) unlistenForEventNotification(
	ctx context.Context,
	conn *pgxpool.Conn,
	channel string,
) {
	_, err := conn.Conn().Exec(
		ctx,
		fmt.Sprintf("UNLISTEN %q", channel),
	)
	if err != nil {
		fmt.Println("Failed to unlisten for event notification", err)
	}
	conn.Release()
}

func (s *PostgresEventsTestSuite) assertEventsEqual(
	expected *manage.Event,
	actual manage.Event,
) {
	s.Assert().Equal(expected.ID, actual.ID)
	s.Assert().Equal(expected.Type, actual.Type)
	s.Assert().Equal(expected.ChannelType, actual.ChannelType)
	s.Assert().Equal(expected.ChannelID, actual.ChannelID)

	if expected.Data != "" {
		var target map[string]any
		err := json.Unmarshal([]byte(expected.Data), &target)
		s.Require().NoError(err)

		var actualData map[string]any
		err = json.Unmarshal([]byte(actual.Data), &actualData)
		s.Require().NoError(err)

		s.Assert().Equal(target, actualData)
	}
	s.Assert().Equal(expected.Timestamp, actual.Timestamp)
}

// Seed events that should be cleaned up.
var eventsShouldBeCleanedUp = []string{
	"0196678c-ae43-7b53-9796-30e84ba07b99",
	"01966793-844a-7a2b-b278-48838bab3835",
	"01966794-8352-767e-a9d7-0ac6275af2e2",
	"01966794-f1e8-7a14-9893-335ca16be0d5",
	"01966795-8929-7f6d-989c-0403037d8131",
}

// Seed events that should not be cleaned up.
// This must not include the IDs of any dynamically generated events
// in the test runs.
var eventsShouldNotBeCleanedUp = []string{
	"01966439-6832-74ba-94e3-9d8d47d98b60",
	"01966439-ff85-760e-9f02-f3572e08a7c2",
	"0196643a-69f6-7d6d-a4c1-c6ee239851a9",
	"0196643c-69b2-7900-bcf7-2ff34d80565e",
}

func TestPostgresEventsTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresEventsTestSuite))
}
