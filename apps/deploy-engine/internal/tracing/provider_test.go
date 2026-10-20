package tracing_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tracing"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// An engine that was asked to export traces and silently did not is the hardest
// telemetry failure to notice, so settings it cannot act on are refused rather than
// quietly ignored.
type SettingsSuite struct {
	suite.Suite
}

func (s *SettingsSuite) Test_accepts_what_the_engine_can_act_on() {
	for _, settings := range []tracing.Settings{
		s.valid(),
		{OTLPEndpoint: "https://collector.example.com:4317", SamplingRate: 0},
		{
			OTLPEndpoint: "http://localhost:4317",
			SamplingRate: 0.1,
			Platform:     tracing.PlatformAWS,
		},
	} {
		s.Assert().NoError(settings.Validate(), "%+v", settings)
	}
}

// The exporter keeps its own default endpoint when given a URL it cannot parse, so a
// typo would have the engine export to localhost rather than report anything. Checking
// the endpoint here is the only thing standing between an operator and that.
func (s *SettingsSuite) Test_refuses_an_endpoint_the_exporter_would_quietly_replace() {
	for _, endpoint := range []string{
		"",
		"localhost:4317",
		"htp://localhost:4317",
		"localhsot:4317",
		"not a collector",
		"http://",
	} {
		settings := s.valid()
		settings.OTLPEndpoint = endpoint

		err := settings.Validate()

		s.Require().Error(err, "%q should be refused", endpoint)
		s.Assert().Contains(err.Error(), "tracing.otlp_endpoint")
	}
}

// Platforms other than AWS reach the engine's traces through a collector that reads W3C
// trace context, so there is nothing for the engine to do differently for them yet.
func (s *SettingsSuite) Test_refuses_a_platform_it_has_nothing_to_do_differently_for() {
	settings := s.valid()
	settings.Platform = "gcp"

	err := settings.Validate()

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "tracing.platform")
	s.Assert().Contains(err.Error(), "gcp")
}

// A rate outside the range is not a rate, and the sampler would silently read it as one
// end of it.
func (s *SettingsSuite) Test_refuses_a_sampling_rate_that_is_not_a_proportion() {
	for _, rate := range []float64{-0.1, 1.5} {
		settings := s.valid()
		settings.SamplingRate = rate

		err := settings.Validate()

		s.Require().Error(err, "%g should be refused", rate)
		s.Assert().Contains(err.Error(), "tracing.sampling_rate")
	}
}

func (s *SettingsSuite) Test_reports_whether_identifiers_have_to_suit_xray() {
	s.Assert().True(tracing.Settings{Platform: tracing.PlatformAWS}.IsAWS())
	s.Assert().False(tracing.Settings{}.IsAWS())
}

// A sampling rate of 0 still exports a trace the caller sampled, because the decision
// is made once per trace by whoever started it. Without that a sampled request would
// produce a trace with holes in it wherever it passed through an engine of its own.
func (s *SettingsSuite) Test_exports_a_trace_the_caller_sampled_whatever_the_rate() {
	settings := s.valid()
	settings.SamplingRate = 0

	exported := s.spanIsRecorded(settings, w3cTraceParent)

	s.Assert().True(exported, "a trace the caller sampled should be exported")
}

func (s *SettingsSuite) Test_exports_nothing_it_started_itself_at_a_rate_of_zero() {
	settings := s.valid()
	settings.SamplingRate = 0

	s.Assert().False(s.spanIsRecorded(settings, ""))
}

// A rate of 1 is what a single-tenant deployment sets, where every operation matters more
// than the volume. It is not the engine's default, which core asserts.
func (s *SettingsSuite) Test_exports_everything_it_started_itself_at_a_rate_of_one() {
	settings := s.valid()
	settings.SamplingRate = 1

	s.Assert().True(s.spanIsRecorded(settings, ""))
}

// A rate between the two ends samples on the trace ID, which is the path a deployment
// with enough traffic to care about volume actually runs. Asserted as a proportion over
// many traces rather than per trace, since which traces are kept is decided by their
// identifiers.
func (s *SettingsSuite) Test_exports_a_share_of_what_it_started_at_a_fractional_rate() {
	settings := s.valid()
	settings.SamplingRate = 0.5

	installed := s.install(settings)
	recorded := 0
	const traces = 400
	for range traces {
		_, span := telemetry.CurrentTracer().Start(context.Background(), "deployment")
		if span.Context().Recorded() {
			recorded++
		}
		span.End()
	}

	s.Assert().NotNil(installed)
	s.Assert().Greater(recorded, 0, "a fractional rate should export some traces")
	s.Assert().Less(recorded, traces, "a fractional rate should decline some traces")
}

