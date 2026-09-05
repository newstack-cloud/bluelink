package specmerge

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ResourceContributionsTestSuite struct {
	suite.Suite
}

func (s *ResourceContributionsTestSuite) Test_sets_a_field_a_link_contributes() {
	result, err := MergeResourceContributions(
		core.MappingNodeFields(
			"handler",
			core.MappingNodeFromString("src/sync.handler"),
		),
		"saveOrderFunction",
		[]LinkResourceContribution{
			s.contribution(
				"saveOrderFunction::ordersTable",
				"saveOrderFunction",
				"spec.environment.variables.TABLE_NAME",
				core.MappingNodeFromString("orders"),
				provider.ContributionActionSet,
			),
		},
	)

	s.Require().NoError(err)
	s.Assert().Empty(result.Unresolved)
	s.Assert().Equal(
		"orders",
		s.valueAt(result.Spec, "$.environment.variables.TABLE_NAME"),
	)
	s.Assert().Equal(
		"src/sync.handler",
		s.valueAt(result.Spec, "$.handler"),
		"what the resource already declared is left alone",
	)
}

// Appending is what lets two links contribute to one list without either having to know
// about the other, which is the case a policy document with a statement per link needs.
func (s *ResourceContributionsTestSuite) Test_appends_contributions_from_two_links_to_one_list() {
	result, err := MergeResourceContributions(
		core.MappingNodeFields(),
		"ordersRole",
		[]LinkResourceContribution{
			s.contribution(
				"saveOrderFunction::ordersTable",
				"ordersRole",
				"spec.policies",
				core.MappingNodeFromString("dynamodb:PutItem"),
				provider.ContributionActionAppend,
			),
			s.contribution(
				"saveOrderFunction::appQueue",
				"ordersRole",
				"spec.policies",
				core.MappingNodeFromString("sqs:SendMessage"),
				provider.ContributionActionAppend,
			),
		},
	)

	s.Require().NoError(err)
	s.Assert().Empty(result.Unresolved)
	s.Assert().Equal(
		[]string{"sqs:SendMessage", "dynamodb:PutItem"},
		s.itemsAt(result.Spec, "$.policies"),
		"both links' statements are present, ordered by link name rather than by "+
			"the order the links happened to finish in",
	)
}

func (s *ResourceContributionsTestSuite) Test_appends_to_a_list_the_resource_already_declares() {
	result, err := MergeResourceContributions(
		core.MappingNodeFields(
			"policies",
			&core.MappingNode{
				Items: []*core.MappingNode{
					core.MappingNodeFromString("logs:CreateLogGroup"),
				},
			},
		),
		"ordersRole",
		[]LinkResourceContribution{
			s.contribution(
				"saveOrderFunction::ordersTable",
				"ordersRole",
				"spec.policies",
				core.MappingNodeFromString("dynamodb:PutItem"),
				provider.ContributionActionAppend,
			),
		},
	)

	s.Require().NoError(err)
	s.Assert().Equal(
		[]string{"logs:CreateLogGroup", "dynamodb:PutItem"},
		s.itemsAt(result.Spec, "$.policies"),
		"the contribution joins what the blueprint declares rather than replacing it",
	)
}

func (s *ResourceContributionsTestSuite) Test_ignores_contributions_to_another_resource() {
	result, err := MergeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		[]LinkResourceContribution{
			s.contribution(
				"saveOrderFunction::ordersTable",
				"ordersRole",
				"spec.policies",
				core.MappingNodeFromString("dynamodb:PutItem"),
				provider.ContributionActionAppend,
			),
		},
	)

	s.Require().NoError(err)
	s.Assert().Empty(result.Unresolved)
	s.Assert().Nil(result.Spec.Fields["policies"])
}

// Reported rather than raised, in the same way a projection that will not apply is, and
// named against the link so the report says whose contribution went missing.
func (s *ResourceContributionsTestSuite) Test_reports_an_append_to_a_field_holding_a_scalar() {
	result, err := MergeResourceContributions(
		core.MappingNodeFields(
			"policies",
			core.MappingNodeFromString("not-a-list"),
		),
		"ordersRole",
		[]LinkResourceContribution{
			s.contribution(
				"saveOrderFunction::ordersTable",
				"ordersRole",
				"spec.policies",
				core.MappingNodeFromString("dynamodb:PutItem"),
				provider.ContributionActionAppend,
			),
		},
	)

	s.Require().NoError(err)
	s.Require().Len(result.Unresolved, 1)
	s.Assert().Equal("saveOrderFunction::ordersTable", result.Unresolved[0].LinkName)
	s.Assert().Equal("spec.policies", result.Unresolved[0].ResourceFieldPath)
	s.Assert().Equal(
		"not-a-list",
		s.valueAt(result.Spec, "$.policies"),
		"what the resource declared is left as it was rather than discarded",
	)
}

