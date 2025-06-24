package memfile

import (
	"context"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	envVarsField = "variables.environment"
)

type MemFileStateContainerExportsTestSuite struct {
	container state.Container
	stateDir  string
	fs        afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerExportsTestSuite) SetupTest() {
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

func (s *MemFileStateContainerExportsTestSuite) Test_retrieves_all_exports_for_blueprint_instance() {
	exportsContainer := s.container.Exports()

	exports, err := exportsContainer.GetAll(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(exports)
	err = testhelpers.Snapshot(exports)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_instance_not_found_for_retrieving_all_exports() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.GetAll(
		context.Background(),
		nonExistentInstanceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_retrieves_single_export_for_blueprint_instance() {
	exportsContainer := s.container.Exports()

	export, err := exportsContainer.Get(
		context.Background(),
		existingBlueprintInstanceID,
		"environment",
	)
	s.Require().NoError(err)
	s.Require().NotNil(export)
	err = testhelpers.Snapshot(export)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_instance_not_found_for_retrieving_single_export() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.Get(
		context.Background(),
		nonExistentInstanceID,
		"region",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_export_not_found_for_retrieving_single_export() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.Get(
		context.Background(),
		existingBlueprintInstanceID,
		"region",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)
}

func (s *MemFileStateContainerExportsTestSuite) Test_saves_multiple_exports_for_blueprint_instance() {
	// SaveAll overrides any existing exports for the instance.
	exportsContainer := s.container.Exports()

	exports := internal.SaveAllExportsInput()

	err := exportsContainer.SaveAll(
		context.Background(),
		existingBlueprintInstanceID,
		exports,
	)
	s.Require().NoError(err)

	savedExports, err := exportsContainer.GetAll(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(savedExports)
	s.Assert().Equal(exports, savedExports)

	s.assertPersistedExports(existingBlueprintInstanceID, exports)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_instance_not_found_for_saving_multiple_exports() {
	exportsContainer := s.container.Exports()

	exports := internal.SaveAllExportsInput()
	err := exportsContainer.SaveAll(
		context.Background(),
		nonExistentInstanceID,
		exports,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_saves_single_export_for_blueprint_instance() {
	exportsContainer := s.container.Exports()

	export := internal.SaveSingleExportInput()

	err := exportsContainer.Save(
		context.Background(),
		existingBlueprintInstanceID,
		"exampleId",
		export,
	)
	s.Require().NoError(err)

	savedExport, err := exportsContainer.Get(
		context.Background(),
		existingBlueprintInstanceID,
		"exampleId",
	)
	s.Require().NoError(err)
	s.Assert().Equal(export, savedExport)

	s.assertPersistedExport(existingBlueprintInstanceID, "exampleId", export)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_instance_not_found_for_saving_single_export() {
	exportsContainer := s.container.Exports()

	export := internal.SaveSingleExportInput()
	err := exportsContainer.Save(
		context.Background(),
		nonExistentInstanceID,
		"exampleId",
		export,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_removes_all_exports_from_blueprint_instance() {
	exportsContainer := s.container.Exports()

	removed, err := exportsContainer.RemoveAll(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)

	expectedRemovedExports := map[string]*state.ExportState{
		"environment": {
			Value: core.MappingNodeFromString("legacy-production-env"),
			Type:  schema.ExportTypeString,
			Field: envVarsField,
		},
	}
	s.Assert().Equal(expectedRemovedExports, removed)

	savedExports, err := exportsContainer.GetAll(
		context.Background(),
		existingBlueprintInstanceID,
	)
	s.Require().NoError(err)
	s.Assert().Len(savedExports, 0)

	s.assertExportsRemovalPersisted(existingBlueprintInstanceID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_instance_not_found_for_removing_all_exports() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.RemoveAll(
		context.Background(),
		nonExistentInstanceID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_removes_single_export_from_blueprint_instance() {
	exportsContainer := s.container.Exports()

	removed, err := exportsContainer.Remove(
		context.Background(),
		existingBlueprintInstanceID,
		"environment",
	)
	s.Require().NoError(err)

	expectedRemovedExport := state.ExportState{
		Value: core.MappingNodeFromString("legacy-production-env"),
		Type:  schema.ExportTypeString,
		Field: envVarsField,
	}
	s.Assert().Equal(expectedRemovedExport, removed)

	_, err = exportsContainer.Get(
		context.Background(),
		existingBlueprintInstanceID,
		"environment",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)

	s.assertExportRemovalPersisted(existingBlueprintInstanceID, "environment")
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_instance_not_found_for_removing_single_export() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.Remove(
		context.Background(),
		nonExistentInstanceID,
		"environment",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
	s.Assert().Equal(nonExistentInstanceID, stateErr.ItemID)
}

func (s *MemFileStateContainerExportsTestSuite) Test_reports_export_not_found_for_removing_single_export() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.Remove(
		context.Background(),
		existingBlueprintInstanceID,
		"environmentMissing",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)
}

func (s *MemFileStateContainerExportsTestSuite) assertPersistedExports(instanceID string, expected map[string]*state.ExportState) {
	// Check that the instance exports were saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	exportsContainer := container.Exports()
	exports, err := exportsContainer.GetAll(
		context.Background(),
		instanceID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(expected, exports)
}

func (s *MemFileStateContainerExportsTestSuite) assertPersistedExport(
	instanceID string,
	exportName string,
	expected state.ExportState,
) {
	// Check that the instance exports were saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	exportsContainer := container.Exports()
	export, err := exportsContainer.Get(
		context.Background(),
		instanceID,
		exportName,
	)
	s.Require().NoError(err)
	s.Assert().Equal(expected, export)
}

func (s *MemFileStateContainerExportsTestSuite) assertExportsRemovalPersisted(instanceID string) {
	// Check that the removal of all instance exports was persisted to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	exportsContainer := container.Exports()
	exports, err := exportsContainer.GetAll(
		context.Background(),
		instanceID,
	)
	s.Require().NoError(err)
	s.Assert().Len(exports, 0)
}

func (s *MemFileStateContainerExportsTestSuite) assertExportRemovalPersisted(instanceID string, exportName string) {
	// Check that the removal of a single export was persisted to "disk" correctly by
	// loading a new state container from persistence and retrieving the instance
	// by instance ID.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	exportsContainer := container.Exports()
	_, err = exportsContainer.Get(
		context.Background(),
		instanceID,
		exportName,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)
}

func TestMemFileStateContainerExportsTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerExportsTestSuite))
}
