package schema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/two-hundred/celerity/libs/blueprint/core"
	"github.com/two-hundred/celerity/libs/blueprint/source"
	"github.com/two-hundred/celerity/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

type ValueTestSuite struct {
	specFixtures map[string][]byte
}

var _ = Suite(&ValueTestSuite{})

func (s *ValueTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string][]byte)
	fixturesToLoad := map[string]string{
		"passYAML":                     "__testdata/values/pass.yml",
		"serialiseExpectedYAML":        "__testdata/values/serialise-expected.yml",
		"failUnsupportedValueTypeYAML": "__testdata/values/fail-unsupported-value-type.yml",
		"passJSON":                     "__testdata/values/pass.json",
		"serialiseExpectedJSON":        "__testdata/values/serialise-expected.json",
		"failUnsupportedValueTypeJSON": "__testdata/values/fail-unsupported-value-type.json",
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

func (s *ValueTestSuite) Test_parses_valid_value_yaml_input(c *C) {
	targetVal := &Value{}
	err := yaml.Unmarshal([]byte(s.specFixtures["passYAML"]), targetVal)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	descriptionStrVal := "This is an example boolean value"
	c.Assert(targetVal.Description, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &descriptionStrVal,
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   3,
						Column: 14,
					},
					EndPosition: &source.Position{
						Line:   3,
						Column: 46,
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   3,
				Column: 14,
			},
			EndPosition: &source.Position{
				Line:   3,
				Column: 46,
			},
		},
	})
	c.Assert(*targetVal.Secret.BoolValue, Equals, false)
	c.Assert(targetVal.Type, DeepEquals, &ValueTypeWrapper{
		Value: ValueTypeBoolean,
		SourceMeta: &source.Meta{
			Position:    source.Position{Line: 1, Column: 7},
			EndPosition: &source.Position{Line: 1, Column: 14},
		},
	})
	c.Assert(targetVal.Value, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					ResourceProperty: &substitutions.SubstitutionResourceProperty{
						ResourceName: "exampleResource",
						Path: []*substitutions.SubstitutionPathItem{
							{
								FieldName: "state",
							},
							{
								FieldName: "enabled",
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   2,
								Column: 10,
							},
							EndPosition: &source.Position{
								Line:   2,
								Column: 49,
							},
						},
					},
					SourceMeta: &source.Meta{
						Position: source.Position{
							Line:   2,
							Column: 10,
						},
						EndPosition: &source.Position{
							Line:   2,
							Column: 49,
						},
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   2,
						Column: 8,
					},
					EndPosition: &source.Position{
						Line:   2,
						Column: 50,
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   2,
				Column: 8,
			},
			EndPosition: &source.Position{
				Line:   2,
				Column: 50,
			},
		},
	})
	c.Assert(targetVal.SourceMeta, NotNil)
	c.Assert(targetVal.SourceMeta.Line, Equals, 1)
	c.Assert(targetVal.SourceMeta.Column, Equals, 1)
}

func (s *ValueTestSuite) Test_serialise_valid_value_yaml_input(c *C) {
	expected := &Value{}
	err := yaml.Unmarshal([]byte(s.specFixtures["serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	region := "eu-west-2"
	secret := false
	descriptionStrVal := "The AWS region to connect to AWS services with"
	serialisedBytes, err := yaml.Marshal(&Value{
		Type: &ValueTypeWrapper{
			Value: ValueTypeString,
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &descriptionStrVal,
				},
			},
		},
		Value: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &region,
				},
			},
		},
		Secret: &core.ScalarValue{BoolValue: &secret},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetVal := &Value{}
	err = yaml.Unmarshal(serialisedBytes, targetVal)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetVal.Type, DeepEquals, &ValueTypeWrapper{
		Value: expected.Type.Value,
		SourceMeta: &source.Meta{
			Position:    source.Position{Line: 1, Column: 7},
			EndPosition: &source.Position{Line: 1, Column: 13},
		},
	})
	c.Assert(targetVal.Description, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &descriptionStrVal,
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   3,
						Column: 14,
					},
					EndPosition: &source.Position{
						Line:   3,
						Column: 60,
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   3,
				Column: 14,
			},
			EndPosition: &source.Position{
				Line:   3,
				Column: 60,
			},
		},
	})
	c.Assert(*targetVal.Secret.BoolValue, Equals, *expected.Secret.BoolValue)
	c.Assert(targetVal.Value, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &region,
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   2,
						Column: 8,
					},
					EndPosition: &source.Position{
						Line:   2,
						Column: 17,
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   2,
				Column: 8,
			},
			EndPosition: &source.Position{
				Line:   2,
				Column: 17,
			},
		},
	})
}

func (s *ValueTestSuite) Test_parses_valid_value_json_input(c *C) {
	targetVal := &Value{}
	err := json.Unmarshal([]byte(s.specFixtures["passJSON"]), targetVal)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetVal.Value, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				SubstitutionValue: &substitutions.Substitution{
					ResourceProperty: &substitutions.SubstitutionResourceProperty{
						ResourceName: "awsAccount",
						Path: []*substitutions.SubstitutionPathItem{
							{
								FieldName: "state",
							},
							{
								FieldName: "accountId",
							},
						},
					},
				},
			},
		},
	})
	description := "This is an example integer value"
	c.Assert(targetVal.Description, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &description,
			},
		},
	})
	c.Assert(*targetVal.Secret.BoolValue, Equals, false)
	c.Assert(targetVal.Type, DeepEquals, &ValueTypeWrapper{
		Value: ValueTypeInteger,
	})
}

func (s *ValueTestSuite) Test_serialise_valid_value_json_input(c *C) {
	expected := &Value{}
	err := json.Unmarshal([]byte(s.specFixtures["serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	secret := true
	description := "The AWS region to connect to AWS services with"
	serialisedBytes, err := json.Marshal(&Value{
		Type: &ValueTypeWrapper{
			Value: ValueTypeString,
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Secret: &core.ScalarValue{BoolValue: &secret},
		Value: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					SubstitutionValue: &substitutions.Substitution{
						ResourceProperty: &substitutions.SubstitutionResourceProperty{
							ResourceName: "awsAccount",
							Path: []*substitutions.SubstitutionPathItem{
								{
									FieldName: "state",
								},
								{
									FieldName: "region",
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetVal := &Value{}
	err = json.Unmarshal(serialisedBytes, targetVal)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetVal.Type, DeepEquals, &ValueTypeWrapper{
		Value: expected.Type.Value,
	})
	c.Assert(targetVal.Description, DeepEquals, expected.Description)
	c.Assert(*targetVal.Secret.BoolValue, Equals, *expected.Secret.BoolValue)
	c.Assert(targetVal.Value, DeepEquals, expected.Value)
}
