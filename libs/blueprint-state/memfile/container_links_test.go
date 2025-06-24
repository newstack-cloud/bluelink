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

const (
	existingLinkID    = "test-link-1"
	existingLinkName  = "saveOrderFunction::ordersTable_0"
	nonExistentLinkID = "non-existent-link"
)

type MemFileStateContainerLinksTestSuite struct {
	container        state.Container
	saveLinkFixtures map[int]internal.SaveLinkFixture
	stateDir         string
	fs               afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerLinksTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir
	// Use a low max guide file size of 100 bytes to trigger the logic that splits
	// instance and resource drift state across multiple chunk files.
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
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

func (s *MemFileStateContainerLinksTestSuite) Test_retrieves_link() {
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

func (s *MemFileStateContainerLinksTestSuite) Test_reports_link_not_found_for_retrieval() {
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

func (s *MemFileStateContainerLinksTestSuite) Test_retrieves_link_by_logical_name() {
	links := s.container.Links()
	linkState, err := links.GetByName(
		context.Background(),
		existingBlueprintInstanceID,
		existingLinkName,
	)
	s.Require().NoError(err)
	s.Require().NotNil(linkState)
	err = testhelpers.Snapshot(linkState)
	s.Require().NoError(err)
}

func (s *MemFileStateContainerLinksTestSuite) Test_reports_link_not_found_for_retrieval_by_logical_name() {
	links := s.container.Links()

	_, err := links.GetByName(
		context.Background(),
		existingBlueprintInstanceID,
		nonExistentLinkID,
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *MemFileStateContainerLinksTestSuite) Test_saves_new_link() {
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
	s.assertPersistedLink(fixture.LinkState)
}

func (s *MemFileStateContainerLinksTestSuite) Test_reports_instance_not_found_for_saving_link() {
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

func (s *MemFileStateContainerLinksTestSuite) Test_updates_existing_link() {
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
	s.assertPersistedLink(fixture.LinkState)
}

func (s *MemFileStateContainerLinksTestSuite) Test_updates_blueprint_link_deployment_status() {
	links := s.container.Links()

	statusInfo := internal.CreateTestLinkStatusInfo()
	err := links.UpdateStatus(
		context.Background(),
		existingLinkID,
		statusInfo,
	)
	s.Require().NoError(err)

	savedState, err := links.Get(
		context.Background(),
		existingLinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkStatusInfo(statusInfo, savedState, &s.Suite)
	s.assertPersistedLink(&savedState)
}

func (s *MemFileStateContainerLinksTestSuite) Test_reports_link_not_found_for_status_update() {
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

func (s *MemFileStateContainerLinksTestSuite) Test_reports_malformed_state_error_for_status_update() {
	// The malformed state for this test case contains a link
	// that references an instance that does not exist.
	container, err := loadMalformedStateContainer(&s.Suite)
	s.Require().NoError(err)

	links := container.Links()
	statusInfo := internal.CreateTestLinkStatusInfo()
	err = links.UpdateStatus(
		context.Background(),
		existingLinkID,
		statusInfo,
	)
	s.Require().Error(err)
	memFileErr, isMemFileErr := err.(*Error)
	s.Assert().True(isMemFileErr)
	s.Assert().Equal(ErrorReasonCodeMalformedState, memFileErr.ReasonCode)
}

func (s *MemFileStateContainerLinksTestSuite) Test_removes_link() {
	links := s.container.Links()
	_, err := links.Remove(context.Background(), existingLinkID)
	s.Require().NoError(err)

	_, err = links.Get(context.Background(), existingLinkID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)

	s.assertLinkRemovedFromPersistence(existingResourceID)
}

func (s *MemFileStateContainerLinksTestSuite) Test_reports_link_not_found_for_removal() {
	links := s.container.Links()
	_, err := links.Remove(context.Background(), nonExistentLinkID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func (s *MemFileStateContainerLinksTestSuite) Test_reports_malformed_state_error_for_removal() {
	// The malformed state for this test case contains a link
	// that references an instance that does not exist.
	container, err := loadMalformedStateContainer(&s.Suite)
	s.Require().NoError(err)

	links := container.Links()
	_, err = links.Remove(
		context.Background(),
		existingLinkID,
	)
	s.Require().Error(err)
	memFileErr, isMemFileErr := err.(*Error)
	s.Assert().True(isMemFileErr)
	s.Assert().Equal(ErrorReasonCodeMalformedState, memFileErr.ReasonCode)
}

func (s *MemFileStateContainerLinksTestSuite) assertPersistedLink(expected *state.LinkState) {
	// Check that the link state was saved to "disk" correctly by
	// loading a new state container from persistence and retrieving the link.
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	links := container.Links()
	savedLinkState, err := links.Get(
		context.Background(),
		expected.LinkID,
	)
	s.Require().NoError(err)
	internal.AssertLinkStatesEqual(expected, &savedLinkState, &s.Suite)
}

func (s *MemFileStateContainerLinksTestSuite) assertLinkRemovedFromPersistence(linkID string) {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	links := container.Links()
	_, err = links.Get(context.Background(), linkID)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrLinkNotFound, stateErr.Code)
}

func TestMemFileStateContainerLinksTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerLinksTestSuite))
}
