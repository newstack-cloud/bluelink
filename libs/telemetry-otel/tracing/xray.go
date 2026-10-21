package tracing

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand/v2"
	"strings"

	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// XRayTraceIDHeader is the header AWS propagates a trace in.
//
// An AWS load balancer in front of the service writes this, so a request arrives with
// the trace the platform has already started for it.
const XRayTraceIDHeader = "X-Amzn-Trace-Id"

// XRayTraceIDField is the log field carrying the X-Ray form of a trace's ID.
//
// Separate from telemetry.TraceIDField because the two are different renderings of the
// same trace, and X-Ray is the only one of them the X-Ray console will find.
const XRayTraceIDField = "xrayTraceId"

// The fields of an X-Ray trace header, of which this reads these three and
// ignores the rest. AWS adds others, Self among them, that describe the hop rather
// than the trace.
const (
	xrayRootField    = "Root"
	xrayParentField  = "Parent"
	xraySampledField = "Sampled"
	// The one value of the sampled field that means the trace is being recorded.
	xraySampledValue = "1"
)

const (
	// Every X-Ray trace ID begins with the version, of which there has only ever
	// been one.
	xrayVersion = "1"
	// The first 8 hex digits of a trace ID are the epoch seconds it started at, and
	// the remaining 24 are random.
	xrayEpochHexLength = 8
	otelTraceIDLength  = 32
)

// ToXRayTraceID renders an OpenTelemetry trace ID in the form X-Ray identifies a trace
// by, which is the version, the epoch seconds the trace started at, and the rest of the
// identifier, separated by hyphens.
//
// Returns an empty string for anything that is not a 32 hex digit trace ID, so that a
// log field is absent rather than carrying an identifier that resolves to nothing.
func ToXRayTraceID(otelTraceID string) string {
	if len(otelTraceID) != otelTraceIDLength {
		return ""
	}

	if _, err := hex.DecodeString(otelTraceID); err != nil {
		return ""
	}

	return xrayVersion + "-" +
		otelTraceID[:xrayEpochHexLength] + "-" +
		otelTraceID[xrayEpochHexLength:]
}

// XRayIDGenerator generates trace IDs in the shape X-Ray accepts.
//
// X-Ray rejects a trace whose ID does not begin with the time it started, so a trace
// a process starts itself has to be given one rather than the random identifier
// OpenTelemetry would otherwise generate. Install it on the tracer provider.
//
// The randomness comes from math/rand/v2, which is seeded for us and safe to call from
// several goroutines at once. An identifier has to be unique, not
// unguessable, which is what makes that the right source rather than crypto/rand.
//
// The SDK starts spans concurrently, so the clock is read concurrently too.
// commoncore.SystemClock is safe for that.
type XRayIDGenerator struct {
	clock commoncore.Clock
}

// NewXRayIDGenerator creates the generator, reading the start time of each trace from
// the given clock.
func NewXRayIDGenerator(clock commoncore.Clock) *XRayIDGenerator {
	return &XRayIDGenerator{clock: clock}
}

// NewIDs generates the IDs for a new trace, the trace ID carrying the time it started.
func (g *XRayIDGenerator) NewIDs(_ context.Context) (oteltrace.TraceID, oteltrace.SpanID) {
	var traceID oteltrace.TraceID
	binary.BigEndian.PutUint32(traceID[:4], uint32(g.clock.Now().Unix()))
	binary.BigEndian.PutUint64(traceID[4:12], rand.Uint64())
	binary.BigEndian.PutUint32(traceID[12:], rand.Uint32())

	return traceID, newSpanID()
}

// NewSpanID generates the ID of a span within a trace that already has one.
func (g *XRayIDGenerator) NewSpanID(_ context.Context, _ oteltrace.TraceID) oteltrace.SpanID {
	return newSpanID()
}

