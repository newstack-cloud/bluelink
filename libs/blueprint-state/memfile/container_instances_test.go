package memfile

import (
	"context"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	existingBlueprintInstanceID = "blueprint-instance-1"
	nonExistentInstanceID       = "non-existent-instance"
)

type MemFileStateContainerInstancesTestSuite struct {
	container             state.Container
	stateDir              string
	fs                    afero.Fs
	saveBlueprintFixtures map[int]internal.SaveBlueprintFixture
	suite.Suite
}

func (s *MemFileStateContainerInstancesTestSuite) SetupTest() {
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

	dirPath := path.Join("__testdata", "save-input", "blueprints")
	fixtures, err := internal.SetupSaveBlueprintFixtures(
		dirPath,
		/* updates */ []int{2},
	)
	s.Require().NoError(err)
	s.saveBlueprintFixtures = fixtures
}

func (s *MemFileStateContainerInstancesTestSuite) Test_retrieves_instance() {
	instances := s.container.Instances()
	instanceState, err := instances.Get(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(instanceState)
	err = testhelpers.Snapshot(instanceState)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_retrieval() {
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

func (s *MemFileStateContainerInstancesTestSuite) Test_saves_new_instance_with_child_blueprint() {
	fixture := s.saveBlueprintFixtures[1]
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
	s.assertPersistedInstance(fixture.InstanceState)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_updates_existing_instance_with_child_blueprint() {
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
	s.assertPersistedInstance(fixture.InstanceState)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_updates_blueprint_instance_deployment_status() {
	instances := s.container.Instances()

	statusInfo := internal.CreateTestInstanceStatusInfo()
	err := instances.UpdateStatus(
		context.Background(),
		existingBlueprintInstanceID,
		statusInfo,
	)
	s.Require().NoError(err)

	savedState, err := instances.Get(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatusInfo(statusInfo, savedState, &s.Suite)
	s.assertPersistedInstance(&savedState)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_status_update() {
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

func (s *MemFileStateContainerInstancesTestSuite) Test_removes_blueprint_instance() {
	instances := s.container.Instances()
	_, err := instances.Remove(context.Background(), existingBlueprintInstanceID)
	s.Require().NoError(err)

	_, err = instances.Get(context.Background(), existingBlueprintInstanceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)

	s.assertInstanceRemovedFromPersistence(existingBlueprintInstanceID)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_removal() {
	instances := s.container.Instances()
	_, err := instances.Remove(context.Background(), nonExistentInstanceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *MemFileStateContainerInstancesTestSuite) assertPersistedInstance(expected *state.InstanceState) {
	// Check that the instance state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	instances := container.Instances()
	savedInstanceState, err := instances.Get(
		context.Background(),
		expected.InstanceID,
	)
	s.Require().NoError(err)
	internal.AssertInstanceStatesEqual(expected, &savedInstanceState, &s.Suite)
}

func (s *MemFileStateContainerInstancesTestSuite) assertInstanceRemovedFromPersistence(instanceID string) {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	instances := container.Instances()
	_, err = instances.Get(context.Background(), instanceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func TestMemFileStateContainerInstancesTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerInstancesTestSuite))
}
