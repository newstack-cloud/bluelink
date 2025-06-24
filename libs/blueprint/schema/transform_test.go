package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

type TransformTestSuite struct {
	specFixtures map[string][]byte
}

var _ = Suite(&TransformTestSuite{})

func (s *TransformTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string][]byte)
	fixturesToLoad := map[string]string{
		"passStringYAML":                             "__testdata/transform/pass-string.yml",
		"passStringListYAML":                         "__testdata/transform/pass-string-list.yml",
		"failUnsupportedTransformTypeYAML":           "__testdata/transform/fail-unsupported-transform-type.yml",
		"failUnsupportedTransformListValueTypeYAML":  "__testdata/transform/fail-unsupported-transform-list-value-type.yml",
		"serialiseExpectedYAML":                      "__testdata/transform/serialise-expected.yml",
		"passStringJSON":                             "__testdata/transform/pass-string.json",
		"passStringListJSON":                         "__testdata/transform/pass-string-list.json",
		"failUnsupportedTransformTypeJSON":           "__testdata/transform/fail-unsupported-transform-type.json",
		"faileUnsupportedTransformListValueTypeJSON": "__testdata/transform/fail-unsupported-transform-list-value-type.json",
		"serialiseExpectedJSON":                      "__testdata/transform/serialise-expected.json",
	}

	for name, filePath := range fixturesToLoad {
		specBytes, err := os.ReadFile(filePath)
		if err != nil {
			c.Error(err)
			c.FailNow()
		}
		s.specFixtures[name] = specBytes
	}
}

func (s *TransformTestSuite) Test_parses_valid_string_transform_yaml_input(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := yaml.Unmarshal([]byte(s.specFixtures["passStringYAML"]), targetTransform)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetTransform.Values, DeepEquals, []string{"celerity-2022-01-22"})
	c.Assert(targetTransform.SourceMeta, HasLen, 1)
	c.Assert(targetTransform.SourceMeta[0].Line, Equals, 1)
	c.Assert(targetTransform.SourceMeta[0].Column, Equals, 1)
}

func (s *TransformTestSuite) Test_parses_valid_string_list_transform_yaml_input(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := yaml.Unmarshal([]byte(s.specFixtures["passStringListYAML"]), targetTransform)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetTransform.Values, DeepEquals, []string{
		"celerity-2022-01-22",
		"custom-transform-2",
		"custom-transform-3",
	})
	c.Assert(targetTransform.SourceMeta, DeepEquals, []*source.Meta{
		{
			Position:    source.Position{Line: 1, Column: 3},
			EndPosition: &source.Position{Line: 1, Column: 22},
		},
		{
			Position:    source.Position{Line: 2, Column: 3},
			EndPosition: &source.Position{Line: 2, Column: 21},
		},
		{
			Position:    source.Position{Line: 3, Column: 3},
			EndPosition: &source.Position{Line: 3, Column: 21},
		},
	})
}

func (s *TransformTestSuite) Test_fails_to_parse_yaml_due_to_unsupported_transform_type(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := yaml.Unmarshal([]byte(s.specFixtures["failUnsupportedTransformTypeYAML"]), targetTransform)
	if err == nil {
		c.Error(errors.New("expected to fail deserialisation due to unsupported transform type"))
		c.FailNow()
	}

	schemaError, isSchemaError := err.(*Error)
	c.Assert(isSchemaError, Equals, true)
	c.Assert(schemaError.ReasonCode, Equals, ErrorSchemaReasonCodeInvalidTransformType)
	c.Assert(*schemaError.SourceLine, Equals, 1)
	c.Assert(*schemaError.SourceColumn, Equals, 1)
}

func (s *TransformTestSuite) Test_fails_to_parse_yaml_due_to_unsupported_transform_list_value_type(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := yaml.Unmarshal([]byte(s.specFixtures["failUnsupportedTransformListValueTypeYAML"]), targetTransform)
	if err == nil {
		c.Error(errors.New("expected to fail deserialisation due to unsupported transform list value type"))
		c.FailNow()
	}

	schemaError, isSchemaError := err.(*Error)
	c.Assert(isSchemaError, Equals, true)
	c.Assert(schemaError.ReasonCode, Equals, ErrorSchemaReasonCodeInvalidTransformType)
	c.Assert(*schemaError.SourceLine, Equals, 1)
	c.Assert(*schemaError.SourceColumn, Equals, 3)
}

func (s *TransformTestSuite) Test_serialise_valid_transform_yaml_input(c *C) {
	expected := &TransformValueWrapper{}
	err := yaml.Unmarshal([]byte(s.specFixtures["serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	serialisedBytes, err := yaml.Marshal(&TransformValueWrapper{
		StringList: StringList{
			Values: []string{
				"celerity-2022-01-22",
				"custom-transform-2023-01-01",
				"custom-transform-2023-02-21",
			},
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetTransform := &TransformValueWrapper{}
	err = yaml.Unmarshal(serialisedBytes, targetTransform)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetTransform.Values, DeepEquals, expected.Values)
}

func (s *TransformTestSuite) Test_parses_valid_string_transform_field_json_input(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := json.Unmarshal([]byte(s.specFixtures["passStringJSON"]), targetTransform)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetTransform.Values, DeepEquals, []string{"celerity-2022-01-22"})
}

func (s *TransformTestSuite) Test_parses_valid_string_list_value_transform_field_json_input(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := json.Unmarshal([]byte(s.specFixtures["passStringListJSON"]), targetTransform)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetTransform.Values, DeepEquals, []string{
		"celerity-2022-01-22",
		"custom-transform-2",
		"custom-transform-3",
	})
}

func (s *TransformTestSuite) Test_fails_to_parse_json_due_to_unsupported_string_transform_type(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := json.Unmarshal([]byte(s.specFixtures["failUnsupportedTransformTypeJSON"]), targetTransform)
	if err == nil {
		c.Error(errors.New("expected to fail deserialisation due to unsupported data source field type"))
		c.FailNow()
	}

	schemaError, isSchemaError := err.(*Error)
	c.Assert(isSchemaError, Equals, true)
	c.Assert(schemaError.ReasonCode, Equals, ErrorSchemaReasonCodeInvalidTransformType)
}

func (s *TransformTestSuite) Test_fails_to_parse_json_due_to_unsupported_list_transform_value_type(c *C) {
	targetTransform := &TransformValueWrapper{}
	err := json.Unmarshal([]byte(s.specFixtures["faileUnsupportedTransformListValueTypeJSON"]), targetTransform)
	if err == nil {
		c.Error(errors.New("expected to fail deserialisation due to unsupported data source field type"))
		c.FailNow()
	}

	schemaError, isSchemaError := err.(*Error)
	c.Assert(isSchemaError, Equals, true)
	c.Assert(schemaError.ReasonCode, Equals, ErrorSchemaReasonCodeInvalidTransformType)
}

func (s *TransformTestSuite) Test_serialise_valid_transform_json_input(c *C) {
	expected := &TransformValueWrapper{}
	err := json.Unmarshal([]byte(s.specFixtures["serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	serialisedBytes, err := json.Marshal(&TransformValueWrapper{
		StringList: StringList{
			Values: []string{
				"celerity-2022-01-22",
				"custom-transform-2023-01-01",
				"custom-transform-2023-02-21",
			},
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetTransform := &TransformValueWrapper{}
	err = json.Unmarshal(serialisedBytes, targetTransform)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetTransform.Values, DeepEquals, expected.Values)
}
