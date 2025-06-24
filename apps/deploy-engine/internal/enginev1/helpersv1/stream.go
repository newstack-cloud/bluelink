package helpersv1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// StreamInfo holds information about the stream channel type and ID
// to be used for streaming events to clients over SSE.
type StreamInfo struct {
	ChannelType string
	ChannelID   string
}

// SSEStreamEvents deals with streaming events from a channel to a client
// using Server-Sent Events (SSE).
func SSEStreamEvents(
	w http.ResponseWriter,
	r *http.Request,
	info *StreamInfo,
	eventStore manage.Events,
	logger core.Logger,
) {
	// Check if the ResponseWriter supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	eventChan := make(chan manage.Event)
	errChan := make(chan error)

	endChan, err := eventStore.Stream(
		r.Context(),
		&manage.EventStreamParams{
			ChannelType:     info.ChannelType,
			ChannelID:       info.ChannelID,
			StartingEventID: r.Header.Get(LastEventIDHeader),
		},
		eventChan,
		errChan,
	)
	if err != nil {
		logger.Error(
			"Failed to start event stream",
			core.ErrorLogField("error", err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

L:
	for {
		select {
		case <-r.Context().Done():
			logger.Debug(
				"stream context cancelled",
				core.ErrorLogField("error", r.Context().Err()),
			)
			break L

		// Listen for incoming messages from messageChan
		case evt := <-eventChan:
			// Write to the ResponseWriter
			// Server Sent Events compatible
			writeEvent(w, evt, flusher)

			// An event at the end of a stream is marked with a special
			// "End" field. This is used to indicate that the stream has ended.
			if evt.End {
				select {
				case endChan <- struct{}{}:
					logger.Debug("End of stream")
				case <-r.Context().Done():
					logger.Debug(
						"stream context cancelled while sending end signal",
						core.ErrorLogField("error", r.Context().Err()),
					)
				}
				break L
			}
		case err := <-errChan:
			writeError(w, err, flusher)
			break L
		}
	}
}

func writeEvent(
	w http.ResponseWriter,
	evt manage.Event,
	flusher http.Flusher,
) {
	fmt.Fprintf(w, "event: %s\n", evt.Type)
	fmt.Fprintf(w, "id: %s\n", evt.ID)
	fmt.Fprintf(w, "data: %s\n\n", evt.Data)

	// Flush the data immediatly instead of buffering it for later.
	flusher.Flush()
}

// writes errors that are not a part of the persisted stream
// to the client. This should only be used for errors that are not
// expected. Validation errors for a blueprint validation
// should be sent as events with IDs that are persisted like any other
// intended event in a stream.
func writeError(
	w http.ResponseWriter,
	err error,
	flusher http.Flusher,
) {
	errBytes, _ := json.Marshal(struct {
		Error string `json:"error"`
	}{Error: err.Error()})
	fmt.Fprintf(w, "event: error\n")
	fmt.Fprintf(w, "data: %s\n\n", string(errBytes))
	flusher.Flush()
}
