package telemetry_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/telemetry"
	"github.com/stretchr/testify/suite"
)

// The seam has to be usable whether or not anything is installed behind it, since a
// deployment that traces nothing runs the same code as one that does.
type TracerSuite struct {
	suite.Suite
}

func (s *TracerSuite) TearDownTest() {
	telemetry.SetTracer(nil)
}

// Nothing installed is the default, and it must not be something a call site has to
// check for.
func (s *TracerSuite) Test_tracing_without_a_tracer_installed_does_nothing() {
	s.Require().NotNil(telemetry.CurrentTracer())

	ran := false
	err := telemetry.TracedCall(
		context.Background(),
		"cleanup.run",
		[]telemetry.Attr{telemetry.StringAttr("cleanupType", "events")},
		func(ctx context.Context, span telemetry.Span) error {
			ran = true
			span.SetAttributes(telemetry.IntAttr("itemsDeleted", 3))
			span.RecordError(errors.New("ignored"))
			s.Assert().False(span.Context().Recorded())
			return nil
		},
	)

	s.Require().NoError(err)
	s.Assert().True(ran, "the work still runs when nothing is tracing it")
}

func (s *TracerSuite) Test_records_a_span_for_traced_work() {
	recorder := telemetry.NewRecorder()
	telemetry.SetTracer(recorder)

	result, err := telemetry.Traced(
		context.Background(),
		"cleanup.run",
		[]telemetry.Attr{telemetry.StringAttr("cleanupType", "audit_records")},
		func(ctx context.Context, span telemetry.Span) (int64, error) {
			span.SetAttributes(telemetry.IntAttr("itemsDeleted", 7))
			return 7, nil
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(int64(7), result)

	span, found := recorder.Span("cleanup.run")
	s.Require().True(found)
	s.Assert().True(span.Ended, "a span that is never ended is never exported")
	s.Assert().NoError(span.Err)

	cleanupType, _ := span.Attr("cleanupType")
	s.Assert().Equal("audit_records", cleanupType)
	itemsDeleted, _ := span.Attr("itemsDeleted")
	s.Assert().Equal(int64(7), itemsDeleted, "detail known only after the work must land")
}

// An error is what a backend filters on, so it has to reach the span without every
// caller remembering to record it.
func (s *TracerSuite) Test_records_an_error_the_work_returned() {
	recorder := telemetry.NewRecorder()
	telemetry.SetTracer(recorder)

	failure := errors.New("the state backend is unreachable")
	err := telemetry.TracedCall(
		context.Background(),
		"cleanup.run",
		nil,
		func(context.Context, telemetry.Span) error { return failure },
	)

	s.Require().ErrorIs(err, failure)
	span, found := recorder.Span("cleanup.run")
	s.Require().True(found)
	s.Assert().ErrorIs(span.Err, failure)
	s.Assert().True(span.Ended)
}

// Work nested inside traced work becomes a child, which is what makes a trace a tree
// rather than a list.
func (s *TracerSuite) Test_nested_work_becomes_a_child_span() {
	recorder := telemetry.NewRecorder()
	telemetry.SetTracer(recorder)

	err := telemetry.TracedCall(
		context.Background(),
		"cleanup.pass",
		nil,
		func(ctx context.Context, _ telemetry.Span) error {
			return telemetry.TracedCall(ctx, "cleanup.run", nil,
				func(context.Context, telemetry.Span) error { return nil })
		},
	)
	s.Require().NoError(err)

	parent, found := recorder.Span("cleanup.pass")
	s.Require().True(found)
	child, found := recorder.Span("cleanup.run")
	s.Require().True(found)
	s.Assert().Equal(parent.SpanID, child.ParentID)
}

// A span is ended even when the work panics, or the span is never exported and the
// trace has a hole where the failure was.
func (s *TracerSuite) Test_a_span_is_ended_when_the_work_panics() {
	recorder := telemetry.NewRecorder()
	telemetry.SetTracer(recorder)

	s.Require().Panics(func() {
		_ = telemetry.TracedCall(
			context.Background(),
			"cleanup.run",
			nil,
			func(context.Context, telemetry.Span) error { panic("gave up") },
		)
	})

	span, found := recorder.Span("cleanup.run")
	s.Require().True(found)
	s.Assert().True(span.Ended)
}

func (s *TracerSuite) Test_setting_a_nil_tracer_restores_the_one_that_does_nothing() {
	telemetry.SetTracer(telemetry.NewRecorder())
	telemetry.SetTracer(nil)

	_, span := telemetry.CurrentTracer().Start(context.Background(), "cleanup.run")
	s.Assert().False(span.Context().Recorded())
}

// The tracer is read on every span and written at startup, so reading it has to be safe
// while it is being installed.
func (s *TracerSuite) Test_is_safe_to_use_while_a_tracer_is_installed() {
	var waitGroup sync.WaitGroup
	for range 20 {
		waitGroup.Go(func() {
			_ = telemetry.TracedCall(
				context.Background(),
				"cleanup.run",
				nil,
				func(context.Context, telemetry.Span) error { return nil },
			)
		})
	}
	waitGroup.Go(func() {
		telemetry.SetTracer(telemetry.NewRecorder())
	})
	waitGroup.Wait()
}

func TestTracerSuite(t *testing.T) {
	suite.Run(t, new(TracerSuite))
}
