package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type LinkContributionTargetsTestSuite struct {
	suite.Suite
}

func (s *LinkContributionTargetsTestSuite) Test_reports_the_resources_a_link_declared() {
	targets := BuildLinkContributionTargets(&changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"saveOrderFunction": {
				OutboundLinkChanges: map[string]provider.LinkChanges{
					"ordersTable": {
						ResourceDataMappings: map[string]string{
							"saveOrderFunction::spec.environment.variables.TABLE_NAME": "saveOrderFunction.tableName",
							"ordersRole::spec.policies[0].statements":                  "saveOrderFunction.policy",
						},
					},
				},
			},
		},
	})

	s.Assert().Equal(
		map[string][]string{
			"saveOrderFunction::ordersTable": {"ordersRole", "saveOrderFunction"},
		},
		targets,
		"a link's targets are every resource its declared mappings name, "+
			"including one that is neither of its endpoints",
	)
}

// Several fields of one resource are one target. The merged update for that resource is
// built once, so waiting on the link once is what the join needs.
func (s *LinkContributionTargetsTestSuite) Test_collapses_repeated_targets_to_one() {
	targets := BuildLinkContributionTargets(&changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"saveOrderFunction": {
				NewOutboundLinks: map[string]provider.LinkChanges{
					"ordersTable": {
						ResourceDataMappings: map[string]string{
							"saveOrderFunction::spec.environment.variables.TABLE_NAME":   "saveOrderFunction.tableName",
							"saveOrderFunction::spec.environment.variables.TABLE_REGION": "saveOrderFunction.tableRegion",
						},
					},
				},
			},
		},
	})

	s.Assert().Equal(
		map[string][]string{
			"saveOrderFunction::ordersTable": {"saveOrderFunction"},
		},
		targets,
	)
}

// Every link that has not moved to contributions declares nothing, and has to appear as a
// link with no targets rather than be absent, so that nothing waits on it.
func (s *LinkContributionTargetsTestSuite) Test_reports_no_targets_for_a_link_that_declares_nothing() {
	targets := BuildLinkContributionTargets(&changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"saveOrderFunction": {
				OutboundLinkChanges: map[string]provider.LinkChanges{
					"ordersTable": {},
				},
			},
		},
	})

	s.Assert().Equal(
		map[string][]string{"saveOrderFunction::ordersTable": {}},
		targets,
	)
}

// A key that is not in the "{resourceName}::{fieldPath}" form names no resource. Treating
// it as one would leave a merged update waiting on a resource that does not exist for the
// rest of the deployment, which is the stall this join exists to make finite.
func (s *LinkContributionTargetsTestSuite) Test_skips_a_mapping_key_naming_no_resource() {
	targets := BuildLinkContributionTargets(&changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"saveOrderFunction": {
				OutboundLinkChanges: map[string]provider.LinkChanges{
					"ordersTable": {
						ResourceDataMappings: map[string]string{
							"spec.environment.variables.TABLE_NAME": "saveOrderFunction.tableName",
							"::spec.policies":                       "saveOrderFunction.policy",
							"ordersRole::spec.policies":             "saveOrderFunction.policy",
						},
					},
				},
			},
		},
	})

	s.Assert().Equal(
		map[string][]string{
			"saveOrderFunction::ordersTable": {"ordersRole"},
		},
		targets,
	)
}

func (s *LinkContributionTargetsTestSuite) Test_reports_no_targets_for_an_absent_change_set() {
	s.Assert().Equal(map[string][]string{}, BuildLinkContributionTargets(nil))
}

func TestLinkContributionTargetsTestSuite(t *testing.T) {
	suite.Run(t, new(LinkContributionTargetsTestSuite))
}
