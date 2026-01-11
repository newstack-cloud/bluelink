package stateio

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type StateExportTestSuite struct {
	suite.Suite
	fs           afero.Fs
	stateDir     string
	engineConfig *EngineConfig
}

func (s *StateExportTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
	s.stateDir = "/test/state"
	s.Require().NoError(s.fs.MkdirAll(s.stateDir, 0755))
	s.engineConfig = &EngineConfig{
		State: StateConfig{
			StorageEngine:   StorageEngineMemfile,
			MemFileStateDir: s.stateDir,
		},
	}
}

func (s *StateExportTestSuite) seedInstances(instances []state.InstanceState) {
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     "/test/seed.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)
}

func (s *StateExportTestSuite) Test_exports_all_instances_to_json() {
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Test Instance 1",
			Status:       core.InstanceStatusDeployed,
		},
		{
			InstanceID:   "inst-002",
			InstanceName: "Test Instance 2",
			Status:       core.InstanceStatusDeployed,
		},
	})

	result, err := Export(ExportParams{
		FilePath:     "/test/export.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(2, result.InstancesCount)
	s.Equal("/test/export.json", result.FilePath)
	s.Contains(result.Message, "Successfully exported 2 instances")

	// Verify the output file was created and contains valid JSON
	data, err := afero.ReadFile(s.fs, "/test/export.json")
	s.Require().NoError(err)

	var exported []state.InstanceState
	err = json.Unmarshal(data, &exported)
	s.Require().NoError(err)
	s.Len(exported, 2)
}

func (s *StateExportTestSuite) Test_exports_empty_state() {
	// Don't seed any instances - state is empty

	result, err := Export(ExportParams{
		FilePath:     "/test/export.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(0, result.InstancesCount)
	s.Contains(result.Message, "Successfully exported 0 instances")

	// Verify the output file contains an empty array
	data, err := afero.ReadFile(s.fs, "/test/export.json")
	s.Require().NoError(err)

	var exported []state.InstanceState
	err = json.Unmarshal(data, &exported)
	s.Require().NoError(err)
	s.Len(exported, 0)
}

func (s *StateExportTestSuite) Test_exports_filtered_instances_by_id() {
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Test Instance 1",
			Status:       core.InstanceStatusDeployed,
		},
		{
			InstanceID:   "inst-002",
			InstanceName: "Test Instance 2",
			Status:       core.InstanceStatusDeployed,
		},
		{
			InstanceID:   "inst-003",
			InstanceName: "Test Instance 3",
			Status:       core.InstanceStatusDeployed,
		},
	})

	result, err := Export(ExportParams{
		FilePath:        "/test/export.json",
		InstanceFilters: []string{"inst-001", "inst-003"},
		EngineConfig:    s.engineConfig,
		FileSystem:      s.fs,
		Logger:          core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(2, result.InstancesCount)

	// Verify only the requested instances are exported
	data, err := afero.ReadFile(s.fs, "/test/export.json")
	s.Require().NoError(err)

	var exported []state.InstanceState
	err = json.Unmarshal(data, &exported)
	s.Require().NoError(err)
	s.Len(exported, 2)

	// Check IDs are correct (order preserved)
	s.Equal("inst-001", exported[0].InstanceID)
	s.Equal("inst-003", exported[1].InstanceID)
}

func (s *StateExportTestSuite) Test_exports_filtered_instances_by_name() {
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "my-app-prod",
			Status:       core.InstanceStatusDeployed,
		},
		{
			InstanceID:   "inst-002",
			InstanceName: "my-app-staging",
			Status:       core.InstanceStatusDeployed,
		},
	})

	result, err := Export(ExportParams{
		FilePath:        "/test/export.json",
		InstanceFilters: []string{"my-app-prod"},
		EngineConfig:    s.engineConfig,
		FileSystem:      s.fs,
		Logger:          core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(1, result.InstancesCount)

	// Verify only the requested instance is exported
	data, err := afero.ReadFile(s.fs, "/test/export.json")
	s.Require().NoError(err)

	var exported []state.InstanceState
	err = json.Unmarshal(data, &exported)
	s.Require().NoError(err)
	s.Len(exported, 1)
	s.Equal("inst-001", exported[0].InstanceID)
	s.Equal("my-app-prod", exported[0].InstanceName)
}

func (s *StateExportTestSuite) Test_exports_instances_with_nested_children() {
	childInstance := state.InstanceState{
		InstanceID:   "child-001",
		InstanceName: "Child Instance",
		Status:       core.InstanceStatusDeployed,
	}
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "parent-001",
			InstanceName: "Parent Instance",
			Status:       core.InstanceStatusDeployed,
			ChildBlueprints: map[string]*state.InstanceState{
				"child": &childInstance,
			},
		},
	})

	// Export specifically the parent to verify child blueprints are included
	result, err := Export(ExportParams{
		FilePath:        "/test/export.json",
		InstanceFilters: []string{"parent-001"},
		EngineConfig:    s.engineConfig,
		FileSystem:      s.fs,
		Logger:          core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(1, result.InstancesCount)

	// Verify nested children are included
	data, err := afero.ReadFile(s.fs, "/test/export.json")
	s.Require().NoError(err)

	var exported []state.InstanceState
	err = json.Unmarshal(data, &exported)
	s.Require().NoError(err)
	s.Len(exported, 1)
	s.Require().NotNil(exported[0].ChildBlueprints)
	s.Contains(exported[0].ChildBlueprints, "child")
	s.Equal("child-001", exported[0].ChildBlueprints["child"].InstanceID)
}

func (s *StateExportTestSuite) Test_export_fails_for_nonexistent_instance() {
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Test Instance",
			Status:       core.InstanceStatusDeployed,
		},
	})

	_, err := Export(ExportParams{
		FilePath:        "/test/export.json",
		InstanceFilters: []string{"nonexistent-id"},
		EngineConfig:    s.engineConfig,
		FileSystem:      s.fs,
		Logger:          core.NewNopLogger(),
	})

	s.Require().Error(err)
	var exportErr *ExportError
	s.ErrorAs(err, &exportErr)
	s.Equal(ErrCodeInstanceNotFound, exportErr.Code)
}

