package specmerge

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type LinkProjectionsTestSuite struct {
	suite.Suite
}

func (s *LinkProjectionsTestSuite) Test_applies_a_contribution_to_the_resource_that_owns_it() {
	spec := core.MappingNodeFields(
		"handler",
		core.MappingNodeFromString("src/handler.handler"),
	)

	result, err := ApplyLinkProjections(spec, "listUsersFunction", []state.LinkState{
		vpcLinkState(),
	})
	s.Require().NoError(err)
	s.Require().Empty(result.Unresolved)

	subnetIds, _ := core.GetPathValue(
		"$.vpcConfig.subnetIds",
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NotNil(subnetIds)
	s.Assert().Len(subnetIds.Items, 2)

	// The source spec must not be modified, callers rely on the declared spec staying
	// as it was so it can be persisted without the contributions.
	_, hasVPCConfig := spec.Fields["vpcConfig"]
	s.Assert().False(hasVPCConfig)
}

// A link contributes to several resources at once, holding mappings for all of them. A
// link that configures a function's networking also grants its execution role the
// permissions that networking needs, so composing the function must not graft the role's
// policy statement into the function's spec.
func (s *LinkProjectionsTestSuite) Test_ignores_contributions_owned_by_another_resource() {
	spec := core.MappingNodeFields(
		"handler",
		core.MappingNodeFromString("src/handler.handler"),
	)

	result, err := ApplyLinkProjections(spec, "listUsersFunction", []state.LinkState{
		vpcLinkState(),
	})
	s.Require().NoError(err)

	policies, _ := core.GetPathValue(
		"$.policies",
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Assert().Nil(policies, "the execution role's policy was applied to the function")
}

// Contributions to a shared resource are array elements selected by a predicate, with one
// element owned by each of many links. Composing must match on the predicate and leave a
// single element per contribution however many times it runs, otherwise a role
// accumulates a duplicate statement on every deployment.
func (s *LinkProjectionsTestSuite) Test_applying_the_same_contributions_twice_is_a_no_op() {
	spec := executionRoleSpec()
	links := []state.LinkState{vpcLinkState(), tableAccessLinkState()}

	once, err := ApplyLinkProjections(spec, "lambdaExecutionRole", links)
	s.Require().NoError(err)

	twice, err := ApplyLinkProjections(once.Spec, "lambdaExecutionRole", links)
	s.Require().NoError(err)

	statements, _ := core.GetPathValue(
		"$.policies[@.policyName = \"bluelink-link-access\"].policyDocument.statement",
		twice.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NotNil(statements)
	s.Assert().Len(
		statements.Items,
		2,
		"each link's statement should appear exactly once however many times composition runs",
	)
}

// Contributions from separate links to the same array must accumulate rather than
// overwrite one another.
func (s *LinkProjectionsTestSuite) Test_applies_contributions_from_several_links_to_one_resource() {
	result, err := ApplyLinkProjections(
		executionRoleSpec(),
		"lambdaExecutionRole",
		[]state.LinkState{vpcLinkState(), tableAccessLinkState()},
	)
	s.Require().NoError(err)
	s.Require().Empty(result.Unresolved)

	statements, _ := core.GetPathValue(
		"$.policies[@.policyName = \"bluelink-link-access\"].policyDocument.statement",
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NotNil(statements)
	s.Assert().Len(statements.Items, 2)

	// The policy links write into is not declared by the blueprint, so composing has to
	// create it alongside the author's own policy rather than replacing it.
	policies, _ := core.GetPathValue(
		"$.policies",
		result.Spec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NotNil(policies)
	s.Assert().Len(policies.Items, 2, "the author's declared policy was lost")
}

// Change staging compares arrays element by element unless the resource type opts into
// sorting them by a field, so the order contributions are appended in is the order they
// are compared in. Neither input is ordered, mappings are held in a map and the order
// links come back in depends on the state store so composing must impose one, or the
// same contributions produce different arrays and the diff reports elements that have not
// changed.
func (s *LinkProjectionsTestSuite) Test_orders_contributions_the_same_way_every_time() {
	forward := []state.LinkState{vpcLinkState(), tableAccessLinkState()}
	reversed := []state.LinkState{tableAccessLinkState(), vpcLinkState()}

	fromForward, err := ApplyLinkProjections(executionRoleSpec(), "lambdaExecutionRole", forward)
	s.Require().NoError(err)

	fromReversed, err := ApplyLinkProjections(executionRoleSpec(), "lambdaExecutionRole", reversed)
	s.Require().NoError(err)

	s.Assert().True(
		core.MappingNodeEqual(fromForward.Spec, fromReversed.Spec),
		"the order links are supplied in changed the composed spec",
	)

	// Repeated composition of identical input must also be stable, since map iteration
	// order varies between runs within a single process.
	for range 20 {
		repeated, err := ApplyLinkProjections(executionRoleSpec(), "lambdaExecutionRole", forward)
		s.Require().NoError(err)
		s.Require().True(
			core.MappingNodeEqual(fromForward.Spec, repeated.Spec),
			"composing the same contributions twice produced different specs",
		)
	}
}

// A mapping the framework holds a record for, with no value behind it, is reported so a
// caller that must not deploy an incomplete spec can refuse to.
func (s *LinkProjectionsTestSuite) Test_reports_a_contribution_that_cannot_be_resolved() {
	link := vpcLinkState()
	link.Data = map[string]*core.MappingNode{}

	result, err := ApplyLinkProjections(
		core.MappingNodeFields("handler", core.MappingNodeFromString("src/handler.handler")),
		"listUsersFunction",
		[]state.LinkState{link},
	)
	s.Require().NoError(err)
	s.Require().Len(result.Unresolved, 1)
	s.Assert().Equal("appVpc::listUsersFunction", result.Unresolved[0].LinkName)
	s.Assert().Equal("spec.vpcConfig.subnetIds", result.Unresolved[0].ResourceFieldPath)
}

func (s *LinkProjectionsTestSuite) Test_applies_nothing_when_no_link_contributes() {
	result, err := ApplyLinkProjections(
		core.MappingNodeFields("handler", core.MappingNodeFromString("src/handler.handler")),
		"listUsersFunction",
		[]state.LinkState{},
	)
	s.Require().NoError(err)
	s.Assert().Empty(result.Unresolved)
	s.Assert().Len(result.Spec.Fields, 1)
}

// Composing and then removing must return the spec to what it was, or state accumulates
// the shape of contributions after the values are gone.
func (s *LinkProjectionsTestSuite) Test_removing_contributions_undoes_composing_them() {
	links := []state.LinkState{vpcLinkState(), tableAccessLinkState()}
	declared := executionRoleSpec()

	composed, err := ApplyLinkProjections(declared, "lambdaExecutionRole", links)
	s.Require().NoError(err)
	s.Require().False(core.MappingNodeEqual(declared, composed.Spec))

	stripped, err := RemoveLinkProjections(composed.Spec, "lambdaExecutionRole", links)
	s.Require().NoError(err)
	s.Assert().True(
		core.MappingNodeEqual(declared, stripped),
		"removing the contributions did not return the spec to what the blueprint declares",
	)
}

// The spec a resource is reconciled from describes the deployed resource, where a link's
// contribution may not be present at all. Its absence is a fact about the resource rather
// than a problem with the removal.
func (s *LinkProjectionsTestSuite) Test_removing_a_contribution_that_is_not_there() {
	declared := executionRoleSpec()

	stripped, err := RemoveLinkProjections(
		declared,
		"lambdaExecutionRole",
		[]state.LinkState{vpcLinkState(), tableAccessLinkState()},
	)
	s.Require().NoError(err)
	s.Assert().True(core.MappingNodeEqual(declared, stripped))
}

// Only what the named resource owns is removed, for the same reason only what it owns is
// applied.
func (s *LinkProjectionsTestSuite) Test_removing_leaves_another_resources_fields_alone() {
	function := core.MappingNodeFields(
		"handler",
		core.MappingNodeFromString("src/handler.handler"),
		"policies",
		core.MappingNodeItems(
			core.MappingNodeFields(
				"policyName",
				core.MappingNodeFromString("bluelink-link-access"),
			),
		),
	)

	stripped, err := RemoveLinkProjections(
		function,
		"listUsersFunction",
		[]state.LinkState{vpcLinkState()},
	)
	s.Require().NoError(err)

	policies, _ := core.GetPathValue("$.policies", stripped, core.MappingNodeMaxTraverseDepth)
	s.Assert().NotNil(policies, "a field owned by the execution role was removed from the function")
}

// A link that writes a function's VPC configuration and grants its execution role the
// matching permissions, mapping into two resources.
func vpcLinkState() state.LinkState {
	return state.LinkState{
		Name: "appVpc::listUsersFunction",
		Data: map[string]*core.MappingNode{
			"listUsersFunction": core.MappingNodeFields(
				"vpcConfig",
				core.MappingNodeFields(
					"subnetIds",
					core.MappingNodeItems(
						core.MappingNodeFromString("subnet-1"),
						core.MappingNodeFromString("subnet-2"),
					),
				),
			),
			"listUsersFunctionExecutionRole": core.MappingNodeFields(
				"permission",
				core.MappingNodeFields(
					"sid",
					core.MappingNodeFromString("VPCNetworkInterfaces"),
					"effect",
					core.MappingNodeFromString("Allow"),
				),
			),
		},
		ResourceDataMappings: map[string]string{
			"listUsersFunction::spec.vpcConfig.subnetIds": "listUsersFunction.vpcConfig.subnetIds",
			"lambdaExecutionRole::spec.policies[@.policyName=\"bluelink-link-access\"]" +
				".policyDocument.statement[@.sid=\"VPCNetworkInterfaces\"]": "listUsersFunctionExecutionRole.permission",
		},
	}
}

// A second link granting the same execution role access to a table, owning its own
// statement in the same policy document.
func tableAccessLinkState() state.LinkState {
	return state.LinkState{
		Name: "listUsersFunction::usersTable",
		Data: map[string]*core.MappingNode{
			"listUsersFunctionExecutionRole": core.MappingNodeFields(
				"permission",
				core.MappingNodeFields(
					"sid",
					core.MappingNodeFromString("DynamoDBAccessUsersTable"),
					"effect",
					core.MappingNodeFromString("Allow"),
				),
			),
		},
		ResourceDataMappings: map[string]string{
			"lambdaExecutionRole::spec.policies[@.policyName=\"bluelink-link-access\"]" +
				".policyDocument.statement[@.sid=\"DynamoDBAccessUsersTable\"]": "listUsersFunctionExecutionRole.permission",
		},
	}
}

func executionRoleSpec() *core.MappingNode {
	// The role as the blueprint declares it: the policy the author wrote, and nothing a
	// link has contributed. The policy links write into does not exist until a link
	// creates it, which is the shape composition has to cope with.
	return core.MappingNodeFields(
		"roleName",
		core.MappingNodeFromString("lambda-exec"),
		"policies",
		core.MappingNodeItems(
			core.MappingNodeFields(
				"policyName",
				core.MappingNodeFromString("declared-by-the-author"),
				"policyDocument",
				core.MappingNodeFields(
					"statement",
					core.MappingNodeItems(
						core.MappingNodeFields(
							"sid",
							core.MappingNodeFromString("DeclaredAccess"),
						),
					),
				),
			),
		),
	)
}

func TestLinkProjectionsTestSuite(t *testing.T) {
	suite.Run(t, new(LinkProjectionsTestSuite))
}
