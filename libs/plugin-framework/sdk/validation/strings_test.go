package validation

import (
	"regexp"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type StringValidationSuite struct {
	suite.Suite
}

func (s *StringValidationSuite) Test_values_in_string_length_range() {
	validValues := []string{
		"hello",
		"world",
		"valid",
		"example",
		"test123",
	}

	for _, value := range validValues {
		diagnostics := StringLengthRange(5, 10)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_values_outside_string_length_range() {
	invalidValues := []string{
		"short",
		"this is a very long string that exceeds the maximum length",
	}

	for _, value := range invalidValues {
		diagnostics := StringLengthRange(6, 10)("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be between 6 and 10 characters")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_length_range() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(42),
		core.ScalarFromFloat(3.26),
	}

	for _, value := range invalidValues {
		diagnostics := StringLengthRange(5, 10)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_max_string_length() {
	validValues := []string{
		"short",
		"medium",
		"exactly10!",
	}

	for _, value := range validValues {
		diagnostics := MaxStringLength(10)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_exceeding_max_string_length() {
	invalidValues := []string{
		"this is too long",
		"exceeds the limit",
	}

	for _, value := range invalidValues {
		diagnostics := MaxStringLength(10)("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be no more than 10 characters long")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_max_string_length() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(320),
		core.ScalarFromFloat(3.26),
	}

	for _, value := range invalidValues {
		diagnostics := MaxStringLength(10)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_min_string_length() {
	validValues := []string{
		"valid string",
		"exactly6",
		"longer than 6",
	}

	for _, value := range validValues {
		diagnostics := MinStringLength(6)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_below_min_string_length() {
	invalidValues := []string{
		"short",
		"tiny",
	}

	for _, value := range invalidValues {
		diagnostics := MinStringLength(6)("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be at least 6 characters long")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_min_string_length() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(43),
		core.ScalarFromFloat(102.14),
	}

	for _, value := range invalidValues {
		diagnostics := MinStringLength(6)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_string_matches_pattern() {
	validValues := []string{
		"valid123",
		"test_456",
		"example-789",
	}

	for _, value := range validValues {
		diagnostics := StringMatchesPattern(
			regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
		)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_string_failures_not_matching_pattern() {
	invalidValues := []string{
		"invalid space",
		"invalid@char",
		"invalid#char",
	}

	for _, value := range invalidValues {
		diagnostics := StringMatchesPattern(
			regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
		)("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must match the pattern")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_pattern() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(301),
		core.ScalarFromFloat(40.26),
	}

	for _, value := range invalidValues {
		diagnostics := StringMatchesPattern(
			regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
		)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_string_does_not_match_pattern() {
	invalidValues := []string{
		"invalid space",
		"invalid@char",
		"invalid#char",
	}

	for _, value := range invalidValues {
		diagnostics := StringDoesNotMatchPattern(
			regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
		)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_string_does_not_match_pattern_failures() {
	validValues := []string{
		"valid123",
		"test_456",
		"example-789",
	}

	for _, value := range validValues {
		diagnostics := StringDoesNotMatchPattern(
			regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
		)("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must not match the pattern")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_does_not_match_pattern() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(412),
		core.ScalarFromFloat(40.26),
	}

	for _, value := range invalidValues {
		diagnostics := StringDoesNotMatchPattern(
			regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
		)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_string_in_list_case_insensitive() {
	validValues := []string{
		"apple",
		"banana",
		"cherry",
	}

	inputValues := []string{
		"apple",
		"BANanA",
		"Cherry",
	}

	for _, value := range inputValues {
		diagnostics := StringInList(
			validValues,
			/* ignoreCase */ true,
		)("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_string_in_list_fails_validation_case_sensitive() {
	validValues := []string{
		"apple",
		"banana",
		"cherry",
	}

	inputValues := []string{
		"apple",
		"BANana", // Should fail
		"Cherry", // Should fail
	}

	for _, value := range inputValues {
		diagnostics := StringInList(
			validValues,
			/* ignoreCase */ false,
		)("exampleField", core.ScalarFromString(value))
		if value == "BANana" || value == "Cherry" {
			s.Assert().NotEmpty(diagnostics)
			s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
			s.Assert().Contains(diagnostics[0].Message, "must be one of apple, banana, cherry")
		} else {
			s.Assert().Empty(diagnostics)
		}
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_in_list() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(42),
		core.ScalarFromFloat(3.26),
	}

	for _, value := range invalidValues {
		diagnostics := StringInList(
			[]string{"apple", "banana", "cherry"},
			/* ignoreCase */ true,
		)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_string_not_in_list_case_insensitive() {
	disallowedValues := []string{
		"apple",
		"banana",
		"cherry",
	}

	inputValues := []string{
		"orange", // Should pass
		"lemon",  // Should pass
		"Banana", // Should fail
	}

	for _, value := range inputValues {
		diagnostics := StringNotInList(
			disallowedValues,
			/* ignoreCase */ true,
		)("exampleField", core.ScalarFromString(value))
		if value == "Banana" {
			s.Assert().NotEmpty(diagnostics)
			s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
			s.Assert().Contains(diagnostics[0].Message, "must not be one of apple, banana, cherry")
		} else {
			s.Assert().Empty(diagnostics)
		}
	}
}

func (s *StringValidationSuite) Test_string_not_in_list_case_sensitive() {
	disallowedValues := []string{
		"apple",
		"banana",
		"cherry",
	}

	inputValues := []string{
		"orange", // Should pass
		"lemon",  // Should pass
		"Banana", // Should pass
		"apple",  // Should fail
	}

	for _, value := range inputValues {
		diagnostics := StringNotInList(
			disallowedValues,
			/* ignoreCase */ false,
		)("exampleField", core.ScalarFromString(value))
		if value == "apple" {
			s.Assert().NotEmpty(diagnostics)
			s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
			s.Assert().Contains(diagnostics[0].Message, "must not be one of apple, banana, cherry")
		} else {
			s.Assert().Empty(diagnostics)
		}
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_not_in_list() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1930),
		core.ScalarFromFloat(24.11),
	}

	for _, value := range invalidValues {
		diagnostics := StringNotInList(
			[]string{"apple", "banana", "cherry"},
			/* ignoreCase */ true,
		)("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_does_not_contain_chars() {
	disallowedChars := "$%&"

	validValues := []string{
		"validString",
		"anotherValidString",
	}

	for _, value := range validValues {
		diagnostics := StringDoesNotContainChars(disallowedChars)(
			"exampleField",
			core.ScalarFromString(value),
		)
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_contains_disallowed_chars() {
	disallowedChars := "$%&"

	invalidValues := []string{
		"invalid$String",
		"anotherInvalid%String",
		"yetAnother&InvalidString",
	}

	for _, value := range invalidValues {
		diagnostics := StringDoesNotContainChars(disallowedChars)(
			"exampleField",
			core.ScalarFromString(value),
		)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must not contain any of \"$%&\"")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_does_not_contain_chars() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromInt(1234),
		core.ScalarFromFloat(56.78),
	}

	for _, value := range invalidValues {
		diagnostics := StringDoesNotContainChars("$%&")("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_string_is_base64() {
	validValues := []string{
		"SGVsbG8sIFdvcmxkIQ==", // "Hello, World!" as a base64-encoded string
		"U29tZSBzdHJpbmc=",     // "Some string" as a base64-encoded string
	}

	for _, value := range validValues {
		diagnostics := StringIsBase64()("exampleField", core.ScalarFromString(value))
		s.Assert().Empty(diagnostics)
	}
}

func (s *StringValidationSuite) Test_invalid_base64_strings() {
	invalidValues := []string{
		"not-a-base64-string",
		"SGVsbG8sIFdvcmxkIQ", // Missing padding
		"U29tZSBzdHJpbmc",    // Missing padding
	}

	for _, value := range invalidValues {
		diagnostics := StringIsBase64()("exampleField", core.ScalarFromString(value))
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a valid base64-encoded string")
	}
}

func (s *StringValidationSuite) Test_invalid_type_for_string_is_base64() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromBool(false),
		core.ScalarFromFloat(56.78),
	}

	for _, value := range invalidValues {
		diagnostics := StringIsBase64()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func (s *StringValidationSuite) Test_json_strings() {
	validJSON := `{"key": "value", "number": 123}`
	invalidJSON := `{"key": "value", "number": }` // Invalid JSON

	// Valid JSON string
	diagnostics := StringIsJSON()("exampleField", core.ScalarFromString(validJSON))
	s.Assert().Empty(diagnostics)

	// Invalid JSON string
	diagnostics = StringIsJSON()("exampleField", core.ScalarFromString(invalidJSON))
	s.Assert().NotEmpty(diagnostics)
	s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
	s.Assert().Contains(diagnostics[0].Message, "must be valid JSON")
}

func (s *StringValidationSuite) Test_invalid_type_for_string_is_json() {
	invalidValues := []*core.ScalarValue{
		core.ScalarFromBool(false),
		core.ScalarFromFloat(123.45),
	}

	for _, value := range invalidValues {
		diagnostics := StringIsJSON()("exampleField", value)
		s.Assert().NotEmpty(diagnostics)
		s.Assert().Equal(core.DiagnosticLevelError, diagnostics[0].Level)
		s.Assert().Contains(diagnostics[0].Message, "must be a string")
	}
}

func TestStringValidationSuite(t *testing.T) {
	suite.Run(t, new(StringValidationSuite))
}
