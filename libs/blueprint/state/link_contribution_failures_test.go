package state

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type LinkContributionFailuresTestSuite struct {
	suite.Suite
}

func (s *LinkContributionFailuresTestSuite) Test_holds_the_first_failure_for_a_layer() {
	failures := UpsertContributionFailure(nil, failureForLayer(0, "the role could not be written"))

	s.Require().Len(failures, 1)
	s.Assert().Equal(0, failures[0].LayerDepth)
	s.Assert().Equal([]string{"the role could not be written"}, failures[0].Reasons)
}

// A layer is identified by its depth alone, so a layer that fails, is retried and fails
// again says so once rather than leaving a reader to work out which entry is current.
func (s *LinkContributionFailuresTestSuite) Test_replaces_the_failure_already_held_for_a_layer() {
	failures := UpsertContributionFailure(nil, failureForLayer(0, "first attempt"))
	failures = UpsertContributionFailure(failures, failureForLayer(0, "second attempt"))

	s.Require().Len(
		failures,
		1,
		"a layer failing twice accumulated an entry per attempt",
	)
	s.Assert().Equal([]string{"second attempt"}, failures[0].Reasons)
}

// A resource whose contributors are ordered has a layer per depth, and the failure of one
// says nothing about the others.
func (s *LinkContributionFailuresTestSuite) Test_holds_a_failure_for_each_layer_that_failed() {
	failures := UpsertContributionFailure(nil, failureForLayer(2, "the access link"))
	failures = UpsertContributionFailure(failures, failureForLayer(0, "the placement link"))

	s.Require().Len(failures, 2)
	s.Assert().Equal(
		[]int{0, 2},
		layerDepthsOf(failures),
		"failures are not read back in the order the layers would have been applied in",
	)
}

func (s *LinkContributionFailuresTestSuite) Test_removes_only_the_layer_that_was_applied() {
	failures := UpsertContributionFailure(nil, failureForLayer(0, "the placement link"))
	failures = UpsertContributionFailure(failures, failureForLayer(2, "the access link"))

	failures = RemoveContributionFailureForLayer(failures, 0)

	s.Require().Len(
		failures,
		1,
		"applying one layer cleared a failure belonging to another",
	)
	s.Assert().Equal(2, failures[0].LayerDepth)
}

// Nil rather than an empty slice, so that a resource with nothing outstanding is
// indistinguishable from one that never failed, in state and in what a client is given.
func (s *LinkContributionFailuresTestSuite) Test_holds_nothing_once_the_last_failure_is_removed() {
	failures := UpsertContributionFailure(nil, failureForLayer(0, "the placement link"))

	s.Assert().Nil(RemoveContributionFailureForLayer(failures, 0))
}

// The common case is a layer that applies successfully having never failed, so clearing
// one that is not held is not a mistake to report.
func (s *LinkContributionFailuresTestSuite) Test_removing_a_layer_that_never_failed_changes_nothing() {
	failures := UpsertContributionFailure(nil, failureForLayer(0, "the placement link"))

	remaining := RemoveContributionFailureForLayer(failures, 3)

	s.Require().Len(remaining, 1)
	s.Assert().Equal(0, remaining[0].LayerDepth)
}

func (s *LinkContributionFailuresTestSuite) Test_removing_from_nothing_does_not_cause_an_error() {
	s.Assert().Nil(RemoveContributionFailureForLayer(nil, 0))
}

func failureForLayer(layerDepth int, reason string) ResourceLinkContributionFailure {
	return ResourceLinkContributionFailure{
		LayerDepth: layerDepth,
		Reasons:    []string{reason},
		Timestamp:  1678901234,
	}
}

func layerDepthsOf(failures []ResourceLinkContributionFailure) []int {
	depths := make([]int, 0, len(failures))
	for _, failure := range failures {
		depths = append(depths, failure.LayerDepth)
	}

	return depths
}

func TestLinkContributionFailuresTestSuite(t *testing.T) {
	suite.Run(t, new(LinkContributionFailuresTestSuite))
}
