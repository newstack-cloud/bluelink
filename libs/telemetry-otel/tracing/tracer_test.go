package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/telemetry"
	"github.com/newstack-cloud/bluelink/libs/telemetry-otel/tracing"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// What the engine records through the seam has to arrive at a backend as the span it
// described, since every call site states its intent in the seam's terms and never in
// OpenTelemetry's.
type TracerSuite struct {
	suite.Suite

	recorder *tracetest.SpanRecorder
	previous oteltrace.TracerProvider
}

func (s *TracerSuite) SetupTest() {
	s.recorder = tracetest.NewSpanRecorder()
	s.previous = otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(s.recorder),
	))
	telemetry.SetTracer(tracing.NewTracer())
}

func (s *TracerSuite) TearDownTest() {
	telemetry.SetTracer(nil)
	otel.SetTracerProvider(s.previous)
}

func (s *TracerSuite) Test_records_the_work_it_was_given() {
	_, err := telemetry.Traced(
		context.Background(),
		"cleanup.task",
		[]telemetry.Attr{telemetry.StringAttr("cleanupType", "events")},
		func(_ context.Context, span telemetry.Span) (int, error) {
			span.SetAttributes(telemetry.IntAttr("itemsDeleted", 7))
			return 7, nil
		},
	)
	s.Require().NoError(err)

	recorded := s.oneSpan()
	s.Assert().Equal("cleanup.task", recorded.Name())
	s.Assert().Equal(codes.Unset, recorded.Status().Code)
	s.Assert().Contains(recorded.Attributes(), attribute.String("cleanupType", "events"))
	s.Assert().Contains(recorded.Attributes(), attribute.Int64("itemsDeleted", 7))
}

// The seam fixes each attribute's type so that a backend is given a number as a number
// rather than as whatever its printed form happened to look like.
func (s *TracerSuite) Test_records_each_kind_of_attribute_as_its_own_type() {
	s.trace(func(span telemetry.Span) {
		span.SetAttributes(
			telemetry.StringAttr("text", "value"),
			telemetry.IntAttr("count", 3),
			telemetry.FloatAttr("rate", 0.25),
			telemetry.BoolAttr("skipped", true),
		)
	})

	attrs := s.oneSpan().Attributes()
	s.Assert().Contains(attrs, attribute.String("text", "value"))
	s.Assert().Contains(attrs, attribute.Int64("count", 3))
	s.Assert().Contains(attrs, attribute.Float64("rate", 0.25))
	s.Assert().Contains(attrs, attribute.Bool("skipped", true))
}

// A span missing an attribute is harder to account for than one carrying a string.
func (s *TracerSuite) Test_records_a_value_of_an_unexpected_type_as_its_printed_form() {
	s.trace(func(span telemetry.Span) {
		span.SetAttributes(telemetry.Attr{Key: "threshold", Value: 42})
	})

	s.Assert().Contains(s.oneSpan().Attributes(), attribute.String("threshold", "42"))
}

// Recording the error and marking the span failed say different things, and a backend
// filters failed spans by the status rather than by the events on them.
func (s *TracerSuite) Test_marks_failed_work_as_failed_and_says_what_failed() {
	failure := errors.New("the state backend refused the delete")

	_, err := telemetry.Traced(
		context.Background(),
		"cleanup.task",
		nil,
		func(context.Context, telemetry.Span) (int, error) { return 0, failure },
	)
	s.Require().ErrorIs(err, failure)

	recorded := s.oneSpan()
	s.Assert().Equal(codes.Error, recorded.Status().Code)
	s.Assert().Equal(failure.Error(), recorded.Status().Description)
	s.Require().Len(recorded.Events(), 1)
	s.Assert().Equal("exception", recorded.Events()[0].Name)
}

// Work nested inside other work belongs to the same trace, which is the whole point of
// handing the context back.
func (s *TracerSuite) Test_work_started_within_a_span_belongs_to_the_same_trace() {
	err := telemetry.TracedCall(
		context.Background(),
		"cleanup.pass",
		nil,
		func(ctx context.Context, _ telemetry.Span) error {
			return telemetry.TracedCall(
				ctx,
				"cleanup.task",
				nil,
				func(context.Context, telemetry.Span) error { return nil },
			)
		},
	)
	s.Require().NoError(err)

	spans := s.recorder.Ended()
	s.Require().Len(spans, 2)
	task, pass := spans[0], spans[1]
	s.Require().Equal("cleanup.task", task.Name())
	s.Require().Equal("cleanup.pass", pass.Name())
	s.Assert().Equal(pass.SpanContext().TraceID(), task.SpanContext().TraceID())
	s.Assert().Equal(pass.SpanContext().SpanID(), task.Parent().SpanID())
}

