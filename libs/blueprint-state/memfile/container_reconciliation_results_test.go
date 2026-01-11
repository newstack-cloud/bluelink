package memfile

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	existingReconciliationResultID    = "d1234567-89ab-cdef-0123-456789abcdef"
	nonExistentReconciliationResultID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	existingChangesetIDForResult      = "08dc456e-cafc-4199-b074-5f04cd4904f2"
	existingInstanceIDForResult       = "46324ee7-b515-4988-98b0-d5445746a997"
	changesetWithMultipleResults      = "2888c908-32e2-4555-af36-319455172c64"
)

type MemFileStateContainerReconciliationResultsSuite struct {
	container                          *StateContainer
	stateDir                           string
	fs                                 afero.Fs
	saveReconciliationResultFixtures   map[int]internal.SaveReconciliationResultFixture
	suite.Suite
}

func (s *MemFileStateContainerReconciliationResultsSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "reconciliation-results")
	fixtures, err := internal.SetupSaveReconciliationResultFixtures(dirPath)
	s.Require().NoError(err)
	s.saveReconciliationResultFixtures = fixtures
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_retrieves_reconciliation_result() {
	results := s.container.ReconciliationResults()
	result, err := results.Get(
		context.Background(),
		existingReconciliationResultID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Assert().Equal(existingReconciliationResultID, result.ID)
	s.Assert().Equal(existingChangesetIDForResult, result.ChangesetID)
	s.Assert().Equal(existingInstanceIDForResult, result.InstanceID)
	s.Assert().NotNil(result.Result)
	s.Assert().True(result.Result.HasDrift)
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_fails_to_retrieve_non_existent_result() {
	results := s.container.ReconciliationResults()

	_, err := results.Get(
		context.Background(),
		nonExistentReconciliationResultID,
	)
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
	s.Assert().True(isNotFoundErr)
	s.Assert().EqualError(
		notFoundErr,
		fmt.Sprintf("reconciliation result with ID %s not found", nonExistentReconciliationResultID),
	)
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_retrieves_latest_by_changeset_id() {
	results := s.container.ReconciliationResults()

	result, err := results.GetLatestByChangesetID(
		context.Background(),
		existingChangesetIDForResult,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Assert().Equal(existingChangesetIDForResult, result.ChangesetID)
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_fails_to_retrieve_latest_by_non_existent_changeset_id() {
	results := s.container.ReconciliationResults()

	_, err := results.GetLatestByChangesetID(
		context.Background(),
		"non-existent-changeset-id",
	)
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
	s.Assert().True(isNotFoundErr)
	s.Assert().Contains(notFoundErr.Error(), "reconciliation result for changeset")
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_retrieves_all_by_changeset_id() {
	results := s.container.ReconciliationResults()

	allResults, err := results.GetAllByChangesetID(
		context.Background(),
		existingChangesetIDForResult,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(allResults)
	for _, r := range allResults {
		s.Assert().Equal(existingChangesetIDForResult, r.ChangesetID)
	}
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_retrieves_latest_by_instance_id() {
	results := s.container.ReconciliationResults()

	result, err := results.GetLatestByInstanceID(
		context.Background(),
		existingInstanceIDForResult,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Assert().Equal(existingInstanceIDForResult, result.InstanceID)
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_fails_to_retrieve_latest_by_non_existent_instance_id() {
	results := s.container.ReconciliationResults()

	_, err := results.GetLatestByInstanceID(
		context.Background(),
		"non-existent-instance-id",
	)
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
	s.Assert().True(isNotFoundErr)
	s.Assert().Contains(notFoundErr.Error(), "reconciliation result for instance")
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_retrieves_all_by_instance_id() {
	results := s.container.ReconciliationResults()

	allResults, err := results.GetAllByInstanceID(
		context.Background(),
		existingInstanceIDForResult,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(allResults)
	s.Assert().Len(allResults, 3)
	for _, r := range allResults {
		s.Assert().Equal(existingInstanceIDForResult, r.InstanceID)
	}
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_saves_new_reconciliation_result() {
	fixture := s.saveReconciliationResultFixtures[1]

	results := s.container.ReconciliationResults()
	err := results.Save(
		context.Background(),
		fixture.ReconciliationResult,
	)
	s.Require().NoError(err)

	savedResult, err := results.Get(
		context.Background(),
		fixture.ReconciliationResult.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedResult)
	s.Assert().Equal(fixture.ReconciliationResult.ID, savedResult.ID)
	s.Assert().Equal(fixture.ReconciliationResult.ChangesetID, savedResult.ChangesetID)
	s.Assert().Equal(fixture.ReconciliationResult.InstanceID, savedResult.InstanceID)

	s.assertPersistedReconciliationResult(fixture.ReconciliationResult)
}

func (s *MemFileStateContainerReconciliationResultsSuite) Test_cleans_up_old_reconciliation_results() {
	_, err := s.container.ReconciliationResults().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	assertReconciliationResultsCleanedUp(
		s.container,
		&s.Suite,
	)

	s.assertReconciliationResultCleanupPersisted()
}

func (s *MemFileStateContainerReconciliationResultsSuite) assertPersistedReconciliationResult(
	expected *manage.ReconciliationResult,
) {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	results := container.ReconciliationResults()
	persistedResult, err := results.Get(
		context.Background(),
		expected.ID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(expected.ID, persistedResult.ID)
	s.Assert().Equal(expected.ChangesetID, persistedResult.ChangesetID)
	s.Assert().Equal(expected.InstanceID, persistedResult.InstanceID)
}

func (s *MemFileStateContainerReconciliationResultsSuite) assertReconciliationResultCleanupPersisted() {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	assertReconciliationResultsCleanedUp(
		container,
		&s.Suite,
	)
}

func assertReconciliationResultsCleanedUp(
	container *StateContainer,
	s *suite.Suite,
) {
	for _, id := range reconciliationResultsShouldBeCleanedUp {
		_, err := container.ReconciliationResults().Get(
			context.Background(),
			id,
		)
		s.Require().Error(err)

		notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
		s.Require().True(isNotFoundErr)
		s.Assert().Equal(
			fmt.Sprintf("reconciliation result with ID %s not found", id),
			notFoundErr.Error(),
		)
	}

	for _, id := range reconciliationResultsShouldNotBeCleanedUp {
		result, err := container.ReconciliationResults().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, result.ID)
	}
}

// Seed reconciliation results that should be cleaned up (created before threshold).
var reconciliationResultsShouldBeCleanedUp = []string{
	"d1234567-89ab-cdef-0123-456789abcdef",
	"e2345678-89ab-cdef-0123-456789abcdef",
}

// Seed reconciliation results that should not be cleaned up (created after threshold).
var reconciliationResultsShouldNotBeCleanedUp = []string{
	"c3456789-89ab-cdef-0123-456789abcdef",
}

func TestMemFileStateContainerReconciliationResultsSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerReconciliationResultsSuite))
}
