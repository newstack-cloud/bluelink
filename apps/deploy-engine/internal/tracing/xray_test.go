package tracing_test

import (
	"context"
	"encoding/binary"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/tracing"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// X-Ray identifies a trace by a rendering of its ID that no other backend uses, and
// propagates it in a header of its own, so neither can be taken from a library.
type XRaySuite struct {
	suite.Suite
}

func (s *XRaySuite) Test_renders_a_trace_id_in_the_form_xray_identifies_a_trace_by() {
	s.Assert().Equal(
		"1-5759e988-bd862e3fe1be46a994272793",
		tracing.ToXRayTraceID("5759e988bd862e3fe1be46a994272793"),
	)
}

// An identifier that resolves to nothing reads as though a trace exists, which is worse
// than a log entry that says nothing about tracing.
func (s *XRaySuite) Test_renders_nothing_for_what_is_not_a_trace_id() {
	for _, notATraceID := range []string{
		"",
		"5759e988",
		"5759e988bd862e3fe1be46a9942727931",
		"5759e988bd862e3fe1be46a99427279z",
	} {
		s.Assert().Empty(
			tracing.ToXRayTraceID(notATraceID),
			"%q is not a trace id",
			notATraceID,
		)
	}
}

// The header an AWS load balancer in front of the engine writes.
func (s *XRaySuite) Test_reads_the_trace_a_request_arrived_carrying() {
	extracted := s.extract(
		"Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1",
	)

	s.Require().True(extracted.IsValid())
	s.Assert().Equal("5759e988bd862e3fe1be46a994272793", extracted.TraceID().String())
	s.Assert().Equal("53995c3f42cd8ad8", extracted.SpanID().String())
	s.Assert().True(extracted.IsSampled())
	s.Assert().True(extracted.IsRemote())
}

// AWS adds fields of its own describing the hop rather than the trace, and leaves the
// sampling decision to whatever receives the request when it has not made one.
func (s *XRaySuite) Test_reads_a_trace_alongside_fields_it_does_not_know() {
	extracted := s.extract(
		"Self=1-67891233-abcdef012345678912345678;" +
			"Root=1-5759e988-bd862e3fe1be46a994272793;" +
			"Parent=53995c3f42cd8ad8;" +
			"Sampled=?;" +
			"Lineage=1:2:3",
	)

	s.Require().True(extracted.IsValid())
	s.Assert().Equal("5759e988bd862e3fe1be46a994272793", extracted.TraceID().String())
	s.Assert().False(
		extracted.IsSampled(),
		"a decision AWS left open is not a decision to sample",
	)
}

// OpenTelemetry generates a new trace ID for a trace whose parent span is unknown, so
// joining one would have the engine log an ID the exported spans do not carry.
//
// Asserted on the trace ID rather than on the span context being valid, since a span
// context with no span ID is invalid whether or not the trace ID was read.
func (s *XRaySuite) Test_reads_nothing_from_a_trace_with_no_parent_span() {
	s.Assert().False(
		s.extract("Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=1").
			TraceID().
			IsValid(),
		"a trace with no parent span should not be joined",
	)
}

// A header the engine cannot read is not a trace, so the request starts one of its own
// rather than joining a trace that does not exist.
func (s *XRaySuite) Test_reads_nothing_from_a_header_it_cannot_make_sense_of() {
	for _, header := range []string{
		"Root=2-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8",
		"Root=1-5759e9-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8",
		"Root=1-5759e988;Parent=53995c3f42cd8ad8",
		"Root=not-a-trace-id;Parent=53995c3f42cd8ad8",
		"Root=1-5759e988-bd862e3fe1be46a994272793;Parent=tooshort",
		"Parent=53995c3f42cd8ad8;Sampled=1",
		"nonsense",
	} {
		s.Assert().False(
			s.extract(header).TraceID().IsValid(),
			"%q should not be read as a trace",
			header,
		)
	}
}

func (s *XRaySuite) Test_writes_the_trace_for_whatever_the_engine_calls_next() {
	carrier := propagation.HeaderCarrier(http.Header{})

	tracing.XRayPropagator{}.Inject(
		oteltrace.ContextWithSpanContext(context.Background(), s.sampledSpanContext()),
		carrier,
	)

	s.Assert().Equal(
		"Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1",
		carrier.Get(tracing.XRayTraceIDHeader),
	)
}

// Carrying a trace that is not being recorded would have whatever is called next record
// against a trace no backend has.
func (s *XRaySuite) Test_writes_the_sampling_decision_it_was_given() {
	carrier := propagation.HeaderCarrier(http.Header{})
	notSampled := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: s.sampledSpanContext().TraceID(),
		SpanID:  s.sampledSpanContext().SpanID(),
	})

	tracing.XRayPropagator{}.Inject(
		oteltrace.ContextWithSpanContext(context.Background(), notSampled),
		carrier,
	)

	s.Assert().Contains(carrier.Get(tracing.XRayTraceIDHeader), "Sampled=0")
}

