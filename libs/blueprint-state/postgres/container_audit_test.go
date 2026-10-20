package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type PostgresAuditTestSuite struct {
	container *StateContainer
	connPool  *pgxpool.Pool
	suite.Suite
}

func (s *PostgresAuditTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

// Each test saves its own records and removes them again, so that the trail one
// test writes is not part of what the next one queries.
func (s *PostgresAuditTestSuite) TearDownTest() {
	_, err := s.connPool.Exec(context.Background(), "DELETE FROM audit_records")
	s.Assert().NoError(err)
	s.connPool.Close()
}

func (s *PostgresAuditTestSuite) Test_saves_and_retrieves_an_audit_record() {
	auditStore := s.container.Audit()
	saved := s.auditRecord(
		"9c0e4a92-3b4a-4a4e-8f6f-5c4d2b1a0001",
		1000,
		"instance.create",
		"user-a",
		manage.OutcomeSuccess,
	)
	s.Require().NoError(auditStore.Save(context.Background(), saved))

	retrieved, err := auditStore.Get(context.Background(), saved.ID)
	s.Require().NoError(err)
	s.Assert().Equal(saved, retrieved)
}

// An audited action has to lead to the trace of the work it did, which means the trace
// identifiers survive a round trip through the columns that hold them.
func (s *PostgresAuditTestSuite) Test_saves_and_retrieves_the_trace_an_action_ran_under() {
	auditStore := s.container.Audit()
	saved := s.auditRecord(
		"9c0e4a92-3b4a-4a4e-8f6f-5c4d2b1a0091",
		1000,
		"instance.deploy",
		"user-a",
		manage.OutcomeSuccess,
	)
	s.Require().NoError(auditStore.Save(context.Background(), saved))

	retrieved, err := auditStore.Get(context.Background(), saved.ID)
	s.Require().NoError(err)

	s.Assert().Equal(saved.TraceID, retrieved.TraceID)
	s.Assert().Equal(saved.SpanID, retrieved.SpanID)
}

// Tracing is off by default, so most records carry no trace, and an empty identifier
// must come back empty rather than as a string that looks like one.
func (s *PostgresAuditTestSuite) Test_saves_a_record_with_no_trace() {
	auditStore := s.container.Audit()
	saved := s.auditRecord(
		"9c0e4a92-3b4a-4a4e-8f6f-5c4d2b1a0092",
		1000,
		"instance.deploy",
		"user-a",
		manage.OutcomeSuccess,
	)
	saved.TraceID = ""
	saved.SpanID = ""
	s.Require().NoError(auditStore.Save(context.Background(), saved))

	retrieved, err := auditStore.Get(context.Background(), saved.ID)
	s.Require().NoError(err)

	s.Assert().Empty(retrieved.TraceID)
	s.Assert().Empty(retrieved.SpanID)
}

func (s *PostgresAuditTestSuite) Test_fails_to_retrieve_a_non_existent_audit_record() {
	_, err := s.container.Audit().Get(
		context.Background(),
		"0b04331f-de5d-44ce-9b74-0a39a3ef3c50",
	)
	s.Require().Error(err)
	notFoundErr := &manage.AuditRecordNotFound{}
	s.Assert().ErrorAs(err, &notFoundErr)
}

func (s *PostgresAuditTestSuite) Test_queries_records_oldest_first() {
	s.saveRecords(
		s.auditRecord(s.recordID(3), 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.update", "user-b", manage.OutcomeFailure),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		[]string{s.recordID(1), s.recordID(2), s.recordID(3)},
		auditRecordIDs(records),
	)
}

func (s *PostgresAuditTestSuite) Test_queries_records_within_a_time_range() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(3), 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{From: 2000, To: 3000},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(2)}, auditRecordIDs(records))
}

func (s *PostgresAuditTestSuite) Test_queries_records_for_an_action() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Action: "instance.destroy"},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(2)}, auditRecordIDs(records))
}

func (s *PostgresAuditTestSuite) Test_queries_records_for_a_principal() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.create", "user-b", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{PrincipalSubject: "user-b"},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(2)}, auditRecordIDs(records))
}

func (s *PostgresAuditTestSuite) Test_queries_records_for_an_outcome() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.create", "user-a", manage.OutcomeFailure),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Outcome: manage.OutcomeFailure},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(2)}, auditRecordIDs(records))
}

