package core

import (
	"encoding/json"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

var (
	testStringValue   = "Test string value"
	testStringValPart = "Test string value for "
)

type MappingNodeTestSuite struct {
	specParseFixtures     map[string][]byte
	specSerialiseFixtures map[string]*MappingNode
	suite.Suite
}

func (s *MappingNodeTestSuite) SetupTest() {
	s.prepareParseInputFixtures()
	s.prepareExpectedFixtures()
}

func (s *MappingNodeTestSuite) Test_parse_string_val_yaml() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["stringValYAML"], targetMappingNode)
	s.Assert().NoError(err)
	s.Assert().NotNil(targetMappingNode.Scalar.StringValue)
	s.Assert().Equal(testStringValue, *targetMappingNode.Scalar.StringValue)
	s.Assert().Equal(1, targetMappingNode.Scalar.SourceMeta.Line)
	s.Assert().Equal(1, targetMappingNode.Scalar.SourceMeta.Column)
}

func (s *MappingNodeTestSuite) Test_parse_string_val_json() {
	targetMappingNode := &MappingNode{}
	err := json.Unmarshal(s.specParseFixtures["stringValJSON"], targetMappingNode)
	s.Assert().NoError(err)
	s.Assert().NotNil(targetMappingNode.Scalar.StringValue)
	s.Assert().Equal(testStringValue, *targetMappingNode.Scalar.StringValue)
}

func (s *MappingNodeTestSuite) Test_parse_string_with_subs_yaml() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["stringWithSubsYAML"], targetMappingNode)
	s.Assert().NoError(err)
	s.Assert().Equal(&MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &testStringValPart,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 1, Column: 1},
						EndPosition: &source.Position{Line: 1, Column: 23},
					},
				},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "environment",
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 1, Column: 25},
								EndPosition: &source.Position{Line: 1, Column: 46},
							},
						},
						SourceMeta: &source.Meta{
							Position:    source.Position{Line: 1, Column: 25},
							EndPosition: &source.Position{Line: 1, Column: 46},
						},
					},
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 1, Column: 23},
						EndPosition: &source.Position{Line: 1, Column: 47},
					},
				},
			},
			SourceMeta: &source.Meta{
				Position:    source.Position{Line: 1, Column: 1},
				EndPosition: &source.Position{Line: 1, Column: 47},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{Line: 1, Column: 1},
		},
	}, targetMappingNode)
}

func (s *MappingNodeTestSuite) Test_parse_string_with_subs_json() {
	targetMappingNode := &MappingNode{}
	err := json.Unmarshal(s.specParseFixtures["stringWithSubsJSON"], targetMappingNode)
	s.Assert().NoError(err)
	s.Assert().Equal(&MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &testStringValPart},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "environment",
						},
					},
				},
			},
		},
	}, targetMappingNode)
}

func (s *MappingNodeTestSuite) Test_parse_int_val() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["intVal"], targetMappingNode)
	s.Assert().NoError(err)
	s.Assert().NotNil(targetMappingNode.Scalar.IntValue)
	s.Assert().Equal(45172131, *targetMappingNode.Scalar.IntValue)
}

func (s *MappingNodeTestSuite) Test_parse_fields_val_yaml() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["fieldsValYAML"], targetMappingNode)
	s.Assert().NoError(err)
	s.assertFieldsNodeYAML(targetMappingNode)
}

func (s *MappingNodeTestSuite) Test_parse_fields_val_json() {
	targetMappingNode := &MappingNode{}
	err := json.Unmarshal(s.specParseFixtures["fieldsValJSON"], targetMappingNode)
	s.Assert().NoError(err)
	s.assertFieldsNodeJSON(targetMappingNode)
}

