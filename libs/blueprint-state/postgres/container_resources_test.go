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
	// See __testdata/seed/blueprint-instances.json
	existingResourceID     = "cf33c61a-bec2-4c51-bd9a-a7c1254151d2"
	existingResourceName   = "saveOrderFunction"
	updateStatusResourceID = "eb5b3b43-5c85-4aa3-bfb8-70e9fb67fddb"
	nonExistentResourceID  = "2e4a7a1f-ddb3-4ad1-be4e-8d5a562ede15"
)

type PostgresStateContainerResourcesTestSuite struct {
	container            state.Container
	saveResourceFixtures map[int]internal.SaveResourceFixture
	connPool             *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerResourcesTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "resources")
	fixtures, err := internal.SetupSaveResourceFixtures(
		dirPath,
		/* updates */ []int{3},
	)
	s.Require().NoError(err)
	s.saveResourceFixtures = fixtures
}

func (s *PostgresStateContainerResourcesTestSuite) TearDownTest() {
	for _, fixture := range s.saveResourceFixtures {
		if !fixture.Update {
			_, _ = s.container.Resources().Remove(
				context.Background(),
				fixture.ResourceState.ResourceID,
			)
		}
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerResourcesTestSuite) Test_retrieves_resource() {
	resources := s.container.Resources()
	resourceState, err := resources.Get(
		context.Background(),
		existingResourceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(resourceState)
	err = testhelpers.Snapshot(resourceState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_retrieval() {
	resources := s.container.Resources()

	_, err := resources.Get(
		context.Background(),
		nonExistentResourceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_retrieves_resource_by_logical_name() {
	resources := s.container.Resources()
	resourceState, err := resources.GetByName(
		context.Background(),
		getTestRootInstanceID,
		existingResourceName,
	)
	s.Require().NoError(err)
	s.Require().NotNil(resourceState)
	err = testhelpers.Snapshot(resourceState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_retrieval_by_logical_name() {
	resources := s.container.Resources()

	_, err := resources.GetByName(
		context.Background(),
		getTestRootInstanceID,
		nonExistentResourceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_saves_new_resource() {
	fixture := s.saveResourceFixtures[1]
	resources := s.container.Resources()
	err := resources.Save(
		context.Background(),
		*fixture.ResourceState,
	)
	s.Require().NoError(err)

	savedState, err := resources.Get(
		context.Background(),
		fixture.ResourceState.ResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceStatesEqual(fixture.ResourceState, &savedState, &s.Suite)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_reports_instance_not_found_for_saving_resource() {
	// Fixture 2 is a resource state that references a non-existent instance.
	fixture := s.saveResourceFixtures[2]
	resources := s.container.Resources()

	err := resources.Save(
		context.Background(),
		*fixture.ResourceState,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_updates_existing_resource() {
	fixture := s.saveResourceFixtures[3]
	resources := s.container.Resources()
	err := resources.Save(
		context.Background(),
		*fixture.ResourceState,
	)
	s.Require().NoError(err)

	savedState, err := resources.Get(
		context.Background(),
		fixture.ResourceState.ResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceStatesEqual(fixture.ResourceState, &savedState, &s.Suite)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_saves_resource_with_system_metadata() {
	fixture := s.saveResourceFixtures[4]
	resources := s.container.Resources()
	err := resources.Save(
		context.Background(),
		*fixture.ResourceState,
	)
	s.Require().NoError(err)

	savedState, err := resources.Get(
		context.Background(),
		fixture.ResourceState.ResourceID,
	)
	s.Require().NoError(err)

	// Verify system metadata was persisted correctly
	s.Require().NotNil(savedState.SystemMetadata)
	s.Require().NotNil(savedState.SystemMetadata.Provenance)
	s.Assert().Equal("bluelink", savedState.SystemMetadata.Provenance.ProvisionedBy)
	s.Assert().Equal("1.0.0", savedState.SystemMetadata.Provenance.DeployEngineVersion)
	s.Assert().Equal("newstack-cloud/aws", savedState.SystemMetadata.Provenance.ProviderPluginID)
	s.Assert().Equal("2.5.0", savedState.SystemMetadata.Provenance.ProviderPluginVersion)
	s.Assert().Equal(int64(1733145428), savedState.SystemMetadata.Provenance.ProvisionedAt)

	internal.AssertResourceStatesEqual(fixture.ResourceState, &savedState, &s.Suite)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_updates_blueprint_resource_deployment_status() {
	resources := s.container.Resources()

	statusInfo := internal.CreateTestResourceStatusInfo()
	err := resources.UpdateStatus(
		context.Background(),
		updateStatusResourceID,
		statusInfo,
	)
	s.Require().NoError(err)

	savedState, err := resources.Get(
		context.Background(),
		updateStatusResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceStatusInfo(statusInfo, savedState, &s.Suite)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_status_update() {
	resources := s.container.Resources()

	statusInfo := internal.CreateTestResourceStatusInfo()
	err := resources.UpdateStatus(
		context.Background(),
		nonExistentResourceID,
		statusInfo,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_removes_resource() {
	fixture := s.saveResourceFixtures[3]

	resources := s.container.Resources()
	// Save the resource to be removed.
	err := resources.Save(
		context.Background(),
		*fixture.ResourceState,
	)
	s.Require().NoError(err)

	_, err = resources.Remove(context.Background(), fixture.ResourceState.ResourceID)
	s.Require().NoError(err)

	_, err = resources.Get(context.Background(), fixture.ResourceState.ResourceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_removal() {
	resources := s.container.Resources()
	_, err := resources.Remove(context.Background(), nonExistentResourceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func TestPostgresStateContainerResourcesTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerResourcesTestSuite))
}
