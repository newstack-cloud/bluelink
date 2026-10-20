package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/stretchr/testify/suite"
)

// The state sink is what makes the trail queryable, so what matters is that a record
// reaches the store intact and that a store failure is reported rather than swallowed.
type StateSinkSuite struct {
	suite.Suite
}

func (s *StateSinkSuite) Test_persists_a_record_through_the_store() {
	store := &stubAuditStore{}
	sink := NewStateSink(store)

	record := &Record{
		ID:               "record-1",
		Action:           "instance.destroy",
		Outcome:          OutcomeFailure,
		PrincipalSubject: "sha256:abcd",
	}
	s.Require().NoError(sink.Write(context.Background(), record))

	s.Require().Len(store.saved, 1)
	s.Assert().Equal(record, store.saved[0])
}

func (s *StateSinkSuite) Test_reports_a_failure_to_persist() {
	store := &stubAuditStore{saveErr: errors.New("state backend unreachable")}
	sink := NewStateSink(store)

	err := sink.Write(context.Background(), &Record{ID: "record-1"})
	s.Require().Error(err)
	s.Assert().ErrorContains(err, "state backend unreachable")
}

func TestStateSinkSuite(t *testing.T) {
	suite.Run(t, new(StateSinkSuite))
}

// A record must still reach the sinks that are working when one is not, since the
// destinations cover different failures and neither substitutes for the other.
type MultiSinkSuite struct {
	suite.Suite
}

func (s *MultiSinkSuite) Test_writes_a_record_to_every_sink() {
	first := &collectingSink{}
	second := &collectingSink{}

	sink := NewMultiSink(first, second)
	s.Require().NoError(sink.Write(context.Background(), &Record{ID: "record-1"}))

	s.Require().Len(first.records, 1)
	s.Require().Len(second.records, 1)
}

func (s *MultiSinkSuite) Test_writes_to_the_remaining_sinks_when_one_fails() {
	working := &collectingSink{}

	sink := NewMultiSink(&failingSink{}, working)
	err := sink.Write(context.Background(), &Record{ID: "record-1"})

	s.Require().Error(err)
	s.Assert().Len(working.records, 1)
}

func (s *MultiSinkSuite) Test_reports_every_failure_together() {
	sink := NewMultiSink(
		&namedFailingSink{name: "first"},
		&namedFailingSink{name: "second"},
	)

	err := sink.Write(context.Background(), &Record{ID: "record-1"})
	s.Require().Error(err)
	s.Assert().ErrorContains(err, "first")
	s.Assert().ErrorContains(err, "second")
}

func (s *MultiSinkSuite) Test_closes_every_sink() {
	first := &closeTrackingSink{}
	second := &closeTrackingSink{}

	s.Require().NoError(NewMultiSink(first, second).Close())
	s.Assert().True(first.closed)
	s.Assert().True(second.closed)
}

func TestMultiSinkSuite(t *testing.T) {
	suite.Run(t, new(MultiSinkSuite))
}

type stubAuditStore struct {
	saved   []*manage.AuditRecord
	saveErr error
}

func (s *stubAuditStore) Save(_ context.Context, record *manage.AuditRecord) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = append(s.saved, record)
	return nil
}

func (s *stubAuditStore) Get(_ context.Context, id string) (*manage.AuditRecord, error) {
	return nil, manage.AuditRecordNotFoundError(id)
}

func (s *stubAuditStore) Query(
	_ context.Context,
	_ *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	return s.saved, nil
}

func (s *stubAuditStore) Cleanup(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

type namedFailingSink struct {
	name string
}

func (s *namedFailingSink) Write(_ context.Context, _ *Record) error {
	return errors.New(s.name + " sink failed")
}

func (s *namedFailingSink) Close() error { return nil }

type closeTrackingSink struct {
	closed bool
}

func (s *closeTrackingSink) Write(_ context.Context, _ *Record) error { return nil }

func (s *closeTrackingSink) Close() error {
	s.closed = true
	return nil
}
