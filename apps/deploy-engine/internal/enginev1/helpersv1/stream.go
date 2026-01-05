package helpersv1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// staleEndEventThreshold defines how old an end event can be before it's
// considered stale when starting a fresh stream (no StartingEventID).
// End events older than this threshold will not close fresh streams,
// allowing new operations on the same channel to proceed without
// interference from old end events from previous operations.
const staleEndEventThreshold = 10 * time.Second

// shouldCloseOnEndEvent determines if an end event should close the stream.
// When a client provides a StartingEventID (resuming a stream), we always
// honor end events since the client explicitly requested events from that point.
// When starting fresh (no StartingEventID), we ignore stale end events to
// prevent race conditions where a new operation starts but old end events
// from previous operations would prematurely close the stream.
func shouldCloseOnEndEvent(eventTimestamp int64, hasStartingEventID bool) bool {
	if hasStartingEventID {
		// Client is resuming from a specific point - always honor end events
		return true
	}
	// Fresh stream - only close on recent end events
	eventTime := time.Unix(eventTimestamp, 0)
	return time.Since(eventTime) <= staleEndEventThreshold
}

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

	startingEventID := r.Header.Get(LastEventIDHeader)
	endChan, err := eventStore.Stream(
		r.Context(),
		&manage.EventStreamParams{
			ChannelType:     info.ChannelType,
			ChannelID:       info.ChannelID,
			StartingEventID: startingEventID,
		},
		eventChan,
		errChan,
	)
	hasStartingEventID := startingEventID != ""
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
			// We check shouldCloseOnEndEvent to handle the case where a client
			// starts a fresh stream (no StartingEventID) and receives a stale
			// end event from a previous operation before new events arrive.
			if evt.End && shouldCloseOnEndEvent(evt.Timestamp, hasStartingEventID) {
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
			fmt.Println("writing error", err)
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
