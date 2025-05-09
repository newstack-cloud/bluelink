package postgres

import (
	"context"
	"path"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/two-hundred/celerity/libs/blueprint-state/internal"
	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/state"
	"github.com/two-hundred/celerity/libs/common/testhelpers"
)

const (
	// See __testdata/seed/blueprint-instances.json
	getTestRootInstanceID   = "46324ee7-b515-4988-98b0-d5445746a997"
	getTestRootInstanceName = "SeedBlueprintInstance1"
	updateStatusInstanceID  = "eb5b3b43-5c85-4aa3-bfb8-70e9fb67fddb"
	nonExistentInstanceID   = "9528bcf0-a42f-4da6-86c8-cb85263f71a9"
	nonExistentInstanceName = "NonExistentInstanceName"
	// See __testdata/seed/save-input/blueprints/3.json
	// The blueprint instance to remove is created as a part of the test case
	// to make it easier to differentiate from all the other seed data.
	removeBlueprintInstanceID = "0082d63f-ef89-406b-a7f5-4e8ce78fc16a"
)

type PostgresStateContainerInstancesTestSuite struct {
	container             state.Container
	connPool              *pgxpool.Pool
	saveBlueprintFixtures map[int]internal.SaveBlueprintFixture
	suite.Suite
}

func (s *PostgresStateContainerInstancesTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "blueprints")
	fixtures, err := internal.SetupSaveBlueprintFixtures(
		dirPath,
		/* updates */ []int{2},
	)
	s.Require().NoError(err)
	s.saveBlueprintFixtures = fixtures
}

func (s *PostgresStateContainerInstancesTestSuite) TearDownTest() {
	for _, fixture := range s.saveBlueprintFixtures {
		if !fixture.Update {
			_, _ = s.container.Instances().Remove(
				context.Background(),
				fixture.InstanceState.InstanceID,
			)
		}
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerInstancesTestSuite) Test_retrieves_blueprint_instance() {
	instances := s.container.Instances()

	instanceState, err := instances.Get(
		context.Background(),
		getTestRootInstanceID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(instanceState)
	err = testhelpers.Snapshot(instanceState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_retrieval() {
	instances := s.container.Instances()

	_, err := instances.Get(
		context.Background(),
		nonExistentInstanceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_look_up_blueprint_instance_id_by_name() {
	instances := s.container.Instances()

	instanceID, err := instances.LookupIDByName(
		context.Background(),
		getTestRootInstanceName,
	)
	s.Require().NoError(err)
	s.Assert().Equal(getTestRootInstanceID, instanceID)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_id_lookup_by_name() {
	instances := s.container.Instances()

	_, err := instances.LookupIDByName(
		context.Background(),
		nonExistentInstanceName,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_saves_new_instance_with_child_blueprint() {
	fixture := s.saveBlueprintFixtures[1]
	instances := s.container.Instances()
	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	savedInstanceState, err := instances.Get(
		context.Background(),
		fixture.InstanceState.InstanceID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedInstanceState)
	internal.AssertInstanceStatesEqual(fixture.InstanceState, &savedInstanceState, &s.Suite)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_updates_existing_instance_with_child_blueprint() {
	fixture := s.saveBlueprintFixtures[2]
	instances := s.container.Instances()
	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	savedState, err := instances.Get(
		context.Background(),
		fixture.InstanceState.InstanceID,
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatesEqual(fixture.InstanceState, &savedState, &s.Suite)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_updates_blueprint_instance_deployment_status() {
	instances := s.container.Instances()

	statusInfo := internal.CreateTestInstanceStatusInfo()
	err := instances.UpdateStatus(
		context.Background(),
		updateStatusInstanceID,
		statusInfo,
	)
	s.Require().NoError(err)

	savedState, err := instances.Get(
		context.Background(),
		updateStatusInstanceID,
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatusInfo(statusInfo, savedState, &s.Suite)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_status_update() {
	instances := s.container.Instances()

	statusInfo := internal.CreateTestInstanceStatusInfo()
	err := instances.UpdateStatus(
		context.Background(),
		nonExistentInstanceID,
		statusInfo,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerInstancesTestSuite) Test_removes_blueprint_instance() {
	fixture := s.saveBlueprintFixtures[3]

	instances := s.container.Instances()
	// Save the full blueprint instance including resources, links and child blueprints
	// for the top-level blueprint to be removed.
	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	_, err = instances.Remove(context.Background(), removeBlueprintInstanceID)
	s.Require().NoError(err)

	_, err = instances.Get(context.Background(), removeBlueprintInstanceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)

	resources := s.container.Resources()
	for _, resource := range fixture.InstanceState.Resources {
		_, err := resources.Get(
			context.Background(),
			resource.ID(),
		)
		s.Require().Error(err)
		stateErr, isStateErr := err.(*state.Error)
		s.Assert().True(isStateErr)
		s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
	}

	links := s.container.Links()
	for _, link := range fixture.InstanceState.Links {
		_, err := links.Get(
			context.Background(),
			link.ID(),
		)
		s.Require().Error(err)
		stateErr, isStateErr := err.(*state.Error)
		s.Assert().True(isStateErr)
		s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
	}
}

func (s *PostgresStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_removal() {
	instances := s.container.Instances()
	_, err := instances.Remove(context.Background(), nonExistentInstanceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func TestPostgresStateContainerInstancesTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerInstancesTestSuite))
}
