package container

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DeploymentStateContributionsTestSuite struct {
	suite.Suite
}

func (s *DeploymentStateContributionsTestSuite) Test_a_target_waits_on_every_link_that_declared_it() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction", "ordersRole"},
		"saveOrderFunction::appVpc":      {"saveOrderFunction"},
	})

	s.Assert().Equal(
		[]string{"saveOrderFunction::appVpc", "saveOrderFunction::ordersTable"},
		deployState.AwaitingContributors("saveOrderFunction"),
		"both links declared a contribution to the function",
	)
	s.Assert().Equal(
		[]string{"saveOrderFunction::ordersTable"},
		deployState.AwaitingContributors("ordersRole"),
	)
	s.Assert().False(deployState.ContributionSetComplete("saveOrderFunction"))
}

func (s *DeploymentStateContributionsTestSuite) Test_a_target_is_complete_once_its_links_settle() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction"},
		"saveOrderFunction::appVpc":      {"saveOrderFunction"},
	})

	deployState.MarkLinkSettled("saveOrderFunction::ordersTable")
	s.Require().False(
		deployState.ContributionSetComplete("saveOrderFunction"),
		"one of the two links has yet to say what it contributes",
	)

	deployState.MarkLinkSettled("saveOrderFunction::appVpc")
	s.Assert().True(deployState.ContributionSetComplete("saveOrderFunction"))
	s.Assert().Empty(deployState.AwaitingContributors("saveOrderFunction"))
}

// Declaring a target and contributing nothing to it is allowed, so the join counts links
// that have settled rather than contributions received.
func (s *DeploymentStateContributionsTestSuite) Test_a_link_declaring_no_targets_holds_nothing_up() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {},
	})

	s.Assert().True(deployState.ContributionSetComplete("saveOrderFunction"))
	s.Assert().Empty(deployState.AwaitingContributors("saveOrderFunction"))
}

// A resource nothing contributes to is complete from the outset, which is every resource
// in a deployment whose links have not moved to contributions.
func (s *DeploymentStateContributionsTestSuite) Test_a_resource_with_no_contributors_is_complete() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction"},
	})

	s.Assert().True(deployState.ContributionSetComplete("ordersTable"))
}

func (s *DeploymentStateContributionsTestSuite) stateWithTargets(
	targets map[string][]string,
) DeploymentState {
	deployState := NewDefaultDeploymentState()
	deployState.SetLinkContributionTargets(targets)

	return deployState
}

func TestDeploymentStateContributionsTestSuite(t *testing.T) {
	suite.Run(t, new(DeploymentStateContributionsTestSuite))
}
