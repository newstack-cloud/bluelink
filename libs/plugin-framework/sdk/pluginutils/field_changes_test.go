package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type FieldChangesSuite struct {
	suite.Suite
}

func (s *FieldChangesSuite) Test_field_changes_to_new_value_map() {
	modifiedFields := []provider.FieldChange{
		{
			FieldPath: "field1",
			NewValue:  core.MappingNodeFromString("newValue1"),
			PrevValue: core.MappingNodeFromString("oldValue1"),
		},
		{
			FieldPath: "field2",
			NewValue:  core.MappingNodeFromString("newValue2"),
			PrevValue: core.MappingNodeFromString("oldValue2"),
		},
	}

	newFields := []provider.FieldChange{
		{
			FieldPath: "field3",
			NewValue:  core.MappingNodeFromString("newValue3"),
		},
		{
			FieldPath: "field4",
			NewValue:  core.MappingNodeFromString("newValue4"),
		},
	}

	expected := map[string]*core.MappingNode{
		"field1": modifiedFields[0].NewValue,
		"field2": modifiedFields[1].NewValue,
		"field3": newFields[0].NewValue,
		"field4": newFields[1].NewValue,
	}

	result := FieldChangesToNewValueMap(modifiedFields, newFields)

	s.Assert().Equal(expected, result)
}

func (s *FieldChangesSuite) Test_field_changes_to_prev_value_map() {
	modifiedFields := []provider.FieldChange{
		{
			FieldPath: "field1",
			NewValue:  core.MappingNodeFromString("newValue1"),
			PrevValue: core.MappingNodeFromString("oldValue1"),
		},
		{
			FieldPath: "field2",
			NewValue:  core.MappingNodeFromString("newValue2"),
			PrevValue: core.MappingNodeFromString("oldValue2"),
		},
	}

	newFields := []provider.FieldChange{
		{
			FieldPath: "field3",
			NewValue:  core.MappingNodeFromString("newValue3"),
		},
		{
			FieldPath: "field4",
			NewValue:  core.MappingNodeFromString("newValue4"),
		},
	}

	expected := map[string]*core.MappingNode{
		"field1": modifiedFields[0].PrevValue,
		"field2": modifiedFields[1].PrevValue,
	}

	result := FieldChangesToPrevValueMap(modifiedFields, newFields)

	s.Assert().Equal(expected, result)
}

func (s *FieldChangesSuite) Test_merge_field_changes() {
	modifiedFields := []provider.FieldChange{
		{
			FieldPath: "field1",
			NewValue:  core.MappingNodeFromString("newValue1"),
			PrevValue: core.MappingNodeFromString("oldValue1"),
		},
	}

	newFields := []provider.FieldChange{
		{
			FieldPath: "field2",
			NewValue:  core.MappingNodeFromString("newValue2"),
		},
	}

	expected := []provider.FieldChange{
		modifiedFields[0],
		newFields[0],
	}

	result := MergeFieldChanges(modifiedFields, newFields)

	s.Assert().Equal(expected, result)
}

func TestFieldChangesSuite(t *testing.T) {
	suite.Run(t, new(FieldChangesSuite))
}
