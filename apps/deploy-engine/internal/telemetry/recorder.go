package telemetry

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

// Recorder is a Tracer that keeps what it was asked to trace, for tests to assert on.
//
// A third implementation of the seam rather than a mock of the OpenTelemetry one, so a
// test can state what work should have been traced without standing up an exporter.
type Recorder struct {
	mu     sync.Mutex
	spans  []*RecordedSpan
	nextID int
}

// RecordedSpan is one span a Recorder was asked for.
type RecordedSpan struct {
	Name     string
	TraceID  string
	SpanID   string
	ParentID string
	// LinkedTo is the span named as having asked for this work, for a span started
	// through StartRoot. Empty for every other span.
	LinkedTo SpanContext
	Attrs    map[string]any
	Err      error
	Ended    bool
}

// Attr returns the value recorded under key, and whether it was recorded at all.
func (s *RecordedSpan) Attr(key string) (any, bool) {
	value, found := s.Attrs[key]

	return value, found
}

// Taken under the recorder's lock, by the readers below.
func (s *RecordedSpan) snapshot() *RecordedSpan {
	copied := *s
	copied.Attrs = maps.Clone(s.Attrs)

	return &copied
}

type recordedSpanHandle struct {
	recorder *Recorder
	span     *RecordedSpan
}

type recorderParentKey struct{}

type recordedParent struct {
	traceID string
	spanID  string
}

// NewRecorder creates a Recorder. Install it with SetTracer.
func NewRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) Start(
	ctx context.Context,
	name string,
	attrs ...Attr,
) (context.Context, Span) {
	return r.start(ctx, name, true, SpanContext{}, attrs)
}

// StartRoot records the span as belonging to a trace of its own, so a test can state
// that an operation was traced separately from the request that asked for it rather
// than only that both were traced.
func (r *Recorder) StartRoot(
	ctx context.Context,
	name string,
	linkedTo SpanContext,
	attrs ...Attr,
) (context.Context, Span) {
	return r.start(ctx, name, false, linkedTo, attrs)
}

func (r *Recorder) start(
	ctx context.Context,
	name string,
	inheritParent bool,
	linkedTo SpanContext,
	attrs []Attr,
) (context.Context, Span) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	// Counted rather than random, so that a test asserting on a correlated log line has
	// something it can predict.
	span := &RecordedSpan{
		Name:     name,
		TraceID:  fmt.Sprintf("trace-%d", r.nextID),
		SpanID:   fmt.Sprintf("span-%d", r.nextID),
		LinkedTo: linkedTo,
		Attrs:    map[string]any{},
	}
	// A span started within another belongs to the same trace, so that a test can state
	// that a unit of work and what it did were traced together rather than only that one
	// names the other as its parent.
	parent, hasParent := ctx.Value(recorderParentKey{}).(recordedParent)
	if hasParent && inheritParent {
		span.TraceID = parent.traceID
		span.ParentID = parent.spanID
	}
	for _, attr := range attrs {
		span.Attrs[attr.Key] = attr.Value
	}
	r.spans = append(r.spans, span)

	return context.WithValue(ctx, recorderParentKey{}, recordedParent{
		traceID: span.TraceID,
		spanID:  span.SpanID,
	}), &recordedSpanHandle{recorder: r, span: span}
}

// Spans returns the spans recorded so far, in the order they were started.
//
// Takes snapshots rather than the spans themselves as traced work
// often keeps writing to its span on another goroutine after this has returned.
func (r *Recorder) Spans() []*RecordedSpan {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshots := make([]*RecordedSpan, len(r.spans))
	for i, span := range r.spans {
		snapshots[i] = span.snapshot()
	}

	return snapshots
}

// Span returns a snapshot of the single span recorded under the given name, and whether
// exactly one was.
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

	if found == nil {
		return nil, false
	}

	return found.snapshot(), true
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
