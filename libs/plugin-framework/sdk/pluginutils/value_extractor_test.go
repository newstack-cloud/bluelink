package pluginutils

import (
	"context"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ValueExtractorSuite struct {
	suite.Suite
}

type testInput struct {
	topLevelField  string
	optionalConfig *testOptionalConfig
}

type testOptionalConfig struct {
	value1 string
	value2 bool
	value3 int
}

type optionalValueExtractorTestCase struct {
	name           string
	input          *testInput
	expectedOutput map[string]*core.MappingNode
}

func (s *ValueExtractorSuite) Test_optional_value_extraction() {
	inputs := []optionalValueExtractorTestCase{
		{
			name: "Test with optional config",
			input: &testInput{
				topLevelField: "test1",
				optionalConfig: &testOptionalConfig{
					value1: "value1",
					value2: true,
					value3: 42,
				},
			},
			expectedOutput: map[string]*core.MappingNode{
				"topLevelField":         core.MappingNodeFromString("test1"),
				"optionalConfig.value1": core.MappingNodeFromString("value1"),
				"optionalConfig.value2": core.MappingNodeFromBool(true),
				"optionalConfig.value3": core.MappingNodeFromInt(42),
			},
		},
		{
			name: "Test without optional config",
			input: &testInput{
				topLevelField:  "test2",
				optionalConfig: nil,
			},
			expectedOutput: map[string]*core.MappingNode{
				"topLevelField": core.MappingNodeFromString("test2"),
			},
		},
	}

	extractors := []OptionalValueExtractor[*testInput]{
		{
			Name:      "Top-level field",
			Condition: func(input *testInput) bool { return input.topLevelField != "" },
			Fields:    []string{"topLevelField"},
			Values: func(input *testInput) ([]*core.MappingNode, error) {
				return []*core.MappingNode{
					core.MappingNodeFromString(input.topLevelField),
				}, nil
			},
		},
		{
			Name:      "Optional config value1",
			Condition: func(input *testInput) bool { return input.optionalConfig != nil },
			Fields:    []string{"optionalConfig.value1", "optionalConfig.value2", "optionalConfig.value3"},
			Values: func(input *testInput) ([]*core.MappingNode, error) {
				return []*core.MappingNode{
					core.MappingNodeFromString(input.optionalConfig.value1),
					core.MappingNodeFromBool(input.optionalConfig.value2),
					core.MappingNodeFromInt(input.optionalConfig.value3),
				}, nil
			},
		},
	}

	for _, tc := range inputs {
		targetMap := map[string]*core.MappingNode{}
		s.Run(tc.name, func() {
			err := RunOptionalValueExtractors(
				tc.input,
				targetMap,
				extractors,
			)
			s.Require().NoError(err)
			s.Assert().Equal(tc.expectedOutput, targetMap)
		})
	}
}

type serviceMock struct{}

type additionalValueExtractorTestCase struct {
	name           string
	extractors     []AdditionalValueExtractor[serviceMock]
	expectedOutput map[string]*core.MappingNode
	expectError    bool
}

func (s *ValueExtractorSuite) Test_additional_value_extraction() {

	testCases := []additionalValueExtractorTestCase{
		{
			name: "Test with valid extractors",
			expectedOutput: map[string]*core.MappingNode{
				"value1": core.MappingNodeFromString("extractedValue1"),
				"value2": core.MappingNodeFromBool(true),
			},
			expectError: false,
			extractors: []AdditionalValueExtractor[serviceMock]{
				{
					Name: "Extract value1",
					Extract: func(
						ctx context.Context,
						filters *provider.ResolvedDataSourceFilters,
						targetData map[string]*core.MappingNode,
						service serviceMock,
					) error {
						targetData["value1"] = core.MappingNodeFromString("extractedValue1")
						return nil
					},
				},
				{
					Name: "Extract value2",
					Extract: func(
						ctx context.Context,
						filters *provider.ResolvedDataSourceFilters,
						targetData map[string]*core.MappingNode,
						service serviceMock,
					) error {
						targetData["value2"] = core.MappingNodeFromBool(true)
						return nil
					},
				},
			},
		},
		{
			name:        "Test with error in extractor",
			expectError: true,
			extractors: []AdditionalValueExtractor[serviceMock]{
				{
					Name: "Extract value1 with error",
					Extract: func(
						ctx context.Context,
						filters *provider.ResolvedDataSourceFilters,
						targetData map[string]*core.MappingNode,
						service serviceMock,
					) error {
						return errors.New("failed to extract value1")
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			targetMap := make(map[string]*core.MappingNode)
			filters := &provider.ResolvedDataSourceFilters{}

			err := RunAdditionalValueExtractors(
				context.Background(),
				filters,
				targetMap,
				tc.extractors,
				serviceMock{},
			)

			if tc.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Assert().Equal(tc.expectedOutput, targetMap)
			}
		})
	}
}

func TestValueExtractorSuite(t *testing.T) {
	suite.Run(t, new(ValueExtractorSuite))
}
