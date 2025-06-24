package schema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

type ConditionTestSuite struct {
	specFixtures map[string][]byte
}

var _ = Suite(&ConditionTestSuite{})

func (s *ConditionTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string][]byte)
	fixturesToLoad := map[string]string{
		"passYAML":               "__testdata/conditions/pass.yml",
		"serialiseExpectedYAML":  "__testdata/conditions/serialise-expected.yml",
		"passJSON":               "__testdata/conditions/pass.json",
		"serialiseExpectedJSON":  "__testdata/conditions/serialise-expected.json",
		"failAndOrNotAllSetYAML": "__testdata/conditions/fail-and-or-not-all-set.yml",
		"failAndOrNotAllSetJSON": "__testdata/conditions/fail-and-or-not-all-set.json",
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

func (s *ConditionTestSuite) Test_parses_valid_condition_yaml_input(c *C) {
	targetVal := &Condition{}
	err := yaml.Unmarshal([]byte(s.specFixtures["passYAML"]), targetVal)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}
	err = testhelpers.Snapshot(targetVal)
	if err != nil {
		c.Error(err)
	}
}

func (s *ConditionTestSuite) Test_serialise_valid_condition_yaml_input(c *C) {
	expected := &Condition{}
	err := yaml.Unmarshal([]byte(s.specFixtures["serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	serialisedBytes, err := yaml.Marshal(serialiseInputCondition())
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetCondition := &Condition{}
	err = yaml.Unmarshal(serialisedBytes, targetCondition)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	NormaliseCondition(targetCondition)
	NormaliseCondition(expected)
	c.Assert(targetCondition, DeepEquals, expected)
}

func (s *ConditionTestSuite) Test_parses_valid_condition_json_input(c *C) {
	targetVal := &Condition{}
	err := json.Unmarshal([]byte(s.specFixtures["passJSON"]), targetVal)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}
	err = testhelpers.Snapshot(targetVal)
	if err != nil {
		c.Error(err)
	}
}

func (s *ConditionTestSuite) Test_serialise_valid_condition_json_input(c *C) {
	expected := &Condition{}
	err := json.Unmarshal([]byte(s.specFixtures["serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	serialisedBytes, err := json.Marshal(serialiseInputCondition())
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetCondition := &Condition{}
	err = json.Unmarshal(serialisedBytes, targetCondition)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	// No need to normalise conditions for JSON as it does not keep track of line numbers
	// and columns.
	c.Assert(targetCondition, DeepEquals, expected)
}

func (s *ConditionTestSuite) Test_fails_to_parse_invalid_condition_yaml_input(c *C) {
	targetVal := &Condition{}
	err := yaml.Unmarshal([]byte(s.specFixtures["failAndOrNotAllSetYAML"]), targetVal)
	if err == nil {
		c.Error("expected error parsing invalid condition yaml input")
		c.FailNow()
	}

	c.Assert(err.Error(), Equals, "an invalid resource condition has been provided, "+
		"only one of \"and\", \"or\" or \"not\" can be set")
}

func (s *ConditionTestSuite) Test_fails_to_parse_invalid_condition_json_input(c *C) {
	targetVal := &Condition{}
	err := yaml.Unmarshal([]byte(s.specFixtures["failAndOrNotAllSetJSON"]), targetVal)
	if err == nil {
		c.Error("expected error parsing invalid condition json input")
		c.FailNow()
	}

	c.Assert(err.Error(), Equals, "an invalid resource condition has been provided, "+
		"only one of \"and\", \"or\" or \"not\" can be set")
}

func serialiseInputCondition() *Condition {
	prefix := "newstack-cloud"
	oss := "oss"
	return &Condition{
		Or: []*Condition{
			{
				Not: &Condition{
					StringValue: &substitutions.StringOrSubstitutions{
						Values: []*substitutions.StringOrSubstitution{
							{
								SubstitutionValue: &substitutions.Substitution{
									Function: &substitutions.SubstitutionFunctionExpr{
										FunctionName: "has_prefix",
										Arguments: []*substitutions.SubstitutionFunctionArg{
											{
												Value: &substitutions.Substitution{
													ResourceProperty: &substitutions.SubstitutionResourceProperty{
														ResourceName: "s3Bucket",
														Path: []*substitutions.SubstitutionPathItem{
															{
																FieldName: "bucketName",
															},
														},
													},
												},
											},
											{
												Value: &substitutions.Substitution{
													StringValue: &prefix,
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
			{
				StringValue: &substitutions.StringOrSubstitutions{
					Values: []*substitutions.StringOrSubstitution{
						{
							SubstitutionValue: &substitutions.Substitution{
								Function: &substitutions.SubstitutionFunctionExpr{
									FunctionName: "contains",
									Arguments: []*substitutions.SubstitutionFunctionArg{
										{
											Value: &substitutions.Substitution{
												ResourceProperty: &substitutions.SubstitutionResourceProperty{
													ResourceName: "s3Bucket",
													Path: []*substitutions.SubstitutionPathItem{
														{
															FieldName: "bucketName",
														},
													},
												},
											},
										},
										{
											Value: &substitutions.Substitution{
												StringValue: &oss,
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
