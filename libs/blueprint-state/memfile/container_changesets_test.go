package memfile

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

const (
	existingChangesetID    = "08dc456e-cafc-4199-b074-5f04cd4904f2"
	nonExistentChangesetID = "e28e895d-fc59-48e4-b385-78d7075b7acf"
)

type MemFileStateContainerChangesetsSuite struct {
	container             *StateContainer
	stateDir              string
	fs                    afero.Fs
	saveChangesetFixtures map[int]internal.SaveChangesetFixture
	suite.Suite
}

func (s *MemFileStateContainerChangesetsSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// Use a low max guide file size of 100 bytes to trigger the logic that splits
	// change set state across multiple chunk files.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "changesets")
	fixtures, err := internal.SetupSaveChangesetFixtures(
		dirPath,
	)
	s.Require().NoError(err)
	s.saveChangesetFixtures = fixtures
}

func (s *MemFileStateContainerChangesetsSuite) Test_retrieves_changeset() {
	changesets := s.container.Changesets()
	changeset, err := changesets.Get(
		context.Background(),
		existingChangesetID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(changeset)
	err = testhelpers.Snapshot(changeset)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerChangesetsSuite) Test_fails_to_retrieve_non_existent_changeset() {
	changesets := s.container.Changesets()

	_, err := changesets.Get(
		context.Background(),
		nonExistentChangesetID,
	)
	s.Require().Error(err)
	changesetNotFoundErr, isChangesetNotFoundErr := err.(*manage.ChangesetNotFound)
	s.Assert().True(isChangesetNotFoundErr)
	s.Assert().EqualError(
		changesetNotFoundErr,
		fmt.Sprintf("change set with ID %s not found", nonExistentChangesetID),
	)
}

func (s *MemFileStateContainerChangesetsSuite) Test_saves_new_changeset() {
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

	s.assertPersistedChangeset(fixture.Changeset)
}

func (s *MemFileStateContainerChangesetsSuite) Test_updates_existing_changeset() {
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

	s.assertPersistedChangeset(fixture.Changeset)
}

func (s *MemFileStateContainerChangesetsSuite) Test_cleans_up_old_changesets() {
	_, err := s.container.Changesets().Cleanup(
		context.Background(),
		time.Unix(cleanupThresholdTimestamp, 0),
	)
	s.Require().NoError(err)

	assertChangesetsCleanedUp(
		s.container,
		&s.Suite,
	)

	// Assert that the change sets are cleaned up when loading a fresh
	// state container from file, ensuring that the cleanup
	// operation was persisted correctly.
	s.assertChangesetCleanupPersisted()
}

func (s *MemFileStateContainerChangesetsSuite) assertPersistedChangeset(
	expected *manage.Changeset,
) {
	// Check that the change set state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the change set.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	changesets := container.Changesets()
	persistedChangeset, err := changesets.Get(
		context.Background(),
		expected.ID,
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		expected,
		persistedChangeset,
	)
}

func (s *MemFileStateContainerChangesetsSuite) assertChangesetCleanupPersisted() {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	assertChangesetsCleanedUp(
		container,
		&s.Suite,
	)
}

func assertChangesetsCleanedUp(
	container *StateContainer,
	s *suite.Suite,
) {
	for _, id := range changesetsShouldBeCleanedUp {
		_, err := container.Changesets().Get(
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
		changeset, err := container.Changesets().Get(
			context.Background(),
			id,
		)
		s.Require().NoError(err)
		s.Assert().Equal(id, changeset.ID)
	}
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

func TestMemFileStateContainerChangesetsSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerChangesetsSuite))
}
