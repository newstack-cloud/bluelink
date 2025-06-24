package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

func Test(t *testing.T) {
	TestingT(t)
}

type VariableTestSuite struct {
	specFixtures map[string][]byte
}

var _ = Suite(&VariableTestSuite{})

func (s *VariableTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string][]byte)
	fixturesToLoad := map[string]string{
		"passYAML":              "__testdata/variables/pass.yml",
		"serialiseExpectedYAML": "__testdata/variables/serialise-expected.yml",
		"passJSON":              "__testdata/variables/pass.json",
		"serialiseExpectedJSON": "__testdata/variables/serialise-expected.json",
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

func (s *VariableTestSuite) Test_parses_valid_variable_yaml_input(c *C) {
	targetVar := &Variable{}
	err := yaml.Unmarshal([]byte(s.specFixtures["passYAML"]), targetVar)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(*targetVar.Default.BoolValue, Equals, true)
	c.Assert(*targetVar.Description.StringValue, Equals, "This is an example boolean variable")
	c.Assert(*targetVar.Secret.BoolValue, Equals, false)
	c.Assert(targetVar.Type.Value, Equals, VariableType("boolean"))
	c.Assert(targetVar.SourceMeta, NotNil)
	c.Assert(targetVar.SourceMeta.Line, Equals, 1)
	c.Assert(targetVar.SourceMeta.Column, Equals, 1)
}

func (s *VariableTestSuite) Test_serialise_valid_variable_yaml_input(c *C) {
	expected := &Variable{}
	err := yaml.Unmarshal([]byte(s.specFixtures["serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	region := "eu-west-2"
	secret := false
	description := "The AWS region to connect to AWS services with"
	serialisedBytes, err := yaml.Marshal(&Variable{
		Type:        &VariableTypeWrapper{Value: VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		Secret:      &core.ScalarValue{BoolValue: &secret},
		Default: &core.ScalarValue{
			StringValue: &region,
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetVar := &Variable{}
	err = yaml.Unmarshal(serialisedBytes, targetVar)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetVar.Type.Value, Equals, expected.Type.Value)
	c.Assert(*targetVar.Description.StringValue, Equals, *expected.Description.StringValue)
	c.Assert(*targetVar.Secret.BoolValue, Equals, *expected.Secret.BoolValue)
	c.Assert(*targetVar.Default.StringValue, Equals, *expected.Default.StringValue)
}

func (s *VariableTestSuite) Test_parses_valid_variable_json_input(c *C) {
	targetVar := &Variable{}
	err := json.Unmarshal([]byte(s.specFixtures["passJSON"]), targetVar)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(*targetVar.Default.IntValue, Equals, 3423)
	c.Assert(*targetVar.Description.StringValue, Equals, "This is an example integer variable")
	c.Assert(*targetVar.Secret.BoolValue, Equals, false)
	c.Assert(targetVar.Type.Value, Equals, VariableType("integer"))
}

func (s *VariableTestSuite) Test_serialise_valid_variable_json_input(c *C) {
	expected := &Variable{}
	err := json.Unmarshal([]byte(s.specFixtures["serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	region := "eu-west-1"
	description := "The AWS region to connect to AWS services with"
	secret := true
	serialisedBytes, err := json.Marshal(&Variable{
		Type:        &VariableTypeWrapper{Value: VariableTypeString},
		Description: &core.ScalarValue{StringValue: &description},
		Secret:      &core.ScalarValue{BoolValue: &secret},
		Default: &core.ScalarValue{
			StringValue: &region,
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetVar := &Variable{}
	err = json.Unmarshal(serialisedBytes, targetVar)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetVar.Type.Value, Equals, expected.Type.Value)
	c.Assert(*targetVar.Description.StringValue, Equals, *expected.Description.StringValue)
	c.Assert(*targetVar.Secret.BoolValue, Equals, *expected.Secret.BoolValue)
	c.Assert(*targetVar.Default.StringValue, Equals, *expected.Default.StringValue)
}
