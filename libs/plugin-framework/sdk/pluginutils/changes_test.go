package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ChangesTestSuite struct {
	suite.Suite
}

type getCurrentResourceStateSpecDataTestCase struct {
	name             string
	inputChanges     *provider.Changes
	expectedSpecData *core.MappingNode
	expectedEmpty    bool
}

func (s *ChangesTestSuite) Test_get_current_resource_state_spec_data() {
	testCases := []getCurrentResourceStateSpecDataTestCase{
		{
			name:         "nil changes",
			inputChanges: nil,
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "no current resource state",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "valid current resource state",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					CurrentResourceState: &state.ResourceState{
						SpecData: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"field1": core.MappingNodeFromString("value1"),
							},
						},
					},
				},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
				},
			},
			expectedEmpty: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetCurrentResourceStateSpecData(tc.inputChanges)
			s.Assert().Equal(tc.expectedSpecData, result)
			if tc.expectedEmpty {
				s.Assert().Len(result.Fields, 0)
			} else {
				s.Assert().NotEmpty(result.Fields)
			}
		})
	}
}

type getSpecDataFromResourceInfoTestCase struct {
	name             string
	resourceInfo     *provider.ResourceInfo
	expectedSpecData *core.MappingNode
	expectedEmpty    bool
}

func (s *ChangesTestSuite) Test_get_spec_data_from_resource_info() {
	testCases := []getSpecDataFromResourceInfoTestCase{
		{
			name:         "nil resource info",
			resourceInfo: nil,
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name:         "no current resource state",
			resourceInfo: &provider.ResourceInfo{},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "valid current resource state",
			resourceInfo: &provider.ResourceInfo{
				CurrentResourceState: &state.ResourceState{
					SpecData: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"field1": core.MappingNodeFromString("value1"),
						},
					},
				},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
				},
			},
			expectedEmpty: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetCurrentStateSpecDataFromResourceInfo(tc.resourceInfo)
			s.Assert().Equal(tc.expectedSpecData, result)
			if tc.expectedEmpty {
				s.Assert().Len(result.Fields, 0)
			} else {
				s.Assert().NotEmpty(result.Fields)
			}
		})
	}
}

type getResolvedResourceSpecDataTestCase struct {
	name             string
	inputChanges     *provider.Changes
	expectedSpecData *core.MappingNode
	expectedEmpty    bool
}

func (s *ChangesTestSuite) Test_get_resolved_resource_spec_data() {
	testCases := []getResolvedResourceSpecDataTestCase{
		{
			name:         "nil changes",
			inputChanges: nil,
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "no resolved resource",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "valid resolved resource",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Spec: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"field1": core.MappingNodeFromString("value1"),
							},
						},
					},
				},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
				},
			},
			expectedEmpty: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetResolvedResourceSpecData(tc.inputChanges)
			s.Assert().Equal(tc.expectedSpecData, result)
			if tc.expectedEmpty {
				s.Assert().Len(result.Fields, 0)
			} else {
				s.Assert().NotEmpty(result.Fields)
			}
		})
	}
}

func (s *ChangesTestSuite) Test_has_modified_fields() {
	testCases := []struct {
		name      string
		changes   *provider.Changes
		fieldPath string
		expected  bool
	}{
		{
			name:      "nil changes",
			changes:   nil,
			fieldPath: "some.field",
			expected:  false,
		},
		{
			name: "no modified fields",
			changes: &provider.Changes{
				ModifiedFields: nil,
			},
			fieldPath: "some.field",
			expected:  false,
		},
		{
			name: "field not modified",
			changes: &provider.Changes{
				ModifiedFields: []provider.FieldChange{},
			},
			fieldPath: "some.field",
			expected:  false,
		},
		{
			name: "field modified",
			changes: &provider.Changes{
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "some.field"},
				},
			},
			fieldPath: "some.field",
			expected:  true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := HasModifiedField(tc.changes, tc.fieldPath)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func (s *ChangesTestSuite) Test_get_modified_field() {
	testCases := []struct {
		name      string
		changes   *provider.Changes
		fieldPath string
		expected  *provider.FieldChange
	}{
		{
			name:      "nil changes",
			changes:   nil,
			fieldPath: "some.field",
			expected:  nil,
		},
		{
			name: "no modified fields",
			changes: &provider.Changes{
				ModifiedFields: nil,
			},
			fieldPath: "some.field",
			expected:  nil,
		},
		{
			name: "field not modified",
			changes: &provider.Changes{
				ModifiedFields: []provider.FieldChange{},
			},
			fieldPath: "some.field",
			expected:  nil,
		},
		{
			name: "field modified",
			changes: &provider.Changes{
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "some.field", NewValue: core.MappingNodeFromString("newValue")},
				},
			},
			fieldPath: "some.field",
			expected: &provider.FieldChange{
				FieldPath: "some.field",
				NewValue:  core.MappingNodeFromString("newValue"),
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetModifiedField(tc.changes, tc.fieldPath)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func (s *ChangesTestSuite) Test_has_new_fields() {
	testCases := []struct {
		name      string
		changes   *provider.Changes
		fieldPath string
		expected  bool
	}{
		{
			name:      "nil changes",
			changes:   nil,
			fieldPath: "some.field",
			expected:  false,
		},
		{
			name: "no new fields",
			changes: &provider.Changes{
				NewFields: nil,
			},
			fieldPath: "some.field",
			expected:  false,
		},
		{
			name: "field not new",
			changes: &provider.Changes{
				NewFields: []provider.FieldChange{},
			},
			fieldPath: "some.field",
			expected:  false,
		},
		{
			name: "field new",
			changes: &provider.Changes{
				NewFields: []provider.FieldChange{
					{FieldPath: "some.field"},
				},
			},
			fieldPath: "some.field",
			expected:  true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := HasNewField(tc.changes, tc.fieldPath)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func (s *ChangesTestSuite) Test_get_new_field() {
	testCases := []struct {
		name      string
		changes   *provider.Changes
		fieldPath string
		expected  *provider.FieldChange
	}{
		{
			name:      "nil changes",
			changes:   nil,
			fieldPath: "some.field",
			expected:  nil,
		},
		{
			name: "no new fields",
			changes: &provider.Changes{
				NewFields: nil,
			},
			fieldPath: "some.field",
			expected:  nil,
		},
		{
			name: "field not new",
			changes: &provider.Changes{
				NewFields: []provider.FieldChange{},
			},
			fieldPath: "some.field",
			expected:  nil,
		},
		{
			name: "field new",
			changes: &provider.Changes{
				NewFields: []provider.FieldChange{
					{FieldPath: "some.field", NewValue: core.MappingNodeFromString("newValue")},
				},
			},
			fieldPath: "some.field",
			expected: &provider.FieldChange{
				FieldPath: "some.field",
				NewValue:  core.MappingNodeFromString("newValue"),
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetNewField(tc.changes, tc.fieldPath)
			s.Assert().Equal(tc.expected, result)
		})
	}
}

func TestChangesTestSuite(t *testing.T) {
	suite.Run(t, new(ChangesTestSuite))
}
