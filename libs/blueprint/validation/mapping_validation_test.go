package validation

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
)

type MappingNodeValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
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
	s.resourceRegistry = internal.NewResourceRegistryMock(
		map[string]provider.Resource{},
	)
	s.dataSourceRegistry = &internal.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{
			"aws/ec2/instance": newTestEC2InstanceDataSource(),
			"aws/vpc":          newTestVPCDataSource(),
			"aws/vpc2":         newTestVPC2DataSource(),
			"aws/vpc3":         newTestVPC3DataSource(),
		},
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
		&ValidationContext{
			BpSchema:           nil,
			Params:             nil,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
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
		&ValidationContext{
			BpSchema:           nil,
			Params:             nil,
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
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
