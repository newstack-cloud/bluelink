package memfile

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	existingEventID           = "01966439-6832-74ba-94e3-9d8d47d98b60"
	nonExistentEventID        = "0196677d-d816-740c-8d99-457fee08eab1"
	cleanupThresholdTimestamp = 1743415200 // 31st Match 2025 10:00 UTC
)

type MemFileStateContainerEventsTestSuite struct {
	container                   *StateContainer
	partitionSizeLimitContainer *StateContainer
	stateDir                    string
	fs                          afero.Fs
	saveEventFixtures           map[int]internal.SaveEventFixture
	streamEventFixtures         []internal.SaveEventFixture
	streamEventFixtures2        []internal.SaveEventFixture
	suite.Suite
}

func (s *MemFileStateContainerEventsTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir

	container, err := LoadStateContainer(
		stateDir,
		memoryFS,
		core.NewNopLogger(),
		WithMaxEventPartitionSize(1048576), // 1MB
		WithClock(
			&internal.MockClock{
				// Wednesday, 23 April 2025 13:27:36 UTC
				// Within 5 minutes of the 3 queued events in the seed
				// data for the event partition used in the stream test case.
				// See the __testdata/initial-state/events__changesets_*.json seed files.
				Timestamp: 1745414856,
			},
		),
	)
	s.Require().NoError(err)
	s.container = container

	// Use a low max partition file size to test reaching the partition
	// size limit.
	memoryFSForPartitionSizeLimit := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFSForPartitionSizeLimit, &s.Suite)
	partitionSizeLimitContainer, err := LoadStateContainer(
		stateDir,
		memoryFSForPartitionSizeLimit,
		core.NewNopLogger(),
		WithMaxEventPartitionSize(1024), // 1KB
		WithClock(
			&internal.MockClock{
				// Wednesday, 23 April 2025 13:27:36 UTC
				// Within 5 minutes of the 3 queued events in the seed
				// data for the event partition used in the stream test case.
				// See the __testdata/initial-state/events__changesets_*.json seed files.
				Timestamp: 1745414856,
			},
		),
	)
	s.Require().NoError(err)
	s.partitionSizeLimitContainer = partitionSizeLimitContainer

	dirPath := path.Join("__testdata", "save-input", "events")
	fixtures, err := internal.SetupSaveEventFixtures(
		dirPath,
	)
	s.Require().NoError(err)
	s.saveEventFixtures = fixtures

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

func (s *MemFileStateContainerEventsTestSuite) Test_retrieves_event() {
	events := s.container.Events()
	event, err := events.Get(
		context.Background(),
		existingEventID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(event)
	err = testhelpers.Snapshot(event)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerEventsTestSuite) Test_reports_event_not_found_for_retrieval() {
	events := s.container.Events()

	_, err := events.Get(
		context.Background(),
		nonExistentEventID,
	)
	s.Require().Error(err)
	eventNotFoundErr, isEventNotFoundErr := err.(*manage.EventNotFound)
	s.Assert().True(isEventNotFoundErr)
	s.Assert().Equal(
		fmt.Sprintf("event with ID %s not found", nonExistentEventID),
		eventNotFoundErr.Error(),
	)
}

func (s *MemFileStateContainerEventsTestSuite) Test_saves_event() {
	fixture := s.saveEventFixtures[1]
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
	s.Assert().Equal(
		fixture.Event,
		&savedEvent,
	)
	s.assertPersistedEvent(fixture.Event)
}

func (s *MemFileStateContainerEventsTestSuite) Test_fails_to_save_event_that_pushes_partition_over_size_limit() {
	// Fixture 2 contains 8KB of serialised JSON data, which is
	// larger than the 1KB partition size limit for the prepared container.
	fixture := s.saveEventFixtures[2]
	events := s.partitionSizeLimitContainer.Events()
	err := events.Save(
		context.Background(),
		fixture.Event,
	)
	s.Require().Error(err)
	memfileErr, isMemfileErr := err.(*Error)
	s.Require().True(isMemfileErr)
	s.Assert().Equal(
		ErrorReasonCodeMaxEventPartitionSizeExceeded,
		memfileErr.ReasonCode,
	)
}

func (s *MemFileStateContainerEventsTestSuite) Test_stream_events() {
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

func (s *MemFileStateContainerEventsTestSuite) Test_excludes_queued_events_outside_recently_queued_time_window() {
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

func (s *MemFileStateContainerEventsTestSuite) Test_streams_event_when_stream_has_ended_with_recently_queued_events() {
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

func (s *MemFileStateContainerEventsTestSuite) Test_ends_stream_when_last_saved_event_is_marked_as_end_of_stream() {
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

func (s *MemFileStateContainerEventsTestSuite) Test_cleans_up_old_events() {
	_, err := s.container.Events().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	assertEventsCleanedUp(
		s.container,
		&s.Suite,
	)

	// Assert that the events are cleaned up when loading a fresh
	// state container from file, ensuring that the cleanup
	// operation was persisted correctly.
	s.assertEventCleanupPersisted()
}

func (s *MemFileStateContainerEventsTestSuite) Test_get_last_event_id_returns_last_event_in_partition() {
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

func (s *MemFileStateContainerEventsTestSuite) Test_get_last_event_id_returns_empty_for_non_existent_channel() {
	events := s.container.Events()
	lastEventID, err := events.GetLastEventID(
		context.Background(),
		"changesets",
		"non-existent-channel-id",
	)
	s.Require().NoError(err)
	s.Assert().Equal("", lastEventID)
}

func (s *MemFileStateContainerEventsTestSuite) assertPersistedEvent(expected *manage.Event) {
	// Check that the event state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the event.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	events := container.Events()
	persistedEvent, err := events.Get(
		context.Background(),
		expected.ID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		expected,
		&persistedEvent,
	)
}

func (s *MemFileStateContainerEventsTestSuite) assertEventCleanupPersisted() {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	assertEventsCleanedUp(
		container,
		&s.Suite,
	)
}

func assertEventsCleanedUp(
	container *StateContainer,
	s *suite.Suite,
) {
	for _, id := range eventsShouldBeCleanedUp {
		_, err := container.Events().Get(
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
		event, err := container.Events().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, event.ID)
	}
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

func TestMemFileStateContainerEventsTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerEventsTestSuite))
}
