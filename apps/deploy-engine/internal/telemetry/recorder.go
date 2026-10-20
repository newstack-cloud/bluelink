package telemetry

import (
	"context"
	"fmt"
	"sync"
)

// Recorder is a Tracer that keeps what it was asked to trace, for tests to assert on.
//
// A third implementation of the seam rather than a mock of the OpenTelemetry one, so a
// test can state what work should have been traced without standing up an exporter.
type Recorder struct {
	mu       sync.Mutex
	spans    []*RecordedSpan
	nextID   int
	parentOf map[*RecordedSpan]string
}

// RecordedSpan is one span a Recorder was asked for.
type RecordedSpan struct {
	Name     string
	TraceID  string
	SpanID   string
	ParentID string
	Attrs    map[string]any
	Err      error
	Ended    bool
}

// Attr returns the value recorded under key, and whether it was recorded at all.
func (s *RecordedSpan) Attr(key string) (any, bool) {
	value, found := s.Attrs[key]

	return value, found
}

type recordedSpanHandle struct {
	recorder *Recorder
	span     *RecordedSpan
}

type recorderParentKey struct{}

// NewRecorder creates a Recorder. Install it with SetTracer.
func NewRecorder() *Recorder {
	return &Recorder{parentOf: map[*RecordedSpan]string{}}
}

func (r *Recorder) Start(
	ctx context.Context,
	name string,
	attrs ...Attr,
) (context.Context, Span) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	// Counted rather than random, so that a test asserting on a correlated log line has
	// something it can predict.
	span := &RecordedSpan{
		Name:    name,
		TraceID: fmt.Sprintf("trace-%d", r.nextID),
		SpanID:  fmt.Sprintf("span-%d", r.nextID),
		Attrs:   map[string]any{},
	}
	if parent, ok := ctx.Value(recorderParentKey{}).(string); ok {
		span.ParentID = parent
	}
	for _, attr := range attrs {
		span.Attrs[attr.Key] = attr.Value
	}
	r.spans = append(r.spans, span)

	return context.WithValue(ctx, recorderParentKey{}, span.SpanID),
		&recordedSpanHandle{recorder: r, span: span}
}

// Spans returns the spans recorded so far, in the order they were started.
func (r *Recorder) Spans() []*RecordedSpan {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]*RecordedSpan(nil), r.spans...)
}

// Span returns the single span recorded under the given name, and whether exactly one
// was.
func (r *Recorder) Span(name string) (*RecordedSpan, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var found *RecordedSpan
	for _, span := range r.spans {
		if span.Name != name {
			continue
		}
		if found != nil {
			return nil, false
		}
		found = span
	}

	return found, found != nil
}

// Names returns the names of the spans recorded so far, in order.
func (r *Recorder) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make([]string, len(r.spans))
	for i, span := range r.spans {
		names[i] = span.Name
	}

	return names
}

func (h *recordedSpanHandle) SetAttributes(attrs ...Attr) {
	h.recorder.mu.Lock()
	defer h.recorder.mu.Unlock()

	for _, attr := range attrs {
		h.span.Attrs[attr.Key] = attr.Value
	}
}

func (h *recordedSpanHandle) RecordError(err error) {
	h.recorder.mu.Lock()
	defer h.recorder.mu.Unlock()
	h.span.Err = err
}

func (h *recordedSpanHandle) End() {
	h.recorder.mu.Lock()
	defer h.recorder.mu.Unlock()
	h.span.Ended = true
}

func (h *recordedSpanHandle) Context() SpanContext {
	return SpanContext{TraceID: h.span.TraceID, SpanID: h.span.SpanID}
}
