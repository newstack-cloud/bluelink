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

func (s *MappingNodeUtilSuite) Test_any_to_mapping_node() {
	testCases := []struct {
		name     string
		input    any
		expected *core.MappingNode
	}{
		{
			name: "map to MappingNode",
			input: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			expected: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"key1": core.MappingNodeFromString("value1"),
					"key2": core.MappingNodeFromInt(42),
				},
			},
		},
		{
			name: "slice to MappingNode",
			input: []any{
				map[string]any{"item1": "value1"},
				map[string]any{"item2": 3.14},
			},
			expected: &core.MappingNode{
				Items: []*core.MappingNode{
					{
						Fields: map[string]*core.MappingNode{
							"item1": core.MappingNodeFromString("value1"),
						},
					},
					{
						Fields: map[string]*core.MappingNode{
							"item2": core.MappingNodeFromFloat(3.14),
						},
					},
				},
			},
		},
		{
			name:     "string to MappingNode",
			input:    "simple string",
			expected: core.MappingNodeFromString("simple string"),
		},
		{
			name:     "int to MappingNode",
			input:    123,
			expected: core.MappingNodeFromInt(123),
		},
		{
			name:     "int32 to MappingNode",
			input:    int32(123),
			expected: core.MappingNodeFromInt(123),
		},
		{
			name:     "int64 to MappingNode",
			input:    int64(59483),
			expected: core.MappingNodeFromInt(59483),
		},
		{
			name:     "float64 to MappingNode",
			input:    float64(123.456),
			expected: core.MappingNodeFromFloat(123.456),
		},
		{
			name:     "bool to MappingNode",
			input:    true,
			expected: core.MappingNodeFromBool(true),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := AnyToMappingNode(tc.input)
			s.Assert().NoError(err, "Expected no error during conversion")
			s.Assert().Equal(tc.expected, result, "Expected converted MappingNode to match")
		})
	}
}

func TestMappingNodeUtilSuite(t *testing.T) {
	suite.Run(t, new(MappingNodeUtilSuite))
}
