package schema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v3"
)

type DataSourceTestSuite struct {
	specFixtures map[string][]byte
}

var _ = Suite(&DataSourceTestSuite{})

func (s *DataSourceTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string][]byte)
	fixturesToLoad := map[string]string{
		"fields-passYAML": "__testdata/datasourcefields/pass.yml",
		"fields-failUnsupportedDataSourceFieldTypeYAML":       "__testdata/datasourcefields/fail-unsupported-data-source-field-type.yml",
		"fields-serialiseExpectedYAML":                        "__testdata/datasourcefields/serialise-expected.yml",
		"fields-passJSON":                                     "__testdata/datasourcefields/pass.json",
		"fields-failUnsupportedDataSourceFieldTypeJSON":       "__testdata/datasourcefields/fail-unsupported-data-source-field-type.json",
		"fields-serialiseExpectedJSON":                        "__testdata/datasourcefields/serialise-expected.json",
		"filters-passYAML":                                    "__testdata/datasourcefilters/pass.yml",
		"filters-failUnsupportedDataSourceFilterOperatorYAML": "__testdata/datasourcefilters/fail-unsupported-data-source-filter-operator.yml",
		"filters-serialiseExpectedYAML":                       "__testdata/datasourcefilters/serialise-expected.yml",
		"filters-passJSON":                                    "__testdata/datasourcefilters/pass.json",
		"filters-failUnsupportedDataSourceFilterOperatorJSON": "__testdata/datasourcefilters/fail-unsupported-data-source-filter-operator.json",
		"filters-serialiseExpectedJSON":                       "__testdata/datasourcefilters/serialise-expected.json",
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

func (s *DataSourceTestSuite) Test_parses_valid_data_source_field_yaml_input(c *C) {
	targetField := &DataSourceFieldExport{}
	err := yaml.Unmarshal([]byte(s.specFixtures["fields-passYAML"]), targetField)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	description := "This is an example boolean data source field"
	c.Assert(targetField.Description, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &description,
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   2,
						Column: 14,
					},
					EndPosition: &source.Position{
						Line:   2,
						Column: 58,
					},
				},
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   2,
				Column: 14,
			},
			EndPosition: &source.Position{
				Line:   2,
				Column: 58,
			},
		},
	})
	c.Assert(targetField.Type.Value, Equals, DataSourceFieldType("boolean"))
}

