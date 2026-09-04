package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type MergedResourceUpdateTestSuite struct {
	suite.Suite
}

func (s *MergedResourceUpdateTestSuite) Test_carries_the_statements_of_links_the_deployment_did_not_run() {
	deployCtx := s.deployContext(
		map[string]*LinkDeployResult{
			"saveOrderFunction::ordersTable": s.produced(
				"ordersRole",
				"spec.policies",
				"dynamodb:PutItem",
			),
		},
		[]*state.LinkState{
			s.storedAppend("archiveFunction::appQueue", "ordersRole", "spec.policies", "sqs:SendMessage"),
			s.storedAppend("reportFunction::appBucket", "ordersRole", "spec.policies", "s3:GetObject"),
		},
		nil,
	)

	result, err := ComposeMergedResourceSpec(
		deployCtx,
		ContributionLayer{ResourceName: "ordersRole"},
		core.MappingNodeFields(),
		[]string{"saveOrderFunction::ordersTable"},
	)

	s.Require().NoError(err)
	s.Assert().ElementsMatch(
		[]string{"sqs:SendMessage", "s3:GetObject", "dynamodb:PutItem"},
		s.itemsAt(result.Spec, "$.policies"),
		"the two links that did not run keep their statements",
	)
}

// A link that ran states everything it needs, so what it recorded before is not read as
// well. Reading both would put back a field it has stopped contributing.
func (s *MergedResourceUpdateTestSuite) Test_a_link_that_ran_is_not_also_read_from_state() {
	deployCtx := s.deployContext(
		map[string]*LinkDeployResult{
			"saveOrderFunction::ordersTable": s.produced(
				"ordersRole",
				"spec.policies",
				"dynamodb:PutItem",
			),
		},
		[]*state.LinkState{
			s.storedAppend(
				"saveOrderFunction::ordersTable",
				"ordersRole",
				"spec.policies",
				"dynamodb:GetItem",
			),
		},
		nil,
	)

	result, err := ComposeMergedResourceSpec(
		deployCtx,
		ContributionLayer{ResourceName: "ordersRole"},
		core.MappingNodeFields(),
		[]string{"saveOrderFunction::ordersTable"},
	)

	s.Require().NoError(err)
	s.Assert().Equal(
		[]string{"dynamodb:PutItem"},
		s.itemsAt(result.Spec, "$.policies"),
		"only what the link just produced, not what it produced last time",
	)
}

// A link with no result held against it has not produced anything, whatever the join
// thinks. Treating it as having run would drop what it recorded before.
func (s *MergedResourceUpdateTestSuite) Test_reads_from_state_a_link_that_produced_no_result() {
	deployCtx := s.deployContext(
		map[string]*LinkDeployResult{},
		[]*state.LinkState{
			s.storedAppend("archiveFunction::appQueue", "ordersRole", "spec.policies", "sqs:SendMessage"),
		},
		nil,
	)

	result, err := ComposeMergedResourceSpec(
		deployCtx,
		ContributionLayer{ResourceName: "ordersRole"},
		core.MappingNodeFields(),
		[]string{"archiveFunction::appQueue"},
	)

	s.Require().NoError(err)
	s.Assert().Equal(
		[]string{"sqs:SendMessage"},
		s.itemsAt(result.Spec, "$.policies"),
	)
}

// The case a resource-level list cannot answer. A user expanding a shared role's policies
// is asking which links put statements there, and every one of them is part of the answer.
func (s *MergedResourceUpdateTestSuite) Test_names_every_link_that_contributed_to_a_field() {
	deployCtx := s.deployContext(
		map[string]*LinkDeployResult{
			"saveOrderFunction::ordersTable": s.produced(
				"ordersRole",
				"spec.policies",
				"dynamodb:PutItem",
			),
		},
		[]*state.LinkState{
			s.storedAppend("archiveFunction::appQueue", "ordersRole", "spec.policies", "sqs:SendMessage"),
			s.storedAppend("reportFunction::appBucket", "ordersRole", "spec.policies", "s3:GetObject"),
		},
		nil,
	)

	contributors := LinkContributorsFor(
		"ordersRole",
		CollectResourceContributionSources(
			deployCtx,
			[]string{"saveOrderFunction::ordersTable"},
		),
	)

	s.Assert().Equal(
		map[string][]string{
			"spec.policies": {
				"archiveFunction::appQueue",
				"reportFunction::appBucket",
				"saveOrderFunction::ordersTable",
			},
		},
		contributors,
		"the link that ran and the two that did not are all named against the field",
	)
}