func (s *ResourceContributionsTestSuite) Test_reports_a_contribution_carrying_no_value() {
	result, err := MergeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		[]LinkResourceContribution{
			s.contribution(
				"saveOrderFunction::ordersTable",
				"saveOrderFunction",
				"spec.environment.variables.TABLE_NAME",
				nil,
				provider.ContributionActionSet,
			),
		},
	)

	s.Require().NoError(err)
	s.Require().Len(result.Unresolved, 1)
	s.Assert().Equal("saveOrderFunction::ordersTable", result.Unresolved[0].LinkName)
}

// Two links each stating the whole value of one field is not a merge, it is a
// disagreement, and applying one of them quietly is the outcome worth refusing.
func (s *ResourceContributionsTestSuite) Test_refuses_to_merge_two_links_setting_one_field() {
	contributions := []LinkResourceContribution{
		s.contribution(
			"saveOrderFunction::ordersTable",
			"saveOrderFunction",
			"spec.environment.variables.TABLE_NAME",
			core.MappingNodeFromString("orders"),
			provider.ContributionActionSet,
		),
		s.contribution(
			"saveOrderFunction::archiveTable",
			"saveOrderFunction",
			"spec.environment.variables.TABLE_NAME",
			core.MappingNodeFromString("archive"),
			provider.ContributionActionSet,
		),
	}

	result, err := MergeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		contributions,
	)
	s.Require().NoError(err)

	s.Assert().Nil(
		result.Spec.Fields["environment"],
		"one of the two values was applied rather than the disagreement being reported",
	)
	s.Require().Len(result.Unresolved, 2, "both links are named, not just one of them")

	for _, unresolved := range result.Unresolved {
		s.Assert().Equal(
			"spec.environment.variables.TABLE_NAME",
			unresolved.ResourceFieldPath,
		)
		s.Assert().Contains(unresolved.Reason, "also set to a different value by")
	}
	s.Assert().Contains(result.Unresolved[0].Reason, "saveOrderFunction::ordersTable")
	s.Assert().Contains(result.Unresolved[1].Reason, "saveOrderFunction::archiveTable")
}

// Links that agree are not in conflict. Refusing a deployment because two links independently
// arrived at the same value would be refusing over the absence of a problem.
func (s *ResourceContributionsTestSuite) Test_merges_two_links_setting_one_field_to_the_same_value() {
	contributions := []LinkResourceContribution{
		s.contribution(
			"saveOrderFunction::appVpc",
			"saveOrderFunction",
			"spec.vpcConfig",
			core.MappingNodeFromString("vpc-1"),
			provider.ContributionActionSet,
		),
		s.contribution(
			"saveOrderFunction::appVpcEndpoint",
			"saveOrderFunction",
			"spec.vpcConfig",
			core.MappingNodeFromString("vpc-1"),
			provider.ContributionActionSet,
		),
	}

	result, err := MergeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		contributions,
	)
	s.Require().NoError(err)

	s.Assert().Empty(result.Unresolved)
	s.Assert().Equal(
		"vpc-1",
		core.StringValue(result.Spec.Fields["vpcConfig"]),
	)
}

func (s *ResourceContributionsTestSuite) Test_merges_links_appending_to_one_field() {
	contributions := []LinkResourceContribution{
		s.contribution(
			"saveOrderFunction::ordersTable",
			"ordersRole",
			"spec.policies",
			core.MappingNodeFromString("dynamodb:PutItem"),
			provider.ContributionActionAppend,
		),
		s.contribution(
			"saveOrderFunction::appQueue",
			"ordersRole",
			"spec.policies",
			core.MappingNodeFromString("sqs:SendMessage"),
			provider.ContributionActionAppend,
		),
	}

	result, err := MergeResourceContributions(
		core.MappingNodeFields(),
		"ordersRole",
		contributions,
	)
	s.Require().NoError(err)

	s.Assert().Empty(result.Unresolved)
	s.Assert().Len(result.Spec.Fields["policies"].Items, 2)
}

func (s *ResourceContributionsTestSuite) contribution(
	linkName string,
	resourceName string,
	fieldPath string,
	value *core.MappingNode,
	action provider.ContributionAction,
) LinkResourceContribution {
	return LinkResourceContribution{
		LinkName: linkName,
		Contribution: &provider.ResourceContribution{
			ResourceName: resourceName,
			FieldPath:    fieldPath,
			Value:        value,
			Action:       action,
		},
	}
}

func (s *ResourceContributionsTestSuite) valueAt(spec *core.MappingNode, path string) string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	return core.StringValue(value)
}

func (s *ResourceContributionsTestSuite) itemsAt(spec *core.MappingNode, path string) []string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	items := []string{}
	for _, item := range value.Items {
		items = append(items, core.StringValue(item))
	}

	return items
}

func TestResourceContributionsTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceContributionsTestSuite))
}
