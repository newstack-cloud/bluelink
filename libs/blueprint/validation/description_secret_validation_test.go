package validation

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	. "gopkg.in/check.v1"
)

type DescriptionSecretValidationTestSuite struct {
	funcRegistry       provider.FunctionRegistry
	refChainCollector  refgraph.RefChainCollector
	resourceRegistry   resourcehelpers.Registry
	dataSourceRegistry provider.DataSourceRegistry
}

var _ = Suite(&DescriptionSecretValidationTestSuite{})

func (s *DescriptionSecretValidationTestSuite) SetUpTest(c *C) {
	s.funcRegistry = &internal.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"trim": corefunctions.NewTrimFunction(),
		},
	}
	s.refChainCollector = refgraph.NewRefChainCollector()
	s.resourceRegistry = internal.NewResourceRegistryMock(map[string]provider.Resource{})
	s.dataSourceRegistry = &internal.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{},
	}
}

func (s *DescriptionSecretValidationTestSuite) Test_warns_when_a_description_interpolates_a_secret_variable(c *C) {
	diagnostics := s.validateExportWithDescription(c, variableSub("dbPassword"))

	c.Assert(secretDescriptionWarnings(diagnostics), HasLen, 1)
	c.Assert(
		secretDescriptionWarnings(diagnostics)[0].Message,
		Matches,
		".*\"variables.dbPassword\".*declared as a secret.*",
	)
}

func (s *DescriptionSecretValidationTestSuite) Test_warns_when_a_description_interpolates_a_secret_value(c *C) {
	diagnostics := s.validateExportWithDescription(c, valueSub("adminToken"))

	c.Assert(secretDescriptionWarnings(diagnostics), HasLen, 1)
}

// A secret handed to a function is still a secret in the text the function produces.
func (s *DescriptionSecretValidationTestSuite) Test_warns_when_a_secret_reaches_a_description_through_a_function(c *C) {
	trimmed := &substitutions.Substitution{
		Function: &substitutions.SubstitutionFunctionExpr{
			FunctionName: "trim",
			Arguments: []*substitutions.SubstitutionFunctionArg{
				{Value: variableSub("dbPassword")},
			},
		},
	}

	c.Assert(secretDescriptionWarnings(s.validateExportWithDescription(c, trimmed)), HasLen, 1)
}

func (s *DescriptionSecretValidationTestSuite) Test_does_not_warn_for_a_plain_variable(c *C) {
	diagnostics := s.validateExportWithDescription(c, variableSub("environment"))

	c.Assert(secretDescriptionWarnings(diagnostics), HasLen, 0)
}

func (s *DescriptionSecretValidationTestSuite) validateExportWithDescription(
	c *C,
	descriptionSub *substitutions.Substitution,
) []*core.Diagnostic {
	field := "variables.environment"
	exportSchema := &schema.Export{
		Type: &schema.ExportTypeWrapper{Value: schema.ExportTypeString},
		Description: &substitutions.StringOrSubstitutions{
			Values: []*substitutions.StringOrSubstitution{
				{SubstitutionValue: descriptionSub},
			},
		},
		Field: &core.ScalarValue{StringValue: &field},
	}
	exportMap := &schema.ExportMap{
		Values: map[string]*schema.Export{"environment": exportSchema},
	}

	diagnostics, err := ValidateExport(
		context.Background(),
		"environment",
		exportSchema,
		exportMap,
		&ValidationContext{
			BpSchema:           secretDescriptionBlueprint(exportMap),
			Params:             &core.ParamsImpl{},
			FuncRegistry:       s.funcRegistry,
			RefChainCollector:  s.refChainCollector,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
	)
	c.Assert(err, IsNil)

	return diagnostics
}

func secretDescriptionBlueprint(exportMap *schema.ExportMap) *schema.Blueprint {
	secret := true
	stringType := schema.VariableTypeString
	tokenValue := "token"

	return &schema.Blueprint{
		Variables: &schema.VariableMap{
			Values: map[string]*schema.Variable{
				"dbPassword": {
					Type:   &schema.VariableTypeWrapper{Value: stringType},
					Secret: &core.ScalarValue{BoolValue: &secret},
				},
				"environment": {
					Type: &schema.VariableTypeWrapper{Value: stringType},
				},
			},
		},
		Values: &schema.ValueMap{
			Values: map[string]*schema.Value{
				"adminToken": {
					Type:   &schema.ValueTypeWrapper{Value: schema.ValueTypeString},
					Secret: &core.ScalarValue{BoolValue: &secret},
					Value:  core.MappingNodeFromString(tokenValue),
				},
			},
		},
		Exports: exportMap,
	}
}

func variableSub(variableName string) *substitutions.Substitution {
	return &substitutions.Substitution{
		Variable: &substitutions.SubstitutionVariable{VariableName: variableName},
	}
}

func valueSub(valueName string) *substitutions.Substitution {
	return &substitutions.Substitution{
		ValueReference: &substitutions.SubstitutionValueReference{ValueName: valueName},
	}
}

func secretDescriptionWarnings(diagnostics []*core.Diagnostic) []*core.Diagnostic {
	warnings := []*core.Diagnostic{}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "declared as a secret") {
			warnings = append(warnings, diagnostic)
		}
	}

	return warnings
}