func (s *DataSourceTestSuite) Test_serialise_valid_data_source_field_yaml_input(c *C) {
	expected := &DataSourceFieldExport{}
	err := yaml.Unmarshal([]byte(s.specFixtures["fields-serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	description := "The AWS region to connect to AWS services with"
	serialisedBytes, err := yaml.Marshal(&DataSourceFieldExport{
		Type: &DataSourceFieldTypeWrapper{
			Value: DataSourceFieldTypeString,
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetField := &DataSourceFieldExport{}
	err = yaml.Unmarshal(serialisedBytes, targetField)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	NormaliseStringOrSubstitutions(targetField.Description)
	NormaliseStringOrSubstitutions(expected.Description)
	c.Assert(targetField.Type.Value, Equals, expected.Type.Value)
	c.Assert(targetField.Description, DeepEquals, expected.Description)
}

func (s *DataSourceTestSuite) Test_parses_valid_data_source_field_json_input(c *C) {
	targetField := &DataSourceFieldExport{}
	err := json.Unmarshal([]byte(s.specFixtures["fields-passJSON"]), targetField)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	description := "This is an example integer data source field"
	c.Assert(targetField.Description, DeepEquals, &substitutions.StringOrSubstitutions{
		Values: []*substitutions.StringOrSubstitution{
			{
				StringValue: &description,
			},
		},
	})
	c.Assert(targetField.Type.Value, Equals, DataSourceFieldType("integer"))
}

func (s *DataSourceTestSuite) Test_serialise_valid_data_source_field_json_input(c *C) {
	expected := &DataSourceFieldExport{}
	err := json.Unmarshal([]byte(s.specFixtures["fields-serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	description := "The AWS region to connect to AWS services with"
	serialisedBytes, err := json.Marshal(&DataSourceFieldExport{
		Type: &DataSourceFieldTypeWrapper{
			Value: DataSourceFieldTypeString,
		},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetField := &DataSourceFieldExport{}
	err = json.Unmarshal(serialisedBytes, targetField)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetField.Type.Value, Equals, expected.Type.Value)
	c.Assert(targetField.Description, DeepEquals, expected.Description)
}

func (s *DataSourceTestSuite) Test_parses_valid_data_source_filter_yaml_input(c *C) {
	targetFilter := &DataSourceFilter{}
	err := yaml.Unmarshal([]byte(s.specFixtures["filters-passYAML"]), targetFilter)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(*targetFilter.Field.StringValue, Equals, "tags")
	c.Assert(targetFilter.Operator.Value, Equals, DataSourceFilterOperatorHasKey)
	c.Assert(
		targetFilter.Search.Values,
		DeepEquals,
		[]*substitutions.StringOrSubstitutions{
			{
				Values: []*substitutions.StringOrSubstitution{
					{
						SubstitutionValue: &substitutions.Substitution{
							Variable: &substitutions.SubstitutionVariable{
								VariableName: "environment",
								SourceMeta: &source.Meta{
									Position: source.Position{
										Line:   3,
										Column: 11,
									},
									EndPosition: &source.Position{
										Line:   3,
										Column: 32,
									},
								},
							},
							SourceMeta: &source.Meta{
								Position: source.Position{
									Line:   3,
									Column: 11,
								},
								EndPosition: &source.Position{
									Line:   3,
									Column: 32,
								},
							},
						},
						SourceMeta: &source.Meta{
							Position: source.Position{
								Line:   3,
								Column: 9,
							},
							EndPosition: &source.Position{
								Line:   3,
								Column: 33,
							},
						},
					},
				},
				SourceMeta: &source.Meta{
					Position: source.Position{
						Line:   3,
						Column: 9,
					},
					EndPosition: &source.Position{
						Line:   3,
						Column: 33,
					},
				},
			},
		},
	)
}

func (s *DataSourceTestSuite) Test_serialise_valid_data_source_filter_yaml_input(c *C) {
	expected := &DataSourceFilter{}
	err := yaml.Unmarshal([]byte(s.specFixtures["filters-serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	searchFor := "ACTIVE"
	field := "configuration.status"
	serialisedBytes, err := yaml.Marshal(&DataSourceFilter{
		Field: &core.ScalarValue{StringValue: &field},
		Operator: &DataSourceFilterOperatorWrapper{
			Value: DataSourceFilterOperatorEquals,
		},
		Search: &DataSourceFilterSearch{
			Values: []*substitutions.StringOrSubstitutions{
				{
					Values: []*substitutions.StringOrSubstitution{
						{
							StringValue: &searchFor,
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

	targetFilter := &DataSourceFilter{}
	err = yaml.Unmarshal(serialisedBytes, targetFilter)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(*targetFilter.Field.StringValue, Equals, *expected.Field.StringValue)
	c.Assert(targetFilter.Operator.Value, Equals, expected.Operator.Value)
	c.Assert(*targetFilter.Search.Values[0].Values[0].StringValue, Equals, *expected.Search.Values[0].Values[0].StringValue)
}

func (s *DataSourceTestSuite) Test_parses_valid_data_source_filter_json_input(c *C) {
	targetFilter := &DataSourceFilter{}
	err := json.Unmarshal([]byte(s.specFixtures["filters-passJSON"]), targetFilter)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(*targetFilter.Field.StringValue, Equals, "tags")
	c.Assert(targetFilter.Operator.Value, Equals, DataSourceFilterOperatorHasKey)
	c.Assert(
		targetFilter.Search.Values,
		DeepEquals,
		[]*substitutions.StringOrSubstitutions{
			{
				Values: []*substitutions.StringOrSubstitution{
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
	)
}

func (s *DataSourceTestSuite) Test_serialise_valid_data_source_filter_json_input(c *C) {
	expected := &DataSourceFilter{}
	err := json.Unmarshal([]byte(s.specFixtures["filters-serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	searchFor := "ACTIVE"
	field := "configuration.status"
	serialisedBytes, err := yaml.Marshal(&DataSourceFilter{
		Field: &core.ScalarValue{StringValue: &field},
		Operator: &DataSourceFilterOperatorWrapper{
			Value: DataSourceFilterOperatorEquals,
		},
		Search: &DataSourceFilterSearch{
			Values: []*substitutions.StringOrSubstitutions{
				{
					Values: []*substitutions.StringOrSubstitution{
						{
							StringValue: &searchFor,
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

	targetFilter := &DataSourceFilter{}
	err = yaml.Unmarshal(serialisedBytes, targetFilter)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(*targetFilter.Field.StringValue, Equals, *expected.Field.StringValue)
	c.Assert(targetFilter.Operator.Value, Equals, expected.Operator.Value)
	c.Assert(*targetFilter.Search.Values[0].Values[0].StringValue, Equals, *expected.Search.Values[0].Values[0].StringValue)
}
