package validation

import (
	"context"
	"fmt"

	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/corefunctions"
	"github.com/two-hundred/celerity/libs/blueprint/internal"
	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/refgraph"
	"github.com/two-hundred/celerity/libs/blueprint/resourcehelpers"
	"github.com/two-hundred/celerity/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
)

type MappingNodeValidationTestSuite struct {
	funcRegistry      provider.FunctionRegistry
	refChainCollector refgraph.RefChainCollector
	resourceRegistry  resourcehelpers.Registry
}

var _ = Suite(&MappingNodeValidationTestSuite{})

func (s *MappingNodeValidationTestSuite) SetUpTest(c *C) {
	s.funcRegistry = &internal.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"trim":       corefunctions.NewTrimFunction(),
			"trimprefix": corefunctions.NewTrimPrefixFunction(),
			"list":       corefunctions.NewListFunction(),
			"object":     corefunctions.NewObjectFunction(),
			"jsondecode": corefunctions.NewJSONDecodeFunction(),
		},
	}
	s.refChainCollector = refgraph.NewRefChainCollector()
	s.resourceRegistry = &internal.ResourceRegistryMock{
		Resources: map[string]provider.Resource{},
	}
}

func (s *MappingNodeValidationTestSuite) Test_succeeds_without_any_issues_for_a_valid_mapping_node(c *C) {
	field1Value := "value1"
	field2ArgValue := " value2 "
	field3Item1Value := 2
	field3Item2Value := 3
	mappingNode := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"field1": {
				Scalar: &core.ScalarValue{
					StringValue: &field1Value,
				},
			},
			"field2": {
				StringWithSubstitutions: &substitutions.StringOrSubstitutions{
					Values: []*substitutions.StringOrSubstitution{
						{
							SubstitutionValue: &substitutions.Substitution{
								Function: &substitutions.SubstitutionFunctionExpr{
									FunctionName: "trim",
									Arguments: []*substitutions.SubstitutionFunctionArg{
										{
											Value: &substitutions.Substitution{
												StringValue: &field2ArgValue,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"field3": {
				Items: []*core.MappingNode{
					{
						Scalar: &core.ScalarValue{
							IntValue: &field3Item1Value,
						},
					},
					{
						Scalar: &core.ScalarValue{
							IntValue: &field3Item2Value,
						},
					},
				},
			},
		},
	}

	diagnostics, err := ValidateMappingNode(
		context.TODO(),
		"datasources.networking",
		"metadata.custom",
		/* usedInResourceDerivedFromTemplate */ false,
		mappingNode,
		nil,
		nil,
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
	)

	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *MappingNodeValidationTestSuite) Test_succeeds_with_info_diagnostic_for_exceeding_max_validation_traversal_depth(c *C) {
	mappingNode := buildTestMappingNode(core.MappingNodeMaxTraverseDepth + 10)

	diagnostics, err := ValidateMappingNode(
		context.TODO(),
		"datasources.networking",
		"metadata.custom",
		/* usedInResourceDerivedFromTemplate */ false,
		mappingNode,
		nil,
		nil,
		s.funcRegistry,
		s.refChainCollector,
		s.resourceRegistry,
	)

	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 1)
	c.Assert(diagnostics[0].Level, Equals, core.DiagnosticLevelInfo)
	c.Assert(diagnostics[0].Message, Equals, fmt.Sprintf(
		"Exceeded max traverse depth of %d. Skipping further validation.",
		core.MappingNodeMaxTraverseDepth,
	))
}

func buildTestMappingNode(depth int) *core.MappingNode {
	root := &core.MappingNode{}
	current := root
	for i := 0; i < depth; i++ {
		next := &core.MappingNode{}
		fieldName := fmt.Sprintf("fieldDepth%d", depth)
		current.Fields = map[string]*core.MappingNode{
			fieldName: next,
		}
		current = next
	}
	return root
}
