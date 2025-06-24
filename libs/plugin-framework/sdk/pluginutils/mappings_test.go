package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type MappingNodeUtilSuite struct {
	suite.Suite
}

type getValueByPathTestCase struct {
	name          string
	path          string
	input         *core.MappingNode
	expectedValue *core.MappingNode
	expectedFound bool
}

func (s *MappingNodeUtilSuite) Test_get_value_by_path() {
	testCases := []getValueByPathTestCase{
		{
			name: "valid path",
			input: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
					"field2": core.MappingNodeFromString("value2"),
				},
			},
			path:          "$.field1",
			expectedValue: core.MappingNodeFromString("value1"),
			expectedFound: true,
		},
		{
			name: "invalid path",
			input: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
					"field2": core.MappingNodeFromString("value2"),
				},
			},
			path:          "$$$$$.>Field323",
			expectedValue: nil,
			expectedFound: false,
		},
		{
			name: "non-existent field",
			input: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
					"field2": core.MappingNodeFromString("value2"),
				},
			},
			path:          "$.field3",
			expectedValue: nil,
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, found := GetValueByPath(tc.path, tc.input)
			s.Assert().Equal(tc.expectedFound, found, "Expected found to be %v", tc.expectedFound)
			if tc.expectedValue != nil {
				s.Assert().Equal(tc.expectedValue.Fields, value.Fields, "Expected value to match")
			} else {
				s.Assert().Nil(value, "Expected value to be nil")
			}
		})
	}
}

func (s *MappingNodeUtilSuite) Test_shallow_copy() {
	testCases := []struct {
		name       string
		input      map[string]*core.MappingNode
		ignoreKeys []string
		expected   map[string]*core.MappingNode
	}{
		{
			name: "copy with ignored keys",
			input: map[string]*core.MappingNode{
				"field1": core.MappingNodeFromString("value1"),
				"field2": core.MappingNodeFromString("value2"),
				"field3": core.MappingNodeFromString("value3"),
			},
			ignoreKeys: []string{"field2"},
			expected: map[string]*core.MappingNode{
				"field1": core.MappingNodeFromString("value1"),
				"field3": core.MappingNodeFromString("value3"),
			},
		},
		{
			name: "copy without ignored keys",
			input: map[string]*core.MappingNode{
				"field1": core.MappingNodeFromString("value1"),
				"field2": core.MappingNodeFromString("value2"),
			},
			ignoreKeys: []string{},
			expected: map[string]*core.MappingNode{
				"field1": core.MappingNodeFromString("value1"),
				"field2": core.MappingNodeFromString("value2"),
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := ShallowCopy(tc.input, tc.ignoreKeys...)
			s.Assert().Equal(tc.expected, result, "Expected shallow copy to match")
		})
	}
}

func TestMappingNodeUtilSuite(t *testing.T) {
	suite.Run(t, new(MappingNodeUtilSuite))
}
