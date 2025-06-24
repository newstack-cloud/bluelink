package validation

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	. "gopkg.in/check.v1"
)

type TransformValidationTestSuite struct{}

var _ = Suite(&TransformValidationTestSuite{})

func (s *TransformValidationTestSuite) Test_succeeds_without_any_issues_for_a_valid_transform(c *C) {
	version := Version2025_05_12
	blueprint := &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: &version},
		Transform: &schema.TransformValueWrapper{
			StringList: schema.StringList{
				Values: []string{"celerity-2025-08-01"},
				SourceMeta: []*source.Meta{
					{Position: source.Position{
						Line:   1,
						Column: 1,
					}},
				},
			},
		},
	}
	diagnostics, err := ValidateTransforms(context.Background(), blueprint, false)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, HasLen, 0)
}

func (s *BlueprintValidationTestSuite) Test_reports_error_for_sub_usage_in_transform(c *C) {
	version := Version2025_05_12
	blueprint := &schema.Blueprint{
		Version: &core.ScalarValue{StringValue: &version},
		Transform: &schema.TransformValueWrapper{
			StringList: schema.StringList{
				Values: []string{"celerity-2025-08-01", "${variables.transform1}"},
				SourceMeta: []*source.Meta{
					{Position: source.Position{
						Line:   1,
						Column: 1,
					}},
					{Position: source.Position{
						Line:   2,
						Column: 1,
					}},
				},
			},
		},
	}
	diagnostics, err := ValidateTransforms(context.Background(), blueprint, false)
	c.Assert(err, IsNil)
	c.Assert(diagnostics, DeepEquals, []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "${..} substitutions can not be used in a transform.",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{Position: source.Position{
					Line:   2,
					Column: 1,
				}},
				End: &source.Meta{Position: source.Position{
					Line:   3,
					Column: 1,
				}},
			},
		},
	})
}
