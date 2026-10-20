package memfile

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type MemFileStateContainerAuditSuite struct {
	container *StateContainer
	stateDir  string
	fs        afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerAuditSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// A low max guide file size so that saved records are split across multiple
	// chunk files, which is what the query and cleanup paths have to read back.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container
}

func (s *MemFileStateContainerAuditSuite) Test_saves_and_retrieves_an_audit_record() {
	auditStore := s.container.Audit()
	saved := s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess)
	s.Require().NoError(auditStore.Save(context.Background(), saved))

	retrieved, err := auditStore.Get(context.Background(), "record-1")
	s.Require().NoError(err)
	s.Assert().Equal(saved, retrieved)
}

func (s *MemFileStateContainerAuditSuite) Test_fails_to_retrieve_a_non_existent_audit_record() {
	_, err := s.container.Audit().Get(context.Background(), "missing-record")
	s.Require().Error(err)
	notFoundErr := &manage.AuditRecordNotFound{}
	s.Assert().ErrorAs(err, &notFoundErr)
}

// The ordering is the contract that makes extraction resumable, so it is asserted
// against records that are saved out of order.
func (s *MemFileStateContainerAuditSuite) Test_queries_records_oldest_first() {
	auditStore := s.container.Audit()
	s.saveRecords(
		s.auditRecord("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.update", "user-b", manage.OutcomeFailure),
	)

	records, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal(
		[]string{"record-1", "record-2", "record-3"},
		recordIDs(records),
	)
}

// Several records can share a timestamp, so the ID has to break the tie for a
// forwarder resuming from a timestamp to neither re-ship nor skip any of them.
func (s *MemFileStateContainerAuditSuite) Test_orders_records_sharing_a_timestamp_by_id() {
	s.saveRecords(
		s.auditRecord("record-c", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-a", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-b", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-a", "record-b", "record-c"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) Test_queries_records_within_a_time_range() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{From: 2000, To: 3000},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-2"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) Test_queries_records_for_an_action() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Action: "instance.destroy"},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-2"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) Test_queries_records_for_a_principal() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.create", "user-b", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{PrincipalSubject: "user-b"},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-2"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) Test_queries_records_for_an_outcome() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.create", "user-a", manage.OutcomeFailure),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Outcome: manage.OutcomeFailure},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-2"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) Test_limits_a_query_to_the_oldest_matching_records() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Limit: 2},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-1", "record-2"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) Test_cleans_up_records_older_than_the_threshold() {
	auditStore := s.container.Audit()
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	removed, err := auditStore.Cleanup(context.Background(), time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), removed)

	records, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-3"}, recordIDs(records))
}

// A trail that only exists until the process restarts is not a trail, so the
// records are read back through a container loaded afresh from the same files.
func (s *MemFileStateContainerAuditSuite) Test_records_survive_a_reload() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.destroy", "user-b", manage.OutcomeFailure),
	)

	reloaded, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	records, err := reloaded.Audit().Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-1", "record-2"}, recordIDs(records))

	retrieved, err := reloaded.Audit().Get(context.Background(), "record-2")
	s.Require().NoError(err)
	s.Assert().Equal("instance.destroy", retrieved.Action)
	s.Assert().Equal("user-b", retrieved.PrincipalSubject)
}

func (s *MemFileStateContainerAuditSuite) Test_resumes_a_listing_after_a_cursor() {
	s.saveRecords(
		s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-3", 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{AfterTimestamp: 2000, AfterID: "record-2"},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-3"}, recordIDs(records))
}

// Without an ID in the cursor a forwarder could never read past a second holding
// more records than the limit, since the limit is applied after ordering.
func (s *MemFileStateContainerAuditSuite) Test_pages_through_records_sharing_a_timestamp() {
	s.saveRecords(
		s.auditRecord("record-a", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-b", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-c", 1000, "instance.create", "user-a", manage.OutcomeSuccess),
	)
	auditStore := s.container.Audit()

	first, err := auditStore.Query(
		context.Background(),
		&manage.AuditRecordQuery{Limit: 2},
	)
	s.Require().NoError(err)
	s.Require().Equal([]string{"record-a", "record-b"}, recordIDs(first))

	last := first[len(first)-1]
	second, err := auditStore.Query(
		context.Background(),
		&manage.AuditRecordQuery{
			Limit:          2,
			AfterTimestamp: last.Timestamp,
			AfterID:        last.ID,
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-c"}, recordIDs(second))
}

// The trail is append-only, so a reused ID must leave the stored record as it is
// rather than amend it or put the same ID in the trail twice.
func (s *MemFileStateContainerAuditSuite) Test_does_not_store_a_reused_id_twice() {
	auditStore := s.container.Audit()
	original := s.auditRecord("record-1", 1000, "instance.create", "user-a", manage.OutcomeSuccess)
	s.Require().NoError(auditStore.Save(context.Background(), original))

	reused := s.auditRecord("record-1", 2000, "instance.destroy", "user-b", manage.OutcomeFailure)
	s.Require().NoError(auditStore.Save(context.Background(), reused))

	records, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Assert().Equal(original, records[0])

	// Get reads the in-memory lookup and Query the chunk files, so the two have to
	// agree on which record is stored.
	retrieved, err := auditStore.Get(context.Background(), "record-1")
	s.Require().NoError(err)
	s.Assert().Equal(original, retrieved)
}

// The boundary is asserted because it is where the backends could disagree: a
// record whose timestamp equals the threshold is removed, not kept.
func (s *MemFileStateContainerAuditSuite) Test_cleans_up_a_record_on_the_threshold() {
	auditStore := s.container.Audit()
	s.saveRecords(
		s.auditRecord("record-1", 2000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord("record-2", 2001, "instance.update", "user-a", manage.OutcomeSuccess),
	)

	removed, err := auditStore.Cleanup(context.Background(), time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), removed)

	records, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-2"}, recordIDs(records))
}

func (s *MemFileStateContainerAuditSuite) saveRecords(records ...*manage.AuditRecord) {
	for _, record := range records {
		s.Require().NoError(s.container.Audit().Save(context.Background(), record))
	}
}

func (s *MemFileStateContainerAuditSuite) auditRecord(
	id string,
	timestamp int64,
	action string,
	principalSubject string,
	outcome string,
) *manage.AuditRecord {
	return &manage.AuditRecord{
		ID:               id,
		Timestamp:        timestamp,
		Action:           action,
		Target:           fmt.Sprintf("target-for-%s", id),
		Outcome:          outcome,
		StatusCode:       200,
		Method:           "POST",
		Path:             "/deployments/instances",
		PrincipalMethod:  "apikey",
		PrincipalSubject: principalSubject,
		SourceIP:         "10.0.0.1",
		RequestID:        fmt.Sprintf("request-for-%s", id),
		TraceID:          fmt.Sprintf("trace-for-%s", id),
		SpanID:           fmt.Sprintf("span-for-%s", id),
	}
}

func recordIDs(records []*manage.AuditRecord) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = record.ID
	}
	return ids
}

func TestMemFileStateContainerAuditSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerAuditSuite))
}
