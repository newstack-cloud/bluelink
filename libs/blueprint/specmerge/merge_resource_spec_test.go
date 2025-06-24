package specmerge

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type MergeResourceSpecTestSuite struct {
	suite.Suite
}

func (s *MergeResourceSpecTestSuite) Test_merges_computed_fields_with_resolved_resource_spec() {
	arn := "arn:aws:lambda:us-west-2:123456789012:function:orders"
	mergedSpec, err := MergeResourceSpec(
		&provider.ResolvedResource{
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"handler": core.MappingNodeFromString("src/orders/handler"),
				},
			},
		},
		"orders",
		map[string]*core.MappingNode{
			"spec.id":                              core.MappingNodeFromString(arn),
			"spec.identifiers[\"ids.v1\"].arns[0]": core.MappingNodeFromString(arn),
		},
		[]string{
			"spec.id",
			"spec.identifiers[\"ids.v1\"].arns[0]",
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(&core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"handler": core.MappingNodeFromString("src/orders/handler"),
			"id":      core.MappingNodeFromString(arn),
			"identifiers": {
				Fields: map[string]*core.MappingNode{
					"ids.v1": {
						Fields: map[string]*core.MappingNode{
							"arns": {
								Items: []*core.MappingNode{
									core.MappingNodeFromString(arn),
								},
							},
						},
					},
				},
			},
		},
	}, mergedSpec)
}

func (s *MergeResourceSpecTestSuite) Test_fails_when_trying_to_merge_non_computed_field() {
	arn := "arn:aws:lambda:us-west-2:223456789012:function:orders"
	_, err := MergeResourceSpec(
		&provider.ResolvedResource{
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"handler": core.MappingNodeFromString("src/orders/handler2"),
				},
			},
		},
		"orders",
		map[string]*core.MappingNode{
			"spec.id":                              core.MappingNodeFromString(arn),
			"spec.identifiers[\"ids.v1\"].arns[0]": core.MappingNodeFromString(arn),
			// Ignored is not in the expected computed fields list.
			"spec.ignored": core.MappingNodeFromString("ignored"),
		},
		[]string{
			"spec.id",
			"spec.identifiers[\"ids.v1\"].arns[0]",
		},
	)
	s.Assert().Error(err)
	runErr, isRunErr := err.(*errors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(ErrorReasonCodeUnexpectedComputedField, runErr.ReasonCode)
	s.Assert().Equal(
		"run error: unexpected computed field \"spec.ignored\" found in"+
			" resource \"orders\", computed fields returned by the resource "+
			"deploy method can include the following: "+
			"spec.id, spec.identifiers[\"ids.v1\"].arns[0]",
		runErr.Error(),
	)
}

func TestMergeResourceSpecTestSuite(t *testing.T) {
	suite.Run(t, new(MergeResourceSpecTestSuite))
}