func (s *StateExportTestSuite) Test_export_fails_for_multiple_nonexistent_instances() {
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Test Instance",
			Status:       core.InstanceStatusDeployed,
		},
	})

	_, err := Export(ExportParams{
		FilePath:        "/test/export.json",
		InstanceFilters: []string{"missing-1", "missing-2"},
		EngineConfig:    s.engineConfig,
		FileSystem:      s.fs,
		Logger:          core.NewNopLogger(),
	})

	s.Require().Error(err)
	var exportErr *ExportError
	s.ErrorAs(err, &exportErr)
	s.Equal(ErrCodeInstanceNotFound, exportErr.Code)
	// Should contain both missing IDs in the error
	s.Contains(exportErr.Message, "missing-1")
	s.Contains(exportErr.Message, "missing-2")
}

func (s *StateExportTestSuite) Test_export_requires_engine_config() {
	_, err := Export(ExportParams{
		FilePath: "/test/export.json",
	})

	s.Require().Error(err)
	s.Contains(err.Error(), "engine config is required")
}

func (s *StateExportTestSuite) Test_export_to_real_filesystem() {
	// Use real filesystem for this test
	tempDir, err := os.MkdirTemp("", "export-test-*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	stateDir := filepath.Join(tempDir, "state")
	s.Require().NoError(os.MkdirAll(stateDir, 0755))

	realFs := afero.NewOsFs()
	realConfig := &EngineConfig{
		State: StateConfig{
			StorageEngine:   StorageEngineMemfile,
			MemFileStateDir: stateDir,
		},
	}

	// First import some data
	instances := []state.InstanceState{
		{
			InstanceID:   "real-001",
			InstanceName: "Real Test Instance",
			Status:       core.InstanceStatusDeployed,
		},
	}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     filepath.Join(tempDir, "input.json"),
		EngineConfig: realConfig,
		FileSystem:   realFs,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)

	// Export to real file
	outputPath := filepath.Join(tempDir, "export.json")
	result, err := Export(ExportParams{
		FilePath:     outputPath,
		EngineConfig: realConfig,
		FileSystem:   realFs,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(1, result.InstancesCount)

	// Verify file exists and can be read
	data, err := os.ReadFile(outputPath)
	s.Require().NoError(err)

	var exported []state.InstanceState
	err = json.Unmarshal(data, &exported)
	s.Require().NoError(err)
	s.Len(exported, 1)
	s.Equal("real-001", exported[0].InstanceID)
}

func (s *StateExportTestSuite) Test_exported_data_can_be_reimported() {
	// Seed with complex data
	s.seedInstances([]state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Instance One",
			Status:       core.InstanceStatusDeployed,
			ResourceIDs: map[string]string{
				"resource1": "res-001",
				"resource2": "res-002",
			},
		},
		{
			InstanceID:   "inst-002",
			InstanceName: "Instance Two",
			Status:       core.InstanceStatusDeployed,
		},
	})

	// Export
	exportPath := "/test/export.json"
	_, err := Export(ExportParams{
		FilePath:     exportPath,
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)

	// Create a fresh state directory for re-import
	freshStateDir := "/test/fresh-state"
	s.Require().NoError(s.fs.MkdirAll(freshStateDir, 0755))
	freshConfig := &EngineConfig{
		State: StateConfig{
			StorageEngine:   StorageEngineMemfile,
			MemFileStateDir: freshStateDir,
		},
	}

	// Read exported data
	exportedData, err := afero.ReadFile(s.fs, exportPath)
	s.Require().NoError(err)

	// Re-import into fresh state
	importResult, err := Import(ImportParams{
		FilePath:     exportPath,
		EngineConfig: freshConfig,
		FileSystem:   s.fs,
		FileData:     exportedData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.Equal(2, importResult.InstancesCount)

	// Verify data integrity
	container, err := memfile.LoadStateContainer(freshStateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	ctx := context.Background()
	inst1, err := container.Instances().Get(ctx, "inst-001")
	s.Require().NoError(err)
	s.Equal("Instance One", inst1.InstanceName)
	s.Require().NotNil(inst1.ResourceIDs)
	s.Equal("res-001", inst1.ResourceIDs["resource1"])
}

func TestStateExportTestSuite(t *testing.T) {
	suite.Run(t, new(StateExportTestSuite))
}
