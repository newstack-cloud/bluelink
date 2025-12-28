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
	existingResourceID    = "test-orders-table-0-id"
	existingResourceName  = "saveOrderFunction"
	nonExistentResourceID = "non-existent-resource"
)

type MemFileStateContainerResourcesTestSuite struct {
	container            state.Container
	saveResourceFixtures map[int]internal.SaveResourceFixture
	stateDir             string
	fs                   afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerResourcesTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// Use a low max guide file size of 100 bytes to trigger the logic that splits
	// instance and resource drift state across multiple chunk files.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "resources")
	saveResourceFixtures, err := internal.SetupSaveResourceFixtures(
		dirPath,
		/* updates */ []int{3},
	)
	s.Require().NoError(err)
	s.saveResourceFixtures = saveResourceFixtures
}

func (s *MemFileStateContainerResourcesTestSuite) Test_retrieves_resource() {
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

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_retrieval() {
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

func (s *MemFileStateContainerResourcesTestSuite) Test_retrieves_resource_by_logical_name() {
	resources := s.container.Resources()
	resourceState, err := resources.GetByName(
		context.Background(),
		existingBlueprintInstanceID,
		existingResourceName,
	)
	s.Require().NoError(err)
	s.Require().NotNil(resourceState)
	err = testhelpers.Snapshot(resourceState)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_retrieval_by_logical_name() {
	resources := s.container.Resources()

	_, err := resources.GetByName(
		context.Background(),
		existingBlueprintInstanceID,
		nonExistentResourceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_saves_new_resource() {
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
	s.assertPersistedResource(fixture.ResourceState)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_instance_not_found_for_saving_resource() {
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

func (s *MemFileStateContainerResourcesTestSuite) Test_updates_existing_resource() {
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
	s.assertPersistedResource(fixture.ResourceState)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_saves_resource_with_system_metadata() {
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
	s.assertPersistedResource(fixture.ResourceState)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_updates_blueprint_resource_deployment_status() {
	resources := s.container.Resources()

	statusInfo := internal.CreateTestResourceStatusInfo()
	err := resources.UpdateStatus(
		context.Background(),
		existingResourceID,
		statusInfo,
	)
	s.Require().NoError(err)

	savedState, err := resources.Get(
		context.Background(),
		existingResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceStatusInfo(statusInfo, savedState, &s.Suite)
	s.assertPersistedResource(&savedState)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_status_update() {
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

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_malformed_state_error_for_status_update() {
	// The malformed state for this test case contains a resource
	// that references an instance that does not exist.
	container, err := loadMalformedStateContainer(&s.Suite)
	s.Require().NoError(err)

	resources := container.Resources()
	statusInfo := internal.CreateTestResourceStatusInfo()
	err = resources.UpdateStatus(
		context.Background(),
		existingResourceID,
		statusInfo,
	)
	s.Require().Error(err)
	memFileErr, isMemFileErr := err.(*Error)
	s.Assert().True(isMemFileErr)
	s.Assert().Equal(ErrorReasonCodeMalformedState, memFileErr.ReasonCode)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_removes_resource() {
	resources := s.container.Resources()
	_, err := resources.Remove(context.Background(), existingResourceID)
	s.Require().NoError(err)

	_, err = resources.Get(context.Background(), existingResourceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)

	s.assertResourceRemovedFromPersistence(existingResourceID)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_resource_not_found_for_removal() {
	resources := s.container.Resources()
	_, err := resources.Remove(context.Background(), nonExistentResourceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *MemFileStateContainerResourcesTestSuite) Test_reports_malformed_state_error_for_removal() {
	// The malformed state for this test case contains a resource
	// that references an instance that does not exist.
	container, err := loadMalformedStateContainer(&s.Suite)
	s.Require().NoError(err)

	resources := container.Resources()
	_, err = resources.Remove(
		context.Background(),
		existingResourceID,
	)
	s.Require().Error(err)
	memFileErr, isMemFileErr := err.(*Error)
	s.Assert().True(isMemFileErr)
	s.Assert().Equal(ErrorReasonCodeMalformedState, memFileErr.ReasonCode)
}

func (s *MemFileStateContainerResourcesTestSuite) assertPersistedResource(expected *state.ResourceState) {
	// Check that the resource state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the resource.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	resources := container.Resources()
	savedResourceState, err := resources.Get(
		context.Background(),
		expected.ResourceID,
	)
	s.Require().NoError(err)
	internal.AssertResourceStatesEqual(expected, &savedResourceState, &s.Suite)
}

func (s *MemFileStateContainerResourcesTestSuite) assertResourceRemovedFromPersistence(resourceID string) {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	resources := container.Resources()
	_, err = resources.Get(context.Background(), resourceID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func TestMemFileStateContainerResourcesTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerResourcesTestSuite))
}