func (s *MappingNodeTestSuite) assertFieldsNodeYAML(actual *MappingNode) {
	expectedIntVal := 45172131
	expectedStrVal := "value1"
	expectedStrSubPrefix := "value with sub "
	s.Assert().Equal(&MappingNode{
		Fields: map[string]*MappingNode{
			"key1": {
				Scalar: &ScalarValue{
					StringValue: &expectedStrVal,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 2, Column: 15},
						EndPosition: &source.Position{Line: 2, Column: 23},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 2, Column: 15}},
			},
			"key2": {
				StringWithSubstitutions: &substitutions.StringOrSubstitutions{
					Values: []*substitutions.StringOrSubstitution{
						{
							StringValue: &expectedStrSubPrefix,
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 3, Column: 15},
								EndPosition: &source.Position{Line: 3, Column: 30},
							},
						},
						{
							SubstitutionValue: &substitutions.Substitution{
								Variable: &substitutions.SubstitutionVariable{
									VariableName: "environment",
									SourceMeta: &source.Meta{
										Position:    source.Position{Line: 3, Column: 33},
										EndPosition: &source.Position{Line: 3, Column: 54},
									},
								},
								SourceMeta: &source.Meta{
									Position:    source.Position{Line: 3, Column: 33},
									EndPosition: &source.Position{Line: 3, Column: 54},
								},
							},
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 3, Column: 30},
								EndPosition: &source.Position{Line: 3, Column: 54},
							},
						},
					},
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 3, Column: 15},
						EndPosition: &source.Position{Line: 3, Column: 56},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 3, Column: 15}},
			},
			"key3": {
				Scalar: &ScalarValue{
					IntValue: &expectedIntVal,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 4, Column: 15},
						EndPosition: &source.Position{Line: 4, Column: 23},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 4, Column: 15}},
			},
		},
		SourceMeta: &source.Meta{Position: source.Position{Line: 2, Column: 9}},
		FieldsSourceMeta: map[string]*source.Meta{
			"key1": {Position: source.Position{Line: 2, Column: 9}},
			"key2": {Position: source.Position{Line: 3, Column: 9}},
			"key3": {Position: source.Position{Line: 4, Column: 9}},
		},
	}, actual)
}

func (s *MappingNodeTestSuite) assertFieldsNodeJSON(actual *MappingNode) {
	expectedIntVal := 45172131
	expectedStrVal := "value1"
	expectedStrSubPrefix := "value with sub "
	s.Assert().Equal(&MappingNode{
		Fields: map[string]*MappingNode{
			"key1": {
				Scalar: &ScalarValue{StringValue: &expectedStrVal},
			},
			"key2": {
				StringWithSubstitutions: &substitutions.StringOrSubstitutions{
					Values: []*substitutions.StringOrSubstitution{
						{StringValue: &expectedStrSubPrefix},
						{
							SubstitutionValue: &substitutions.Substitution{
								Variable: &substitutions.SubstitutionVariable{
									VariableName: "environment",
								},
							},
						},
					},
				},
			},
			"key3": {
				Scalar: &ScalarValue{IntValue: &expectedIntVal},
			},
		},
	}, actual)
}

func (s *MappingNodeTestSuite) Test_parse_items_val_yaml() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["itemsValYAML"], targetMappingNode)
	s.Assert().NoError(err)
	s.assertItemsNodeYAML(targetMappingNode)
}

func (s *MappingNodeTestSuite) Test_parse_items_val_json() {
	targetMappingNode := &MappingNode{}
	err := json.Unmarshal(s.specParseFixtures["itemsValJSON"], targetMappingNode)
	s.Assert().NoError(err)
	s.assertItemsNodeJSON(targetMappingNode)
}

