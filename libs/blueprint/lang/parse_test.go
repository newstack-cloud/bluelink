package lang_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/stretchr/testify/suite"
)

type ParseSuite struct {
	suite.Suite
}

func (s *ParseSuite) parseFixture(name string) (string, error) {
	contents, err := os.ReadFile(filepath.Join("__testdata", name+".bp"))
	return string(contents), err
}

// snapshotFixtures parses each named .bp file and snapshots the resulting
// blueprint AST. Used by every per-group success test.
func (s *ParseSuite) snapshotFixtures(fixtures []string) {
	for _, name := range fixtures {
		s.Run(name, func() {
			src, err := s.parseFixture(name)
			s.Require().NoError(err)

			blueprint, err := lang.ParseString(src)
			s.Require().NoError(err)
			s.Require().NotNil(blueprint)
			s.Require().NoError(cupaloy.SnapshotWithName(name, blueprint))
		})
	}
}

// snapshotErrors parses each src string, expects an error, and snapshots the
// formatted error message. Used by every per-group error test.
func (s *ParseSuite) snapshotErrors(cases []errorCase) {
	for _, c := range cases {
		s.Run(c.name, func() {
			_, err := lang.ParseString(c.src)
			s.Require().Error(err)
			s.Require().NoError(cupaloy.SnapshotWithName(c.name, err.Error()))
		})
	}
}

type errorCase struct {
	name string
	src  string
}

func (s *ParseSuite) Test_parses_directive_fixtures() {
	s.snapshotFixtures([]string{
		"version-only",
		"transform-single",
		"transform-multiple",
	})
}

func (s *ParseSuite) Test_parses_variable_fixtures() {
	s.snapshotFixtures([]string{
		"variable-string",
		"variable-integer",
		"variable-float",
		"variable-boolean",
		"variable-secret",
		"variable-element-types",
		"variable-empty-block",
		"variable-separators",
		"variable-quoted-name",
		"variable-multiple",
		"variable-multiline-description",
	})
}

func (s *ParseSuite) Test_parses_value_fixtures() {
	s.snapshotFixtures([]string{
		"value-string",
		"value-array",
		"value-object",
		"value-expression",
		"value-call",
	})
}

func (s *ParseSuite) Test_parses_export_fixtures() {
	s.snapshotFixtures([]string{
		"export-bare-ref",
		"export-string-form",
		"export-each-type",
		"export-with-description",
	})
}

func (s *ParseSuite) Test_parses_include_fixtures() {
	s.snapshotFixtures([]string{
		"include-minimal",
		"include-with-description",
		"include-all",
	})
}

func (s *ParseSuite) Test_parses_metadata_fixtures() {
	s.snapshotFixtures([]string{
		"metadata-dotted-keys",
		"metadata-nested",
	})
}

func (s *ParseSuite) Test_parses_data_fixtures() {
	s.snapshotFixtures([]string{
		"data-minimal",
		"data-metadata",
		"data-filter-equality",
		"data-filter-comparison",
		"data-filter-collection",
		"data-filter-text",
		"data-export-forms",
		"data-description",
	})
}

func (s *ParseSuite) Test_parses_resource_fixtures() {
	s.snapshotFixtures([]string{
		"resource-minimal",
		"resource-condition-bare",
		"resource-condition-object",
		"resource-metadata-full",
		"resource-select",
		"resource-select-exclude",
		"resource-dependson",
		"resource-removal-policy",
		"resource-foreach",
		"resource-spec-complex",
	})
}

func (s *ParseSuite) Test_parses_expression_fixtures() {
	s.snapshotFixtures([]string{
		"expr-precedence",
		"expr-multiline-ops",
		"expr-function-call",
		"expr-multiline-string",
		"expr-resource-path",
		"expr-resource-quoted-accessor",
		"expr-none-literal",
	})
}

func (s *ParseSuite) Test_parses_lexical_fixtures() {
	s.snapshotFixtures([]string{
		"comments-basic",
	})
}

func (s *ParseSuite) Test_reports_directive_errors() {
	s.snapshotErrors([]errorCase{
		{"duplicate-version", "version \"2025-11-02\"\nversion \"2025-11-02\""},
		{"transform-interpolation", "transform \"${variables.region}\""},
		{"empty-transform-list", "version \"2025-11-02\"\ntransform []"},
		{"unexpected-top-level-token", "notAKeyword"},
		{"multiline-version", "version \"\"\"\n2025-11-02\n\"\"\""},
		{"transform-not-a-string", "transform 123"},
		{"missing-version", "variable region: string {}"},
	})
}

