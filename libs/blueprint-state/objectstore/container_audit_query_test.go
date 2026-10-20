package objectstore_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/stretchr/testify/suite"
)

// A trail is one object per record here, so reading it to answer every query does not
// scale with the trail. The record's timestamp leads its key, which is what lets a query's
// time range, cursor and limit be answered before anything is read.
type ObjectstoreAuditQuerySuite struct {
	objectstoreSuite
}

func (s *ObjectstoreAuditQuerySuite) Test_reads_only_the_records_in_the_time_range() {
	s.saveTrail(60)
	reads := s.countReads()

	records, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		From: 1020,
		To:   1030,
	})
	s.Require().NoError(err)

	s.Assert().Len(records, 10)
	s.Assert().Equal(10, reads.total(), "only the records in range should be read")
}

func (s *ObjectstoreAuditQuerySuite) Test_reads_only_as_far_as_the_limit() {
	s.saveTrail(60)
	reads := s.countReads()

	records, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		Limit: 5,
	})
	s.Require().NoError(err)

	s.Require().Len(records, 5)
	s.Assert().Equal(5, reads.total(), "reading should stop once the limit is satisfied")
	s.Assert().Equal(int64(1000), records[0].Timestamp, "the oldest records come first")
}

// Paging through the trail should cost the page, not the trail, which is what makes the
// cursor worth having.
func (s *ObjectstoreAuditQuerySuite) Test_reads_only_a_page_when_resuming_from_a_cursor() {
	s.saveTrail(60)
	auditStore := s.newContainer().Audit()

	first, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{Limit: 10})
	s.Require().NoError(err)
	s.Require().Len(first, 10)

	last := first[len(first)-1]
	reads := s.countReads()
	second, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{
		AfterTimestamp: last.Timestamp,
		AfterID:        last.ID,
		Limit:          10,
	})
	s.Require().NoError(err)

	s.Require().Len(second, 10)
	s.Assert().Equal(10, reads.total(), "a page should cost a page's worth of reads")
	s.Assert().Greater(second[0].Timestamp, last.Timestamp)
}

// Narrowing by something the key does not carry still has to be correct, even though it
// costs reads.
func (s *ObjectstoreAuditQuerySuite) Test_narrows_by_action_which_the_key_does_not_carry() {
	s.saveTrail(60)

	records, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		Action: "instance.deploy.0007",
	})
	s.Require().NoError(err)

	s.Require().Len(records, 1)
	s.Assert().Equal(int64(1007), records[0].Timestamp)
}

func (s *ObjectstoreAuditQuerySuite) Test_cleanup_removes_expired_records_without_reading_any() {
	s.saveTrail(60)
	auditStore := s.newContainer().Audit()
	reads := s.countReads()

	removed, err := auditStore.Cleanup(context.Background(), time.Unix(1030, 0))
	s.Require().NoError(err)

	s.Assert().Equal(int64(31), removed, "records up to and including the threshold go")
	s.Assert().Zero(reads.total(), "the threshold is answered from the key")

	remaining, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{Limit: 100})
	s.Require().NoError(err)
	s.Assert().Len(remaining, 29)
}

// A record read back by ID has no timestamp to build a key from, so it is found by
// listing. That is a listing and one read, not a read of the trail.
func (s *ObjectstoreAuditQuerySuite) Test_reads_one_record_when_fetching_by_id() {
	s.saveTrail(60)
	reads := s.countReads()

	record, err := s.newContainer().Audit().Get(context.Background(), "record-0042")
	s.Require().NoError(err)

	s.Assert().Equal(int64(1042), record.Timestamp)
	s.Assert().Equal(1, reads.total())
}

// A bounded query should list the slice of the trail its range can touch, not the whole
// trail, which is what the timestamp leading the key buys beyond skipping reads.
func (s *ObjectstoreAuditQuerySuite) Test_lists_only_the_slice_the_range_can_touch() {
	s.saveTrail(60)
	listed := s.capturePrefixes()

	_, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		From: 1040,
		To:   1050,
	})
	s.Require().NoError(err)

	s.Require().NotEmpty(listed.prefixes())
	// 1040 and 1049 pad to 00000001040 and 00000001049, agreeing on all but the last digit.
	s.Assert().Equal(testPrefix+"audit_records/0000000104", listed.prefixes()[0])
}

// Without an upper bound there is no shared prefix, so the whole trail is listed. Stated
// so the narrowing is understood as applying to bounded reads rather than all of them.
func (s *ObjectstoreAuditQuerySuite) Test_lists_the_whole_trail_for_an_unbounded_query() {
	s.saveTrail(60)
	listed := s.capturePrefixes()

	_, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		Limit: 5,
	})
	s.Require().NoError(err)

	s.Require().NotEmpty(listed.prefixes())
	s.Assert().Equal(testPrefix+"audit_records/", listed.prefixes()[0])
}

// Paging within a range narrows as the page advances, since the cursor is a lower bound.
func (s *ObjectstoreAuditQuerySuite) Test_narrows_further_as_a_cursor_advances() {
	s.saveTrail(60)
	listed := s.capturePrefixes()

	_, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		From:           1000,
		To:             1060,
		AfterTimestamp: 1052,
		AfterID:        "record-0052",
	})
	s.Require().NoError(err)

	s.Require().NotEmpty(listed.prefixes())
	// 1052 and 1059 agree on all but the last digit, where 1000 and 1059 agree on less.
	s.Assert().Equal(testPrefix+"audit_records/0000000105", listed.prefixes()[0])
}