// A link that ran is named from what it just produced rather than twice, once from each
// account of it.
func (s *MergedResourceUpdateTestSuite) Test_names_a_link_that_ran_once() {
	deployCtx := s.deployContext(
		map[string]*LinkDeployResult{
			"saveOrderFunction::ordersTable": s.produced(
				"ordersRole",
				"spec.policies",
				"dynamodb:PutItem",
			),
		},
		[]*state.LinkState{
			s.storedAppend(
				"saveOrderFunction::ordersTable",
				"ordersRole",
				"spec.policies",
				"dynamodb:GetItem",
			),
		},
		nil,
	)

	contributors := LinkContributorsFor(
		"ordersRole",
		CollectResourceContributionSources(
			deployCtx,
			[]string{"saveOrderFunction::ordersTable"},
		),
	)

	s.Assert().Equal(
		map[string][]string{"spec.policies": {"saveOrderFunction::ordersTable"}},
		contributors,
	)
}

func (s *MergedResourceUpdateTestSuite) Test_does_not_name_a_link_against_a_field_it_withdrew() {
	stored := s.storedAppend(
		"saveOrderFunction::ordersTable",
		"ordersRole",
		"spec.oldPolicies",
		"dynamodb:GetItem",
	)

	deployCtx := s.deployContext(
		map[string]*LinkDeployResult{
			"saveOrderFunction::ordersTable": s.produced(
				"ordersRole",
				"spec.policies",
				"dynamodb:PutItem",
			),
		},
		[]*state.LinkState{stored},
		nil,
	)

	contributors := LinkContributorsFor(
		"ordersRole",
		CollectResourceContributionSources(
			deployCtx,
			[]string{"saveOrderFunction::ordersTable"},
		),
	)

	// oldPolicies should not be in the map at all,
	// because the link that contributed to it has withdrawn that contribution.
	s.Assert().Equal(
		map[string][]string{"spec.policies": {"saveOrderFunction::ordersTable"}},
		contributors,
		"the field it no longer contributes is not attributed to it",
	)
}

func (s *MergedResourceUpdateTestSuite) deployContext(
	deployResults map[string]*LinkDeployResult,
	storedLinks []*state.LinkState,
	removedLinks []string,
) *DeployContext {
	deployState := NewDefaultDeploymentState()
	for linkName, result := range deployResults {
		deployState.SetLinkDeployResult(linkName, result)
	}

	links := map[string]*state.LinkState{}
	for _, linkState := range storedLinks {
		links[linkState.Name] = linkState
	}

	return &DeployContext{
		State:                 deployState,
		InstanceStateSnapshot: &state.InstanceState{Links: links},
		InputChanges:          &changes.BlueprintChanges{RemovedLinks: removedLinks},
	}
}

func (s *MergedResourceUpdateTestSuite) produced(
	resourceName string,
	fieldPath string,
	value string,
) *LinkDeployResult {
	return &LinkDeployResult{
		Contributions: []*provider.ResourceContribution{
			{
				ResourceName: resourceName,
				FieldPath:    fieldPath,
				Value:        core.MappingNodeFromString(value),
				Action:       provider.ContributionActionAppend,
			},
		},
	}
}

// A link as state holds it once the framework has recorded what it contributed.
func (s *MergedResourceUpdateTestSuite) storedAppend(
	linkName string,
	resourceName string,
	fieldPath string,
	value string,
) *state.LinkState {
	contributed := ContributionsToLinkData([]*provider.ResourceContribution{
		{
			ResourceName: resourceName,
			FieldPath:    fieldPath,
			Value:        core.MappingNodeFromString(value),
			Action:       provider.ContributionActionAppend,
		},
	})

	return &state.LinkState{
		Name:                 linkName,
		Data:                 contributed.Data,
		ResourceDataMappings: contributed.ResourceDataMappings,
		ContributionRecords:  contributed.ContributionRecords,
	}
}

func (s *MergedResourceUpdateTestSuite) itemsAt(spec *core.MappingNode, path string) []string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	items := []string{}
	for _, item := range value.Items {
		items = append(items, core.StringValue(item))
	}

	return items
}

func TestMergedResourceUpdateTestSuite(t *testing.T) {
	suite.Run(t, new(MergedResourceUpdateTestSuite))
}
