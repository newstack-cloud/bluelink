package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

const (
	saveMetadataForInstanceID   = "7836c96c-2875-4338-9bed-d437c69558fe"
	removeMetadataForInstanceID = "a5fa44c0-d9d5-4c49-92ac-ba08d090898f"
)

type PostgresStateContainerMetadataTestSuite struct {
	container state.Container
	connPool  *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerMetadataTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *PostgresStateContainerMetadataTestSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresStateContainerMetadataTestSuite) Test_retrieves_metadata_for_blueprint_instance() {
	metadataContainer := s.container.Metadata()

	metadata, err := metadataContainer.Get(
		context.Background(),
		getTestRootInstanceID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(metadata)
	err = testhelpers.Snapshot(metadata)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerMetadataTestSuite) Test_reports_instance_not_found_when_retrieving_metadata() {
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

func (s *PostgresStateContainerMetadataTestSuite) Test_saves_metadata_for_blueprint_instance() {
	metadataContainer := s.container.Metadata()

	metadata := internal.SaveMetadataInput()

	err := metadataContainer.Save(
		context.Background(),
		saveMetadataForInstanceID,
		metadata,
	)
	s.Require().NoError(err)

	savedMetadata, err := metadataContainer.Get(
		context.Background(),
		saveMetadataForInstanceID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(metadata, savedMetadata)
}

func (s *PostgresStateContainerMetadataTestSuite) Test_reports_instance_not_found_when_saving_metadata() {
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

func (s *PostgresStateContainerMetadataTestSuite) Test_saving_no_metadata_leaves_the_instance_with_none_without_failing() {
	ctx := context.Background()
	instanceID := uuid.NewString()
	err := s.container.Instances().Save(ctx, state.InstanceState{
		InstanceID:   instanceID,
		InstanceName: "SaveNoMetadata" + instanceID,
		Status:       core.InstanceStatusDeployed,
		Metadata: map[string]*core.MappingNode{
			"build": core.MappingNodeFromString("tsc"),
		},
		Exports: map[string]*state.ExportState{},
	})
	s.Require().NoError(err)

	metadataContainer := s.container.Metadata()
	err = metadataContainer.Save(ctx, instanceID, nil)
	s.Require().NoError(err)

	saved, err := metadataContainer.Get(ctx, instanceID)
	s.Require().NoError(err)
	s.Assert().Empty(saved)
}

func (s *PostgresStateContainerMetadataTestSuite) Test_removes_metadata_from_blueprint_instance() {
	metadataContainer := s.container.Metadata()

	removed, err := metadataContainer.Remove(
		context.Background(),
		removeMetadataForInstanceID,
	)
	s.Require().NoError(err)

	expectedRemovedMetadata := map[string]*core.MappingNode{
		"build": core.MappingNodeFromString("tsc"),
	}
	s.Assert().Equal(expectedRemovedMetadata, removed)
}

func (s *PostgresStateContainerMetadataTestSuite) Test_reports_instance_not_found_when_removing_metadata() {
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

func TestPostgresStateContainerMetadataTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerMetadataTestSuite))
}