func (s *XRaySuite) Test_writes_nothing_where_there_is_no_span_to_write() {
	carrier := propagation.HeaderCarrier(http.Header{})

	tracing.XRayPropagator{}.Inject(context.Background(), carrier)

	s.Assert().Empty(carrier.Get(tracing.XRayTraceIDHeader))
}

// What the engine writes has to be what it can read back, since a trace commonly passes
// through more than one engine.
func (s *XRaySuite) Test_reads_back_the_trace_it_wrote() {
	carrier := propagation.HeaderCarrier(http.Header{})
	written := s.sampledSpanContext()

	tracing.XRayPropagator{}.Inject(
		oteltrace.ContextWithSpanContext(context.Background(), written),
		carrier,
	)
	read := s.extract(carrier.Get(tracing.XRayTraceIDHeader))

	s.Assert().Equal(written.TraceID(), read.TraceID())
	s.Assert().Equal(written.SpanID(), read.SpanID())
	s.Assert().True(read.IsSampled())
}

func (s *XRaySuite) Test_names_the_header_it_writes() {
	s.Assert().Equal(
		[]string{tracing.XRayTraceIDHeader},
		tracing.XRayPropagator{}.Fields(),
	)
}

// X-Ray rejects a trace whose ID does not begin with the time it started, so this is
// what makes a trace the engine starts itself arrive at all.
func (s *XRaySuite) Test_generates_a_trace_id_beginning_with_the_time_it_started() {
	started := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)

	traceID, spanID := tracing.NewXRayIDGenerator(
		&testutils.MockClock{StaticTime: started},
	).NewIDs(context.Background())

	s.Require().True(traceID.IsValid())
	s.Require().True(spanID.IsValid())
	s.Assert().Equal(
		uint32(started.Unix()),
		binary.BigEndian.Uint32(traceID[:4]),
		"the first four bytes should be the epoch seconds the trace started at",
	)
}

func (s *XRaySuite) Test_generates_a_distinct_id_each_time() {
	generator := tracing.NewXRayIDGenerator(&testutils.MockClock{StaticTime: time.Now()})

	seen := map[string]bool{}
	for range 100 {
		traceID, spanID := generator.NewIDs(context.Background())
		s.Require().False(seen[traceID.String()], "trace ids should not repeat")
		s.Require().False(seen[spanID.String()], "span ids should not repeat")
		seen[traceID.String()] = true
		seen[spanID.String()] = true

		within := generator.NewSpanID(context.Background(), traceID)
		s.Require().True(within.IsValid())
		s.Require().False(seen[within.String()], "span ids should not repeat")
		seen[within.String()] = true
	}
}

// The SDK starts spans from many goroutines, and the generator holds no lock, so this is
// the property that justifies not having one. Run under -race it also catches shared
// mutable state being reintroduced.
func (s *XRaySuite) Test_generates_ids_concurrently() {
	generator := tracing.NewXRayIDGenerator(
		&testutils.MockClock{StaticTime: time.Now()},
	)

	const goroutines, perGoroutine = 8, 50
	results := make(chan oteltrace.TraceID, goroutines*perGoroutine)

	var waitGroup sync.WaitGroup
	for range goroutines {
		waitGroup.Go(func() {
			for range perGoroutine {
				traceID, spanID := generator.NewIDs(context.Background())
				s.Require().True(traceID.IsValid())
				s.Require().True(spanID.IsValid())
				results <- traceID
			}
		})
	}
	waitGroup.Wait()
	close(results)

	seen := map[oteltrace.TraceID]bool{}
	for traceID := range results {
		s.Require().False(seen[traceID], "trace ids should not repeat across goroutines")
		seen[traceID] = true
	}
	s.Assert().Len(seen, goroutines*perGoroutine)
}

func (s *XRaySuite) extract(header string) oteltrace.SpanContext {
	carrier := propagation.HeaderCarrier(http.Header{})
	carrier.Set(tracing.XRayTraceIDHeader, header)

	return oteltrace.SpanContextFromContext(
		tracing.XRayPropagator{}.Extract(context.Background(), carrier),
	)
}

func (s *XRaySuite) sampledSpanContext() oteltrace.SpanContext {
	traceID, err := oteltrace.TraceIDFromHex("5759e988bd862e3fe1be46a994272793")
	s.Require().NoError(err)
	spanID, err := oteltrace.SpanIDFromHex("53995c3f42cd8ad8")
	s.Require().NoError(err)

	return oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
	})
}

func TestXRaySuite(t *testing.T) {
	suite.Run(t, new(XRaySuite))
}
