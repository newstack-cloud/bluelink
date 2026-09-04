package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type LinkContributionDataTestSuite struct {
	suite.Suite
}

// What the framework records has to be enough to rebuild the contribution in a later
// deployment that does not run the link, which is the whole reason it is recorded.
func (s *LinkContributionDataTestSuite) Test_records_a_contribution_so_it_can_be_read_back() {
	contributed := ContributionsToLinkData([]*provider.ResourceContribution{
		{
			ResourceName:    "ordersRole",
			FieldPath:       "spec.policies[0].statements",
			Value:           core.MappingNodeFromString("dynamodb:PutItem"),
			Action:          provider.ContributionActionAppend,
			RetainOnRemoval: true,
		},
	})

	stored := state.LinkState{
		Name:                 "saveOrderFunction::ordersTable",
		Data:                 contributed.Data,
		ResourceDataMappings: contributed.ResourceDataMappings,
		ContributionRecords:  contributed.ContributionRecords,
	}

	readBack := specmerge.StoredResourceContributions(
		"ordersRole",
		stored,
		/* retainedOnly */ false,
	)

	s.Require().Len(readBack, 1)
	s.Assert().Equal("saveOrderFunction::ordersTable", readBack[0].LinkName)
	s.Assert().Equal("spec.policies[0].statements", readBack[0].Contribution.FieldPath)
	s.Assert().Equal(
		"dynamodb:PutItem",
		core.StringValue(readBack[0].Contribution.Value),
		"the value is found through the mapping the framework recorded",
	)
	s.Assert().Equal(
		provider.ContributionActionAppend,
		readBack[0].Contribution.Action,
		"appending is not inferred from the value, so it has to survive the round trip",
	)
	s.Assert().True(readBack[0].Contribution.RetainOnRemoval)
}

// A link data path is traversed to find the value it points at, and a resource field path
// is written in that same syntax of dots and brackets. Recording
// "spec.policies[0].statements" as the link data path would have it read as a traversal,
// spec to policies to index 0 to statements, of data that has no such structure. Nothing
// is found, no error is raised, and the resource is deployed without a field one of its
// links owns.
//
// The recorded path is an index into a list the framework owns instead, so what the
// resource's field path happens to look like cannot change where the value is found.
func (s *LinkContributionDataTestSuite) Test_does_not_record_a_resource_field_path_as_a_link_data_path() {
	fieldPath := "spec.policies[0].statements"

	contributed := ContributionsToLinkData([]*provider.ResourceContribution{
		{
			ResourceName: "ordersRole",
			FieldPath:    fieldPath,
			Value:        core.MappingNodeFromString("dynamodb:PutItem"),
		},
	})

	linkDataPath := contributed.ResourceDataMappings["ordersRole::"+fieldPath]
	s.Require().NotEmpty(linkDataPath, "the contribution was not recorded at all")
	s.Assert().NotContains(
		linkDataPath,
		fieldPath,
		"the resource's field path was reused as the path the value is read back through",
	)
	s.Assert().Equal("contributions[0]", linkDataPath)

	// The mapping has to resolve through the same traversal the reader makes, or it
	// points at nothing while looking perfectly reasonable.
	value, err := core.GetPathValue(
		"$."+linkDataPath,
		&core.MappingNode{Fields: contributed.Data},
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NoError(err)
	s.Assert().Equal("dynamodb:PutItem", core.StringValue(value))
}

// Two contributions have to keep pointing at their own values, so the mappings and the
// list they index have to agree however the contributions arrive.
func (s *LinkContributionDataTestSuite) Test_records_two_contributions_against_their_own_values() {
	contributions := []*provider.ResourceContribution{
		{
			ResourceName: "saveOrderFunction",
			FieldPath:    "spec.environment.variables.TABLE_REGION",
			Value:        core.MappingNodeFromString("eu-west-2"),
		},
		{
			ResourceName: "saveOrderFunction",
			FieldPath:    "spec.environment.variables.TABLE_NAME",
			Value:        core.MappingNodeFromString("orders"),
		},
	}

	contributed := ContributionsToLinkData(contributions)
	stored := state.LinkState{
		Name:                 "saveOrderFunction::ordersTable",
		Data:                 contributed.Data,
		ResourceDataMappings: contributed.ResourceDataMappings,
		ContributionRecords:  contributed.ContributionRecords,
	}

	readBack := specmerge.StoredResourceContributions(
		"saveOrderFunction",
		stored,
		/* retainedOnly */ false,
	)
	s.Require().Len(readBack, 2)

	byField := map[string]string{}
	for _, contribution := range readBack {
		byField[contribution.Contribution.FieldPath] = core.StringValue(
			contribution.Contribution.Value,
		)
	}
	s.Assert().Equal(
		map[string]string{
			"spec.environment.variables.TABLE_NAME":   "orders",
			"spec.environment.variables.TABLE_REGION": "eu-west-2",
		},
		byField,
	)
}

// Every link that has not moved to contributions produces none, and has to record nothing
// rather than recording that it recorded nothing.
func (s *LinkContributionDataTestSuite) Test_records_nothing_for_a_link_that_contributes_nothing() {
	contributed := ContributionsToLinkData(nil)

	s.Assert().Nil(contributed.Data)
	s.Assert().Nil(contributed.ResourceDataMappings)
	s.Assert().Nil(contributed.ContributionRecords)
}

func TestLinkContributionDataTestSuite(t *testing.T) {
	suite.Run(t, new(LinkContributionDataTestSuite))
}
