package deployengine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/errors"
	"github.com/two-hundred/celerity/libs/deploy-engine-client/types"
)

func handleStreamEvents[EventType any](
	streamType string,
	client *sse.Client,
	internalEventChan chan *sse.Event,
	streamTo chan<- EventType,
	errChan chan<- error,
	transformEvent func(*sse.Event) EventType,
	checkIsEnd func(EventType) bool,
	streamTimeout time.Duration,
	logger core.Logger,
) {
	defer client.Unsubscribe(internalEventChan)

	for {
		select {
		case <-time.After(streamTimeout):
			logger.Debug(
				fmt.Sprintf(
					"%s stream timed out, stopping stream",
					streamType,
				),
			)
			close(streamTo)
			return
		case event := <-internalEventChan:
			if string(event.Event) == "error" {
				eventErr := errFromStreamEvent(event)
				select {
				case <-time.After(sendToClientStreamTimeout):
					logger.Debug(
						"timed out waiting for client error listener channel (most likely has been closed)",
					)
					close(streamTo)
					return
				case errChan <- eventErr:
				}
			}

			transformedEvent := transformEvent(event)
			select {
			case <-time.After(sendToClientStreamTimeout):
				logger.Debug(
					fmt.Sprintf(
						"timed out waiting for %s stream client listener channel (most likely has been closed)",
						streamType,
					),
				)
				return
			case streamTo <- transformedEvent:
				isEnd := checkIsEnd(transformedEvent)
				if isEnd {
					logger.Debug(
						fmt.Sprintf(
							"%s stream ended, stopping stream",
							streamType,
						),
					)
					close(streamTo)
					return
				}
			}
		}
	}
}

func errFromStreamEvent(event *sse.Event) error {
	streamErr := &errors.StreamError{
		Event: &types.StreamErrorMessageEvent{},
	}
	err := json.Unmarshal(event.Data, streamErr.Event)
	if err != nil {
		return fmt.Errorf(
			"a stream error was received but could not be "+
				"deserialised into a stream error event: %w",
			err,
		)
	}

	streamErr.Event.ID = string(event.ID)
	return streamErr
}
