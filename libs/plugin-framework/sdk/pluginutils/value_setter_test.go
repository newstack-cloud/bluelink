package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ValueSetterSuite struct {
	suite.Suite
}

type testTarget struct {
	FieldValue string
}

func (s *ValueSetterSuite) Test_sets_value_on_target_when_value_is_not_nil() {
	setter := NewValueSetter(
		"$.testField",
		func(value *core.MappingNode, target *testTarget) {
			target.FieldValue = core.StringValue(value)
		},
	)

	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"testField": core.MappingNodeFromString("testValue"),
		},
	}

	target := &testTarget{}

	setter.Set(specData, target)

	s.Assert().Equal("testValue", target.FieldValue)
	s.Assert().True(setter.DidSet())
}

func (s *ValueSetterSuite) Test_skips_setting_value_for_non_existent_path() {
	setter := NewValueSetter(
		"$.testFieldMissing",
		func(value *core.MappingNode, target *testTarget) {
			target.FieldValue = core.StringValue(value)
		},
	)

	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"testField": core.MappingNodeFromString("testValue"),
		},
	}

	target := &testTarget{}

	setter.Set(specData, target)

	s.Assert().False(setter.DidSet())
	s.Assert().Equal("", target.FieldValue)
}

func (s *ValueSetterSuite) Test_skips_setting_value_for_value_explicity_set_to_nil() {
	setter := NewValueSetter(
		"$.testField",
		func(value *core.MappingNode, target *testTarget) {
			target.FieldValue = core.StringValue(value)
		},
	)

	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"testField": nil,
		},
	}

	target := &testTarget{}

	setter.Set(specData, target)

	s.Assert().False(setter.DidSet())
	s.Assert().Equal("", target.FieldValue)
}

func (s *ValueSetterSuite) Test_does_not_set_value_if_not_in_modified_fields() {
	setter := NewValueSetter(
		"$.testFieldMissing",
		func(value *core.MappingNode, target *testTarget) {
			target.FieldValue = core.StringValue(value)
		},
		WithValueSetterCheckIfChanged[*testTarget](true),
		WithValueSetterModifiedFields[*testTarget](
			[]provider.FieldChange{
				{
					FieldPath: "spec.testField",
					NewValue:  core.MappingNodeFromString("testValue"),
					PrevValue: core.MappingNodeFromString("oldValue"),
				},
			},
			"spec",
		),
	)

	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"testField": core.MappingNodeFromString("testValue"),
		},
	}

	target := &testTarget{}

	setter.Set(specData, target)

	s.Assert().False(setter.DidSet())
	s.Assert().Equal("", target.FieldValue)
}

func (s *ValueSetterSuite) Test_sets_value_if_in_modified_fields() {
	setter := NewValueSetter(
		"$[\"testField.name\"]",
		func(value *core.MappingNode, target *testTarget) {
			target.FieldValue = core.StringValue(value)
		},
		WithValueSetterCheckIfChanged[*testTarget](true),
		WithValueSetterModifiedFields[*testTarget](
			[]provider.FieldChange{
				{
					FieldPath: "spec[\"testField.name\"]",
					NewValue:  core.MappingNodeFromString("testValue"),
					PrevValue: core.MappingNodeFromString("oldValue"),
				},
			},
			"spec",
		),
	)

	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"testField.name": core.MappingNodeFromString("testValue"),
		},
	}

	target := &testTarget{}

	setter.Set(specData, target)

	s.Assert().True(setter.DidSet())
	s.Assert().Equal("testValue", target.FieldValue)
}

func (s *ValueSetterSuite) Test_sets_value_if_in_modified_fields_nested() {
	setter := NewValueSetter(
		"$[\"test.nested.field\"]",
		func(value *core.MappingNode, target *testTarget) {
			target.FieldValue = core.StringValue(value)
		},
		WithValueSetterCheckIfChanged[*testTarget](true),
		WithValueSetterModifiedFields[*testTarget](
			[]provider.FieldChange{
				{
					// Relative to a root object 2 levels up.
					FieldPath: "spec.nestedConfig.values[\"test.nested.field\"]",
					NewValue:  core.MappingNodeFromString("testValue"),
					PrevValue: core.MappingNodeFromString("oldValue"),
				},
			},
			"spec.nestedConfig.values",
		),
	)

	specData := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"test.nested.field": core.MappingNodeFromString("testValue"),
		},
	}

	target := &testTarget{}

	setter.Set(specData, target)

	s.Assert().True(setter.DidSet())
	s.Assert().Equal("testValue", target.FieldValue)
}

func TestValueSetterSuite(t *testing.T) {
	suite.Run(t, new(ValueSetterSuite))
}
