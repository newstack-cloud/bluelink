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
	// See files in __testdata/seed/
	existingLinkID     = "c0d6d914-21a6-4a99-afb3-f6f45eefbdd3"
	existingLinkName   = "saveOrderFunction::ordersTable_0"
	updateStatusLinkID = "d97e379c-7e85-4c70-bd31-d7a6f9a5dbd6"
	nonExistentLinkID  = "423d094a-4fac-4869-b0af-1f373c0e6820"
)

type PostgresStateContainerLinksTestSuite struct {
	container        state.Container
	saveLinkFixtures map[int]internal.SaveLinkFixture
	connPool         *pgxpool.Pool
	suite.Suite
}

func (s *PostgresStateContainerLinksTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "links")
	fixtures, err := internal.SetupSaveLinkFixtures(
		dirPath,
		/* updates */ []int{3},
	)
	s.Require().NoError(err)
	s.saveLinkFixtures = fixtures
}

func (s *PostgresStateContainerLinksTestSuite) TearDownTest() {
	for _, fixture := range s.saveLinkFixtures {
		if !fixture.Update {
			_, _ = s.container.Links().Remove(
				context.Background(),
				fixture.LinkState.LinkID,
			)
		}
	}
	s.connPool.Close()
}

func (s *PostgresStateContainerLinksTestSuite) Test_retrieves_link() {
	links := s.container.Links()
	linkState, err := links.Get(
		context.Background(),
		existingLinkID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(linkState)
	err = testhelpers.Snapshot(linkState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerLinksTestSuite) Test_reports_link_not_found_for_retrieval() {
	links := s.container.Links()

	_, err := links.Get(
		context.Background(),
		nonExistentLinkID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinksTestSuite) Test_retrieves_link_by_logical_name() {
	links := s.container.Links()
	linkState, err := links.GetByName(
		context.Background(),
		getTestRootInstanceID,
		existingLinkName,
	)
	s.Require().NoError(err)
	s.Require().NotNil(linkState)
	err = testhelpers.Snapshot(linkState)
	s.Require().NoError(err)
}

func (s *PostgresStateContainerLinksTestSuite) Test_reports_link_not_found_for_retrieval_by_logical_name() {
	links := s.container.Links()

	_, err := links.GetByName(
		context.Background(),
		getTestRootInstanceID,
		nonExistentLinkID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinksTestSuite) Test_saves_new_link() {
	fixture := s.saveLinkFixtures[1]
	links := s.container.Links()
	err := links.Save(
		context.Background(),
		*fixture.LinkState,
	)
	s.Require().NoError(err)

	savedState, err := links.Get(
		context.Background(),
		fixture.LinkState.LinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkStatesEqual(fixture.LinkState, &savedState, &s.Suite)
}

func (s *PostgresStateContainerLinksTestSuite) Test_reports_instance_not_found_for_saving_link() {
	// Fixture 2 is a link state that references a non-existent instance.
	fixture := s.saveLinkFixtures[2]
	links := s.container.Links()

	err := links.Save(
		context.Background(),
		*fixture.LinkState,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinksTestSuite) Test_updates_existing_link() {
	fixture := s.saveLinkFixtures[3]
	links := s.container.Links()
	err := links.Save(
		context.Background(),
		*fixture.LinkState,
	)
	s.Require().NoError(err)

	savedState, err := links.Get(
		context.Background(),
		fixture.LinkState.LinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkStatesEqual(fixture.LinkState, &savedState, &s.Suite)
}

func (s *PostgresStateContainerLinksTestSuite) Test_updates_blueprint_link_deployment_status() {
	links := s.container.Links()

	statusInfo := internal.CreateTestLinkStatusInfo()
	err := links.UpdateStatus(
		context.Background(),
		updateStatusLinkID,
		statusInfo,
	)
	s.Require().NoError(err)

	savedState, err := links.Get(
		context.Background(),
		updateStatusLinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkStatusInfo(statusInfo, savedState, &s.Suite)
}

func (s *PostgresStateContainerLinksTestSuite) Test_reports_link_not_found_for_status_update() {
	links := s.container.Links()

	statusInfo := internal.CreateTestLinkStatusInfo()
	err := links.UpdateStatus(
		context.Background(),
		nonExistentLinkID,
		statusInfo,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinksTestSuite) Test_removes_link() {
	fixture := s.saveLinkFixtures[4]

	links := s.container.Links()
	// Save the link to be removed.
	err := links.Save(
		context.Background(),
		*fixture.LinkState,
	)
	s.Require().NoError(err)

	_, err = links.Remove(context.Background(), fixture.LinkState.LinkID)
	s.Require().NoError(err)

	_, err = links.Get(context.Background(), fixture.LinkState.LinkID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *PostgresStateContainerLinksTestSuite) Test_reports_link_not_found_for_removal() {
	links := s.container.Links()
	_, err := links.Remove(context.Background(), nonExistentLinkID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func TestPostgresStateContainerLinksTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresStateContainerLinksTestSuite))
}
