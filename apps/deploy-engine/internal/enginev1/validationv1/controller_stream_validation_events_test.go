package validationv1

import (
	"context"
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/celerity/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/celerity/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/celerity/libs/blueprint-state/manage"
	"github.com/r3labs/sse/v2"
)

const (
	testBlueprintValidationID = "43e3ca9d-6d7f-424c-b08c-ff607d06eefa"
)

func (s *ControllerTestSuite) Test_stream_blueprint_validation_events_handler() {
	// Store some events in the mock event store to be streamed.
	expectedEvents, err := s.saveBlueprintValidationEvents()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/validations/{id}/stream",
		s.ctrl.StreamEventsHandler,
	).Methods("GET")

	// We need to create a server to be able to stream events asynchronously,
	// the httptest recording test tools do not support response streaming
	// as the Result() method is to be used after response writing is done.
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/validations/%s/stream",
		streamServer.URL,
		testBlueprintValidationID,
	)

	client := sse.NewClient(url)

	eventChan := make(chan *sse.Event)
	client.SubscribeChan("messages", eventChan)
	defer client.Unsubscribe(eventChan)

	collected := []*manage.Event{}
	for len(collected) < len(expectedEvents) {
		select {
		case event := <-eventChan:
			manageEvent := testutils.SSEToManageEvent(event)
			collected = append(collected, manageEvent)
			s.Require().NotNil(event)
		case <-time.After(5 * time.Second):
			s.Fail("Timed out waiting for events")
		}
	}

	// Only the ID, type and data fields are retained in the SSE events.
	for i, event := range collected {
		s.Assert().Equal(
			expectedEvents[i].ID,
			event.ID,
		)
		s.Assert().Equal(
			expectedEvents[i].Type,
			event.Type,
		)
		s.Assert().Equal(
			expectedEvents[i].Data,
			event.Data,
		)
	}
}

func (s *ControllerTestSuite) saveBlueprintValidationEvents() ([]*manage.Event, error) {
	ctx := context.Background()

	events := make([]*manage.Event, 10)
	for i := 0; i < len(events); i += 1 {
		event := &manage.Event{
			ID:          fmt.Sprintf("event-%d", i),
			Type:        eventTypeDiagnostic,
			ChannelType: helpersv1.ChannelTypeValidation,
			ChannelID:   testBlueprintValidationID,
			Data: fmt.Sprintf(
				"{\"message\": \"streaming event %d for validation %s\"}",
				i,
				testBlueprintValidationID,
			),
			Timestamp: testTime.Unix(),
			End:       i == len(events)-1,
		}
		err := s.eventStore.Save(ctx, event)
		if err != nil {
			return nil, err
		}
		events[i] = event
	}

	return events, nil
}
