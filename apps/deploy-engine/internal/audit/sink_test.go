package audit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Records go out as one JSON object per line, which is what a log collector already reads,
// so collection and retention are the collector's rather than the engine's.
type WriterSinkSuite struct {
	suite.Suite
}

func (s *WriterSinkSuite) Test_writes_one_record_per_line() {
	var buffer bytes.Buffer
	sink := NewWriterSink(&buffer)

	s.Require().NoError(sink.Write(context.Background(), &Record{
		Action:  "instance.create",
		Outcome: OutcomeSuccess,
	}))
	s.Require().NoError(sink.Write(context.Background(), &Record{
		Action:  "instance.destroy",
		Outcome: OutcomeSuccess,
	}))

	records := s.readRecords(&buffer)
	s.Require().Len(records, 2)
	s.Assert().Equal("instance.create", records[0].Action)
	s.Assert().Equal("instance.destroy", records[1].Action)
}

// Two requests finishing at once must not interleave within a line, which would leave a
// collector with unparseable output rather than records.
func (s *WriterSinkSuite) Test_writes_whole_lines_under_concurrent_writers() {
	var buffer bytes.Buffer
	sink := NewWriterSink(&buffer)

	var waitGroup sync.WaitGroup
	for index := range 50 {
		waitGroup.Go(func() {
			s.Assert().NoError(sink.Write(context.Background(), &Record{
				Action: "instance.create",
				Target: strings.Repeat(fmt.Sprintf("%d", index), 256),
			}))
		})
	}
	waitGroup.Wait()

	s.Assert().Len(s.readRecords(&buffer), 50)
}

// Stdout is not the sink's to close, and a sink whose destination does own a handle has to
// be swappable without changing its callers.
func (s *WriterSinkSuite) Test_closing_a_stdout_sink_does_nothing() {
	s.Assert().NoError(NewStdoutSink().Close())
}

func (s *WriterSinkSuite) readRecords(reader io.Reader) []*Record {
	records := []*Record{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		record := &Record{}
		s.Require().NoError(json.Unmarshal(scanner.Bytes(), record))
		records = append(records, record)
	}
	s.Require().NoError(scanner.Err())

	return records
}

func TestWriterSinkSuite(t *testing.T) {
	suite.Run(t, new(WriterSinkSuite))
}
