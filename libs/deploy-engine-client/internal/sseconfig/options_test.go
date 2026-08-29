package sseconfig

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Larger than the 64KB the SSE library defaults to, and well within what a
// change set for a real blueprint reaches.
const oversizedEventBytes = 200 * 1024

func serverSendingOneEvent(t *testing.T, dataSize int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		fmt.Fprintf(w, "event: message\n")
		fmt.Fprintf(w, "id: 1\n")
		fmt.Fprintf(w, "data: %s\n\n", strings.Repeat("x", dataSize))
		w.(http.Flusher).Flush()
	}))
}

func receiveOneEvent(t *testing.T, url string, opts ...Option) *sse.Event {
	t.Helper()

	client := sse.NewClient(url)
	for _, opt := range opts {
		opt(client)
	}

	events := make(chan *sse.Event, 1)
	require.NoError(t, client.SubscribeChan("messages", events))
	defer client.Unsubscribe(events)

	select {
	case event := <-events:
		return event
	case <-time.After(5 * time.Second):
		return nil
	}
}

// An event over the limit stops the read with nothing delivered and no error
// reaching the caller, which shows up as a client waiting on a stream the
// server has already finished rather than as a failure.
func Test_an_event_over_the_default_limit_is_never_delivered(t *testing.T) {
	server := serverSendingOneEvent(t, oversizedEventBytes)
	defer server.Close()

	event := receiveOneEvent(t, server.URL)

	assert.Nil(t, event, "the default 64KB limit should drop this event")
}

func Test_an_event_over_the_default_limit_is_delivered_with_a_raised_limit(t *testing.T) {
	server := serverSendingOneEvent(t, oversizedEventBytes)
	defer server.Close()

	event := receiveOneEvent(t, server.URL, WithMaxBufferSize(10*1024*1024))

	require.NotNil(t, event, "a raised limit should let the event through")
	assert.Len(t, event.Data, oversizedEventBytes)
}