func (s *MappingNodeTestSuite) assertItemsNodeYAML(actual *MappingNode) {
	expectedIntVal := 45172131
	expectedStrVal := "value1"
	expectedStrSubPrefix := "value with sub "
	s.Assert().Equal(&MappingNode{
		Items: []*MappingNode{
			{
				Scalar: &ScalarValue{
					StringValue: &expectedStrVal,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 2, Column: 11},
						EndPosition: &source.Position{Line: 2, Column: 19},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 2, Column: 11}},
			},
			{
				StringWithSubstitutions: &substitutions.StringOrSubstitutions{
					Values: []*substitutions.StringOrSubstitution{
						{
							StringValue: &expectedStrSubPrefix,
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 3, Column: 11},
								EndPosition: &source.Position{Line: 3, Column: 26},
							},
						},
						{
							SubstitutionValue: &substitutions.Substitution{
								Variable: &substitutions.SubstitutionVariable{
									VariableName: "environment",
									SourceMeta: &source.Meta{
										Position:    source.Position{Line: 3, Column: 29},
										EndPosition: &source.Position{Line: 3, Column: 50},
									},
								},
								SourceMeta: &source.Meta{
									Position:    source.Position{Line: 3, Column: 29},
									EndPosition: &source.Position{Line: 3, Column: 50},
								},
							},
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 3, Column: 26},
								EndPosition: &source.Position{Line: 3, Column: 50},
							},
						},
					},
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 3, Column: 11},
						EndPosition: &source.Position{Line: 3, Column: 52},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 3, Column: 11}},
			},
			{
				Scalar: &ScalarValue{
					IntValue: &expectedIntVal,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 4, Column: 11},
						EndPosition: &source.Position{Line: 4, Column: 19},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 4, Column: 11}},
			},
		},
		SourceMeta: &source.Meta{Position: source.Position{Line: 2, Column: 9}},
	}, actual)
}

func (s *MappingNodeTestSuite) assertItemsNodeJSON(actual *MappingNode) {
	expectedIntVal := 45172131
	expectedStrVal := "value1"
	expectedStrSubPrefix := "value with sub "
	s.Assert().Equal(&MappingNode{
		Items: []*MappingNode{
			{
				Scalar: &ScalarValue{
					StringValue: &expectedStrVal,
				},
			},
			{
				StringWithSubstitutions: &substitutions.StringOrSubstitutions{
					Values: []*substitutions.StringOrSubstitution{
						{StringValue: &expectedStrSubPrefix},
						{
							SubstitutionValue: &substitutions.Substitution{
								Variable: &substitutions.SubstitutionVariable{
									VariableName: "environment",
								},
							},
						},
					},
				},
			},
			{
				Scalar: &ScalarValue{
					IntValue: &expectedIntVal,
				},
			},
		},
	}, actual)
}

func (s *MappingNodeTestSuite) Test_parse_nested_val_yaml() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["nestedValYAML"], targetMappingNode)
	s.Assert().NoError(err)
	s.assertNestedNodeYAML(targetMappingNode)
}

func (s *MappingNodeTestSuite) Test_parse_nested_val_json() {
	targetMappingNode := &MappingNode{}
	err := json.Unmarshal(s.specParseFixtures["nestedValJSON"], targetMappingNode)
	s.Assert().NoError(err)
	s.assertNestedNodeJSON(targetMappingNode)
}

func (s *MappingNodeTestSuite) assertNestedNodeJSON(actual *MappingNode) {
	expectedIntVal := 931721304
	expectedStrVal1 := "value10"
	expectedStrVal2 := "value11"
	expectedStrVal3 := "value12"
	expectedStrSubPrefix := "value13 with sub "
	s.Assert().Equal(&MappingNode{
		Fields: map[string]*MappingNode{
			"key1": {
				Scalar: &ScalarValue{StringValue: &expectedStrVal1},
			},
			"key2": {
				Fields: map[string]*MappingNode{
					"key3": {
						Scalar: &ScalarValue{StringValue: &expectedStrVal2},
					},
				},
			},
			"key4": {
				Items: []*MappingNode{
					{
						Scalar: &ScalarValue{StringValue: &expectedStrVal3},
					},
					{
						StringWithSubstitutions: &substitutions.StringOrSubstitutions{
							Values: []*substitutions.StringOrSubstitution{
								{StringValue: &expectedStrSubPrefix},
								{
									SubstitutionValue: &substitutions.Substitution{
										Variable: &substitutions.SubstitutionVariable{
											VariableName: "environment",
										},
									},
								},
							},
						},
					},
				},
			},
			"key5": {
				Scalar: &ScalarValue{IntValue: &expectedIntVal},
			},
		},
	}, actual)
}

