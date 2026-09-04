package container

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DeploymentStateContributionsTestSuite struct {
	suite.Suite
}

func (s *DeploymentStateContributionsTestSuite) Test_a_layer_waits_on_every_link_that_declared_it() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction", "ordersRole"},
		"saveOrderFunction::appVpc":      {"saveOrderFunction"},
	})

	s.Assert().Equal(
		[]string{"saveOrderFunction::appVpc", "saveOrderFunction::ordersTable"},
		deployState.AwaitingContributors(ContributionLayer{ResourceName: "saveOrderFunction"}),
		"both links declared a contribution to the function",
	)
	s.Assert().Equal(
		[]string{"saveOrderFunction::ordersTable"},
		deployState.AwaitingContributors(ContributionLayer{ResourceName: "ordersRole"}),
	)
	s.Assert().False(deployState.ContributionSetComplete(ContributionLayer{ResourceName: "saveOrderFunction"}))
}

func (s *DeploymentStateContributionsTestSuite) Test_a_layer_is_complete_once_its_links_settle() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction"},
		"saveOrderFunction::appVpc":      {"saveOrderFunction"},
	})

	deployState.MarkLinkSettled("saveOrderFunction::ordersTable")
	s.Require().False(
		deployState.ContributionSetComplete(ContributionLayer{ResourceName: "saveOrderFunction"}),
		"one of the two links has yet to say what it contributes",
	)

	deployState.MarkLinkSettled("saveOrderFunction::appVpc")
	s.Assert().True(deployState.ContributionSetComplete(ContributionLayer{ResourceName: "saveOrderFunction"}))
	s.Assert().Empty(deployState.AwaitingContributors(ContributionLayer{ResourceName: "saveOrderFunction"}))
}

// Declaring a target and contributing nothing to it is allowed, so the join counts links
// that have settled rather than contributions received.
func (s *DeploymentStateContributionsTestSuite) Test_a_link_declaring_no_targets_holds_nothing_up() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {},
	})

	s.Assert().True(deployState.ContributionSetComplete(ContributionLayer{ResourceName: "saveOrderFunction"}))
	s.Assert().Empty(deployState.AwaitingContributors(ContributionLayer{ResourceName: "saveOrderFunction"}))
}

// A resource nothing contributes to is complete from the outset, which is every resource
// in a deployment whose links have not moved to contributions.
func (s *DeploymentStateContributionsTestSuite) Test_a_resource_with_no_contributors_is_complete() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction"},
	})

	s.Assert().True(deployState.ContributionSetComplete(ContributionLayer{ResourceName: "ordersTable"}))
}

func (s *DeploymentStateContributionsTestSuite) Test_claims_a_layer_once_its_links_have_all_settled() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction"},
		"saveOrderFunction::appVpc":      {"saveOrderFunction"},
	})

	deployState.MarkLinkSettled("saveOrderFunction::ordersTable")
	s.Assert().Empty(
		deployState.ClaimContributionLayersReadyToUpdate(),
		"a layer one of whose links has yet to settle is not ready",
	)

	deployState.MarkLinkSettled("saveOrderFunction::appVpc")
	s.Assert().Equal(
		[]ContributionLayer{{ResourceName: "saveOrderFunction"}},
		deployState.ClaimContributionLayersReadyToUpdate(),
	)
}

// Links settle from several goroutines, and the last contributors to a resource can settle
// together, so more than one of them can find the resource complete. A resource updated
// once per link that noticed would be written concurrently with itself.
func (s *DeploymentStateContributionsTestSuite) Test_claims_a_layer_only_once() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {"saveOrderFunction"},
	})
	deployState.MarkLinkSettled("saveOrderFunction::ordersTable")

	s.Require().Equal(
		[]ContributionLayer{{ResourceName: "saveOrderFunction"}},
		deployState.ClaimContributionLayersReadyToUpdate(),
	)
	s.Assert().Empty(
		deployState.ClaimContributionLayersReadyToUpdate(),
		"the second caller finds the layer already claimed",
	)
}

// A resource nothing contributes to has no merged update to issue, so it must never be
// claimed as one, however many links the deployment runs.
func (s *DeploymentStateContributionsTestSuite) Test_never_claims_a_resource_with_no_contributors() {
	deployState := s.stateWithTargets(map[string][]string{
		"saveOrderFunction::ordersTable": {},
	})
	deployState.MarkLinkSettled("saveOrderFunction::ordersTable")

	s.Assert().Empty(deployState.ClaimContributionLayersReadyToUpdate())
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
