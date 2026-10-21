// Package telemetry is the seam the engine traces through.
//
// Nothing here imports OpenTelemetry. The engine depends on this interface, and an
// implementation is installed at startup when tracing is configured, so a build that
// traces nothing carries no tracing machinery and every call site reads the same either
// way.
package telemetry

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Tracer starts spans for units of work.
//
// This is deliberately not HTTP-shaped, a scheduled cleanup pass
// and a request are both work worth a span, and only one of them has a request.
type Tracer interface {
	// Start begins a span for the named unit of work, returning a context that carries
	// it so that work nested inside becomes a child.
	Start(ctx context.Context, name string, attrs ...Attr) (context.Context, Span)
	// StartRoot begins a span for work that outlives whatever asked for it, as the root
	// of a trace of its own rather than as a child of any span ctx carries.
	//
	// A deployment runs for minutes to hours where the request asking for it lasts
	// milliseconds. Parenting one on the other would hold the request's span open for
	// the whole deployment and produce a child outliving its parent by three orders of
	// magnitude, which reads badly in every backend. So linkedTo names the span that
	// asked for the work instead, letting a backend lead from the request to the trace
	// of what it started. The zero value links to nothing, which is what an operation
	// started by a scheduled or internal path has.
	StartRoot(
		ctx context.Context,
		name string,
		linkedTo SpanContext,
		attrs ...Attr,
	) (context.Context, Span)
}

// Span is one unit of work in progress.
type Span interface {
	// SetAttributes records what was learned about the work, for anything only known
	// once it has been done.
	SetAttributes(attrs ...Attr)
	// RecordError marks the work as having failed, with the error that failed it.
	RecordError(err error)
	// End closes the span. A span that is never ended is never exported.
	End()
	// Context identifies the span, for correlating it with what was logged.
	Context() SpanContext
}

// SpanContext identifies a span well enough to find it in a trace backend.
type SpanContext struct {
	TraceID string
	SpanID  string
}

// Recorded reports whether the span is one a backend will have.
//
// What this answers is whether it is worth putting the IDs on a log line: an unrecorded
// span's IDs point at nothing, and a line carrying them reads as though a trace exists.
func (c SpanContext) Recorded() bool {
	return c.TraceID != ""
}

// Log field names for correlating a log line with the span it was written under.
const (
	TraceIDField = "traceId"
	SpanIDField  = "spanId"
)

// Attr is one piece of detail on a span.
//
// The constructors below fix the value's type, so a backend is given a number as a
// number rather than as whatever its string happened to look like.
type Attr struct {
	Key   string
	Value any
}

// StringAttr records a string.
func StringAttr(key string, value string) Attr {
	return Attr{Key: key, Value: value}
}

// IntAttr records an integer.
func IntAttr(key string, value int64) Attr {
	return Attr{Key: key, Value: value}
}

// FloatAttr records a floating-point number.
func FloatAttr(key string, value float64) Attr {
	return Attr{Key: key, Value: value}
}

// BoolAttr records a boolean.
func BoolAttr(key string, value bool) Attr {
	return Attr{Key: key, Value: value}
}

// installed holds the tracer the engine traces through.
//
// Process-wide rather than threaded through every caller, because which implementation
// is installed is the deployment's decision and the same for every piece of work. An
// atomic pointer rather than a mutex since it is read on every span and written once at
// startup.
var installed atomic.Pointer[Tracer]

// SetTracer installs the tracer the engine traces through. A nil tracer restores the
// one that does nothing.
func SetTracer(tracer Tracer) {
	if tracer == nil {
		installed.Store(nil)
		return
	}

	installed.Store(&tracer)
}

// CurrentTracer returns the installed tracer, or one that does nothing.
//
// Never nil, so that a call site pays a call that does nothing rather than a branch at
// every point it might trace.
func CurrentTracer() Tracer {
	if tracer := installed.Load(); tracer != nil {
		return *tracer
	}

	return noopTracer{}
}

// Traced runs fn inside a span, recording an error it returns against the span.
//
// The span is handed to fn as well as the context, so that detail only known once the
// work is done, a count of what it removed for instance, can be recorded on it.
//
// A panic ends the span without recording anything. Describing it belongs to whatever
// recovers it, which knows what it means.
func Traced[T any](
	ctx context.Context,
	name string,
	attrs []Attr,
	fn func(ctx context.Context, span Span) (T, error),
) (T, error) {
	ctx, span := CurrentTracer().Start(ctx, name, attrs...)
	defer span.End()

	result, err := fn(ctx, span)
	if err != nil {
		span.RecordError(err)
	}

	return result, err
}

// TracedCall runs fn inside a span, for work that has no result to return.
func TracedCall(
	ctx context.Context,
	name string,
	attrs []Attr,
	fn func(ctx context.Context, span Span) error,
) error {
	_, err := Traced(ctx, name, attrs, func(ctx context.Context, span Span) (struct{}, error) {
		return struct{}{}, fn(ctx, span)
	})

	return err
}

// AttrString renders an attribute's value for an implementation that can only carry
// strings, so that an unexpected type is recorded rather than dropped.
func AttrString(value any) string {
	return fmt.Sprint(value)
}

type noopTracer struct{}

func (noopTracer) Start(ctx context.Context, _ string, _ ...Attr) (context.Context, Span) {
	return ctx, noopSpan{}
}

func (noopTracer) StartRoot(
	ctx context.Context,
	_ string,
	_ SpanContext,
	_ ...Attr,
) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) SetAttributes(_ ...Attr) {}

func (noopSpan) RecordError(_ error) {}

func (noopSpan) End() {}

func (noopSpan) Context() SpanContext { return SpanContext{} }
