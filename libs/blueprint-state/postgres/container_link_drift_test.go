package postgres

import (
	"context"
	"path"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

const (
	existingDriftLinkID   = "c0d6d914-21a6-4a99-afb3-f6f45eefbdd3"
	removeDriftLinkID     = "e4a4a3be-f494-4dc8-87e0-3bd0ac33707b"
	existingLinkIDNoDrift = "d97e379c-7e85-4c70-bd31-d7a6f9a5dbd6"
)

type PostgresStateContainerLinkDriftTestSuite struct {
	container             state.Container
	saveLinkDriftFixtures map[int]internal.SaveLinkDriftFixture
	connPool              *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerLinkDriftTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "link-drift")
	fixtures, err := internal.SetupSaveLinkDriftFixtures(
		dirPath,
		/* updates */ []int{2},
	)
	s.Require().NoError(err)
	s.saveLinkDriftFixtures = fixtures
}

func (s *PostgresStateContainerLinkDriftTestSuite) TearDownTest() {
	for _, fixture := range s.saveLinkDriftFixtures {
		if !fixture.Update {
			_, _ = s.container.Links().RemoveDrift(
				context.Background(),
				fixture.DriftState.LinkID,
			)
		}
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_retrieves_link_drift() {
	links := s.container.Links()
	linkDriftState, err := links.GetDrift(
		context.Background(),
		existingDriftLinkID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(linkDriftState)
	err = testhelpers.Snapshot(linkDriftState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_reports_link_not_found_for_drift_retrieval() {
	links := s.container.Links()

	_, err := links.GetDrift(
		context.Background(),
		nonExistentLinkID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_saves_new_link_drift() {
	fixture := s.saveLinkDriftFixtures[1]
	links := s.container.Links()
	err := links.SaveDrift(
		context.Background(),
		*fixture.DriftState,
	)
	s.Require().NoError(err)

	savedDriftState, err := links.GetDrift(
		context.Background(),
		fixture.DriftState.LinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkDriftEqual(fixture.DriftState, &savedDriftState, &s.Suite)

	updatedLink, err := links.Get(
		context.Background(),
		fixture.DriftState.LinkID,
	)
	s.Require().NoError(err)
	s.Assert().True(updatedLink.Drifted)
	s.Assert().Equal(fixture.DriftState.Timestamp, updatedLink.LastDriftDetectedTimestamp)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_updates_existing_link_drift() {
	fixture := s.saveLinkDriftFixtures[2]
	links := s.container.Links()
	err := links.SaveDrift(
		context.Background(),
		*fixture.DriftState,
	)
	s.Require().NoError(err)

	savedDriftState, err := links.GetDrift(
		context.Background(),
		fixture.DriftState.LinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkDriftEqual(fixture.DriftState, &savedDriftState, &s.Suite)

	updatedLink, err := links.Get(
		context.Background(),
		fixture.DriftState.LinkID,
	)
	s.Require().NoError(err)
	s.Assert().True(updatedLink.Drifted)
	s.Assert().Equal(fixture.DriftState.Timestamp, updatedLink.LastDriftDetectedTimestamp)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_reports_link_not_found_for_saving_drift() {
	// Fixture 3 is a drift state that references a non-existent link.
	fixture := s.saveLinkDriftFixtures[3]
	links := s.container.Links()

	err := links.SaveDrift(
		context.Background(),
		*fixture.DriftState,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_removes_link_drift() {
	links := s.container.Links()
	_, err := links.RemoveDrift(context.Background(), removeDriftLinkID)
	s.Require().NoError(err)

	drift, err := links.GetDrift(context.Background(), removeDriftLinkID)
	s.Require().NoError(err)
	// The link should still exist but the drift should be an empty value.
	s.Assert().True(internal.IsEmptyLinkDriftState(drift))

	link, err := links.Get(context.Background(), removeDriftLinkID)
	s.Require().NoError(err)
	s.Assert().False(link.Drifted)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_reports_link_not_found_for_removing_drift() {
	links := s.container.Links()

	_, err := links.RemoveDrift(
		context.Background(),
		nonExistentLinkID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinkDriftTestSuite) Test_does_nothing_for_missing_drift_entry_for_existing_link() {
	links := s.container.Links()

	drift, err := links.RemoveDrift(
		context.Background(),
		existingLinkIDNoDrift,
	)
	s.Require().NoError(err)
	s.Assert().True(internal.IsEmptyLinkDriftState(drift))
}

func TestPostgresStateContainerLinkDriftTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerLinkDriftTestSuite))
}
