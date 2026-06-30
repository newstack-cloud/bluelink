package core

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MappingPathsTestSuite struct {
	suite.Suite
}

func (s *MappingPathsTestSuite) Test_get_value_by_path_for_complex_path() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	node, expectedEndpoint := fixtureMappingNode1()
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(&MappingNode{
		Scalar: &ScalarValue{
			StringValue: &expectedEndpoint,
		},
	}, value)
}

func (s *MappingPathsTestSuite) Test_get_value_by_path_for_path_with_array_selector() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[@.endpoint = \"https://other.example.com\"].protocol"
	node, expectedProtocol := fixtureMappingNode2()
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(&MappingNode{
		Scalar: &ScalarValue{
			StringValue: &expectedProtocol,
		},
	}, value)
}

func (s *MappingPathsTestSuite) Test_returns_passed_in_node_for_root_identity_path() {
	path := "$"
	node, _ := fixtureMappingNode1()
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(node, value)
}

func (s *MappingPathsTestSuite) Test_returns_nil_for_non_existent_path() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].missingField"
	node, _ := fixtureMappingNode1()
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Nil(value)
}

func (s *MappingPathsTestSuite) Test_returns_nil_for_path_that_goes_beyond_max_depth() {
	node, path := fixtureDeepMappingNode(30)
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Nil(value)
}

func (s *MappingPathsTestSuite) Test_fails_to_get_value_by_path_for_invalid_path() {
	path := "$[\"cluster.v1\"].config.environments[0unexpected].hosts[0].missingField"
	node, _ := fixtureMappingNode1()
	_, err := GetPathValue(path, node, 10)
	s.Require().Error(err)
	mappingPathErr, isMappingPathErr := err.(*MappingPathError)
	s.Require().True(isMappingPathErr)
	s.Assert().Equal(ErrInvalidMappingPath, mappingPathErr.ReasonCode)
	s.Assert().Equal(path, mappingPathErr.Path)
}

func (s *MappingPathsTestSuite) Test_fails_to_get_value_by_path_for_invalid_path_2() {
	// Missing $ for the root object of the path.
	path := ".config[\"hosts\"][0]"
	node, _ := fixtureMappingNode1()
	_, err := GetPathValue(path, node, 10)
	s.Require().Error(err)
	mappingPathErr, isMappingPathErr := err.(*MappingPathError)
	s.Require().True(isMappingPathErr)
	s.Assert().Equal(ErrInvalidMappingPath, mappingPathErr.ReasonCode)
	s.Assert().Equal(path, mappingPathErr.Path)
}

func (s *MappingPathsTestSuite) Test_inject_value_for_map_field() {
	path := "$[\"cluster.v1\"].config.endpoint"
	node := fixtureInjectMappingNode1()
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValue(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(endpoint))
}

func (s *MappingPathsTestSuite) Test_inject_value_for_array_item() {
	path := "$[\"cluster.v1\"].config.environments[0]"
	node := fixtureInjectMappingNode1()
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValue(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(endpoint))
}

func (s *MappingPathsTestSuite) Test_inject_value_for_complex_path() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	node := fixtureInjectMappingNode1()
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValue(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(endpoint))
}

func (s *MappingPathsTestSuite) Test_inject_value_for_array_item_selector_path() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[@.endpoint = \"https://old.example.com\"].protocol"
	node := fixtureInjectMappingNode3()
	protocol := "http/1.1"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &protocol,
		},
	}
	err := InjectPathValue(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(protocol))
}

func (s *MappingPathsTestSuite) Test_inject_value_replace_for_array_item_selector_path() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[@.endpoint = \"https://other.example.com\"].protocol"
	node := fixtureInjectMappingNode3()
	protocol := "http/1.1"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &protocol,
		},
	}
	err := InjectPathValueReplaceFields(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(protocol))
}

func (s *MappingPathsTestSuite) Test_inject_value_replace_for_complex_path() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	node := fixtureInjectMappingNode2()
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValueReplaceFields(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(endpoint))
}

func (s *MappingPathsTestSuite) Test_reports_error_for_trying_to_inject_value_for_non_existent_path() {
	// Config is a map, not an array in the target node so we can't inject into it.
	path := "$[\"cluster.v1\"].config[0]"
	node := fixtureInjectMappingNode1()
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValue(path, value, node, 10)
	s.Assert().Error(err)
	s.Assert().Equal(
		"path \"$[\\\"cluster.v1\\\"].config[0]\" could not be injected into the mapping node, "+
			"the structure of the mapping node does not match the path",
		err.Error(),
	)
}

