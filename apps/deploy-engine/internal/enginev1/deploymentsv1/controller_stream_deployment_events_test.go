package deploymentsv1

import (
	"context"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/r3labs/sse/v2"
)

func (s *ControllerTestSuite) Test_stream_deployment_events_handler() {
	// Store some events in the mock event store to be streamed.
	expectedEvents, err := s.saveDeploymentEvents()
	s.Require().NoError(err)

	router := mux.NewRouter()
	router.HandleFunc(
		"/deployments/instances/{id}/stream",
		s.ctrl.StreamDeploymentEventsHandler,
	).Methods("GET")

	// We need to create a server to be able to stream events asynchronously,
	// the httptest recording test tools do not support response streaming
	// as the Result() method is to be used after response writing is done.
	streamServer := httptest.NewServer(router)
	defer streamServer.Close()

	url := fmt.Sprintf(
		"%s/deployments/instances/%s/stream",
		streamServer.URL,
		testInstanceID,
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

func (s *ControllerTestSuite) saveDeploymentEvents() ([]*manage.Event, error) {
	ctx := context.Background()

	events := make([]*manage.Event, 10)
	for i := 0; i < len(events); i += 1 {
		event := &manage.Event{
			ID:          fmt.Sprintf("event-%d", i),
			Type:        selectDeploymentEventType(i, len(events)),
			ChannelType: helpersv1.ChannelTypeDeployment,
			ChannelID:   testInstanceID,
			Data: fmt.Sprintf(
				"{\"message\": \"streaming deployment event %d for blueprint instance %s\"}",
				i,
				testInstanceID,
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

var deploymentEventTypes = []string{
	eventTypeChildUpdate,
	eventTypeResourceUpdate,
	eventTypeLinkUpdate,
	eventTypeInstanceUpdate,
}

func selectDeploymentEventType(i int, total int) string {
	if i == total-1 {
		return eventTypeDeployFinished
	}

	index := rand.Intn(len(deploymentEventTypes))
	return deploymentEventTypes[index]
}