// Every key the engine writes leads with the record's timestamp, so one that does not is
// a foreign object or a bug. Answering around it would report part of a trail as the whole
// of it, which is the one thing a read of an audit trail must not do.
func (s *ObjectstoreAuditQuerySuite) Test_refuses_to_read_a_trail_holding_a_key_it_cannot_parse() {
	s.saveTrail(4)
	_, err := s.svc.Put(
		context.Background(),
		testPrefix+"audit_records/not-a-record-key.json",
		[]byte(`{"id":"stray"}`),
		nil,
	)
	s.Require().NoError(err)

	_, err = s.newContainer().Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Limit: 10},
	)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not an audit record key")
}

// Cleanup stops for the same reason: deleting around an object it cannot identify is
// worse than reporting it.
func (s *ObjectstoreAuditQuerySuite) Test_cleanup_refuses_a_key_it_cannot_parse() {
	s.saveTrail(4)
	_, err := s.svc.Put(
		context.Background(),
		testPrefix+"audit_records/not-a-record-key.json",
		[]byte(`{"id":"stray"}`),
		nil,
	)
	s.Require().NoError(err)

	_, err = s.newContainer().Audit().Cleanup(context.Background(), time.Unix(2000, 0))

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not an audit record key")
}

// A lower bound has no upper bound to share digits with, so a prefix cannot narrow it.
// A store that can begin a listing at a key answers it exactly, which is the read an
// extraction paging forward from a checkpoint makes over and over.
func (s *ObjectstoreAuditQuerySuite) Test_lists_from_the_cursor_rather_than_the_whole_trail() {
	s.saveTrail(60)
	starts := s.captureStarts()

	records, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		AfterTimestamp: 1052,
		AfterID:        "record-0052",
		Limit:          10,
	})
	s.Require().NoError(err)

	s.Require().Len(records, 7)
	s.Assert().Equal("record-0053", records[0].ID)
	s.Require().Len(starts.seen(), 1)
	s.Assert().Equal(testPrefix+"audit_records/00000001052", starts.seen()[0])
}

// An unbounded read has nothing to start from, so it lists the prefix as before.
func (s *ObjectstoreAuditQuerySuite) Test_lists_the_prefix_when_there_is_no_lower_bound() {
	s.saveTrail(20)
	starts := s.captureStarts()

	_, err := s.newContainer().Audit().Query(context.Background(), &manage.AuditRecordQuery{
		To: 1010,
	})
	s.Require().NoError(err)

	s.Assert().Empty(starts.seen(), "no lower bound means no start key")
}

func (s *ObjectstoreAuditQuerySuite) saveTrail(count int) {
	auditStore := s.newContainer().Audit()
	for i := range count {
		s.Require().NoError(auditStore.Save(context.Background(), &manage.AuditRecord{
			ID:               fmt.Sprintf("record-%04d", i),
			Timestamp:        int64(1000 + i),
			Action:           fmt.Sprintf("instance.deploy.%04d", i),
			Outcome:          manage.OutcomeSuccess,
			StatusCode:       200,
			Method:           "POST",
			Path:             "/deployments/instances",
			PrincipalMethod:  "apikey",
			PrincipalSubject: "user-a",
		}))
	}
}

type startRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *startRecorder) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.calls...)
}

// Records the keys narrowed listings were asked to start from, so a test can assert that
// a lower-bounded read asked the store for only the part of the trail it needed.
func (s *ObjectstoreAuditQuerySuite) captureStarts() *startRecorder {
	recorder := &startRecorder{}
	s.svc.SetHooks(mockservice.Hooks{
		BeforeListFrom: func(_ context.Context, _ string, startAfter string) error {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			recorder.calls = append(recorder.calls, startAfter)

			return nil
		},
	})

	return recorder
}

type prefixRecorder struct {
	mu   sync.Mutex
	seen []string
}

func (r *prefixRecorder) prefixes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.seen...)
}

// Records the prefixes listings are made under, so a test can assert how much of the
// trail a query asked the store for.
func (s *ObjectstoreAuditQuerySuite) capturePrefixes() *prefixRecorder {
	recorder := &prefixRecorder{}
	s.svc.SetHooks(mockservice.Hooks{
		BeforeList: func(_ context.Context, prefix string) error {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			recorder.seen = append(recorder.seen, prefix)

			return nil
		},
	})

	return recorder
}

type readCounter struct {
	mu    sync.Mutex
	count int
}

func (c *readCounter) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.count
}

// Counts object reads from this point on, so a test can assert what a query cost rather
// than only what it returned.
func (s *ObjectstoreAuditQuerySuite) countReads() *readCounter {
	counter := &readCounter{}
	s.svc.SetHooks(mockservice.Hooks{
		BeforeGet: func(context.Context, string) error {
			counter.mu.Lock()
			defer counter.mu.Unlock()
			counter.count++

			return nil
		},
	})

	return counter
}

func TestObjectstoreAuditQuerySuite(t *testing.T) {
	suite.Run(t, new(ObjectstoreAuditQuerySuite))
}