func (s *ParseSuite) Test_reports_name_and_type_errors() {
	s.snapshotErrors([]errorCase{
		{"reserved-word-as-name", "variable resource: string {}"},
		{"invalid-quoted-name", "variable \"with space\": string {}"},
		{"invalid-type-segment", "variable instance: aws-x/ec2 {}"},
		{"element-type-single-segment", "variable instance: aws {}"},
	})
}

func (s *ParseSuite) Test_reports_variable_errors() {
	s.snapshotErrors([]errorCase{
		{"variable-unknown-field", "variable region: string { foo = \"bar\" }"},
		{"variable-missing-assign", "variable region: string { default \"us-east-1\" }"},
		{"variable-non-scalar-default", "variable region: string { default = [\"a\"] }"},
		{"variable-allowedvalues-not-array", "variable region: string { allowedValues = \"a\" }"},
		{"variable-unterminated-block", "variable region: string { default = \"us-east-1\""},
		{"variable-secret-not-bool", "variable region: string { secret = \"yes\" }"},
		{"variable-description-not-string", "version \"2025-11-02\"\nvariable region: string { description = 42 }"},
		{"variable-boolean-allowedvalues", "version \"2025-11-02\"\nvariable flag: boolean { allowedValues = [true, false] }"},
	})
}

func (s *ParseSuite) Test_reports_value_errors() {
	s.snapshotErrors([]errorCase{
		{"value-unknown-field", "value x: string { foo = \"bar\" }"},
		{"value-bad-type", "value x: notAType { value = \"x\" }"},
	})
}

func (s *ParseSuite) Test_reports_export_errors() {
	s.snapshotErrors([]errorCase{
		{"export-unknown-field", "export x: string { foo = \"bar\" }"},
		{"export-fn-call", "export x: string { field = jsonencode(variables.x) }"},
		{"export-bad-type", "export x: notAType { field = variables.x }"},
	})
}

func (s *ParseSuite) Test_reports_include_errors() {
	s.snapshotErrors([]errorCase{
		{"include-missing-path", "include child {}"},
		{"include-unknown-field", "include child \"a.yaml\" { foo = \"bar\" }"},
		{"include-unterminated", "include child \"a.yaml\" {"},
	})
}

func (s *ParseSuite) Test_reports_metadata_errors() {
	s.snapshotErrors([]errorCase{
		{"metadata-missing-assign", "metadata { foo \"bar\" }"},
		{"metadata-duplicate", "version \"2025-11-02\"\nmetadata { a = 1 }\nmetadata { b = 2 }"},
	})
}

func (s *ParseSuite) Test_reports_data_errors() {
	s.snapshotErrors([]errorCase{
		{
			"data-export-type-object",
			"data n: aws/vpc { filter \"x\" == \"y\"\nexport id: object }",
		},
		{
			"data-not-before-eq",
			"data n: aws/vpc { filter \"x\" not == \"y\"\nexport id: string }",
		},
		{
			"data-has-without-key",
			"data n: aws/vpc { filter \"x\" has \"y\"\nexport id: string }",
		},
		{
			"data-unknown-field",
			"data n: aws/vpc { foo = \"bar\" }",
		},
	})
}

func (s *ParseSuite) Test_reports_resource_errors() {
	s.snapshotErrors([]errorCase{
		{
			"resource-condition-two-keys",
			"resource r: aws/x { condition = { and = [variables.x], or = [variables.y] }\nspec {} }",
		},
		{
			"resource-labels-non-string",
			"resource r: aws/x { metadata { labels = { service = 42 } }\nspec {} }",
		},
		{
			"resource-removal-policy-non-literal",
			"resource r: aws/x { removalPolicy = variables.policy\nspec {} }",
		},
		{
			"resource-removal-policy-invalid",
			"version \"2025-11-02\"\nresource r: aws/x { removalPolicy = \"destroy\"\nspec {} }",
		},
		{
			"resource-spec-missing",
			"version \"2025-11-02\"\nresource r: aws/x { description = \"desc\" }",
		},
		{
			"resource-select-missing-by",
			"resource r: aws/x { select { service = \"ordersApi\" }\nspec {} }",
		},
		{
			"resource-unknown-field",
			"resource r: aws/x { foo = \"bar\"\nspec {} }",
		},
	})
}

func (s *ParseSuite) Test_reports_expression_errors() {
	s.snapshotErrors([]errorCase{
		{
			"expr-dangling-op",
			"value x: boolean { value = variables.a && }",
		},
		{
			"expr-unterminated-call",
			"value x: object { value = object(",
		},
	})
}

func TestParseSuite(t *testing.T) {
	suite.Run(t, new(ParseSuite))
}
