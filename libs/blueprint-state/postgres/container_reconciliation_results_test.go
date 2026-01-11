package postgres

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

const (
	existingReconciliationResultID    = "d1234567-89ab-cdef-0123-456789abcdef"
	nonExistentReconciliationResultID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	// Use a changeset that won't be cleaned up by the changesets cleanup test
	existingChangesetIDForResult = "2888c908-32e2-4555-af36-319455172c64"
	existingInstanceIDForResult  = "46324ee7-b515-4988-98b0-d5445746a997"
)

type PostgresReconciliationResultsTestSuite struct {
	container                        *StateContainer
	connPool                         *pgxpool.Pool
	saveReconciliationResultFixtures map[int]internal.SaveReconciliationResultFixture
	suite.Suite
}

func (s *PostgresReconciliationResultsTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "reconciliation-results")
	saveFixtures, err := internal.SetupSaveReconciliationResultFixtures(dirPath)
	s.Require().NoError(err)
	s.saveReconciliationResultFixtures = saveFixtures
}

func (s *PostgresReconciliationResultsTestSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresReconciliationResultsTestSuite) Test_retrieve_existing_reconciliation_result() {
	ctx := context.Background()
	result, err := s.container.ReconciliationResults().Get(ctx, existingReconciliationResultID)
	s.Require().NoError(err)

	s.Require().Equal(existingReconciliationResultID, result.ID)
	s.Require().Equal(existingChangesetIDForResult, result.ChangesetID)
	s.Require().Equal(existingInstanceIDForResult, result.InstanceID)
	s.Require().NotNil(result.Result)
	s.Require().True(result.Result.HasDrift)
}

func (s *PostgresReconciliationResultsTestSuite) Test_fails_to_retrieve_non_existent_result() {
	ctx := context.Background()
	_, err := s.container.ReconciliationResults().Get(ctx, nonExistentReconciliationResultID)
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
	s.Require().True(isNotFoundErr)
	s.Require().Equal(nonExistentReconciliationResultID, notFoundErr.ID)
	s.Require().Equal(
		fmt.Sprintf("reconciliation result with ID %s not found", nonExistentReconciliationResultID),
		notFoundErr.Error(),
	)
}

func (s *PostgresReconciliationResultsTestSuite) Test_retrieves_latest_by_changeset_id() {
	ctx := context.Background()
	result, err := s.container.ReconciliationResults().GetLatestByChangesetID(ctx, existingChangesetIDForResult)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(existingChangesetIDForResult, result.ChangesetID)
}

func (s *PostgresReconciliationResultsTestSuite) Test_fails_to_retrieve_latest_by_non_existent_changeset_id() {
	ctx := context.Background()
	_, err := s.container.ReconciliationResults().GetLatestByChangesetID(ctx, "00000000-0000-0000-0000-000000000000")
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
	s.Require().True(isNotFoundErr)
	s.Require().Contains(notFoundErr.Error(), "reconciliation result for changeset")
}

func (s *PostgresReconciliationResultsTestSuite) Test_retrieves_all_by_changeset_id() {
	ctx := context.Background()
	results, err := s.container.ReconciliationResults().GetAllByChangesetID(ctx, existingChangesetIDForResult)
	s.Require().NoError(err)
	s.Require().NotEmpty(results)
	for _, r := range results {
		s.Assert().Equal(existingChangesetIDForResult, r.ChangesetID)
	}
}

func (s *PostgresReconciliationResultsTestSuite) Test_retrieves_latest_by_instance_id() {
	ctx := context.Background()
	result, err := s.container.ReconciliationResults().GetLatestByInstanceID(ctx, existingInstanceIDForResult)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(existingInstanceIDForResult, result.InstanceID)
}

func (s *PostgresReconciliationResultsTestSuite) Test_fails_to_retrieve_latest_by_non_existent_instance_id() {
	ctx := context.Background()
	_, err := s.container.ReconciliationResults().GetLatestByInstanceID(ctx, "00000000-0000-0000-0000-000000000000")
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ReconciliationResultNotFound)
	s.Require().True(isNotFoundErr)
	s.Require().Contains(notFoundErr.Error(), "reconciliation result for instance")
}

func (s *PostgresReconciliationResultsTestSuite) Test_retrieves_all_by_instance_id() {
	ctx := context.Background()
	results, err := s.container.ReconciliationResults().GetAllByInstanceID(ctx, existingInstanceIDForResult)
	s.Require().NoError(err)
	s.Require().NotEmpty(results)
	// Note: The changeset cleanup test may have already deleted some reconciliation results
	// via cascade, so we just check that we get at least 1 result
	s.Assert().GreaterOrEqual(len(results), 1)
	for _, r := range results {
		s.Assert().Equal(existingInstanceIDForResult, r.InstanceID)
	}
}

func (s *PostgresReconciliationResultsTestSuite) Test_saves_reconciliation_result() {
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
}

func (s *PostgresReconciliationResultsTestSuite) Test_cleans_up_old_reconciliation_results() {
	_, err := s.container.ReconciliationResults().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	for _, id := range reconciliationResultsShouldBeCleanedUp {
		_, err := s.container.ReconciliationResults().Get(
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
		result, err := s.container.ReconciliationResults().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, result.ID)
	}
}

// Seed reconciliation results that should be cleaned up (created before threshold).
// Note: e2345678 references a changeset that also gets cleaned up, so it may be deleted by cascade.
var reconciliationResultsShouldBeCleanedUp = []string{
	"e2345678-89ab-cdef-0123-456789abcdef",
}

// Seed reconciliation results that should not be cleaned up (created after threshold).
var reconciliationResultsShouldNotBeCleanedUp = []string{
	"d1234567-89ab-cdef-0123-456789abcdef",
	"c3456789-89ab-cdef-0123-456789abcdef",
}

func TestPostgresReconciliationResultsTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresReconciliationResultsTestSuite))
}
