package validation

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
)

type ConflictValidationSuite struct {
	suite.Suite
}

func (s *ConflictValidationSuite) Test_conflicts_with_plugin_config_field() {
	conflictsWithKey := "test.conflict.key"
	validationFunc := ConflictsWithPluginConfig(conflictsWithKey)

	// Test case where the plugin config contains the conflicting key
	pluginConfig := core.PluginConfig{
		conflictsWithKey: core.ScalarFromString("conflicting value"),
	}

	diagnostics := validationFunc("testField", core.ScalarFromString("main value"), pluginConfig)
	s.Assert().NotEmpty(diagnostics)
	s.Assert().Equal(
		"\"testField\" cannot be set because it conflicts with"+
			" the plugin configuration key \"test.conflict.key\".",
		diagnostics[0].Message,
	)
}

func (s *ConflictValidationSuite) Test_does_not_conflict_with_plugin_config_field() {
	conflictsWithKey := "test.conflict.key"
	validationFunc := ConflictsWithPluginConfig(conflictsWithKey)

	// Test case where the plugin config does not contain the conflicting key
	pluginConfig := core.PluginConfig{
		"another.key": core.ScalarFromString("some value"),
	}

	diagnostics := validationFunc("testField", core.ScalarFromString("main value"), pluginConfig)
	s.Assert().Empty(diagnostics)
}

func (s *ConflictValidationSuite) Test_conflicts_with_resource_definition_field() {
	conflictsWithKey := "$.testMap.conflictKey"
	validationFunc := ConflictsWithResourceDefinition(conflictsWithKey)

	// Test case where the resource definition contains the conflicting key
	resource := &schema.Resource{
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"testMap": {
					Fields: map[string]*core.MappingNode{
						"conflictKey": core.MappingNodeFromString("conflicting value"),
					},
				},
			},
		},
	}

	diagnostics := validationFunc("testField", core.MappingNodeFromString("main value"), resource)
	s.Assert().NotEmpty(diagnostics)
	s.Assert().Equal(
		"\"testField\" cannot be set because it conflicts with"+
			" the resource spec field \"testMap.conflictKey\".",
		diagnostics[0].Message,
	)
}

func (s *ConflictValidationSuite) Test_does_not_conflict_with_resource_definition_field() {
	conflictsWithKey := "$.testMap.conflictKey"
	validationFunc := ConflictsWithResourceDefinition(conflictsWithKey)

	// Test case where the resource definition does not contain the conflicting key
	resource := &schema.Resource{
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"testMap": {
					Fields: map[string]*core.MappingNode{
						"anotherKey": core.MappingNodeFromString("some value"),
					},
				},
			},
		},
	}

	diagnostics := validationFunc("testField", core.MappingNodeFromString("main value"), resource)
	s.Assert().Empty(diagnostics)
}

func TestConflictValidationSuite(t *testing.T) {
	suite.Run(t, new(ConflictValidationSuite))
}