func (s *MappingNodeTestSuite) assertNestedNodeYAML(actual *MappingNode) {
	expectedIntVal := 931721304
	expectedStrVal1 := "value10"
	expectedStrVal2 := "value11"
	expectedStrVal3 := "value12"
	expectedStrSubPrefix := "value13 with sub "
	s.Assert().Equal(&MappingNode{
		Fields: map[string]*MappingNode{
			"key1": {
				Scalar: &ScalarValue{
					StringValue: &expectedStrVal1,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 2, Column: 17},
						EndPosition: &source.Position{Line: 2, Column: 26},
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{Line: 2, Column: 17},
				},
			},
			"key2": {
				Fields: map[string]*MappingNode{
					"key3": {
						Scalar: &ScalarValue{
							StringValue: &expectedStrVal2,
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 4, Column: 19},
								EndPosition: &source.Position{Line: 4, Column: 28},
							},
						},
						SourceMeta: &source.Meta{Position: source.Position{Line: 4, Column: 19}},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 4, Column: 13}},
				FieldsSourceMeta: map[string]*source.Meta{
					"key3": {Position: source.Position{Line: 4, Column: 13}},
				},
			},
			"key4": {
				Items: []*MappingNode{
					{
						Scalar: &ScalarValue{
							StringValue: &expectedStrVal3,
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 6, Column: 14},
								EndPosition: &source.Position{Line: 6, Column: 23},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{Line: 6, Column: 14},
						},
					},
					{
						StringWithSubstitutions: &substitutions.StringOrSubstitutions{
							Values: []*substitutions.StringOrSubstitution{
								{
									StringValue: &expectedStrSubPrefix,
									SourceMeta: &source.Meta{
										Position:    source.Position{Line: 7, Column: 14},
										EndPosition: &source.Position{Line: 7, Column: 31},
									},
								},
								{
									SubstitutionValue: &substitutions.Substitution{
										Variable: &substitutions.SubstitutionVariable{
											VariableName: "environment",
											SourceMeta: &source.Meta{
												Position:    source.Position{Line: 7, Column: 34},
												EndPosition: &source.Position{Line: 7, Column: 55},
											},
										},
										SourceMeta: &source.Meta{
											Position:    source.Position{Line: 7, Column: 34},
											EndPosition: &source.Position{Line: 7, Column: 55},
										},
									},
									SourceMeta: &source.Meta{
										Position:    source.Position{Line: 7, Column: 31},
										EndPosition: &source.Position{Line: 7, Column: 55},
									},
								},
							},
							SourceMeta: &source.Meta{
								Position:    source.Position{Line: 7, Column: 14},
								EndPosition: &source.Position{Line: 7, Column: 57},
							},
						},
						SourceMeta: &source.Meta{Position: source.Position{Line: 7, Column: 14}},
					},
				},
				SourceMeta: &source.Meta{Position: source.Position{Line: 6, Column: 12}},
			},
			"key5": {
				Scalar: &ScalarValue{
					IntValue: &expectedIntVal,
					SourceMeta: &source.Meta{
						Position:    source.Position{Line: 8, Column: 17},
						EndPosition: &source.Position{Line: 8, Column: 26},
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{Line: 8, Column: 17},
				},
			},
		},
		SourceMeta: &source.Meta{Position: source.Position{Line: 2, Column: 11}},
		FieldsSourceMeta: map[string]*source.Meta{
			"key1": {Position: source.Position{Line: 2, Column: 11}},
			"key2": {Position: source.Position{Line: 3, Column: 11}},
			"key4": {Position: source.Position{Line: 5, Column: 11}},
			"key5": {Position: source.Position{Line: 8, Column: 11}},
		},
	}, actual)
}