// A trace the caller declined is not exported, whatever this engine's own rate is.
func (s *SettingsSuite) Test_declines_a_trace_the_caller_declined() {
	notSampled := "00-" + w3cTraceID + "-00f067aa0ba902b7-00"

	s.Assert().False(s.spanIsRecorded(s.valid(), notSampled))
}

// Whether a span started under the given incoming trace, if any, is one a backend will
// have.
func (s *SettingsSuite) spanIsRecorded(
	settings tracing.Settings,
	traceParent string,
) bool {
	installed := s.install(settings)

	ctx := context.Background()
	if traceParent != "" {
		ctx = installed.Extract(ctx, propagation.HeaderCarrier(http.Header{
			"Traceparent": []string{traceParent},
		}))
	}

	_, span := telemetry.CurrentTracer().Start(ctx, "deployment")
	defer span.End()

	return span.Context().Recorded()
}

// Which propagators are installed, and which of them wins, is the difference between a
// trace that spans the caller and the engine and two unrelated traces.
func (s *SettingsSuite) Test_reads_w3c_trace_context_everywhere() {
	ctx := s.extractWith(s.valid(), http.Header{
		"Traceparent": []string{w3cTraceParent},
	})

	s.Assert().Equal(w3cTraceID, ctx.TraceID().String())
}

func (s *SettingsSuite) Test_ignores_an_xray_header_off_aws() {
	ctx := s.extractWith(s.valid(), http.Header{
		tracing.XRayTraceIDHeader: []string{xrayTraceHeader},
	})

	s.Assert().False(ctx.IsValid(), "X-Ray is not propagated unless the platform is AWS")
}

func (s *SettingsSuite) Test_reads_an_xray_header_on_aws() {
	ctx := s.extractWith(s.aws(), http.Header{
		tracing.XRayTraceIDHeader: []string{xrayTraceHeader},
	})

	s.Assert().Equal(xrayTraceIDAsOtel, ctx.TraceID().String())
}

// A request carrying both is one whose caller started a W3C trace and then passed
// through AWS, so the caller's trace is the one to join.
func (s *SettingsSuite) Test_prefers_w3c_where_a_request_carries_both() {
	ctx := s.extractWith(s.aws(), http.Header{
		"Traceparent":             []string{w3cTraceParent},
		tracing.XRayTraceIDHeader: []string{xrayTraceHeader},
	})

	s.Assert().Equal(w3cTraceID, ctx.TraceID().String())
}

// Baggage has to survive either way, since it is how a caller labels a trace with what
// it is for.
func (s *SettingsSuite) Test_reads_baggage_on_either_platform() {
	for _, settings := range []tracing.Settings{s.valid(), s.aws()} {
		installed := s.install(settings)
		carried := installed.Extract(
			context.Background(),
			propagation.HeaderCarrier(http.Header{"Baggage": []string{"deployment=one"}}),
		)

		s.Assert().Equal("one", baggage.FromContext(carried).Member("deployment").Value())
	}
}

// Installs the propagators the settings ask for, by building a provider, and reports
// what they then read from the given headers.
func (s *SettingsSuite) extractWith(
	settings tracing.Settings,
	headers http.Header,
) oteltrace.SpanContext {
	return oteltrace.SpanContextFromContext(s.install(settings).Extract(
		context.Background(),
		propagation.HeaderCarrier(headers),
	))
}

func (s *SettingsSuite) install(settings tracing.Settings) propagation.TextMapPropagator {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	s.T().Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
		telemetry.SetTracer(nil)
	})

	shutdown, err := tracing.Setup(
		context.Background(),
		settings,
		&testutils.MockClock{StaticTime: time.Now()},
	)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		// Bounded, and its error ignored, because no collector is listening: the flush
		// this attempts cannot succeed and its failure says nothing about the test.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_ = shutdown(ctx)
	})

	return otel.GetTextMapPropagator()
}

const (
	w3cTraceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
	w3cTraceParent = "00-" + w3cTraceID + "-00f067aa0ba902b7-01"

	xrayTraceIDAsOtel = "5759e988bd862e3fe1be46a994272793"
	xrayTraceHeader   = "Root=1-5759e988-bd862e3fe1be46a994272793;" +
		"Parent=53995c3f42cd8ad8;Sampled=1"
)

func (s *SettingsSuite) aws() tracing.Settings {
	settings := s.valid()
	settings.Platform = tracing.PlatformAWS

	return settings
}

// Settings the validator accepts, for a test that is about something other than
// validation. Deliberately not the engine's shipped defaults, which are asserted in the
// core package against the configuration that produces them.
func (s *SettingsSuite) valid() tracing.Settings {
	return tracing.Settings{
		OTLPEndpoint: "http://localhost:4317",
		ServiceName:  "bluelink-deploy-engine",
		SamplingRate: 1,
	}
}

func TestSettingsSuite(t *testing.T) {
	suite.Run(t, new(SettingsSuite))
}
