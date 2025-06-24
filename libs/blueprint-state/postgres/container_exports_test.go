package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

const (
	// See __testdata/seed/blueprint-instances.json for instance IDs.
	saveAllExportsForInstanceID     = "47b34a02-80a4-4e70-b55c-00b43564ec0d"
	saveSingleExportForInstanceID   = "8d614d5d-82ac-42e5-b211-a796624c8217"
	removeExportsForInstanceID      = "476438b9-3ba0-4681-a778-989cfb4408e9"
	removeSingleExportForInstanceID = "6d313fc9-dacb-422f-9b48-f713891b91e3"
	envVarsField                    = "variables.environment"
)

type PostgresStateContainerExportsTestSuite struct {
	container state.Container
	connPool  *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerExportsTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *PostgresStateContainerExportsTestSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresStateContainerExportsTestSuite) Test_retrieves_all_exports_for_blueprint_instance() {
	exportsContainer := s.container.Exports()

	exports, err := exportsContainer.GetAll(
		context.Background(),
		getTestRootInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(exports)
	err = testhelpers.Snapshot(exports)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerExportsTestSuite) Test_reports_instance_not_found_for_retrieving_all_exports() {
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

func (s *PostgresStateContainerExportsTestSuite) Test_retrieves_single_export_for_blueprint_instance() {
	exportsContainer := s.container.Exports()

	export, err := exportsContainer.Get(
		context.Background(),
		getTestRootInstanceID,
		"environment",
	)
	s.Require().NoError(err)
	s.Require().NotNil(export)
	err = testhelpers.Snapshot(export)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerExportsTestSuite) Test_reports_instance_not_found_for_retrieving_single_export() {
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

func (s *PostgresStateContainerExportsTestSuite) Test_reports_export_not_found_for_retrieving_single_export() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.Get(
		context.Background(),
		getTestRootInstanceID,
		"region_504932",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)
}

func (s *PostgresStateContainerExportsTestSuite) Test_saves_multiple_exports_for_blueprint_instance() {
	// SaveAll overrides any existing exports for the instance.
	exportsContainer := s.container.Exports()

	exports := internal.SaveAllExportsInput()

	err := exportsContainer.SaveAll(
		context.Background(),
		saveAllExportsForInstanceID,
		exports,
	)
	s.Require().NoError(err)

	savedExports, err := exportsContainer.GetAll(
		context.Background(),
		saveAllExportsForInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(savedExports)
	s.Assert().Equal(exports, savedExports)
}

func (s *PostgresStateContainerExportsTestSuite) Test_reports_instance_not_found_for_saving_multiple_exports() {
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

func (s *PostgresStateContainerExportsTestSuite) Test_saves_single_export_for_blueprint_instance() {
	exportsContainer := s.container.Exports()

	export := internal.SaveSingleExportInput()

	err := exportsContainer.Save(
		context.Background(),
		saveSingleExportForInstanceID,
		"exampleId",
		export,
	)
	s.Require().NoError(err)

	savedExport, err := exportsContainer.Get(
		context.Background(),
		saveSingleExportForInstanceID,
		"exampleId",
	)
	s.Require().NoError(err)
	s.Assert().Equal(export, savedExport)
}

func (s *PostgresStateContainerExportsTestSuite) Test_reports_instance_not_found_for_saving_single_export() {
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

func (s *PostgresStateContainerExportsTestSuite) Test_removes_all_exports_from_blueprint_instance() {
	exportsContainer := s.container.Exports()

	removed, err := exportsContainer.RemoveAll(
		context.Background(),
		removeExportsForInstanceID,
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
		removeExportsForInstanceID,
	)
	s.Require().NoError(err)
	s.Assert().Len(savedExports, 0)
}

func (s *PostgresStateContainerExportsTestSuite) Test_reports_instance_not_found_for_removing_all_exports() {
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

func (s *PostgresStateContainerExportsTestSuite) Test_removes_single_export_from_blueprint_instance() {
	exportsContainer := s.container.Exports()

	removed, err := exportsContainer.Remove(
		context.Background(),
		removeSingleExportForInstanceID,
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
		removeSingleExportForInstanceID,
		"environment",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)
}

func (s *PostgresStateContainerExportsTestSuite) Test_reports_instance_not_found_for_removing_single_export() {
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

func (s *PostgresStateContainerExportsTestSuite) Test_reports_export_not_found_for_removing_single_export() {
	exportsContainer := s.container.Exports()

	_, err := exportsContainer.Remove(
		context.Background(),
		removeSingleExportForInstanceID,
		"environmentMissing",
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrExportNotFound, stateErr.Code)
}

func TestPostgresStateContainerExportsTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerExportsTestSuite))
}
