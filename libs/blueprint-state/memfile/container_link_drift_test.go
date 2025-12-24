package memfile

import (
	"context"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type MemFileStateContainerLinkDriftTestSuite struct {
	container             state.Container
	saveLinkDriftFixtures map[int]internal.SaveLinkDriftFixture
	stateDir              string
	fs                    afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerLinkDriftTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// Use a low max guide file size of 100 bytes to trigger the logic that splits
	// link drift state across multiple chunk files.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container

	dirPath := path.Join("__testdata", "save-input", "link-drift")
	saveLinkDriftFixtures, err := internal.SetupSaveLinkDriftFixtures(
		dirPath,
		/* updates */ []int{2},
	)
	s.Require().NoError(err)
	s.saveLinkDriftFixtures = saveLinkDriftFixtures
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_retrieves_link_drift() {
	links := s.container.Links()
	linkDriftState, err := links.GetDrift(
		context.Background(),
		existingLinkID,
	)
	s.Require().NoError(err)
	s.Require().NotNil(linkDriftState)
	err = testhelpers.Snapshot(linkDriftState)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_reports_link_not_found_for_drift_retrieval() {
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

func (s *MemFileStateContainerLinkDriftTestSuite) Test_saves_new_link_drift() {
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
	s.assertPersistedLinkDrift(fixture.DriftState)
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_updates_existing_link_drift() {
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
	s.assertPersistedLinkDrift(fixture.DriftState)
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_reports_link_not_found_for_saving_drift() {
	// Fixture 3 is a link drift state that references a non-existent link.
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

func (s *MemFileStateContainerLinkDriftTestSuite) Test_reports_malformed_state_error_for_saving_drift() {
	// The malformed state for this test case contains a link
	// that references an instance that does not exist.
	container, err := loadMalformedStateContainer(&s.Suite)
	s.Require().NoError(err)

	links := container.Links()
	err = links.SaveDrift(
		context.Background(),
		state.LinkDriftState{
			LinkID:   existingLinkID,
			LinkName: existingLinkName,
		},
	)
	s.Require().Error(err)
	memFileErr, isMemFileErr := err.(*Error)
	s.Assert().True(isMemFileErr)
	s.Assert().Equal(ErrorReasonCodeMalformedState, memFileErr.ReasonCode)
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_removes_link_drift() {
	links := s.container.Links()
	_, err := links.RemoveDrift(context.Background(), existingLinkID)
	s.Require().NoError(err)

	drift, err := links.GetDrift(context.Background(), existingLinkID)
	s.Require().NoError(err)
	// The link should still exist but the drift should be an empty value.
	s.Assert().True(internal.IsEmptyLinkDriftState(drift))

	link, err := links.Get(context.Background(), existingLinkID)
	s.Require().NoError(err)
	s.Assert().False(link.Drifted)

	s.assertLinkDriftRemovedFromPersistence(existingLinkID)
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_reports_link_not_found_for_removing_drift() {
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

func (s *MemFileStateContainerLinkDriftTestSuite) Test_does_nothing_for_missing_drift_entry_for_existing_link() {
	links := s.container.Links()

	// test-link-2 exists but has no drift entry
	drift, err := links.RemoveDrift(
		context.Background(),
		"test-link-2",
	)
	s.Require().NoError(err)
	s.Assert().True(internal.IsEmptyLinkDriftState(drift))
}

func (s *MemFileStateContainerLinkDriftTestSuite) Test_reports_malformed_state_error_for_removing_drift() {
	// The malformed state for this test case contains a link
	// that references an instance that does not exist.
	container, err := loadMalformedStateContainer(&s.Suite)
	s.Require().NoError(err)

	links := container.Links()
	_, err = links.RemoveDrift(
		context.Background(),
		existingLinkID,
	)
	s.Require().Error(err)
	memFileErr, isMemFileErr := err.(*Error)
	s.Assert().True(isMemFileErr)
	s.Assert().Equal(ErrorReasonCodeMalformedState, memFileErr.ReasonCode)
}

func (s *MemFileStateContainerLinkDriftTestSuite) assertPersistedLinkDrift(expected *state.LinkDriftState) {
	// Check that the link drift state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the link drift.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	links := container.Links()
	savedDrift, err := links.GetDrift(
		context.Background(),
		expected.LinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkDriftEqual(expected, &savedDrift, &s.Suite)

	savedLink, err := links.Get(
		context.Background(),
		expected.LinkID,
	)
	s.Require().NoError(err)
	s.Assert().True(savedLink.Drifted)
	s.Assert().Equal(expected.Timestamp, savedLink.LastDriftDetectedTimestamp)
}

func (s *MemFileStateContainerLinkDriftTestSuite) assertLinkDriftRemovedFromPersistence(linkID string) {
	// Check that the link drift state was removed from "disk" correctly by
	// loading a new state container from persistence and retrieving the link drift.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	links := container.Links()
	drift, err := links.GetDrift(context.Background(), linkID)
	s.Require().NoError(err)
	s.Assert().True(internal.IsEmptyLinkDriftState(drift))

	link, err := links.Get(context.Background(), linkID)
	s.Require().NoError(err)
	s.Assert().False(link.Drifted)
}

func TestMemFileStateContainerLinkDriftTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerLinkDriftTestSuite))
}
