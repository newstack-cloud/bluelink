// Package tracing exports distributed traces over OpenTelemetry.
//
// This is the OpenTelemetry implementation of the telemetry package's seam. Spans are
// started through the seam rather than here, so a caller's code reads the same whether
// or not traces are being exported and the decision is confined to startup.
//
// [Setup] builds a tracer provider from what an operator configured and installs it
// behind the seam. Where tracing is off nothing is built and the seam's own
// implementation, which does nothing, stays in place.
package tracing

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/common/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Names this instrumentation in the spans it produces, which is what a backend groups
// them by.
const scope = "github.com/newstack-cloud/bluelink/libs/telemetry-otel"

// TraceKindAttr marks the root of a trace with the kind of work it covers.
//
// This is one of the two things the sampler classifies on, so that the two kinds of
// trace a process starts can be sampled at rates of their own. It marks the detached
// work a process starts for itself; a request that changes something is recognised by
// its method instead, since it has no opportunity to mark itself. An attribute rather
// than the span name, so that renaming a span cannot quietly change how it is sampled.
// It stays on the span afterwards, which also lets a backend filter by it.
const TraceKindAttr = "bluelink.trace.kind"

// TraceKindOperation marks the root of a trace covering one operation, meaning work a
// process carries on with after answering the request that asked for it. In the deploy
// engine that is a deployment, a destroy, a rollback, change staging, a blueprint
// validation or a state cleanup pass.
const TraceKindOperation = "operation"

// NewTracer returns a telemetry.Tracer starting spans on whatever tracer provider
// OpenTelemetry has been given.
//
// Exported for a test, and for a host that builds a provider of its own. [Setup]
// installs one for an ordinary deployment.
func NewTracer() telemetry.Tracer {
	return tracer{}
}

type tracer struct{}

// Resolves the provider per call rather than capturing one, so that a provider
// installed after this tracer was does not leave it starting spans on the no-op that
// preceded it.
func (tracer) Start(
	ctx context.Context,
	name string,
	attrs ...telemetry.Attr,
) (context.Context, telemetry.Span) {
	ctx, started := otel.Tracer(scope).Start(
		ctx,
		name,
		oteltrace.WithAttributes(toKeyValues(attrs)...),
	)

	return ctx, span{inner: started}
}

func (tracer) StartRoot(
	ctx context.Context,
	name string,
	linkedTo telemetry.SpanContext,
	attrs ...telemetry.Attr,
) (context.Context, telemetry.Span) {
	options := []oteltrace.SpanStartOption{
		oteltrace.WithNewRoot(),
		oteltrace.WithAttributes(toKeyValues(attrs)...),
		oteltrace.WithAttributes(attribute.String(TraceKindAttr, TraceKindOperation)),
	}
	if link, ok := linkOf(linkedTo); ok {
		options = append(options, oteltrace.WithLinks(link))
	}

	ctx, started := otel.Tracer(scope).Start(ctx, name, options...)

	return ctx, span{inner: started}
}

// Rebuilt from the identifiers rather than taken from a context, since the work is
// started from a struct a request handler filled in, by which point the request's own
// context has been cancelled.
//
// Marked sampled because the seam only reports the identity of a span that was sampled
// in the first place. See spanContextOf.
func linkOf(spanCtx telemetry.SpanContext) (oteltrace.Link, bool) {
	traceID, err := oteltrace.TraceIDFromHex(spanCtx.TraceID)
	if err != nil {
		return oteltrace.Link{}, false
	}

	spanID, err := oteltrace.SpanIDFromHex(spanCtx.SpanID)
	if err != nil {
		return oteltrace.Link{}, false
	}

	return oteltrace.Link{
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: oteltrace.FlagsSampled,
		}),
	}, true
}

func (tracer) SpanFromContext(ctx context.Context) telemetry.Span {
	return span{inner: oteltrace.SpanFromContext(ctx)}
}

type span struct {
	inner oteltrace.Span
}

func (s span) SetAttributes(attrs ...telemetry.Attr) {
	s.inner.SetAttributes(toKeyValues(attrs)...)
}

// Records the error as an event and sets the status, which is what a backend filters
// failed spans by.
func (s span) RecordError(err error) {
	if err == nil {
		return
	}

	s.inner.RecordError(err)
	s.inner.SetStatus(codes.Error, err.Error())
}

func (s span) End() {
	s.inner.End()
}

// The span's place in its trace, as the strings a log entry carries.
func (s span) Context() telemetry.SpanContext {
	return spanContextOf(s.inner.SpanContext())
}

// SpanContextFromContext returns the identity of the span ctx is carrying, and zero
// where it carries none or nothing is recording it.
//
// This is for correlating what was logged with the span it was logged under, where the
// span was started by something that does not hand it over, request instrumentation
// being the case that matters.
func SpanContextFromContext(ctx context.Context) telemetry.SpanContext {
	return spanContextOf(oteltrace.SpanContextFromContext(ctx))
}

// Sampling is part of the question, not just validity as a span the sampler declined has
// perfectly good identifiers and is never exported, so correlating against it points at
// a trace nothing holds.
func spanContextOf(spanCtx oteltrace.SpanContext) telemetry.SpanContext {
	if !spanCtx.IsValid() || !spanCtx.IsSampled() {
		return telemetry.SpanContext{}
	}

	return telemetry.SpanContext{
		TraceID: spanCtx.TraceID().String(),
		SpanID:  spanCtx.SpanID().String(),
	}
}

// Turns the seam's attributes into OpenTelemetry's.
//
// A value of a type the seam's constructors cannot produce is recorded as its printed
// form rather than dropped, since a span missing an attribute is harder to account for
// than one carrying a string.
func toKeyValues(attrs []telemetry.Attr) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	converted := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		switch value := attr.Value.(type) {
		case string:
			converted = append(converted, attribute.String(attr.Key, value))
		case int64:
			converted = append(converted, attribute.Int64(attr.Key, value))
		case float64:
			converted = append(converted, attribute.Float64(attr.Key, value))
		case bool:
			converted = append(converted, attribute.Bool(attr.Key, value))
		default:
			converted = append(
				converted,
				attribute.String(attr.Key, telemetry.AttrString(value)),
			)
		}
	}

	return converted
}
