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

type ExportTestSuite struct {
	specFixtures map[string][]byte
}

var _ = Suite(&ExportTestSuite{})

func exportExactColAccuracy() *source.ColumnAccuracy {
	ca := source.ColumnAccuracyExact
	return &ca
}

func (s *ExportTestSuite) SetUpSuite(c *C) {
	s.specFixtures = make(map[string][]byte)
	fixturesToLoad := map[string]string{
		"passYAML":              "__testdata/exports/pass.yml",
		"serialiseExpectedYAML": "__testdata/exports/serialise-expected.yml",
		"passJSON":              "__testdata/exports/pass.json",
		"serialiseExpectedJSON": "__testdata/exports/serialise-expected.json",
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

func (s *ExportTestSuite) Test_parses_valid_string_export_yaml_input(c *C) {
	targetExport := &Export{}
	err := yaml.Unmarshal([]byte(s.specFixtures["passYAML"]), targetExport)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	field := "resources.orderQueue.spec.arn"
	description := "The arn of the queue used to send order workloads to"
	c.Assert(targetExport, DeepEquals, &Export{
		Type: &ExportTypeWrapper{
			Value: ExportTypeString,
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   1,
					Column: 7,
				},
				EndPosition: &source.Position{
					Line:   1,
					Column: 13,
				},
			ColumnAccuracy: exportExactColAccuracy(),
			},
		},
		Description: &substitutions.StringOrSubstitutions{
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
							Column: 66,
						},
					ColumnAccuracy: exportExactColAccuracy(),
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
					Column: 66,
				},
			ColumnAccuracy: exportExactColAccuracy(),
			},
		},
		Field: &core.ScalarValue{
			StringValue: &field,
			SourceMeta: &source.Meta{
				Position: source.Position{
					Line:   3,
					Column: 8,
				},
				EndPosition: &source.Position{
					Line:   3,
					Column: 37,
				},
			ColumnAccuracy: exportExactColAccuracy(),
			},
		},
		SourceMeta: &source.Meta{
			Position: source.Position{
				Line:   1,
				Column: 1,
			},
		},
	})
}

func (s *ExportTestSuite) Test_serialise_valid_export_yaml_input(c *C) {
	expected := &Export{}
	err := yaml.Unmarshal([]byte(s.specFixtures["serialiseExpectedYAML"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	description := "The ARN of the function used to save orders to the system."
	field := "resources.saveOrdersFunction.spec.functionArn"
	serialisedBytes, err := yaml.Marshal(&Export{
		Type: &ExportTypeWrapper{Value: ExportTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Field: &core.ScalarValue{StringValue: &field},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetExport := &Export{}
	err = yaml.Unmarshal(serialisedBytes, targetExport)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	NormaliseExport(targetExport)
	NormaliseExport(expected)
	c.Assert(targetExport, DeepEquals, expected)
}

func (s *ExportTestSuite) Test_parses_valid_export_json_input(c *C) {
	targetExport := &Export{}
	err := json.Unmarshal([]byte(s.specFixtures["passJSON"]), targetExport)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	description := "The arn of the queue used to send order workloads to"
	field := "resources.orderQueue.spec.arn"
	c.Assert(targetExport, DeepEquals, &Export{
		Type: &ExportTypeWrapper{Value: ExportTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Field: &core.ScalarValue{StringValue: &field},
	})
}

func (s *ExportTestSuite) Test_serialise_valid_export_json_input(c *C) {
	expected := &Export{}
	err := json.Unmarshal([]byte(s.specFixtures["serialiseExpectedJSON"]), expected)
	if err != nil {
		c.Error(fmt.Errorf("failed to parse expected fixture to compare with: %s", err.Error()))
		c.FailNow()
	}

	description := "The ARN of the function used to save orders to the system."
	field := "resources.saveOrdersFunction.spec.functionArn"
	serialisedBytes, err := json.Marshal(&Export{
		Type: &ExportTypeWrapper{Value: ExportTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{
					StringValue: &description,
				},
			},
		},
		Field: &core.ScalarValue{StringValue: &field},
	})
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	targetExport := &Export{}
	err = json.Unmarshal(serialisedBytes, targetExport)
	if err != nil {
		c.Error(err)
		c.FailNow()
	}

	c.Assert(targetExport, DeepEquals, expected)
}
