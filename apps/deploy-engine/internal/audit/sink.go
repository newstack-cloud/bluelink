package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

// Sink receives audit records.
//
// Writing a record must not fail a request, the action has already happened by the time
// the record is written, so refusing the request afterwards would misreport it. A sink
// reports its error for the caller to log instead.
type Sink interface {
	// Write records one action.
	//
	// The context is for the write itself rather than the request the record describes.
	// The request has been answered by the time a record is written, so a sink is given a
	// context that outlives it.
	Write(ctx context.Context, record *Record) error
	io.Closer
}

// WriterSink writes records as one JSON object per line, which is what every log collector
// already reads.
type WriterSink struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewWriterSink writes records to the given writer.
//
// The destination is the caller's to own and to close: this sink serialises writes to it
// and nothing more.
func NewWriterSink(writer io.Writer) *WriterSink {
	return &WriterSink{writer: writer}
}

// NewStdoutSink writes records to stdout.
//
// The record leaves the process immediately, so whatever already collects and rotates the
// engine's output collects this too. Deliberately not a file: a file on the engine's own
// host is no further from the state it describes than stdout is, and it brings rotation,
// pruning and disk headroom along with it.
//
// Written directly rather than through the engine's logger so that a log level cannot
// discard it. An audit trail that disappears when somebody raises the log level is not one.
func NewStdoutSink() *WriterSink {
	return &WriterSink{writer: os.Stdout}
}

// Write appends one record.
//
// Serialised under the lock so that two requests finishing at once cannot interleave within
// a line, which would leave a collector with two unparseable records rather than two
// records.
func (s *WriterSink) Write(_ context.Context, record *Record) error {
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to serialise an audit record: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.writer.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to write an audit record: %w", err)
	}

	return nil
}

// Close is a no-op as the destination is the caller's to close.
// Present so that a sink whose destination does own a handle
// can be swapped in without changing its callers.
func (s *WriterSink) Close() error {
	return nil
}

// StateSink persists records through the engine's state backend.
//
// This is the sink that makes the trail queryable.
// Records land beside the state they
// describe, the operational endpoints can read them back and a deployment can
// forward them on or extract them from the backend directly.
type StateSink struct {
	audit manage.Audit
}

// NewStateSink writes records to the given audit state container.
func NewStateSink(audit manage.Audit) *StateSink {
	return &StateSink{audit: audit}
}

func (s *StateSink) Write(ctx context.Context, record *Record) error {
	if err := s.audit.Save(ctx, record); err != nil {
		return fmt.Errorf("failed to persist an audit record: %w", err)
	}

	return nil
}

// Close is a no-op as the state backend's lifetime is the engine's to manage,
// not this sink's.
func (s *StateSink) Close() error {
	return nil
}

// MultiSink writes each record to every sink it is given.
//
// Every sink is attempted even after one fails, and the failures are reported together. a
// For example, state backend being unreachable must not also cost the record its line on stdout.
type MultiSink struct {
	sinks []Sink
}

// NewMultiSink writes records to all of the given sinks.
func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

func (s *MultiSink) Write(ctx context.Context, record *Record) error {
	errs := make([]error, 0, len(s.sinks))
	for _, sink := range s.sinks {
		if err := sink.Write(ctx, record); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *MultiSink) Close() error {
	errs := make([]error, 0, len(s.sinks))
	for _, sink := range s.sinks {
		if err := sink.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// NopSink discards records, for a deployment that has turned the trail off.
type NopSink struct{}

func (s *NopSink) Write(_ context.Context, _ *Record) error {
	return nil
}

func (s *NopSink) Close() error {
	return nil
}
