package stateio

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// Test UUIDs for postgres export integration tests
const (
	testExportInstanceID1      = "e1f2a3b4-c5d6-7890-abcd-ef1234567801"
	testExportInstanceID2      = "e1f2a3b4-c5d6-7890-abcd-ef1234567802"
	testExportInstanceID3      = "e1f2a3b4-c5d6-7890-abcd-ef1234567803"
	testExportParentInstanceID = "f2a3b4c5-d6e7-8901-bcde-f12345678901"
	testExportChildInstanceID  = "f2a3b4c5-d6e7-8901-bcde-f12345678902"
)

type PostgresExportIntegrationSuite struct {
	suite.Suite
	connPool     *pgxpool.Pool
	engineConfig *EngineConfig
	container    state.Container
}

func TestPostgresExportIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PostgresExportIntegrationSuite))
}

func (s *PostgresExportIntegrationSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestPostgresDatabaseURL())
	s.Require().NoError(err)
	s.connPool = connPool

	port, err := strconv.Atoi(os.Getenv("STATEIO_POSTGRES_PORT"))
	s.Require().NoError(err)

	poolMaxConns, _ := strconv.Atoi(os.Getenv("STATEIO_POSTGRES_POOL_MAX_CONNS"))
	if poolMaxConns == 0 {
		poolMaxConns = 10
	}

	poolMaxConnLifetime := os.Getenv("STATEIO_POSTGRES_POOL_MAX_CONN_LIFETIME")
	if poolMaxConnLifetime == "" {
		poolMaxConnLifetime = "1h30m"
	}

	s.engineConfig = &EngineConfig{
		State: StateConfig{
			StorageEngine:               StorageEnginePostgres,
			PostgresHost:                os.Getenv("STATEIO_POSTGRES_HOST"),
			PostgresPort:                port,
			PostgresUser:                os.Getenv("STATEIO_POSTGRES_USER"),
			PostgresPassword:            os.Getenv("STATEIO_POSTGRES_PASSWORD"),
			PostgresDatabase:            os.Getenv("STATEIO_POSTGRES_DATABASE"),
			PostgresSSLMode:             "disable",
			PostgresPoolMaxConns:        poolMaxConns,
			PostgresPoolMaxConnLifetime: poolMaxConnLifetime,
		},
	}

	container, err := postgres.LoadStateContainer(ctx, s.connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *PostgresExportIntegrationSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresExportIntegrationSuite) seedInstance(inst state.InstanceState) {
	ctx := context.Background()
	err := s.container.Instances().Save(ctx, inst)
	s.Require().NoError(err)
}

func (s *PostgresExportIntegrationSuite) cleanupInstance(instanceID string) {
	ctx := context.Background()
	_, _ = s.container.Instances().Remove(ctx, instanceID)
}

func (s *PostgresExportIntegrationSuite) Test_exports_instances_from_postgres() {
	s.seedInstance(newTestInstance(testExportInstanceID1, "Postgres Export Instance 1", core.InstanceStatusDeployed))
	s.seedInstance(newTestInstance(testExportInstanceID2, "Postgres Export Instance 2", core.InstanceStatusDeployed))
	defer s.cleanupInstance(testExportInstanceID1)
	defer s.cleanupInstance(testExportInstanceID2)

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	// Use exporter directly to avoid file system operations
	result, err := ExecuteInstancesExport(ctx, exporter, []string{testExportInstanceID1, testExportInstanceID2})

	s.Require().NoError(err)
	s.Equal(2, result.InstancesCount)
	s.NotEmpty(result.Data)

	// Verify data is valid JSON
	var instances []state.InstanceState
	err = json.Unmarshal(result.Data, &instances)
	s.Require().NoError(err)
	s.Len(instances, 2)
}

func (s *PostgresExportIntegrationSuite) Test_exports_filtered_instances_from_postgres() {
	s.seedInstance(newTestInstance(testExportInstanceID1, "Postgres Filter Test 1", core.InstanceStatusDeployed))
	s.seedInstance(newTestInstance(testExportInstanceID2, "Postgres Filter Test 2", core.InstanceStatusDeployed))
	s.seedInstance(newTestInstance(testExportInstanceID3, "Postgres Filter Test 3", core.InstanceStatusDeployed))
	defer s.cleanupInstance(testExportInstanceID1)
	defer s.cleanupInstance(testExportInstanceID2)
	defer s.cleanupInstance(testExportInstanceID3)

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	// Export only two of the three instances
	result, err := ExecuteInstancesExport(ctx, exporter, []string{testExportInstanceID1, testExportInstanceID3})

	s.Require().NoError(err)
	s.Equal(2, result.InstancesCount)

	// Verify data contains only filtered instances
	var instances []state.InstanceState
	err = json.Unmarshal(result.Data, &instances)
	s.Require().NoError(err)
	s.Len(instances, 2)
}

