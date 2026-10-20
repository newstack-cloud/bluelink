// Package tracing exports distributed traces of the work the deploy engine carries out.
//
// This is the OpenTelemetry implementation of the telemetry package's seam. Spans are
// started through the seam rather than here, so the engine's code reads the same
// whether or not traces are being exported and the decision is confined to startup.
//
// [Setup] builds a tracer provider from what an operator configured and installs it
// behind the seam. Where tracing is off nothing is built and the seam's own
// implementation, which does nothing, stays in place.
package tracing

import (
	"context"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Names this instrumentation in the spans it produces, which is how a backend tells
// the engine's own spans from those of a library it traces through.
const scope = "github.com/newstack-cloud/bluelink/apps/deploy-engine"

// NewTracer returns a telemetry.Tracer starting spans on whatever tracer provider
// OpenTelemetry has been given.
//
// Exported for a test, and for a host embedding the engine that builds a provider of
// its own. [Setup] installs one for an ordinary deployment.
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
