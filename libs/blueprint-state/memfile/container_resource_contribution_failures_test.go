package memfile

import (
	"context"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type MemFileStateContainerContributionFailuresTestSuite struct {
	container state.Container
	stateDir  string
	fs        afero.Fs
	suite.Suite
}

func (s *MemFileStateContainerContributionFailuresTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	s.fs = memoryFS
	s.stateDir = stateDir

	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger(), WithMaxGuideFileSize(100))
	s.Require().NoError(err)
	s.container = container
}

func (s *MemFileStateContainerContributionFailuresTestSuite) Test_saves_a_contribution_failure_against_a_resource() {
	resources := s.container.Resources()

	err := resources.SaveContributionFailure(
		context.Background(),
		existingResourceID,
		contributionFailureFixture(0, "the role could not be given what its links contribute"),
	)
	s.Require().NoError(err)

	resourceState, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)
	s.Require().Len(resourceState.LinkContributionFailures, 1)
	s.Assert().Equal(
		[]string{"the role could not be given what its links contribute"},
		resourceState.LinkContributionFailures[0].Reasons,
	)

	s.assertPersistedContributionFailureLayers(existingResourceID, []int{0})
}

// The resource's own deployment is a separate outcome, and a reader cannot tell a resource
// that failed to deploy from one that deployed and was then left without its contributions
// if saving one disturbs the other.
func (s *MemFileStateContainerContributionFailuresTestSuite) Test_saving_a_failure_leaves_the_resources_own_status_alone() {
	resources := s.container.Resources()

	before, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)

	err = resources.SaveContributionFailure(
		context.Background(),
		existingResourceID,
		contributionFailureFixture(0, "the role could not be written"),
	)
	s.Require().NoError(err)

	after, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)
	s.Assert().Equal(before.Status, after.Status)
	s.Assert().Equal(before.PreciseStatus, after.PreciseStatus)
	s.Assert().Equal(before.FailureReasons, after.FailureReasons)
	s.Assert().Equal(before.LastDeployedTimestamp, after.LastDeployedTimestamp)
}

func (s *MemFileStateContainerContributionFailuresTestSuite) Test_replaces_the_failure_held_for_the_same_layer() {
	resources := s.container.Resources()
	ctx := context.Background()

	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, contributionFailureFixture(0, "first attempt"),
	))
	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, contributionFailureFixture(0, "second attempt"),
	))

	resourceState, err := resources.Get(ctx, existingResourceID)
	s.Require().NoError(err)
	s.Require().Len(
		resourceState.LinkContributionFailures,
		1,
		"a layer failing twice accumulated an entry per attempt",
	)
	s.Assert().Equal([]string{"second attempt"}, resourceState.LinkContributionFailures[0].Reasons)
}

func (s *MemFileStateContainerContributionFailuresTestSuite) Test_removes_only_the_layer_that_was_applied() {
	resources := s.container.Resources()
	ctx := context.Background()

	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, contributionFailureFixture(0, "the placement link"),
	))
	s.Require().NoError(resources.SaveContributionFailure(
		ctx, existingResourceID, contributionFailureFixture(2, "the access link"),
	))

	s.Require().NoError(resources.RemoveContributionFailure(ctx, existingResourceID, 0))

	resourceState, err := resources.Get(ctx, existingResourceID)
	s.Require().NoError(err)
	s.Require().Len(
		resourceState.LinkContributionFailures,
		1,
		"applying one layer cleared a failure belonging to another",
	)
	s.Assert().Equal(2, resourceState.LinkContributionFailures[0].LayerDepth)

	s.assertPersistedContributionFailureLayers(existingResourceID, []int{2})
}

// The common case is a layer that applies successfully having never failed, so clearing one
// that is not held must not be reported as a mistake.
func (s *MemFileStateContainerContributionFailuresTestSuite) Test_removing_a_layer_that_never_failed_is_not_an_error() {
	resources := s.container.Resources()

	err := resources.RemoveContributionFailure(context.Background(), existingResourceID, 0)
	s.Require().NoError(err)

	resourceState, err := resources.Get(context.Background(), existingResourceID)
	s.Require().NoError(err)
	s.Assert().Empty(resourceState.LinkContributionFailures)
}

func (s *MemFileStateContainerContributionFailuresTestSuite) Test_reports_resource_not_found_for_saving_a_failure() {
	resources := s.container.Resources()

	err := resources.SaveContributionFailure(
		context.Background(),
		nonExistentResourceID,
		contributionFailureFixture(0, "the role could not be written"),
	)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

func (s *MemFileStateContainerContributionFailuresTestSuite) Test_reports_resource_not_found_for_removing_a_failure() {
	resources := s.container.Resources()

	err := resources.RemoveContributionFailure(context.Background(), nonExistentResourceID, 0)
	s.Require().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrResourceNotFound, stateErr.Code)
}

// Read back from a container loaded afresh from persistence, since the resource is copied
// on the way out and a field left out of that copy is dropped without anything failing.
func (s *MemFileStateContainerContributionFailuresTestSuite) assertPersistedContributionFailureLayers(
	resourceID string,
	expectedLayerDepths []int,
) {
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)

	resourceState, err := container.Resources().Get(context.Background(), resourceID)
	s.Require().NoError(err)

	layerDepths := make([]int, 0, len(resourceState.LinkContributionFailures))
	for _, failure := range resourceState.LinkContributionFailures {
		layerDepths = append(layerDepths, failure.LayerDepth)
	}

	s.Assert().Equal(
		expectedLayerDepths,
		layerDepths,
		"the contribution failures did not survive being persisted and loaded back",
	)
}

func contributionFailureFixture(
	layerDepth int,
	reason string,
) state.ResourceLinkContributionFailure {
	return state.ResourceLinkContributionFailure{
		LayerDepth: layerDepth,
		Reasons:    []string{reason},
		UnappliedContributions: []state.UnappliedLinkContribution{
			{
				LinkName:  "saveOrderFunction::ordersRole",
				FieldPath: "spec.policies",
				Reason:    "the value to inject does not match the selector that would locate it",
			},
		},
		ContributingLinks: []string{"saveOrderFunction::ordersRole"},
		Timestamp:         1678901234,
	}
}

func TestMemFileStateContainerContributionFailuresTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerContributionFailuresTestSuite))
}