func (s *PostgresExportIntegrationSuite) Test_exports_single_instance_from_postgres() {
	s.seedInstance(newTestInstance(testExportInstanceID1, "postgres-single-export-test", core.InstanceStatusDeployed))
	defer s.cleanupInstance(testExportInstanceID1)

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	// Export single instance by ID
	result, err := ExecuteInstancesExport(ctx, exporter, []string{testExportInstanceID1})

	s.Require().NoError(err)
	s.Equal(1, result.InstancesCount)

	// Verify data contains the instance
	var instances []state.InstanceState
	err = json.Unmarshal(result.Data, &instances)
	s.Require().NoError(err)
	s.Len(instances, 1)
	s.Equal(testExportInstanceID1, instances[0].InstanceID)
	s.Equal("postgres-single-export-test", instances[0].InstanceName)
}

func (s *PostgresExportIntegrationSuite) Test_exports_nested_children_from_postgres() {
	childInstance := newTestInstance(testExportChildInstanceID, "Postgres Export Child", core.InstanceStatusDeployed)
	parentInstance := newTestInstance(testExportParentInstanceID, "Postgres Export Parent", core.InstanceStatusDeployed)
	parentInstance.ChildBlueprints = map[string]*state.InstanceState{
		"child": &childInstance,
	}
	s.seedInstance(parentInstance)
	defer s.cleanupInstance(testExportParentInstanceID)

	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	result, err := ExecuteInstancesExport(ctx, exporter, []string{testExportParentInstanceID})

	s.Require().NoError(err)
	s.Equal(1, result.InstancesCount)

	// Verify nested children are included
	var instances []state.InstanceState
	err = json.Unmarshal(result.Data, &instances)
	s.Require().NoError(err)
	s.Len(instances, 1)
	s.Require().NotNil(instances[0].ChildBlueprints)
	s.Contains(instances[0].ChildBlueprints, "child")
}

func (s *PostgresExportIntegrationSuite) Test_export_returns_error_for_nonexistent_instance() {
	exporter := NewContainerStateExporter(s.container)
	ctx := context.Background()

	// Use a valid UUID format that doesn't exist
	nonexistentID := "00000000-0000-0000-0000-000000000000"
	_, err := ExecuteInstancesExport(ctx, exporter, []string{nonexistentID})

	s.Require().Error(err)
	var exportErr *ExportError
	s.ErrorAs(err, &exportErr)
	s.Equal(ErrCodeInstanceNotFound, exportErr.Code)
}

func (s *PostgresExportIntegrationSuite) Test_exported_data_can_be_reimported_to_postgres() {
	// Seed test data
	s.seedInstance(newTestInstance(testExportInstanceID1, "Postgres Roundtrip Test", core.InstanceStatusDeployed))
	defer s.cleanupInstance(testExportInstanceID1)

	exporter := NewContainerStateExporter(s.container)

	// Export using the exporter
	ctx := context.Background()
	instances, err := exporter.ExportInstances(ctx, []string{testExportInstanceID1})
	s.Require().NoError(err)

	// Serialize to JSON
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	// Remove the original
	s.cleanupInstance(testExportInstanceID1)

	// Create temp directory for import file path
	tempDir, err := os.MkdirTemp("", "postgres-reimport-test-*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	// Re-import
	importResult, err := Import(ImportParams{
		FilePath:     path.Join(tempDir, "postgres-reimport.json"),
		EngineConfig: s.engineConfig,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(importResult.Success)
	s.Equal(1, importResult.InstancesCount)

	// Verify the data is back
	inst, err := s.container.Instances().Get(ctx, testExportInstanceID1)
	s.Require().NoError(err)
	s.Equal("Postgres Roundtrip Test", inst.InstanceName)
	s.Equal(core.InstanceStatusDeployed, inst.Status)
}
