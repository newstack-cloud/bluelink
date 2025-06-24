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

type MemFileStateContainerMetadataTestSuite struct {
	container state.Container
	stateDir  string
	fs        afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerMetadataTestSuite) SetupTest() {
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
}

func (s *MemFileStateContainerMetadataTestSuite) Test_retrieves_metadata_for_blueprint_instance() {
	metadataContainer := s.container.Metadata()

	metadata, err := metadataContainer.Get(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(metadata)
	err = testhelpers.Snapshot(metadata)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerMetadataTestSuite) Test_reports_instance_not_found_when_retrieving_metadata() {
	metadataContainer := s.container.Metadata()

	_, err := metadataContainer.Get(
		context.Background(),
		nonExistentInstanceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerMetadataTestSuite) Test_saves_metadata_for_blueprint_instance() {
	metadataContainer := s.container.Metadata()

	metadata := internal.SaveMetadataInput()

	err := metadataContainer.Save(
		context.Background(),
		existingBlueprintInstanceID,
		metadata,
	)
	s.Require().NoError(err)

	savedMetadata, err := metadataContainer.Get(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(metadata, savedMetadata)

	s.assertPersistedMetadata(existingBlueprintInstanceID, metadata)
}

func (s *MemFileStateContainerMetadataTestSuite) Test_reports_instance_not_found_when_saving_metadata() {
	metadataContainer := s.container.Metadata()

	metadata := internal.SaveMetadataInput()
	err := metadataContainer.Save(
		context.Background(),
		nonExistentInstanceID,
		metadata,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerMetadataTestSuite) Test_removes_metadata_from_blueprint_instance() {
	metadataContainer := s.container.Metadata()

	removed, err := metadataContainer.Remove(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)

	expectedRemovedMetadata := map[string]*core.MappingNode{
		"build": core.MappingNodeFromString("tsc"),
	}
	s.Assert().Equal(expectedRemovedMetadata, removed)

	s.assertMetadataRemovalPersisted(existingBlueprintInstanceID)
}

func (s *MemFileStateContainerMetadataTestSuite) Test_reports_instance_not_found_when_removing_metadata() {
	metadataContainer := s.container.Metadata()

	_, err := metadataContainer.Remove(
		context.Background(),
		nonExistentInstanceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerMetadataTestSuite) assertPersistedMetadata(
	instanceID string,
	expected map[string]*core.MappingNode,
) {
	// Check that the instance metadata was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	metadataContainer := container.Metadata()
	metadata, err := metadataContainer.Get(
		context.Background(),
		instanceID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(expected, metadata)
}

func (s *MemFileStateContainerMetadataTestSuite) assertMetadataRemovalPersisted(instanceID string) {
	// Check that the instance metadata removal was persisted to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	metadataContainer := container.Metadata()
	metadata, err := metadataContainer.Get(
		context.Background(),
		instanceID,
	)
	s.Require().NoError(err)
	s.Assert().Len(metadata, 0)
}

func TestMemFileStateContainerMetadataTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerMetadataTestSuite))
}
