package memfile

import (
	"context"
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

var errStorageUnavailable = errors.New("storage unavailable")

// What happens to the in-memory view when persisting a record fails. Memory and
// storage have to agree, because Get reads one and Query the other.
type MemFileAuditPersistenceSuite struct {
	suite.Suite
}

func (s *MemFileAuditPersistenceSuite) Test_a_failed_persist_leaves_no_record_behind() {
	container, storage := s.containerWithFailableStorage()
	auditStore := container.Audit()
	ctx := context.Background()

	storage.failAuditWrites = true
	err := auditStore.Save(ctx, auditRecordFor("record-1", 1000))
	s.Require().ErrorIs(err, errStorageUnavailable)

	_, err = auditStore.Get(ctx, "record-1")
	s.Require().Error(err)
	notFoundErr := &manage.AuditRecordNotFound{}
	s.Assert().ErrorAs(err, &notFoundErr, "Get must not answer from memory with an unpersisted record")

	records, err := auditStore.Query(ctx, &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Empty(records)
}

// The ID must not be left blocked by a memory entry, or the record could never be
// written even once storage recovers.
func (s *MemFileAuditPersistenceSuite) Test_a_record_can_be_saved_again_after_a_failed_persist() {
	container, storage := s.containerWithFailableStorage()
	auditStore := container.Audit()
	ctx := context.Background()

	storage.failAuditWrites = true
	s.Require().Error(auditStore.Save(ctx, auditRecordFor("record-1", 1000)))

	storage.failAuditWrites = false
	s.Require().NoError(auditStore.Save(ctx, auditRecordFor("record-1", 1000)))

	retrieved, err := auditStore.Get(ctx, "record-1")
	s.Require().NoError(err)
	s.Assert().Equal("record-1", retrieved.ID)

	records, err := auditStore.Query(ctx, &manage.AuditRecordQuery{})
	s.Require().NoError(err)
	s.Assert().Equal([]string{"record-1"}, recordIDs(records))
}

func (s *MemFileAuditPersistenceSuite) containerWithFailableStorage() (
	*StateContainer,
	*failableStorage,
) {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)

	storage := &failableStorage{Storage: NewFSStorage(memoryFS)}
	container, err := LoadStateContainerWithStorage(stateDir, storage, core.NewNopLogger())
	s.Require().NoError(err)

	return container, storage
}

// Fails writes to the audit chunk files only, so a save can be made to fail without
// disturbing the rest of the store's startup or state.
type failableStorage struct {
	statestore.Storage
	failAuditWrites bool
}

func (s *failableStorage) Write(ctx context.Context, key string, data []byte) error {
	if s.failAuditWrites && strings.Contains(key, "audit_record") {
		return errStorageUnavailable
	}

	return s.Storage.Write(ctx, key, data)
}

func auditRecordFor(id string, timestamp int64) *manage.AuditRecord {
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

func TestMemFileAuditPersistenceSuite(t *testing.T) {
	suite.Run(t, new(MemFileAuditPersistenceSuite))
}