func (s *MappingPathsTestSuite) Test_reports_error_for_trying_to_inject_value_for_path_that_goes_beyond_max_depth() {
	node, path := fixtureDeepMappingNode(30)
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValue(path, value, node, 10)
	s.Assert().Error(err)
	s.Assert().Equal(
		"path \"$.field0.field1.field2.field3.field4.field5.field6."+
			"field7.field8.field9.field10.field11.field12.field13.field14."+
			"field15.field16.field17.field18.field19.field20.field21.field22."+
			"field23.field24.field25.field26.field27.field28.field29\" "+
			"could not be injected into the mapping node, "+
			"the path goes beyond the maximum depth of the node",
		err.Error(),
	)
}

func (s *MappingPathsTestSuite) Test_path_matches_pattern_for_exact_match() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	pattern := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	matches, err := PathMatchesPattern(path, pattern)
	s.Require().NoError(err)
	s.Assert().True(matches)
}

func (s *MappingPathsTestSuite) Test_path_matches_pattern_for_different_property_notation() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	pattern := "$[\"cluster.v1\"].config[\"environments\"][0].hosts[0][\"endpoint\"]"
	matches, err := PathMatchesPattern(path, pattern)
	s.Require().NoError(err)
	s.Assert().True(matches)
}

func (s *MappingPathsTestSuite) Test_path_matches_pattern_with_wildcards_1() {
	path := "$[\"cluster.v1\"].config.environments[0].hosts[0].endpoint"
	pattern := "$[\"cluster.v1\"].config.*[0].hosts[*].endpoint"
	matches, err := PathMatchesPattern(path, pattern)
	s.Require().NoError(err)
	s.Assert().True(matches)
}

func (s *MappingPathsTestSuite) Test_path_matches_pattern_with_wildcards_2() {
	path := "$.cluster.config[5].environments[2].hosts[0].endpoint"
	pattern := "$.cluster.*[*].*[2].hosts[*].endpoint"
	matches, err := PathMatchesPattern(path, pattern)
	s.Require().NoError(err)
	s.Assert().True(matches)
}

