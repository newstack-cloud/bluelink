package stateio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// Test UUIDs for postgres integration tests
const (
	testInstanceID1       = "a1b2c3d4-e5f6-7890-abcd-ef1234567801"
	testInstanceID2       = "a1b2c3d4-e5f6-7890-abcd-ef1234567802"
	testParentInstanceID  = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	testChildInstanceID   = "b2c3d4e5-f6a7-8901-bcde-f12345678902"
	testLookupInstanceID  = "c3d4e5f6-a7b8-9012-cdef-123456789001"
	testOverwriteInstID   = "d4e5f6a7-b8c9-0123-def0-123456789001"
)

type PostgresImportIntegrationSuite struct {
	suite.Suite
	connPool     *pgxpool.Pool
	engineConfig *EngineConfig
}

func TestPostgresImportIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PostgresImportIntegrationSuite))
}

func (s *PostgresImportIntegrationSuite) SetupTest() {
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
}

func (s *PostgresImportIntegrationSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresImportIntegrationSuite) Test_imports_instances_to_postgres() {
	instances := []state.InstanceState{
		newTestInstance(testInstanceID1, "Postgres Test Instance 1", core.InstanceStatusDeployed),
		newTestInstance(testInstanceID2, "Postgres Test Instance 2", core.InstanceStatusDeployed),
	}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	result, err := Import(ImportParams{
		FilePath:     "/test/postgres-state.json",
		EngineConfig: s.engineConfig,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(2, result.InstancesCount)

	// Verify instances are persisted in postgres
	ctx := context.Background()
	container, err := postgres.LoadStateContainer(ctx, s.connPool, core.NewNopLogger())
	s.Require().NoError(err)

	inst1, err := container.Instances().Get(ctx, testInstanceID1)
	s.Require().NoError(err)
	s.Equal(testInstanceID1, inst1.InstanceID)
	s.Equal("Postgres Test Instance 1", inst1.InstanceName)
	s.Equal(core.InstanceStatusDeployed, inst1.Status)

	inst2, err := container.Instances().Get(ctx, testInstanceID2)
	s.Require().NoError(err)
	s.Equal(testInstanceID2, inst2.InstanceID)
	s.Equal("Postgres Test Instance 2", inst2.InstanceName)

	// Cleanup test instances
	_, _ = container.Instances().Remove(ctx, testInstanceID1)
	_, _ = container.Instances().Remove(ctx, testInstanceID2)
}

func (s *PostgresImportIntegrationSuite) Test_imports_instances_with_nested_children_to_postgres() {
	childInstance := newTestInstance(testChildInstanceID, "Postgres Child Instance", core.InstanceStatusDeployed)
	parentInstance := newTestInstance(testParentInstanceID, "Postgres Parent Instance", core.InstanceStatusDeployed)
	parentInstance.ChildBlueprints = map[string]*state.InstanceState{
		"child": &childInstance,
	}

	instances := []state.InstanceState{parentInstance}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	result, err := Import(ImportParams{
		FilePath:     "/test/postgres-nested.json",
		EngineConfig: s.engineConfig,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})

	s.Require().NoError(err)
	s.True(result.Success)
	s.Equal(1, result.InstancesCount)

	// Verify parent instance with nested child is persisted
	ctx := context.Background()
	container, err := postgres.LoadStateContainer(ctx, s.connPool, core.NewNopLogger())
	s.Require().NoError(err)

	parent, err := container.Instances().Get(ctx, testParentInstanceID)
	s.Require().NoError(err)
	s.Equal(testParentInstanceID, parent.InstanceID)
	s.Equal("Postgres Parent Instance", parent.InstanceName)
	s.Require().NotNil(parent.ChildBlueprints)
	s.Require().Contains(parent.ChildBlueprints, "child")
	s.Equal(testChildInstanceID, parent.ChildBlueprints["child"].InstanceID)
	s.Equal("Postgres Child Instance", parent.ChildBlueprints["child"].InstanceName)

	// Cleanup
	_, _ = container.Instances().Remove(ctx, testParentInstanceID)
}

func (s *PostgresImportIntegrationSuite) Test_imports_can_lookup_instance_by_name_in_postgres() {
	instances := []state.InstanceState{
		newTestInstance(testLookupInstanceID, "postgres-unique-lookup-test", core.InstanceStatusDeployed),
	}
	jsonData, err := json.Marshal(instances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     "/test/postgres-lookup.json",
		EngineConfig: s.engineConfig,
		FileData:     jsonData,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)

	// Verify we can look up the instance by name
	ctx := context.Background()
	container, err := postgres.LoadStateContainer(ctx, s.connPool, core.NewNopLogger())
	s.Require().NoError(err)

	instanceID, err := container.Instances().LookupIDByName(ctx, "postgres-unique-lookup-test")
	s.Require().NoError(err)
	s.Equal(testLookupInstanceID, instanceID)

	// Cleanup
	_, _ = container.Instances().Remove(ctx, testLookupInstanceID)
}

func (s *PostgresImportIntegrationSuite) Test_import_overwrites_existing_instance_in_postgres() {
	// First import
	originalInstances := []state.InstanceState{
		newTestInstance(testOverwriteInstID, "Original Postgres Name", core.InstanceStatusDeploying),
	}
	originalJSON, err := json.Marshal(originalInstances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		FilePath:     "/test/postgres-original.json",
		EngineConfig: s.engineConfig,
		FileData:     originalJSON,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)

	// Second import with updated data
	updatedInstances := []state.InstanceState{
		newTestInstance(testOverwriteInstID, "Updated Postgres Name", core.InstanceStatusDeployed),
	}
	updatedJSON, err := json.Marshal(updatedInstances)
	s.Require().NoError(err)

	result, err := Import(ImportParams{
		FilePath:     "/test/postgres-updated.json",
		EngineConfig: s.engineConfig,
		FileData:     updatedJSON,
		Logger:       core.NewNopLogger(),
	})
	s.Require().NoError(err)
	s.True(result.Success)

	// Verify instance was updated
	ctx := context.Background()
	container, err := postgres.LoadStateContainer(ctx, s.connPool, core.NewNopLogger())
	s.Require().NoError(err)

	inst, err := container.Instances().Get(ctx, testOverwriteInstID)
	s.Require().NoError(err)
	s.Equal("Updated Postgres Name", inst.InstanceName)
	s.Equal(core.InstanceStatusDeployed, inst.Status)

	// Cleanup
	_, _ = container.Instances().Remove(ctx, testOverwriteInstID)
}

func newTestInstance(id, name string, status core.InstanceStatus) state.InstanceState {
	return state.InstanceState{
		InstanceID:   id,
		InstanceName: name,
		Status:       status,
		Metadata:     map[string]*core.MappingNode{},
		Exports:      map[string]*state.ExportState{},
	}
}

func buildTestPostgresDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_max_conns=%s&pool_max_conn_lifetime=%s",
		os.Getenv("STATEIO_POSTGRES_USER"),
		os.Getenv("STATEIO_POSTGRES_PASSWORD"),
		os.Getenv("STATEIO_POSTGRES_HOST"),
		os.Getenv("STATEIO_POSTGRES_PORT"),
		os.Getenv("STATEIO_POSTGRES_DATABASE"),
		os.Getenv("STATEIO_POSTGRES_POOL_MAX_CONNS"),
		os.Getenv("STATEIO_POSTGRES_POOL_MAX_CONN_LIFETIME"),
	)
}
