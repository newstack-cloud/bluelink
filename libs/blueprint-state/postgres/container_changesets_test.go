package postgres

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

const (
	existingChangesetID    = "2888c908-32e2-4555-af36-319455172c64"
	nonExistentChangesetID = "e28e895d-fc59-48e4-b385-78d7075b7acf"
)

type PostgresChangesetsTestSuite struct {
	container             *StateContainer
	connPool              *pgxpool.Pool
	saveChangesetFixtures map[int]internal.SaveChangesetFixture
	suite.Suite
}

func (s *PostgresChangesetsTestSuite) SetupTest() {
	ctx := context.Background()
	connPool, err := pgxpool.New(ctx, buildTestDatabaseURL())
	s.connPool = connPool
	s.Require().NoError(err)
	container, err := LoadStateContainer(ctx, connPool, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "changesets")
	saveFixtures, err := internal.SetupSaveChangesetFixtures(
		dirPath,
	)
	s.Require().NoError(err)
	s.saveChangesetFixtures = saveFixtures
}

func (s *PostgresChangesetsTestSuite) TearDownTest() {
	s.connPool.Close()
}

func (s *PostgresChangesetsTestSuite) Test_retrieve_existing_changeset() {
	ctx := context.Background()
	changeset, err := s.container.Changesets().Get(ctx, existingChangesetID)
	s.Require().NoError(err)

	s.Require().Equal(expectedExistingChangeSet, changeset)
}

func (s *PostgresChangesetsTestSuite) Test_fails_to_retrieve_non_existent_changeset() {
	ctx := context.Background()
	_, err := s.container.Changesets().Get(ctx, nonExistentChangesetID)
	s.Require().Error(err)
	notFoundErr, isNotFoundErr := err.(*manage.ChangesetNotFound)
	s.Require().True(isNotFoundErr)
	s.Require().Equal(nonExistentChangesetID, notFoundErr.ID)
	s.Require().Equal(
		fmt.Sprintf("change set with ID %s not found", nonExistentChangesetID),
		notFoundErr.Error(),
	)
}

func (s *PostgresChangesetsTestSuite) Test_saves_changeset() {
	fixture := s.saveChangesetFixtures[1]

	changesets := s.container.Changesets()
	err := changesets.Save(
		context.Background(),
		fixture.Changeset,
	)
	s.Require().NoError(err)

	savedChangeset, err := changesets.Get(
		context.Background(),
		fixture.Changeset.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedChangeset)
	s.Assert().Equal(fixture.Changeset, savedChangeset)
}

func (s *PostgresChangesetsTestSuite) Test_updates_existing_changeset() {
	// Fixture 2 represents a change set that already exists
	// but with changes to the "status" and "changes" fields.
	fixture := s.saveChangesetFixtures[2]

	changesets := s.container.Changesets()
	err := changesets.Save(
		context.Background(),
		fixture.Changeset,
	)
	s.Require().NoError(err)

	savedChangeset, err := changesets.Get(
		context.Background(),
		fixture.Changeset.ID,
	)
	s.Require().NoError(err)
	s.Assert().NotNil(savedChangeset)
	s.Assert().Equal(fixture.Changeset, savedChangeset)
}

func (s *PostgresChangesetsTestSuite) Test_cleans_up_old_changesets() {
	_, err := s.container.Changesets().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	for _, id := range changesetsShouldBeCleanedUp {
		_, err := s.container.Changesets().Get(
			context.Background(),
			id,
		)
		s.Require().Error(err)

		notFoundErr, isNotFoundErr := err.(*manage.ChangesetNotFound)
		s.Require().True(isNotFoundErr)
		s.Assert().Equal(
			fmt.Sprintf("change set with ID %s not found", id),
			notFoundErr.Error(),
		)
	}

	for _, id := range changesetsShouldNotBeCleanedUp {
		changeset, err := s.container.Changesets().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, changeset.ID)
	}
}

var expectedExistingChangeSet = &manage.Changeset{
	ID: existingChangesetID,
	// This has a foreign key constraint, so it must exist in the database.
	// See __testdata/seed/blueprint-instances.json
	InstanceID:        "46324ee7-b515-4988-98b0-d5445746a997",
	Destroy:           true,
	Status:            manage.ChangesetStatusChangesStaged,
	BlueprintLocation: "s3://bluelink-test/project1/project.blueprint.yml",
	Changes: &changes.BlueprintChanges{
		RemovedResources: []string{
			"resource-1",
			"resource-2",
			"resource-3",
		},
		RemovedLinks: []string{
			"resource-1::resource-3",
		},
		RemovedExports: []string{
			"bucket_id",
		},
	},
	Created: 1745496983,
}

// Seed change sets that should be cleaned up.
var changesetsShouldBeCleanedUp = []string{
	"08dc456e-cafc-4199-b074-5f04cd4904f2",
	"3d234a23-abd8-4633-8f43-654f8788413b",
	"ff50a0f8-96e9-41c1-b729-4f3a6a82d8d8",
	"341c10bd-7c1e-4bfe-bdd2-10c5c16a871d",
}

// Seed change sets that should not be cleaned up.
// This must not include the IDs of any dynamically generated change sets
// in the test runs.
var changesetsShouldNotBeCleanedUp = []string{
	"2888c908-32e2-4555-af36-319455172c64",
}

func TestPostgresChangesetsTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresChangesetsTestSuite))
}
