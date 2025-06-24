package postgres

import (
	"context"
	"path"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

const (
	existingDriftResourceID   = "e78707ae-f21e-42e3-b3e1-ce295bf907dd"
	removeDriftResourceID     = "1710c9d9-4276-46a2-af77-fa59cb69a586"
	existingResourceIDNoDrift = "9d171038-b7e0-417f-a79f-57eaab9a9a11"
)

type PostgresStateContainerResourceDriftTestSuite struct {
	container                 state.Container
	saveResourceDriftFixtures map[int]internal.SaveResourceDriftFixture
	connPool                  *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerResourceDriftTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "resource-drift")
	fixtures, err := internal.SetupSaveResourceDriftFixtures(
		dirPath,
		/* updates */ []int{2},
	)
	s.Require().NoError(err)
	s.saveResourceDriftFixtures = fixtures
}

func (s *PostgresStateContainerResourceDriftTestSuite) TearDownTest() {
	for _, fixture := range s.saveResourceDriftFixtures {
		if !fixture.Update {
			_, _ = s.container.Resources().RemoveDrift(
				context.Background(),
				fixture.DriftState.ResourceID,
			)
		}
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_retrieves_resource_drift() {
	resources := s.container.Resources()
	resourceDriftState, err := resources.GetDrift(
		context.Background(),
		existingDriftResourceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(resourceDriftState)
	err = testhelpers.Snapshot(resourceDriftState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_reports_resource_not_found_for_drift_retrieval() {
	resources := s.container.Resources()

	_, err := resources.GetDrift(
		context.Background(),
		nonExistentResourceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_saves_new_resource_drift() {
	fixture := s.saveResourceDriftFixtures[1]
	resources := s.container.Resources()
	err := resources.SaveDrift(
		context.Background(),
		*fixture.DriftState,
	)
	s.Require().NoError(err)

	savedDriftState, err := resources.GetDrift(
		context.Background(),
		fixture.DriftState.ResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceDriftEqual(fixture.DriftState, &savedDriftState, &s.Suite)

	updatedResource, err := resources.Get(
		context.Background(),
		fixture.DriftState.ResourceID,
	)
	s.Require().NoError(err)
	s.Assert().True(updatedResource.Drifted)
	s.Assert().Equal(fixture.DriftState.Timestamp, updatedResource.LastDriftDetectedTimestamp)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_updates_existing_resource_drift() {
	fixture := s.saveResourceDriftFixtures[2]
	resources := s.container.Resources()
	err := resources.SaveDrift(
		context.Background(),
		*fixture.DriftState,
	)
	s.Require().NoError(err)

	savedDriftState, err := resources.GetDrift(
		context.Background(),
		fixture.DriftState.ResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceDriftEqual(fixture.DriftState, &savedDriftState, &s.Suite)

	updatedResource, err := resources.Get(
		context.Background(),
		fixture.DriftState.ResourceID,
	)
	s.Require().NoError(err)
	s.Assert().True(updatedResource.Drifted)
	s.Assert().Equal(fixture.DriftState.Timestamp, updatedResource.LastDriftDetectedTimestamp)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_reports_resource_not_found_for_saving_drift() {
	// Fixture 3 is a drift state that references a non-existent resource.
	fixture := s.saveResourceDriftFixtures[3]
	resources := s.container.Resources()

	err := resources.SaveDrift(
		context.Background(),
		*fixture.DriftState,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_removes_resource_drift() {
	resources := s.container.Resources()
	_, err := resources.RemoveDrift(context.Background(), removeDriftResourceID)
	s.Require().NoError(err)

	drift, err := resources.GetDrift(context.Background(), removeDriftResourceID)
	s.Require().NoError(err)
	// The resource should still exist but the drift should be an empty value.
	s.Assert().True(internal.IsEmptyDriftState(drift))

	resource, err := resources.Get(context.Background(), removeDriftResourceID)
	s.Require().NoError(err)
	s.Assert().False(resource.Drifted)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_reports_resource_not_found_for_removing_drift() {
	resources := s.container.Resources()

	_, err := resources.RemoveDrift(
		context.Background(),
		nonExistentResourceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourceDriftTestSuite) Test_does_nothing_for_missing_drift_entry_for_existing_resource() {
	resources := s.container.Resources()

	drift, err := resources.RemoveDrift(
		context.Background(),
		existingResourceIDNoDrift,
	)
	s.Require().NoError(err)
	s.Assert().True(internal.IsEmptyDriftState(drift))
}

func TestPostgresStateContainerResourceDriftTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerResourceDriftTestSuite))
}
