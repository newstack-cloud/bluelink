package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MappingPathsTestSuite struct {
	suite.Suite
}

func (s *MappingPathsTestSuite) Test_get_value_by_path_for_complex_path() {
	path := "$[\"cluster\\\".v1\"].config.environments[0].hosts[0].endpoint"
	node, expectedEndpoint := fixtureMappingNode1()
	value, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(&MappingNode{
		Scalar: &ScalarValue{
			StringValue: &expectedEndpoint,
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
	path := "$[\"cluster\\\".v1\"].config.environments[0].hosts[0].missingField"
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
	path := "$[\"cluster\\\".v1\"].config.environments[0unexpected].hosts[0].missingField"
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
	path := "$[\"cluster\\\".v1\"].config.endpoint"
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
	path := "$[\"cluster\\\".v1\"].config.environments[0]"
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
	path := "$[\"cluster\\\".v1\"].config.environments[0].hosts[0].endpoint"
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

func (s *MappingPathsTestSuite) Test_inject_value_replace_for_complex_path() {
	path := "$[\"cluster\\\".v1\"].config.environments[0].hosts[0].endpoint"
	node := fixtureInjectMappingNode2()
	endpoint := "https://sfg94831-api.example.com"
	value := &MappingNode{
		Scalar: &ScalarValue{
			StringValue: &endpoint,
		},
	}
	err := InjectPathValueReplace(path, value, node, 10)
	s.Require().NoError(err)
	injected, err := GetPathValue(path, node, 10)
	s.Require().NoError(err)
	s.Assert().Equal(injected, MappingNodeFromString(endpoint))
}

func (s *MappingPathsTestSuite) Test_reports_error_for_trying_to_inject_value_for_non_existent_path() {
	// Config is a map, not an array in the target node so we can't inject into it.
	path := "$[\"cluster\\\".v1\"].config[0]"
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
		"path \"$[\\\"cluster\\\\\\\".v1\\\"].config[0]\" could not be injected into the mapping node, "+
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

func fixtureMappingNode1() (*MappingNode, string) {
	endpoint := "https://sfg94832-api.example.com"
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"cluster\".v1": {
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

func fixtureInjectMappingNode1() *MappingNode {
	return &MappingNode{
		Fields: map[string]*MappingNode{
			"cluster\".v1": {
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
			"cluster\".v1": {
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

func fixtureDeepMappingNode(depth int) (*MappingNode, string) {
	node := &MappingNode{}
	path := "$"
	current := node
	for i := 0; i < depth; i++ {
		fieldName := fmt.Sprintf("field%d", i)
		path += "." + fieldName
		current.Fields = map[string]*MappingNode{
			fieldName: {
				Fields: map[string]*MappingNode{},
			},
		}
		current = current.Fields[fieldName]
	}

	return node, path
}

func TestMappingPathsTestSuite(t *testing.T) {
	suite.Run(t, new(MappingPathsTestSuite))
}
