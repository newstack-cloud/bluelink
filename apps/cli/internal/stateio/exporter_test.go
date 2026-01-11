package stateio

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type ExporterTestSuite struct {
	suite.Suite
	fs        afero.Fs
	stateDir  string
	container state.Container
}

func (s *ExporterTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
	s.stateDir = "/test/state"
	s.Require().NoError(s.fs.MkdirAll(s.stateDir, 0755))

	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *ExporterTestSuite) seedInstance(inst state.InstanceState) {
	ctx := context.Background()
	err := s.container.Instances().Save(ctx, inst)
	s.Require().NoError(err)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_exports_all_instances() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "Instance One",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-002",
		InstanceName: "Instance Two",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	instances, err := exporter.ExportInstances(ctx, nil)

	s.Require().NoError(err)
	s.Len(instances, 2)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_exports_empty_list() {
	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	instances, err := exporter.ExportInstances(ctx, nil)

	s.Require().NoError(err)
	s.Len(instances, 0)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_exports_filtered_by_id() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "Instance One",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-002",
		InstanceName: "Instance Two",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	instances, err := exporter.ExportInstances(ctx, []string{"inst-001"})

	s.Require().NoError(err)
	s.Len(instances, 1)
	s.Equal("inst-001", instances[0].InstanceID)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_exports_filtered_by_name() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "my-app-prod",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-002",
		InstanceName: "my-app-staging",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	instances, err := exporter.ExportInstances(ctx, []string{"my-app-prod"})

	s.Require().NoError(err)
	s.Len(instances, 1)
	s.Equal("inst-001", instances[0].InstanceID)
	s.Equal("my-app-prod", instances[0].InstanceName)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_exports_with_mixed_ids_and_names() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "Instance One",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-002",
		InstanceName: "Instance Two",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-003",
		InstanceName: "Instance Three",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	// Mix of ID and name
	instances, err := exporter.ExportInstances(ctx, []string{"inst-001", "Instance Three"})

	s.Require().NoError(err)
	s.Len(instances, 2)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_returns_error_for_nonexistent() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "Instance One",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	_, err := exporter.ExportInstances(ctx, []string{"nonexistent"})

	s.Require().Error(err)
	var exportErr *ExportError
	s.ErrorAs(err, &exportErr)
	s.Equal(ErrCodeInstanceNotFound, exportErr.Code)
}

func (s *ExporterTestSuite) Test_ContainerStateExporter_exports_nested_children() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "parent-001",
		InstanceName: "Parent Instance",
		Status:       core.InstanceStatusDeployed,
		ChildBlueprints: map[string]*state.InstanceState{
			"child": {
				InstanceID:   "child-001",
				InstanceName: "Child Instance",
				Status:       core.InstanceStatusDeployed,
			},
		},
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	// Export specifically the parent to verify child blueprints are included
	instances, err := exporter.ExportInstances(ctx, []string{"parent-001"})

	s.Require().NoError(err)
	s.Len(instances, 1)
	s.Require().NotNil(instances[0].ChildBlueprints)
	s.Contains(instances[0].ChildBlueprints, "child")
	s.Equal("child-001", instances[0].ChildBlueprints["child"].InstanceID)
}

func (s *ExporterTestSuite) Test_SerializeInstancesJSON_produces_valid_json() {
	instances := []state.InstanceState{
		{
			InstanceID:   "inst-001",
			InstanceName: "Test Instance",
			Status:       core.InstanceStatusDeployed,
		},
	}

	data, err := SerializeInstancesJSON(instances)

	s.Require().NoError(err)
	s.NotEmpty(data)

	// Verify it's valid JSON by parsing it back
	var parsed []state.InstanceState
	err = json.Unmarshal(data, &parsed)
	s.Require().NoError(err)
	s.Len(parsed, 1)
	s.Equal("inst-001", parsed[0].InstanceID)
}

func (s *ExporterTestSuite) Test_SerializeInstancesJSON_handles_empty_slice() {
	instances := []state.InstanceState{}

	data, err := SerializeInstancesJSON(instances)

	s.Require().NoError(err)
	s.Equal("[]", string(data))
}

func (s *ExporterTestSuite) Test_ExecuteInstancesExport_returns_count_and_data() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "Instance One",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-002",
		InstanceName: "Instance Two",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	result, err := ExecuteInstancesExport(ctx, exporter, nil)

	s.Require().NoError(err)
	s.Equal(2, result.InstancesCount)
	s.NotEmpty(result.Data)

	// Verify data is valid JSON
	var parsed []state.InstanceState
	err = json.Unmarshal(result.Data, &parsed)
	s.Require().NoError(err)
	s.Len(parsed, 2)
}

func (s *ExporterTestSuite) Test_ExecuteInstancesExport_with_filter() {
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-001",
		InstanceName: "Instance One",
		Status:       core.InstanceStatusDeployed,
	})
	s.seedInstance(state.InstanceState{
		InstanceID:   "inst-002",
		InstanceName: "Instance Two",
		Status:       core.InstanceStatusDeployed,
	})

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	result, err := ExecuteInstancesExport(ctx, exporter, []string{"inst-002"})

	s.Require().NoError(err)
	s.Equal(1, result.InstancesCount)

	var parsed []state.InstanceState
	err = json.Unmarshal(result.Data, &parsed)
	s.Require().NoError(err)
	s.Len(parsed, 1)
	s.Equal("inst-002", parsed[0].InstanceID)
}

func TestExporterTestSuite(t *testing.T) {
	suite.Run(t, new(ExporterTestSuite))
}
