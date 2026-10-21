package tracing

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// The trace kind as an attribute key, for the sampler to match against.
var traceKindKey = attribute.Key(TraceKindAttr)

// PlatformAWS is the platform whose tracing has an identifier format of its own.
const PlatformAWS = "aws"

// Settings is the tracing configuration, resolved.
//
// Resolved from configuration by the caller rather than read from the environment
// here, so that tracing is configured the same way as everything else the engine does.
type Settings struct {
	// OTLPEndpoint is the collector to export to.
	OTLPEndpoint string
	// ServiceName and ServiceVersion name the engine in the traces it produces.
	ServiceName    string
	ServiceVersion string
	// SamplingRate is the proportion of traces the engine starts that are exported,
	// between 0 and 1.
	SamplingRate float64
	// OperationSamplingRate is the proportion of the traces the engine starts for work
	// it carries on with after answering the request that asked for it, between 0 and 1.
	//
	// Separate from SamplingRate because the two are not alike. One status poll is
	// interchangeable with the next, so a fraction of them is a fair sample, whereas an
	// operation is a discrete thing someone asks about afterwards by name, and the one
	// they ask about is the one that failed.
	OperationSamplingRate float64
	// Platform is PlatformAWS where traces reach AWS X-Ray, and empty otherwise.
	Platform string
}

// IsAWS reports whether traces have to carry identifiers in AWS X-Ray's format.
func (s Settings) IsAWS() bool {
	return s.Platform == PlatformAWS
}

// Validate reports what about these settings the engine cannot act on.
//
// Called whether or not tracing is on, so that settings the engine would refuse are
// refused when they are written rather than when they are first switched on.
func (s Settings) Validate() error {
	if err := validateEndpoint(s.OTLPEndpoint); err != nil {
		return err
	}

	if s.Platform != "" && !s.IsAWS() {
		return fmt.Errorf(
			"%q is not a supported tracing.platform, expected %q or an empty value",
			s.Platform,
			PlatformAWS,
		)
	}

	if s.SamplingRate < 0 || s.SamplingRate > 1 {
		return fmt.Errorf(
			"tracing.sampling_rate must be between 0 and 1, got %g",
			s.SamplingRate,
		)
	}

	if s.OperationSamplingRate < 0 || s.OperationSamplingRate > 1 {
		return fmt.Errorf(
			"tracing.operation_sampling_rate must be between 0 and 1, got %g",
			s.OperationSamplingRate,
		)
	}

	return nil
}

// Checked here because the exporter does not check it for us.
// When given a URL it cannot parse, it keeps its own default
// and exports there instead, so a typo would have the
// engine export to localhost rather than report anything.
//
// A scheme and a host are what that amounts to. A host that does not resolve still
// cannot be caught, since the gRPC exporter connects lazily and nothing here dials it.
func validateEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("tracing.otlp_endpoint %q is not a URL: %w", endpoint, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf(
			"tracing.otlp_endpoint %q must begin with http:// or https://",
			endpoint,
		)
	}

	if parsed.Host == "" {
		return fmt.Errorf("tracing.otlp_endpoint %q names no host", endpoint)
	}

	return nil
}

// Setup builds a tracer provider from the given settings, installs it behind the
// telemetry seam, and returns the function that shuts it down.
//
// The returned shutdown is never nil, so a caller can defer it without checking. It is
// the only thing that flushes the spans still held when the process stops.
func Setup(
	ctx context.Context,
	settings Settings,
	clock commoncore.Clock,
) (func(context.Context) error, error) {
	provider, err := providerFor(ctx, settings, clock)
	if err != nil {
		return noShutdown, err
	}

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagatorFor(settings))
	telemetry.SetTracer(NewTracer())

	return provider.Shutdown, nil
}

func noShutdown(_ context.Context) error {
	return nil
}

func providerFor(
	ctx context.Context,
	settings Settings,
	clock commoncore.Clock,
) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpointURL(settings.OTLPEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to build the exporter sending traces to %q: %w",
			settings.OTLPEndpoint,
			err,
		)
	}

	described, err := describeService(settings)
	if err != nil {
		return nil, err
	}

	options := []sdktrace.TracerProviderOption{
		// Batched rather than a request per span, so that exporting traces does not
		// put a round trip in the path of the work being traced.
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(described),
		sdktrace.WithSampler(samplerFor(settings)),
	}
	if settings.IsAWS() {
		// X-Ray will not accept a trace ID it does not recognise the shape of, so a
		// trace the engine starts has to be given one.
		options = append(options, sdktrace.WithIDGenerator(NewXRayIDGenerator(clock)))
	}

	return sdktrace.NewTracerProvider(options...), nil
}