func fixtureMappingNode1() (*MappingNode, string) {
	endpoint := "https://sfg94832-api.example.com"
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"cluster.v1": {
				Fields: map[string]*MappingNode{
					"config": {
						Fields: map[string]*MappingNode{
							"environments": {
								Items: []*MappingNode{
									{
										Fields: map[string]*MappingNode{
											"hosts": {
												Items: []*MappingNode{
													{
														Fields: map[string]*MappingNode{
															"endpoint": {
																Scalar: &ScalarValue{
																	StringValue: &endpoint,
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, endpoint
}

func fixtureMappingNode2() (*MappingNode, string) {
	return fixtureInjectMappingNode3(), "http/2.0"
}

func fixtureInjectMappingNode1() *MappingNode {
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"cluster.v1": {
				Fields: map[string]*MappingNode{
					"config": {
						Fields: map[string]*MappingNode{
							"environments": {
								Items: []*MappingNode{},
							},
						},
					},
				},
			},
		},
	}
}

func fixtureInjectMappingNode2() *MappingNode {
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"cluster.v1": {
				Fields: map[string]*MappingNode{
					"config": {
						Fields: map[string]*MappingNode{
							"environments": {
								Items: []*MappingNode{
									{
										Fields: map[string]*MappingNode{
											"hosts": {
												Items: []*MappingNode{
													{
														Fields: map[string]*MappingNode{
															"endpoint": {
																Scalar: ScalarFromString("https://old-endpoint.example.com"),
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func fixtureInjectMappingNode3() *MappingNode {
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"cluster.v1": {
				Fields: map[string]*MappingNode{
					"config": {
						Fields: map[string]*MappingNode{
							"environments": {
								Items: []*MappingNode{
									{
										Fields: map[string]*MappingNode{
											"hosts": {
												Items: []*MappingNode{
													{
														Fields: map[string]*MappingNode{
															"endpoint": MappingNodeFromString("https://other.example.com"),
															"protocol": MappingNodeFromString("http/2.0"),
														},
													},
													{
														Fields: map[string]*MappingNode{
															"endpoint": MappingNodeFromString("https://old.example.com"),
															// Protocol will be injected here.
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func fixtureDeepMappingNode(depth int) (*MappingNode, string) {
	node := &MappingNode{}
	var path strings.Builder
	path.WriteString("$")
	current := node
	for i := range depth {
		fieldName := fmt.Sprintf("field%d", i)
		path.WriteString("." + fieldName)
		current.Fields = map[string]*MappingNode{
			fieldName: {
				Fields: map[string]*MappingNode{},
			},
		}
		current = current.Fields[fieldName]
	}

	return node, path.String()
}

func fixtureScalarArrayNode() *MappingNode {
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"managedPolicyArns": {
				Items: []*MappingNode{
					MappingNodeFromString("arn:aws:iam::aws:policy/ReadOnlyAccess"),
					MappingNodeFromString("arn:aws:iam::123456789012:policy/bluelink-link-access-1"),
				},
			},
		},
	}
}

func (s *MappingPathsTestSuite) Test_get_value_by_path_for_scalar_array_selector() {
	path := "$.managedPolicyArns[@ = \"arn:aws:iam::123456789012:policy/bluelink-link-access-1\"]"
	value, err := GetPathValue(path, fixtureScalarArrayNode(), 10)
	s.Require().NoError(err)
	s.Assert().Equal(
		MappingNodeFromString("arn:aws:iam::123456789012:policy/bluelink-link-access-1"),
		value,
	)
}

func (s *MappingPathsTestSuite) Test_get_value_returns_nil_for_scalar_array_selector_miss() {
	path := "$.managedPolicyArns[@ = \"arn:aws:iam::123456789012:policy/absent\"]"
	value, err := GetPathValue(path, fixtureScalarArrayNode(), 10)
	s.Require().NoError(err)
	s.Assert().Nil(value)
}

func (s *MappingPathsTestSuite) Test_inject_value_for_existing_scalar_array_selector() {
	arn := "arn:aws:iam::123456789012:policy/bluelink-link-access-1"
	path := fmt.Sprintf("$.managedPolicyArns[@ = %q]", arn)
	node := fixtureScalarArrayNode()
	err := InjectPathValue(path, MappingNodeFromString(arn), node, 10)
	s.Require().NoError(err)
	// The array length is unchanged; the matching item is still present.
	s.Assert().Len(node.Fields["managedPolicyArns"].Items, 2)
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(MappingNodeFromString(arn), value)
}

func (s *MappingPathsTestSuite) Test_inject_value_appends_missing_scalar_array_item() {
	arn := "arn:aws:iam::123456789012:policy/bluelink-link-access-2"
	path := fmt.Sprintf("$.managedPolicyArns[@ = %q]", arn)
	node := fixtureScalarArrayNode()
	err := InjectPathValue(path, MappingNodeFromString(arn), node, 10)
	s.Require().NoError(err)
	// The missing scalar item is appended so the path now resolves.
	s.Assert().Len(node.Fields["managedPolicyArns"].Items, 3)
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(MappingNodeFromString(arn), value)
}

func (s *MappingPathsTestSuite) Test_get_value_by_path_with_compound_array_selector() {
	// Two array items share the "function" attribute, so neither attribute alone is
	// unique; the compound selector targets the item by "function" AND "event".
	node := fixtureCompoundSelectorNode()
	path := "$.lambdaConfigurations[@.function = \"arn:fn\" && @.event = \"s3:ObjectRemoved:*\"].event"
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(MappingNodeFromString("s3:ObjectRemoved:*"), value)
}

func (s *MappingPathsTestSuite) Test_inject_value_with_compound_array_selector() {
	node := fixtureCompoundSelectorNode()
	// Replace the (function=arn:fn, event=ObjectCreated) item's filter via the compound key.
	path := "$.lambdaConfigurations[@.function = \"arn:fn\" && @.event = \"s3:ObjectCreated:*\"].filterPrefix"
	err := InjectPathValue(path, MappingNodeFromString("incoming/"), node, 10)
	s.Require().NoError(err)

	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(MappingNodeFromString("incoming/"), value)

	// The sibling item (same function, different event) is left untouched.
	other, err := GetPathValue(
		"$.lambdaConfigurations[@.function = \"arn:fn\" && @.event = \"s3:ObjectRemoved:*\"].filterPrefix",
		node,
		10,
	)
	s.Require().NoError(err)
	s.Assert().Nil(other)
}

func fixtureCompoundSelectorNode() *MappingNode {
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"lambdaConfigurations": {
				Items: []*MappingNode{
					{Fields: map[string]*MappingNode{
						"function": MappingNodeFromString("arn:fn"),
						"event":    MappingNodeFromString("s3:ObjectCreated:*"),
					}},
					{Fields: map[string]*MappingNode{
						"function": MappingNodeFromString("arn:fn"),
						"event":    MappingNodeFromString("s3:ObjectRemoved:*"),
					}},
				},
			},
		},
	}
}

func TestMappingPathsTestSuite(t *testing.T) {
	suite.Run(t, new(MappingPathsTestSuite))
}