// This is what a log entry is correlated against, so it has to be the span that was
// actually recorded.
func (s *TracerSuite) Test_reports_the_identity_of_the_span_it_recorded() {
	var reported telemetry.SpanContext
	var fromContext telemetry.SpanContext

	err := telemetry.TracedCall(
		context.Background(),
		"cleanup.task",
		nil,
		func(ctx context.Context, span telemetry.Span) error {
			reported = span.Context()
			fromContext = tracing.SpanContextFromContext(ctx)
			return nil
		},
	)
	s.Require().NoError(err)

	recorded := s.oneSpan()
	s.Require().True(reported.Recorded())
	s.Assert().Equal(recorded.SpanContext().TraceID().String(), reported.TraceID)
	s.Assert().Equal(recorded.SpanContext().SpanID().String(), reported.SpanID)
	s.Assert().Equal(
		reported,
		fromContext,
		"the span in the context must be the one that was handed over",
	)
}

func (s *TracerSuite) Test_reports_no_span_where_the_context_carries_none() {
	s.Assert().False(
		tracing.SpanContextFromContext(context.Background()).Recorded(),
	)
}

// A deployment runs for minutes to hours after the request that asked for it has been
// answered, so it begins a trace of its own rather than becoming a child of a span that
// ended long before it.
func (s *TracerSuite) Test_root_work_begins_a_trace_of_its_own() {
	ctx, request := telemetry.CurrentTracer().Start(context.Background(), "request")
	asked := request.Context()

	_, operation := telemetry.CurrentTracer().StartRoot(ctx, "deployment", asked)
	operation.End()
	request.End()

	recorded := s.spanNamed("deployment")
	s.Assert().False(recorded.Parent().IsValid(), "a root span should have no parent")
	s.Assert().NotEqual(
		asked.TraceID,
		recorded.SpanContext().TraceID().String(),
		"a root span should not continue the trace it was started from",
	)
}

// The link is what leads from the request to the trace of the work it started, which is
// the part parenting would otherwise have expressed.
func (s *TracerSuite) Test_root_work_links_to_what_asked_for_it() {
	ctx, request := telemetry.CurrentTracer().Start(context.Background(), "request")
	asked := request.Context()

	_, operation := telemetry.CurrentTracer().StartRoot(ctx, "deployment", asked)
	operation.End()
	request.End()

	links := s.spanNamed("deployment").Links()
	s.Require().Len(links, 1, "a root span should link to the span that asked for it")
	s.Assert().Equal(asked.TraceID, links[0].SpanContext.TraceID().String())
	s.Assert().Equal(asked.SpanID, links[0].SpanContext.SpanID().String())
}

// An operation started by a scheduled or internal path has nothing to link to, and a
// link to a span no backend holds leads a reader nowhere.
func (s *TracerSuite) Test_root_work_links_to_nothing_where_nothing_asked_for_it() {
	_, operation := telemetry.CurrentTracer().StartRoot(
		context.Background(),
		"deployment",
		telemetry.SpanContext{},
	)
	operation.End()

	s.Assert().Empty(s.oneSpan().Links())
}

// The sampler classifies on this attribute, so a root span that did not carry it would
// be sampled as though it were a request.
func (s *TracerSuite) Test_root_work_is_marked_as_an_operation() {
	_, operation := telemetry.CurrentTracer().StartRoot(
		context.Background(),
		"deployment",
		telemetry.SpanContext{},
	)
	operation.End()

	s.Assert().Contains(
		s.oneSpan().Attributes(),
		attribute.String(tracing.TraceKindAttr, tracing.TraceKindOperation),
	)
}

func (s *TracerSuite) trace(record func(telemetry.Span)) {
	err := telemetry.TracedCall(
		context.Background(),
		"cleanup.task",
		nil,
		func(_ context.Context, span telemetry.Span) error {
			record(span)
			return nil
		},
	)
	s.Require().NoError(err)
}

func (s *TracerSuite) spanNamed(name string) sdktrace.ReadOnlySpan {
	for _, ended := range s.recorder.Ended() {
		if ended.Name() == name {
			return ended
		}
	}

	s.Require().FailNow("no span was recorded under the name " + name)

	return nil
}

func (s *TracerSuite) oneSpan() sdktrace.ReadOnlySpan {
	ended := s.recorder.Ended()
	s.Require().Len(ended, 1, "exactly one span should have been recorded")

	return ended[0]
}

func TestTracerSuite(t *testing.T) {
	suite.Run(t, new(TracerSuite))
}
