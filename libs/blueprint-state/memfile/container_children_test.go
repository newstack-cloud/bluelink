package memfile

import (
	"context"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	existingChildName           = "coreInfra"
	nonExistentChildName        = "non-existent-child"
	existingChildBlueprintID    = "blueprint-instance-1-child-core-infra"
	nonExistentChildBlueprintID = "non-existent-child-id"
)

type MemFileStateContainerChildrenTestSuite struct {
	container                  state.Container
	saveChildBlueprintFixtures map[int]internal.SaveBlueprintFixture
	stateDir                   string
	fs                         afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerChildrenTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// Use a low max guide file size of 100 bytes to trigger the logic that splits
	// instance state across multiple chunk files.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "children")
	fixtures, err := internal.SetupSaveBlueprintFixtures(dirPath, []int{})
	s.Require().NoError(err)
	s.saveChildBlueprintFixtures = fixtures
}

func (s *MemFileStateContainerChildrenTestSuite) Test_retrieves_child_blueprint_instance() {
	children := s.container.Children()
	childInstanceState, err := children.Get(
		context.Background(),
		existingBlueprintInstanceID,
		existingChildName,
	)
	s.Require().NoError(err)
	s.Require().NotNil(childInstanceState)
	err = testhelpers.Snapshot(childInstanceState)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_reports_parent_instance_not_found_for_retrieval() {
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
	s.Assert().Equal(stateErr.ItemID, nonExistentInstanceID)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_reports_child_instance_not_found_for_retrieval() {
	children := s.container.Children()

	_, err := children.Get(
		context.Background(),
		existingBlueprintInstanceID,
		nonExistentChildName,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(existingBlueprintInstanceID, nonExistentChildName)
	s.Assert().Equal(stateErr.ItemID, itemID)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_attaches_child_blueprint_to_parent() {
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
		existingBlueprintInstanceID,
		fixture.InstanceState.InstanceID,
		"networking",
	)
	s.Require().NoError(err)

	savedChild, err := children.Get(
		context.Background(),
		existingBlueprintInstanceID,
		"networking",
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatesEqual(fixture.InstanceState, &savedChild, &s.Suite)
	s.assertPersistedChild(existingBlueprintInstanceID, "networking", fixture.InstanceState)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_reports_parent_instance_not_found_for_attaching() {
	children := s.container.Children()

	err := children.Attach(
		context.Background(),
		nonExistentInstanceID,
		existingChildBlueprintID,
		"coreInfra",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(stateErr.ItemID, nonExistentInstanceID)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_reports_child_instance_not_found_for_attaching() {
	children := s.container.Children()

	err := children.Attach(
		context.Background(),
		existingBlueprintInstanceID,
		nonExistentChildBlueprintID,
		"coreInfra",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentChildBlueprintID, stateErr.ItemID)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_detaches_child_blueprint_from_parent() {
	children := s.container.Children()

	err := children.Detach(
		context.Background(),
		existingBlueprintInstanceID,
		existingChildName,
	)
	s.Require().NoError(err)

	_, err = children.Get(
		context.Background(),
		existingBlueprintInstanceID,
		existingChildName,
	)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(existingBlueprintInstanceID, existingChildName)
	s.Assert().Equal(stateErr.ItemID, itemID)
	s.assertChildDetachPersisted(existingBlueprintInstanceID, existingChildName)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_reports_child_instance_not_found_for_detaching() {
	children := s.container.Children()

	err := children.Detach(
		context.Background(),
		existingBlueprintInstanceID,
		nonExistentChildBlueprintID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(existingBlueprintInstanceID, nonExistentChildBlueprintID)
	s.Assert().Equal(itemID, stateErr.ItemID)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_saves_dependencies_for_child_blueprint_in_parent_context() {
	children := s.container.Children()
	instances := s.container.Instances()

	inputDependencyInfo := &state.DependencyInfo{
		DependsOnResources: []string{"saveOrderFunction"},
		DependsOnChildren:  []string{"networking"},
	}
	err := children.SaveDependencies(
		context.Background(),
		existingBlueprintInstanceID,
		existingChildName,
		inputDependencyInfo,
	)
	s.Require().NoError(err)

	instanceState, err := instances.Get(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	savedDeps := instanceState.ChildDependencies[existingChildName]
	s.Assert().Equal(inputDependencyInfo, savedDeps)
	s.assertPersistedDependencies(existingBlueprintInstanceID, inputDependencyInfo)
}

func (s *MemFileStateContainerChildrenTestSuite) Test_reports_parent_instance_not_found_for_saving_dependencies() {
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

func (s *MemFileStateContainerChildrenTestSuite) assertPersistedChild(instanceID string, childName string, expected *state.InstanceState) {
	// Check that the child instance relationship was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by child name relative to the parent blueprint.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	children := container.Children()
	savedInstanceState, err := children.Get(
		context.Background(),
		instanceID,
		childName,
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatesEqual(expected, &savedInstanceState, &s.Suite)
}

func (s *MemFileStateContainerChildrenTestSuite) assertChildDetachPersisted(instanceID string, childName string) {
	// Check that the child instance relationship removal was persisted to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by child name relative to the parent blueprint.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	children := container.Children()
	_, err = children.Get(
		context.Background(),
		instanceID,
		childName,
	)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	itemID := idutils.ChildInBlueprintID(instanceID, childName)
	s.Assert().Equal(stateErr.ItemID, itemID)
}

func (s *MemFileStateContainerChildrenTestSuite) assertPersistedDependencies(instanceID string, expected *state.DependencyInfo) {
	// Check that the child instance dependencies were saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	instances := container.Instances()
	instanceState, err := instances.Get(
		context.Background(),
		instanceID,
	)
	s.Require().NoError(err)
	savedDeps := instanceState.ChildDependencies[existingChildName]
	s.Assert().Equal(expected, savedDeps)
}

func TestMemFileStateContainerChildrenTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerChildrenTestSuite))
}
