package postgres

import (
	"context"
	"path"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

const (
	existingChildName           = "coreInfra"
	attachToParentInstanceID    = "8029c6d0-66f6-43ba-85c1-943403f4139d"
	nonExistentChildBlueprintID = "297dbe5b-eb78-4474-b70b-67fb4d21a6e7"
	detachFromParentInstanceID  = "2b585ad0-543e-4ab2-8bf5-371854968ea0"
	dependenciesTestInstanceID  = "203de8c1-027f-458b-82a4-e77a5c9ddf62"
)

type PostgresStateContainerChildrenTestSuite struct {
	container                  state.Container
	connPool                   *pgxpool.Pool
	saveChildBlueprintFixtures map[int]internal.SaveBlueprintFixture
	suite.Suite
}

func (s *PostgresStateContainerChildrenTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "children")
	fixtures, err := internal.SetupSaveBlueprintFixtures(
		dirPath,
		/* updates */ []int{2},
	)
	s.Require().NoError(err)
	s.saveChildBlueprintFixtures = fixtures
}

func (s *PostgresStateContainerChildrenTestSuite) TearDownTest() {
	for _, fixture := range s.saveChildBlueprintFixtures {
		if !fixture.Update {
			_, _ = s.container.Instances().Remove(
				context.Background(),
				fixture.InstanceState.InstanceID,
			)
		}
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerChildrenTestSuite) Test_retrieves_child_blueprint_instance() {
	children := s.container.Children()
	childInstanceState, err := children.Get(
		context.Background(),
		getTestRootInstanceID,
		existingChildName,
	)
	s.Require().NoError(err)
	s.Require().NotNil(childInstanceState)
	err = testhelpers.Snapshot(childInstanceState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_reports_child_not_found_for_retrieval() {
	children := s.container.Children()

	_, err := children.Get(
		context.Background(),
		nonExistentInstanceID,
		existingChildName,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(nonExistentInstanceID, existingChildName)
	s.Assert().Equal(stateErr.ItemID, itemID)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_attaches_child_blueprint_to_parent() {
	children := s.container.Children()
	instances := s.container.Instances()

	fixture := s.saveChildBlueprintFixtures[1]

	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	err = children.Attach(
		context.Background(),
		attachToParentInstanceID,
		fixture.InstanceState.InstanceID,
		"networking",
	)
	s.Require().NoError(err)

	savedChild, err := children.Get(
		context.Background(),
		attachToParentInstanceID,
		"networking",
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatesEqual(fixture.InstanceState, &savedChild, &s.Suite)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_reports_parent_instance_not_found_for_attaching() {
	children := s.container.Children()
	fixture := s.saveChildBlueprintFixtures[1]

	err := children.Attach(
		context.Background(),
		nonExistentInstanceID,
		fixture.InstanceState.InstanceID,
		"coreInfra",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(stateErr.ItemID, nonExistentInstanceID)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_reports_child_instance_not_found_for_attaching() {
	children := s.container.Children()

	err := children.Attach(
		context.Background(),
		attachToParentInstanceID,
		nonExistentChildBlueprintID,
		"coreInfra",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentChildBlueprintID, stateErr.ItemID)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_detaches_child_blueprint_from_parent() {
	children := s.container.Children()
	instances := s.container.Instances()
	fixture := s.saveChildBlueprintFixtures[2]

	// Save, attach and then detach on the fly to make the operation as atomic as possible,
	// avoiding side effects that could cause confusion leading to malformed state and unreliable tests.
	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	err = children.Attach(
		context.Background(),
		detachFromParentInstanceID,
		fixture.InstanceState.InstanceID,
		existingChildName,
	)
	s.Require().NoError(err)

	err = children.Detach(
		context.Background(),
		detachFromParentInstanceID,
		existingChildName,
	)
	s.Require().NoError(err)

	_, err = children.Get(
		context.Background(),
		detachFromParentInstanceID,
		existingChildName,
	)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(detachFromParentInstanceID, existingChildName)
	s.Assert().Equal(stateErr.ItemID, itemID)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_reports_child_instance_not_found_for_detaching() {
	children := s.container.Children()

	err := children.Detach(
		context.Background(),
		detachFromParentInstanceID,
		nonExistentChildBlueprintID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(detachFromParentInstanceID, nonExistentChildBlueprintID)
	s.Assert().Equal(itemID, stateErr.ItemID)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_saves_dependencies_for_child_blueprint_in_parent_context() {
	children := s.container.Children()
	instances := s.container.Instances()

	fixture := s.saveChildBlueprintFixtures[3]

	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	err = children.Attach(
		context.Background(),
		dependenciesTestInstanceID,
		fixture.InstanceState.InstanceID,
		existingChildName,
	)
	s.Require().NoError(err)

	inputDependencyInfo := &state.DependencyInfo{
		DependsOnResources: []string{"saveOrderFunction"},
		DependsOnChildren:  []string{"networking"},
	}
	err = children.SaveDependencies(
		context.Background(),
		dependenciesTestInstanceID,
		existingChildName,
		inputDependencyInfo,
	)
	s.Require().NoError(err)

	instanceState, err := instances.Get(
		context.Background(),
		dependenciesTestInstanceID,
	)
	s.Require().NoError(err)
	savedDeps := instanceState.ChildDependencies[existingChildName]
	s.Assert().Equal(inputDependencyInfo, savedDeps)
}

func (s *PostgresStateContainerChildrenTestSuite) Test_reports_parent_instance_not_found_for_saving_dependencies() {
	children := s.container.Children()

	err := children.SaveDependencies(
		context.Background(),
		nonExistentInstanceID,
		existingChildName,
		&state.DependencyInfo{},
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func TestPostgresStateContainerChildrenTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerChildrenTestSuite))
}
