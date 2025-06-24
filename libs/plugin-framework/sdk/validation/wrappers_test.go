package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
)

type WrappersValidationSuite struct {
	suite.Suite
}

func (s *WrappersValidationSuite) Test_plugin_config_wrapper() {
	diagnostics := WrapForPluginConfig(
		IsWebURL(),
	)(
		"exampleField",
		core.ScalarFromString("https://example.com"),
		core.PluginConfig{},
	)
	s.Assert().Empty(diagnostics)
}

func (s *WrappersValidationSuite) Test_resource_definition_wrapper() {
	diagnostics := WrapForResourceDefinition(
		IsWebURL(),
	)(
		"exampleField",
		core.MappingNodeFromString("https://example.com"),
		&schema.Resource{},
	)
	s.Assert().Empty(diagnostics)
}

func (s *WrappersValidationSuite) Test_resource_definition_wrapper_fails_for_non_scalar_value() {
	diagnostics := WrapForResourceDefinition(
		IsWebURL(),
	)(
		"exampleField",
		&core.MappingNode{
			Items: []*core.MappingNode{
				core.MappingNodeFromString("https://example.com"),
				core.MappingNodeFromString("not-a-url"),
			},
		},
		&schema.Resource{},
	)
	s.Assert().NotEmpty(diagnostics)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Equal(
		diagnostics[0].Message,
		"exampleField is not a valid type for the configured "+
			"validator, expected a scalar (string, integer, float or boolean), but got array.",
	)
}

func TestWrappersValidationSuite(t *testing.T) {
	suite.Run(t, new(WrappersValidationSuite))
}