// Names the engine in every span it produces.
//
// Schemaless, because merging two resources that carry different schema versions is a
// conflict and the SDK's default resource moves to whatever version it was built
// against.
func describeService(settings Settings) (*resource.Resource, error) {
	described, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName(settings.ServiceName),
		semconv.ServiceVersion(settings.ServiceVersion),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to describe the deploy engine in its traces: %w", err)
	}

	return described, nil
}

// Chooses the sampler, which decides how much is exported.
//
// Parent-based, so that the decision is made once per trace by whoever started it and
// every engine the trace passes through agrees. Without it, a sampled request would
// produce a trace with holes in it where a span was dropped on its own.
func samplerFor(settings Settings) sdktrace.Sampler {
	return rootKindSampler{
		operations: ratioSampler(settings.OperationSamplingRate),
		requests:   ratioSampler(settings.SamplingRate),
	}
}

func ratioSampler(rate float64) sdktrace.Sampler {
	if rate >= 1 {
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}

	if rate <= 0 {
		return sdktrace.ParentBased(sdktrace.NeverSample())
	}

	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(rate))
}

// Samples the two kinds of trace the engine starts at rates of their own.
//
// A provider has one sampler, so telling the two apart has to happen inside it. Only a
// root span needs classifying, every span within an operation has a valid local parent,
// and each delegate being ParentBased means the decision taken at the root is inherited
// the whole way down.
//
// Each delegate staying ParentBased is also the escape hatch. A request that arrived
// with a sampled traceparent is honoured whatever either rate says, so a client that
// needs a trace for one particular deployment can force one by sampling the request
// that starts it.
type rootKindSampler struct {
	operations sdktrace.Sampler
	requests   sdktrace.Sampler
}

func (s rootKindSampler) ShouldSample(
	params sdktrace.SamplingParameters,
) sdktrace.SamplingResult {
	if samplesAsOperation(params.Attributes) {
		return s.operations.ShouldSample(params)
	}

	return s.requests.ShouldSample(params)
}

func (s rootKindSampler) Description() string {
	return fmt.Sprintf(
		"BluelinkRootKind{operations=%s,requests=%s}",
		s.operations.Description(),
		s.requests.Description(),
	)
}

// Whether a span belongs to an operation rather than to request traffic.
//
// Two things make it one. The engine marks a root it starts for detached work, and a
// request that changes something is one in its own right: every one of the engine's
// mutating routes either starts an operation or, in the case of a drift check,
// is one carried out while the caller waits. The reads, the polls, the lists and the
// event streams are the volume the request rate is for.
//
// Classified on the method rather than on a list of routes, so that an endpoint added
// later is classified without anyone needing to remember to add it, and so that renaming a
// route cannot change how it is sampled. Anything other than a read counts, so a method
// this does not know is traced rather than quietly sampled.
//
// Only a root can be classified wrongly here and it does not matter if it is, a child
// span of an outgoing request carries a method of its own, but both delegates are
// ParentBased, so a child inherits the decision taken at its root either way.
func samplesAsOperation(attrs []attribute.KeyValue) bool {
	for _, attr := range attrs {
		switch attr.Key {
		case traceKindKey:
			if attr.Value.AsString() == TraceKindOperation {
				return true
			}
		case semconv.HTTPRequestMethodKey:
			if !isReadMethod(attr.Value.AsString()) {
				return true
			}
		}
	}

	return false
}

func isReadMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions
}

// Chooses how a trace arriving from upstream is read and how one leaving is written.
//
// W3C always, since that is what a caller of the engine's API propagates, and X-Ray's
// own header alongside it on AWS, where the platform in front of the engine propagates
// in that format and a trace would otherwise break at the boundary. W3C baggage travels
// with both.
//
// X-Ray goes first because extracting runs the propagators in order and the last one to
// find a trace wins. A request carrying both headers is one whose caller started a W3C
// trace and then passed through AWS, so the caller's trace is the one to join.
func propagatorFor(settings Settings) propagation.TextMapPropagator {
	carried := []propagation.TextMapPropagator{}
	if settings.IsAWS() {
		carried = append(carried, XRayPropagator{})
	}

	return propagation.NewCompositeTextMapPropagator(append(
		carried,
		propagation.TraceContext{},
		propagation.Baggage{},
	)...)
}