func (s *PostgresAuditTestSuite) Test_limits_a_query_to_the_oldest_matching_records() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(3), 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{Limit: 2},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		[]string{s.recordID(1), s.recordID(2)},
		auditRecordIDs(records),
	)
}

func (s *PostgresAuditTestSuite) Test_cleans_up_records_older_than_the_threshold() {
	auditStore := s.container.Audit()
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(3), 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	removed, err := auditStore.Cleanup(context.Background(), time.Unix(2500, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), removed)

	records, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(3)}, auditRecordIDs(records))
}

func (s *PostgresAuditTestSuite) Test_resumes_a_listing_after_a_cursor() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2000, "instance.update", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(3), 3000, "instance.destroy", "user-a", manage.OutcomeSuccess),
	)

	records, err := s.container.Audit().Query(
		context.Background(),
		&manage.AuditRecordQuery{AfterTimestamp: 2000, AfterID: s.recordID(2)},
	)
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(3)}, auditRecordIDs(records))
}

// Without an ID in the cursor a forwarder could never read past a second holding
// more records than the limit, since the limit is applied after ordering.
func (s *PostgresAuditTestSuite) Test_pages_through_records_sharing_a_timestamp() {
	s.saveRecords(
		s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(3), 1000, "instance.create", "user-a", manage.OutcomeSuccess),
	)
	auditStore := s.container.Audit()

	first, err := auditStore.Query(
		context.Background(),
		&manage.AuditRecordQuery{Limit: 2},
	)
	s.Require().NoError(err)
	s.Require().Equal(
		[]string{s.recordID(1), s.recordID(2)},
		auditRecordIDs(first),
	)

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
	s.Assert().Equal([]string{s.recordID(3)}, auditRecordIDs(second))
}

// The boundary is asserted because it is where the backends could disagree: a
// record whose timestamp equals the threshold is removed, not kept.
func (s *PostgresAuditTestSuite) Test_cleans_up_a_record_on_the_threshold() {
	auditStore := s.container.Audit()
	s.saveRecords(
		s.auditRecord(s.recordID(1), 2000, "instance.create", "user-a", manage.OutcomeSuccess),
		s.auditRecord(s.recordID(2), 2001, "instance.update", "user-a", manage.OutcomeSuccess),
	)

	removed, err := auditStore.Cleanup(context.Background(), time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), removed)

	records, err := auditStore.Query(context.Background(), &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{s.recordID(2)}, auditRecordIDs(records))
}

// The trail is append-only, so a reused ID must leave the stored record as it is
// rather than overwrite what it already says.
func (s *PostgresAuditTestSuite) Test_does_not_overwrite_a_record_with_a_reused_id() {
	auditStore := s.container.Audit()
	original := s.auditRecord(s.recordID(1), 1000, "instance.create", "user-a", manage.OutcomeSuccess)
	s.Require().NoError(auditStore.Save(context.Background(), original))

	reused := s.auditRecord(s.recordID(1), 2000, "instance.destroy", "user-b", manage.OutcomeFailure)
	s.Require().NoError(auditStore.Save(context.Background(), reused))

	retrieved, err := auditStore.Get(context.Background(), s.recordID(1))
	s.Require().NoError(err)
	s.Assert().Equal(original, retrieved)
}

func (s *PostgresAuditTestSuite) saveRecords(records ...*manage.AuditRecord) {
	for _, record := range records {
		s.Require().NoError(s.container.Audit().Save(context.Background(), record))
	}
}

func (s *PostgresAuditTestSuite) recordID(index int) string {
	return fmt.Sprintf("9c0e4a92-3b4a-4a4e-8f6f-5c4d2b1a00%02d", index)
}

func (s *PostgresAuditTestSuite) auditRecord(
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
		PrincipalName:    "a-name",
		OnBehalfOf:       "an-end-user",
		SourceIP:         "10.0.0.1",
		UserAgent:        "bluelink-cli/1.0",
		RequestID:        fmt.Sprintf("request-for-%s", id),
		TraceID:          fmt.Sprintf("trace-for-%s", id),
		SpanID:           fmt.Sprintf("span-for-%s", id),
	}
}

func auditRecordIDs(records []*manage.AuditRecord) []string {
	ids := make([]string, len(records))
	for i, record := range records {
		ids[i] = record.ID
	}
	return ids
}

func TestPostgresAuditTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresAuditTestSuite))
}
