package stateio

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type StateImportTestSuite struct {
	suite.Suite
	fs           afero.Fs
	stateDir     string
	engineConfig *EngineConfig
}

func (s *StateImportTestSuite) SetupTest() {
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

func (s *StateImportTestSuite) Test_imports_instances_from_json() {
	instances := []state.InstanceState{
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
	}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	result, err := Import(ImportParams{
		FilePath:     "/test/state.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(2, result.InstancesCount)

	// Verify instances are actually persisted in the state container
	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	ctx := context.Background()

	inst1, err := container.Instances().Get(ctx, "inst-001")
	s.Require().NoError(err)
	s.Equal("inst-001", inst1.InstanceID)
	s.Equal("Test Instance 1", inst1.InstanceName)
	s.Equal(core.InstanceStatusDeployed, inst1.Status)

	inst2, err := container.Instances().Get(ctx, "inst-002")
	s.Require().NoError(err)
	s.Equal("inst-002", inst2.InstanceID)
	s.Equal("Test Instance 2", inst2.InstanceName)
	s.Equal(core.InstanceStatusDeployed, inst2.Status)
}

func (s *StateImportTestSuite) Test_imports_empty_array() {
	jsonData := []byte("[]")

	result, err := Import(ImportParams{
		FilePath:     "/test/empty.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(0, result.InstancesCount)

	// Verify no instances in the state container
	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	ctx := context.Background()
	listResult, err := container.Instances().List(ctx, state.ListInstancesParams{})
	s.Require().NoError(err)
	s.Equal(0, listResult.TotalCount)
}

func (s *StateImportTestSuite) Test_rejects_invalid_json() {
	_, err := Import(ImportParams{
		FilePath:     "/test/instances.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     []byte("not valid json"),
		Logger:       core.NewNopLogger(),
	})

	s.Require().Error(err)

	var importErr *ImportError
	s.True(errors.As(err, &importErr))
	s.Equal(ErrCodeInvalidJSON, importErr.Code)
}

func (s *StateImportTestSuite) Test_rejects_non_array_json() {
	_, err := Import(ImportParams{
		FilePath:     "/test/instances.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     []byte(`{"instanceId": "single"}`),
		Logger:       core.NewNopLogger(),
	})

	s.Require().Error(err)

	var importErr *ImportError
	s.True(errors.As(err, &importErr))
	s.Equal(ErrCodeInvalidJSON, importErr.Code)
}

func (s *StateImportTestSuite) Test_import_requires_engine_config() {
	params := ImportParams{
		FilePath: "/nonexistent/file.json",
		FileData: []byte("[]"),
	}

	_, err := Import(params)
	s.Require().Error(err)
	s.Contains(err.Error(), "engine config is required")
}

func (s *StateImportTestSuite) Test_imports_instances_with_nested_children() {
	childInstance := state.InstanceState{
		InstanceID:   "child-001",
		InstanceName: "Child Instance",
		Status:       core.InstanceStatusDeployed,
	}
	parentInstance := state.InstanceState{
		InstanceID:   "parent-001",
		InstanceName: "Parent Instance",
		Status:       core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"child": &childInstance,
		},
	}

	instances := []state.InstanceState{parentInstance}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	result, err := Import(ImportParams{
		FilePath:     "/test/nested.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(1, result.InstancesCount)

	// Verify parent instance is persisted with nested child
	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	ctx := context.Background()
	parent, err := container.Instances().Get(ctx, "parent-001")
	s.Require().NoError(err)
	s.Equal("parent-001", parent.InstanceID)
	s.Equal("Parent Instance", parent.InstanceName)
	s.Require().NotNil(parent.ChildBlueprints)
	s.Require().Contains(parent.ChildBlueprints, "child")
	s.Equal("child-001", parent.ChildBlueprints["child"].InstanceID)
	s.Equal("Child Instance", parent.ChildBlueprints["child"].InstanceName)
}

func (s *StateImportTestSuite) Test_import_overwrites_existing_instance() {
	// First, import an instance
	originalInstances := []state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Original Name",
			Status:       core.InstanceStatusDeploying,
		},
	}
	originalJSON, err := json.Marshal(originalInstances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     "/test/original.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     originalJSON,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)

	// Now import the same instance with updated data
	updatedInstances := []state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Updated Name",
			Status:       core.InstanceStatusDeployed,
		},
	}
	updatedJSON, err := json.Marshal(updatedInstances)
	s.Require().NoError(err)

	result, err := Import(ImportParams{
		FilePath:     "/test/updated.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     updatedJSON,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(1, result.InstancesCount)

	// Verify the instance was updated
	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	ctx := context.Background()
	inst, err := container.Instances().Get(ctx, "inst-001")
	s.Require().NoError(err)
	s.Equal("Updated Name", inst.InstanceName)
	s.Equal(core.InstanceStatusDeployed, inst.Status)
}

func (s *StateImportTestSuite) Test_import_can_lookup_instance_by_name() {
	instances := []state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "my-unique-instance",
			Status:       core.InstanceStatusDeployed,
		},
	}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     "/test/state.json",
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)

	// Verify we can look up the instance by name
	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	ctx := context.Background()
	instanceID, err := container.Instances().LookupIDByName(ctx, "my-unique-instance")
	s.Require().NoError(err)
	s.Equal("inst-001", instanceID)
}

func TestStateImportTestSuite(t *testing.T) {
	suite.Run(t, new(StateImportTestSuite))
}
