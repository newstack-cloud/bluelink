package statestore

import (
	"context"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// The in-memory lookup holds the whole trail only under ModeEager. These exercise
// the case where it holds a subset, which is what a lazy backend sees, and where
// counting the change in the map's size would answer with a number bearing no
// relation to what was removed.
type AuditContainerSuite struct {
	suite.Suite
}

func (s *AuditContainerSuite) Test_cleanup_counts_what_storage_held_not_what_was_cached() {
	storage := newMapStorage()
	ctx := context.Background()

	// Written through a container whose lookup is complete, so storage ends up
	// holding the whole trail.
	writerState := NewState()
	writer := NewAuditContainer(writerState, s.persisterFor(writerState, storage), core.NewNopLogger())
	for _, record := range []*manage.AuditRecord{
		auditRecordFixture("record-1", 1000),
		auditRecordFixture("record-2", 2000),
		auditRecordFixture("record-3", 3000),
	} {
		s.Require().NoError(writer.Save(ctx, record))
	}

	// Read back through a container that has only one of the three cached, which is
	// the state a lazy backend is in after resolving a single record by ID.
	cachedState := NewState(
		WithSharedAuditRecords(map[string]*manage.AuditRecord{
			"record-3": auditRecordFixture("record-3", 3000),
		}),
	)
	cached := NewAuditContainer(cachedState, s.persisterFor(cachedState, storage), core.NewNopLogger())

	removed, err := cached.Cleanup(ctx, time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), removed)

	records, err := cached.Query(ctx, &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Assert().Equal("record-3", records[0].ID)
}

func (s *AuditContainerSuite) Test_cleanup_reports_nothing_removed_for_an_empty_trail() {
	storage := newMapStorage()
	st := NewState()
	container := NewAuditContainer(st, s.persisterFor(st, storage), core.NewNopLogger())

	removed, err := container.Cleanup(context.Background(), time.Unix(2000, 0))
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), removed)
}

func (s *AuditContainerSuite) persisterFor(st *State, storage Storage) *Persister {
	return NewPersister(
		st,
		storage,
		Config{
			AuditRecords: CategoryConfig{Layout: LayoutChunked, Scope: ScopeGlobal},
		},
		/* prefix */ "",
	)
}

func auditRecordFixture(id string, timestamp int64) *manage.AuditRecord {
	return &manage.AuditRecord{
		ID:               id,
		Timestamp:        timestamp,
		Action:           "instance.create",
		Outcome:          manage.OutcomeSuccess,
		StatusCode:       200,
		Method:           "POST",
		Path:             "/deployments/instances",
		PrincipalMethod:  "apikey",
		PrincipalSubject: "user-a",
	}
}

func TestAuditContainerSuite(t *testing.T) {
	suite.Run(t, new(AuditContainerSuite))
}
