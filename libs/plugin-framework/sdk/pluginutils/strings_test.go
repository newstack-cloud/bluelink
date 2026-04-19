package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/stretchr/testify/suite"
)

type StringsTestSuite struct {
	suite.Suite
}

func (s *StringsTestSuite) Test_strip_non_alphanumeric_chars() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only alphanumeric characters",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "with spaces and punctuation",
			input:    "Hello, World! 123",
			expected: "HelloWorld123",
		},
		{
			name:     "special characters",
			input:    "@#$$%^&*()_+{}|:\"<>?",
			expected: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := StripNonAlphaNumericChars(tc.input)
			s.Equal(tc.expected, result)
		})
	}
}

func (s *StringsTestSuite) Test_string_to_substitutions() {
	testCases := []struct {
		name     string
		input    string
		expected *substitutions.StringOrSubstitutions
	}{
		{
			name:  "simple string",
			input: "OrderFunction",
			expected: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{StringValue: strPtr("OrderFunction")},
				},
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: &substitutions.StringOrSubstitutions{
				Values: []*substitutions.StringOrSubstitution{
					{StringValue: strPtr("")},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := StringToSubstitutions(tc.input)
			s.Equal(tc.expected, result)
		})
	}
}

func TestStringsTestSuite(t *testing.T) {
	suite.Run(t, new(StringsTestSuite))
}
