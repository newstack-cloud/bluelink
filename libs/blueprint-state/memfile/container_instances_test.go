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
	existingBlueprintInstanceID   = "blueprint-instance-1"
	existingBlueprintInstanceName = "BlueprintInstance1"
	nonExistentInstanceID         = "non-existent-instance"
	nonExistentInstanceName       = "NonExistentInstance"
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

func (s *MemFileStateContainerInstancesTestSuite) Test_looks_up_instance_id_by_name() {
	instances := s.container.Instances()
	instanceID, err := instances.LookupIDByName(
		context.Background(),
		existingBlueprintInstanceName,
	)
	s.Require().NoError(err)
	s.Assert().Equal(existingBlueprintInstanceID, instanceID)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_reports_instance_not_found_for_lookup_by_name() {
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

func (s *MemFileStateContainerInstancesTestSuite) Test_looks_up_newly_saved_instance_by_name() {
	fixture := s.saveBlueprintFixtures[1]
	instances := s.container.Instances()

	// Save a new instance
	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	// Lookup by name should work immediately after save
	instanceID, err := instances.LookupIDByName(
		context.Background(),
		fixture.InstanceState.InstanceName,
	)
	s.Require().NoError(err)
	s.Assert().Equal(fixture.InstanceState.InstanceID, instanceID)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lookup_by_name_works_after_reloading_state() {
	fixture := s.saveBlueprintFixtures[1]
	instances := s.container.Instances()

	// Save a new instance
	err := instances.Save(
		context.Background(),
		*fixture.InstanceState,
	)
	s.Require().NoError(err)

	// Load a fresh state container from disk
	freshContainer, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	// Lookup by name should work after reload
	freshInstances := freshContainer.Instances()
	instanceID, err := freshInstances.LookupIDByName(
		context.Background(),
		fixture.InstanceState.InstanceName,
	)
	s.Require().NoError(err)
	s.Assert().Equal(fixture.InstanceState.InstanceID, instanceID)
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

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_all_instances() {
	instances := s.container.Instances()
	result, err := instances.List(context.Background(), state.ListInstancesParams{})
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(result.TotalCount, 1)
	s.Assert().GreaterOrEqual(len(result.Instances), 1)

	var found bool
	for _, inst := range result.Instances {
		if inst.InstanceID == existingBlueprintInstanceID {
			s.Assert().Equal(existingBlueprintInstanceName, inst.InstanceName)
			found = true
			break
		}
	}
	s.Assert().True(found, "expected to find the existing blueprint instance")
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_with_search_filter() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.List(context.Background(), state.ListInstancesParams{
		Search: "prod",
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, result.TotalCount)
	s.Assert().Len(result.Instances, 1)
	s.Assert().Equal("my-app-production", result.Instances[0].InstanceName)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_search_is_case_insensitive() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.List(context.Background(), state.ListInstancesParams{
		Search: "STAGING",
	})
	s.Require().NoError(err)
	s.Assert().Equal(1, result.TotalCount)
	s.Assert().Equal("my-app-staging", result.Instances[0].InstanceName)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_returns_empty_when_no_matches() {
	instances := s.container.Instances()

	result, err := instances.List(context.Background(), state.ListInstancesParams{
		Search: "nonexistent-search-term",
	})
	s.Require().NoError(err)
	s.Assert().Equal(0, result.TotalCount)
	s.Assert().Empty(result.Instances)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_sorted_by_name() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.List(context.Background(), state.ListInstancesParams{})
	s.Require().NoError(err)

	names := make([]string, len(result.Instances))
	for i, inst := range result.Instances {
		names[i] = inst.InstanceName
	}

	sortedNames := make([]string, len(names))
	copy(sortedNames, names)
	s.Assert().Equal(sortedNames, names, "instances should be sorted alphabetically by name")
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_with_pagination() {
	s.saveTestInstances()
	instances := s.container.Instances()

	// Get total count first so test is resilient to initial data changes
	allResult, err := instances.List(context.Background(), state.ListInstancesParams{})
	s.Require().NoError(err)
	totalCount := allResult.TotalCount

	result, err := instances.List(context.Background(), state.ListInstancesParams{
		Offset: 1,
		Limit:  2,
	})
	s.Require().NoError(err)
	s.Assert().Equal(totalCount, result.TotalCount)
	expectedLen := min(2, totalCount-1)
	s.Assert().Len(result.Instances, expectedLen)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_returns_total_count_before_pagination() {
	s.saveTestInstances()
	instances := s.container.Instances()

	// Get total count first
	allResult, err := instances.List(context.Background(), state.ListInstancesParams{})
	s.Require().NoError(err)
	totalCount := allResult.TotalCount

	result, err := instances.List(context.Background(), state.ListInstancesParams{
		Limit: 1,
	})
	s.Require().NoError(err)
	s.Assert().Equal(totalCount, result.TotalCount)
	s.Assert().Len(result.Instances, 1)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_lists_instances_handles_offset_beyond_results() {
	instances := s.container.Instances()

	// Get total count first
	allResult, err := instances.List(context.Background(), state.ListInstancesParams{})
	s.Require().NoError(err)
	totalCount := allResult.TotalCount

	result, err := instances.List(context.Background(), state.ListInstancesParams{
		Offset: 100,
	})
	s.Require().NoError(err)
	s.Assert().Equal(totalCount, result.TotalCount)
	s.Assert().Empty(result.Instances)
}

func (s *MemFileStateContainerInstancesTestSuite) saveTestInstances() {
	instances := s.container.Instances()

	testInstances := []state.InstanceState{
		{
			InstanceID:            "test-instance-prod",
			InstanceName:          "my-app-production",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1704067200,
		},
		{
			InstanceID:            "test-instance-staging",
			InstanceName:          "my-app-staging",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1704067100,
		},
	}

	for _, inst := range testInstances {
		err := instances.Save(context.Background(), inst)
		s.Require().NoError(err)
	}
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

func (s *MemFileStateContainerInstancesTestSuite) Test_get_batch_retrieves_instances_by_id() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.GetBatch(context.Background(), []string{
		"test-instance-prod",
		"test-instance-staging",
	})
	s.Require().NoError(err)
	s.Require().Len(result, 2)
	s.Assert().Equal("test-instance-prod", result[0].InstanceID)
	s.Assert().Equal("test-instance-staging", result[1].InstanceID)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_get_batch_retrieves_instances_by_name() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.GetBatch(context.Background(), []string{
		"my-app-production",
		"my-app-staging",
	})
	s.Require().NoError(err)
	s.Require().Len(result, 2)
	s.Assert().Equal("my-app-production", result[0].InstanceName)
	s.Assert().Equal("my-app-staging", result[1].InstanceName)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_get_batch_retrieves_instances_by_mixed_id_and_name() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.GetBatch(context.Background(), []string{
		"test-instance-prod",
		"my-app-staging",
	})
	s.Require().NoError(err)
	s.Require().Len(result, 2)
	s.Assert().Equal("test-instance-prod", result[0].InstanceID)
	s.Assert().Equal("my-app-staging", result[1].InstanceName)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_get_batch_reports_all_missing_instances() {
	s.saveTestInstances()
	instances := s.container.Instances()

	_, err := instances.GetBatch(context.Background(), []string{
		"test-instance-prod",
		"nonexistent-instance-1",
		"nonexistent-instance-2",
	})
	s.Require().Error(err)
	s.Assert().True(state.IsInstancesNotFound(err))
	notFoundErr, ok := err.(*state.InstancesNotFoundError)
	s.Require().True(ok)
	s.Assert().ElementsMatch(
		[]string{"nonexistent-instance-1", "nonexistent-instance-2"},
		notFoundErr.MissingIDsOrNames,
	)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_get_batch_returns_empty_for_empty_input() {
	instances := s.container.Instances()

	result, err := instances.GetBatch(context.Background(), []string{})
	s.Require().NoError(err)
	s.Assert().Empty(result)
}

func (s *MemFileStateContainerInstancesTestSuite) Test_get_batch_preserves_order() {
	s.saveTestInstances()
	instances := s.container.Instances()

	result, err := instances.GetBatch(context.Background(), []string{
		"my-app-staging",
		"my-app-production",
	})
	s.Require().NoError(err)
	s.Require().Len(result, 2)
	s.Assert().Equal("my-app-staging", result[0].InstanceName)
	s.Assert().Equal("my-app-production", result[1].InstanceName)
}

func TestMemFileStateContainerInstancesTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerInstancesTestSuite))
}