// Repeats until the ID is one OpenTelemetry will accept, which rules out the all-zero
// span ID it treats as absent.
func newSpanID() oteltrace.SpanID {
	var spanID oteltrace.SpanID
	for !spanID.IsValid() {
		binary.BigEndian.PutUint64(spanID[:], rand.Uint64())
	}

	return spanID
}

// XRayPropagator reads and writes a trace in the header AWS propagates one in.
//
// Carried alongside W3C trace context rather than instead of it: AWS itself propagates
// in this format, and a caller that speaks W3C is every other caller.
type XRayPropagator struct{}

// Inject writes the span's trace into the carrier for whatever is called next.
func (XRayPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if !spanCtx.TraceID().IsValid() || !spanCtx.SpanID().IsValid() {
		return
	}

	sampled := "0"
	if spanCtx.IsSampled() {
		sampled = xraySampledValue
	}

	carrier.Set(XRayTraceIDHeader, strings.Join([]string{
		xrayRootField + "=" + ToXRayTraceID(spanCtx.TraceID().String()),
		xrayParentField + "=" + spanCtx.SpanID().String(),
		xraySampledField + "=" + sampled,
	}, ";"))
}

// Extract reads the trace a request arrived carrying, returning the context unchanged
// where there is no usable trace in the carrier.
func (XRayPropagator) Extract(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) context.Context {
	header := carrier.Get(XRayTraceIDHeader)
	if header == "" {
		return ctx
	}

	spanCtx, ok := parseXRayHeader(header)
	if !ok {
		return ctx
	}

	return oteltrace.ContextWithRemoteSpanContext(ctx, spanCtx)
}

// Fields names the headers this writes, so that a caller can clear them before reusing
// a carrier.
func (XRayPropagator) Fields() []string {
	return []string{XRayTraceIDHeader}
}

// Both the trace and the span within it have to be readable for the header to be a
// trace this process can join. AWS omits the parent at the first hop, and OpenTelemetry
// has nowhere to put a trace whose parent span is unknown, it generates a new trace ID
// for one, so such a request starts a trace of its own rather than carrying an ID the
// exported spans would not agree with.
func parseXRayHeader(header string) (oteltrace.SpanContext, bool) {
	var config oteltrace.SpanContextConfig

	for field := range strings.SplitSeq(header, ";") {
		name, value, found := strings.Cut(strings.TrimSpace(field), "=")
		if !found {
			continue
		}

		switch name {
		case xrayRootField:
			traceID, err := parseXRayTraceID(value)
			if err != nil {
				return oteltrace.SpanContext{}, false
			}
			config.TraceID = traceID
		case xrayParentField:
			spanID, err := oteltrace.SpanIDFromHex(value)
			if err != nil {
				return oteltrace.SpanContext{}, false
			}
			config.SpanID = spanID
		case xraySampledField:
			// Compared against the one value that means sampled rather than parsed as
			// a boolean, because AWS sends "?" for a decision it has left open and
			// anything other than "1" is not a decision to sample.
			if value == xraySampledValue {
				config.TraceFlags = oteltrace.FlagsSampled
			}
		}
	}

	spanCtx := oteltrace.NewSpanContext(config)

	return spanCtx, spanCtx.IsValid()
}

// Reverses ToXRayTraceID, taking the version, epoch and identifier back to the 32 hex
// digits OpenTelemetry holds a trace ID as.
func parseXRayTraceID(value string) (oteltrace.TraceID, error) {
	version, rest, found := strings.Cut(value, "-")
	if !found || version != xrayVersion {
		return oteltrace.TraceID{}, errMalformedXRayTraceID
	}

	epoch, identifier, found := strings.Cut(rest, "-")
	if !found || len(epoch) != xrayEpochHexLength {
		return oteltrace.TraceID{}, errMalformedXRayTraceID
	}

	return oteltrace.TraceIDFromHex(epoch + identifier)
}

var errMalformedXRayTraceID = errors.New("not an X-Ray trace id")

// Stated so that a change to what the SDK expects of a propagator is a failure here
// rather than where one is installed.
var _ propagation.TextMapPropagator = XRayPropagator{}
