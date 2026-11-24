package helpersv1_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type StreamSuite struct {
	suite.Suite
}

// The store's own error is logged, not sent to the client.
func (s *StreamSuite) Test_a_stream_that_cannot_start_responds_with_a_json_error() {
	w := httptest.NewRecorder()

	helpersv1.SSEStreamEvents(
		w,
		httptest.NewRequest(http.MethodGet, "/stream", nil),
		&helpersv1.StreamInfo{ChannelType: helpersv1.ChannelTypeDeployment, ChannelID: "instance-1"},
		&failingEventStore{},
		core.NewNopLogger(),
	)

	s.Assert().Equal(http.StatusInternalServerError, w.Code)
	s.Assert().Equal("application/json", w.Header().Get("Content-Type"))
	s.Assert().JSONEq(`{"message": "an unexpected error occurred"}`, w.Body.String())
}

func (s *StreamSuite) Test_a_response_that_cannot_stream_responds_with_a_json_error() {
	recorder := httptest.NewRecorder()
	w := &unflushableResponseWriter{recorder: recorder}

	helpersv1.SSEStreamEvents(
		w,
		httptest.NewRequest(http.MethodGet, "/stream", nil),
		&helpersv1.StreamInfo{ChannelType: helpersv1.ChannelTypeDeployment, ChannelID: "instance-1"},
		&failingEventStore{},
		core.NewNopLogger(),
	)

	s.Assert().Equal(http.StatusInternalServerError, recorder.Code)
	s.Assert().Equal("application/json", recorder.Header().Get("Content-Type"))
	s.Assert().JSONEq(`{"message": "streaming is not supported"}`, recorder.Body.String())
}

func TestStreamSuite(t *testing.T) {
	suite.Run(t, new(StreamSuite))
}

type failingEventStore struct {
	manage.Events
}

func (f *failingEventStore) Stream(
	_ context.Context,
	_ *manage.EventStreamParams,
	_ chan manage.Event,
	_ chan error,
) (chan struct{}, error) {
	return nil, errors.New("connection to the event store was lost")
}

// Holds the recorder rather than embedding it so that its Flush is not promoted, as a
// response wrapped by a middleware may not have one.
type unflushableResponseWriter struct {
	recorder *httptest.ResponseRecorder
}

func (w *unflushableResponseWriter) Header() http.Header {
	return w.recorder.Header()
}

func (w *unflushableResponseWriter) Write(data []byte) (int, error) {
	return w.recorder.Write(data)
}

func (w *unflushableResponseWriter) WriteHeader(statusCode int) {
	w.recorder.WriteHeader(statusCode)
}