func (s *MappingNodeTestSuite) Test_fails_to_parse_invalid_value() {
	targetMappingNode := &MappingNode{}
	err := yaml.Unmarshal(s.specParseFixtures["failInvalidValue"], targetMappingNode)
	s.Assert().Error(err)
	s.Assert().Equal("a blueprint mapping node must be a valid scalar, mapping or sequence", err.Error())
	coreErr, isCoreErr := err.(*Error)
	s.Assert().Equal(true, isCoreErr)
	s.Assert().Equal(ErrorCoreReasonCodeInvalidMappingNode, coreErr.ReasonCode)
	s.Assert().Equal(3, *coreErr.SourceLine)
	s.Assert().Equal(11, *coreErr.SourceColumn)
}

func (s *MappingNodeTestSuite) Test_serialise_string_val_yaml() {
	actual, err := yaml.Marshal(s.specSerialiseFixtures["stringValYAML"])
	s.Assert().NoError(err)
	s.Assert().Equal("Test string value\n", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_string_val_json() {
	actual, err := json.Marshal(s.specSerialiseFixtures["stringValJSON"])
	s.Assert().NoError(err)
	s.Assert().Equal("\"Test string value\"", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_string_with_subs_yaml() {
	actual, err := yaml.Marshal(s.specSerialiseFixtures["stringWithSubsYAML"])
	s.Assert().NoError(err)
	s.Assert().Equal("Test string value for ${variables.environment}\n", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_string_with_subs_json() {
	actual, err := json.Marshal(s.specSerialiseFixtures["stringWithSubsJSON"])
	s.Assert().NoError(err)
	s.Assert().Equal("\"Test string value for ${variables.environment}\"", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_int_val_yaml() {
	actual, err := yaml.Marshal(s.specSerialiseFixtures["intVal"])
	s.Assert().NoError(err)
	s.Assert().Equal("45172131\n", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_int_val_json() {
	actual, err := json.Marshal(s.specSerialiseFixtures["intVal"])
	s.Assert().NoError(err)
	s.Assert().Equal("45172131", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_fields_val_yaml() {
	actual, err := yaml.Marshal(s.specSerialiseFixtures["fieldsValYAML"])
	s.Assert().NoError(err)
	s.Assert().Equal("key1: Test string value\nkey2: Test string value for ${variables.environment}\nkey3: 45172131\n", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_fields_val_json() {
	actual, err := json.Marshal(s.specSerialiseFixtures["fieldsValJSON"])
	s.Assert().NoError(err)
	s.Assert().Equal("{\"key1\":\"Test string value\",\"key2\":\"Test string value for ${variables.environment}\",\"key3\":45172131}", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_items_val_yaml() {
	actual, err := yaml.Marshal(s.specSerialiseFixtures["itemsValYAML"])
	s.Assert().NoError(err)
	s.Assert().Equal("- Test string value\n- Test string value for ${variables.environment}\n- 45172131\n", string(actual))
}

func (s *MappingNodeTestSuite) Test_serialise_items_val_json() {
	actual, err := json.Marshal(s.specSerialiseFixtures["itemsValJSON"])
	s.Assert().NoError(err)
	s.Assert().Equal("[\"Test string value\",\"Test string value for ${variables.environment}\",45172131]", string(actual))
}

func (s *MappingNodeTestSuite) Test_fails_to_serialise_invalid_mapping_node_yaml() {
	_, err := yaml.Marshal(s.specSerialiseFixtures["failInvalidYAML"])
	s.Assert().Error(err)
	s.Assert().Equal("a blueprint mapping node must have a valid value set", err.Error())
}

func (s *MappingNodeTestSuite) Test_fails_to_serialise_invalid_mapping_node_json() {
	_, err := json.Marshal(s.specSerialiseFixtures["failInvalidJSON"])
	s.Assert().Error(err)
	s.Assert().Equal("json: error calling MarshalJSON for type *core.MappingNode: a blueprint mapping node must have a valid value set", err.Error())
}

func (s *MappingNodeTestSuite) prepareParseInputFixtures() {
	s.specParseFixtures = map[string][]byte{
		"stringValYAML":      []byte(testStringValue),
		"stringValJSON":      []byte("\"Test string value\""),
		"stringWithSubsYAML": []byte("Test string value for ${variables.environment}"),
		"stringWithSubsJSON": []byte("\"Test string value for ${variables.environment}\""),
		// We only need one example of a non-string scalar value as scalar value
		// parsing is tested in the ScalarValue test suite.
		"intVal": []byte("45172131"),
		"fieldsValYAML": []byte(`
        key1: "value1"
        key2: "value with sub ${variables.environment}"
        key3: 45172131`),
		"fieldsValJSON": []byte(`{
			"key1": "value1",
			"key2": "value with sub ${variables.environment}",
			"key3": 45172131
			}`),
		"itemsValYAML": []byte(`
        - "value1"
        - "value with sub ${variables.environment}"
        - 45172131`),
		"itemsValJSON": []byte(`[
			"value1",
			"value with sub ${variables.environment}",
			45172131
			]`),
		"nestedValYAML": []byte(`
          key1: "value10"
          key2:
            key3: "value11"
          key4:
           - "value12"
           - "value13 with sub ${variables.environment}"
          key5: 931721304`),
		"nestedValJSON": []byte(`{
			"key1": "value10",
			"key2": {
				"key3": "value11"
			},
			"key4": [
				"value12",
				"value13 with sub ${variables.environment}"
			],
			"key5": 931721304
			}`),
		// YAML anchors and aliases do not represent valid values for a mapping node.
		"failInvalidValue": []byte(`
        - &unsupportedAlias
        - *unsupportedAlias`),
	}
}

func (s *MappingNodeTestSuite) prepareExpectedFixtures() {
	expectedStrVal := testStringValue
	expectedSubStrVal := "Test string value for "
	expectedMappingNodeWithSubs := &MappingNode{
		StringWithSubstitutions: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{StringValue: &expectedSubStrVal},
				{
					SubstitutionValue: &substitutions.Substitution{
						Variable: &substitutions.SubstitutionVariable{
							VariableName: "environment",
						},
					},
				},
			},
		},
	}
	expectedIntVal := 45172131
	expectedFields := map[string]*MappingNode{
		"key1": {
			Scalar: &ScalarValue{StringValue: &expectedStrVal},
		},
		"key2": {
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{StringValue: &expectedSubStrVal},
					{
						SubstitutionValue: &substitutions.Substitution{
							Variable: &substitutions.SubstitutionVariable{
								VariableName: "environment",
							},
						},
					},
				},
			},
		},
		"key3": {
			Scalar: &ScalarValue{IntValue: &expectedIntVal},
		},
	}
	expectedItems := []*MappingNode{
		{
			Scalar: &ScalarValue{StringValue: &expectedStrVal},
		},
		{
			StringWithSubstitutions: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{StringValue: &expectedSubStrVal},
					{
						SubstitutionValue: &substitutions.Substitution{
							Variable: &substitutions.SubstitutionVariable{
								VariableName: "environment",
							},
						},
					},
				},
			},
		},
		{
			Scalar: &ScalarValue{IntValue: &expectedIntVal},
		},
	}

	s.specSerialiseFixtures = map[string]*MappingNode{
		"stringValYAML": {
			Scalar: &ScalarValue{StringValue: &expectedStrVal},
		},
		"stringValJSON": {
			Scalar: &ScalarValue{StringValue: &expectedStrVal},
		},
		"stringWithSubsYAML": expectedMappingNodeWithSubs,
		"stringWithSubsJSON": expectedMappingNodeWithSubs,
		"intVal": {
			Scalar: &ScalarValue{IntValue: &expectedIntVal},
		},
		"fieldsValYAML": {
			Fields: expectedFields,
		},
		"fieldsValJSON": {
			Fields: expectedFields,
		},
		"itemsValYAML": {
			Items: expectedItems,
		},
		"itemsValJSON": {
			Items: expectedItems,
		},
		// Empty mapping node is invalid.
		"failInvalidYAML": {},
		"failInvalidJSON": {},
	}
}

func TestMappingNodeTestSuite(t *testing.T) {
	suite.Run(t, new(MappingNodeTestSuite))
}
