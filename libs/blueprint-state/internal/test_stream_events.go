package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/stretchr/testify/suite"
)

func TestStreamEvents(
	fixtures []SaveEventFixture,
	events manage.Events,
	channelType string,
	channelID string,
	expectedInitialEventIDs []string,
	s *suite.Suite,
) {
	expectedInitialCount := len(expectedInitialEventIDs)

	collectedEvents := []*manage.Event{}
	streamTo := make(chan manage.Event)
	errChan := make(chan error)
	endChan, err := events.Stream(
		context.Background(),
		&manage.EventStreamParams{
			ChannelType: channelType,
			ChannelID:   channelID,
		},
		streamTo,
		errChan,
	)
	s.Require().NoError(err)

	go func() {
		for _, fixture := range fixtures {
			err := events.Save(
				context.Background(),
				fixture.Event,
			)
			if err != nil {
				fmt.Println("Failed to save event", err)
			}
		}
	}()

	// N existing events are streamed in addition to the new events
	// saved as a part of this test.
	totalToCollect := len(fixtures) + expectedInitialCount
	for len(collectedEvents) < totalToCollect && err == nil {
		select {
		case event := <-streamTo:
			collectedEvents = append(collectedEvents, &event)
		case err = <-errChan:
			s.Fail("Error in event stream", err)
		case <-time.After(20 * time.Second):
			s.Fail("Timeout waiting for event stream")
		}
	}

	s.Require().NoError(err)

	select {
	case endChan <- struct{}{}:
	case <-time.After(5 * time.Second):
		s.Fail("Timeout waiting for listener to handle end signal")
	}

	s.Assert().Len(collectedEvents, len(fixtures)+expectedInitialCount)
	for i, expectedID := range expectedInitialEventIDs {
		s.Assert().Equal(
			expectedID,
			collectedEvents[i].ID,
		)
	}

	for i := expectedInitialCount; i < len(collectedEvents); i += 1 {
		// The deterministic value in the set of generated events is the data
		// field, which is a JSON encoded string of the generated event index.
		// The order of collected events will tell us if events were sent in the expected
		// order in the stream.
		generatedEventIndex := i - expectedInitialCount
		var actualData map[string]any
		err := json.Unmarshal([]byte(collectedEvents[i].Data), &actualData)
		s.Require().NoError(err)

		s.Assert().Equal(
			map[string]any{
				"value": fmt.Sprintf("%d", generatedEventIndex),
			},
			actualData,
		)
	}
}

func TestEndOfEventStream(
	events manage.Events,
	channelType string,
	channelID string,
	s *suite.Suite,
) {
	collectedEvents := []*manage.Event{}
	streamTo := make(chan manage.Event)
	errChan := make(chan error)
	endChan, err := events.Stream(
		context.Background(),
		&manage.EventStreamParams{
			ChannelType: channelType,
			ChannelID:   channelID,
		},
		streamTo,
		errChan,
	)
	s.Require().NoError(err)

	select {
	case event := <-streamTo:
		collectedEvents = append(collectedEvents, &event)
	case err = <-errChan:
		s.Fail("Error in event stream", err)
	case <-time.After(20 * time.Second):
		s.Fail("Timeout waiting for event stream")
	}

	s.Require().NoError(err)

	s.Assert().Len(collectedEvents, 1)
	s.Assert().True(
		collectedEvents[0].End,
	)

	// The event store should be listening for the end signal
	// which will close the stream on the event store side.
	select {
	case endChan <- struct{}{}:
	case <-time.After(5 * time.Second):
		s.Fail("Timeout waiting for listener to handle end signal")
	}
}
