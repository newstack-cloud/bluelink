package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExpandEnvTestSuite struct {
	suite.Suite
}

func (s *ExpandEnvTestSuite) SetupTest() {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("HOME", "/home/user")
	os.Setenv("PATH", "/usr/bin:/usr/local/bin")
	os.Setenv("USERNAME", "testuser")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
}

func (s *ExpandEnvTestSuite) TearDownTest() {
	// Clean up test environment variables
	os.Unsetenv("TEST_VAR")
	os.Unsetenv("HOME")
	os.Unsetenv("PATH")
	os.Unsetenv("USERNAME")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
}

func (s *ExpandEnvTestSuite) Test_expands_single_unix_style_variable() {
	input := "The value is ${TEST_VAR}"
	expected := "The value is test_value"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_single_windows_style_variable() {
	input := "The value is %TEST_VAR%"
	expected := "The value is test_value"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_multiple_unix_style_variables() {
	input := "User ${USERNAME} has home directory ${HOME}"
	expected := "User testuser has home directory /home/user"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_multiple_windows_style_variables() {
	input := "User %USERNAME% has home directory %HOME%"
	expected := "User testuser has home directory /home/user"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_unix_style_variable_at_start_of_string() {
	input := "${HOME}/documents"
	expected := "/home/user/documents"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_windows_style_variable_at_start_of_string() {
	input := "%HOME%/documents"
	expected := "/home/user/documents"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_unix_style_variable_at_end_of_string() {
	input := "Path is ${PATH}"
	expected := "Path is /usr/bin:/usr/local/bin"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_windows_style_variable_at_end_of_string() {
	input := "Path is %PATH%"
	expected := "Path is /usr/bin:/usr/local/bin"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_unix_style_variable_as_entire_string() {
	input := "${TEST_VAR}"
	expected := "test_value"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_windows_style_variable_as_entire_string() {
	input := "%TEST_VAR%"
	expected := "test_value"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_returns_empty_string_for_undefined_unix_style_variable() {
	input := "${UNDEFINED_VAR}"
	expected := ""

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_returns_empty_string_for_undefined_windows_style_variable() {
	input := "%UNDEFINED_VAR%"
	expected := ""

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_variables_in_connection_string_with_windows_style() {
	input := "postgres://%USERNAME%@%DB_HOST%:%DB_PORT%/mydb"
	expected := "postgres://testuser@localhost:5432/mydb"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_variables_in_connection_string_with_unix_style() {
	input := "postgres://${USERNAME}@${DB_HOST}:${DB_PORT}/mydb"
	expected := "postgres://testuser@localhost:5432/mydb"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_handles_string_without_variables() {
	input := "This is a plain string without any variables"
	expected := "This is a plain string without any variables"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_handles_empty_string() {
	input := ""
	expected := ""

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_handles_string_with_literal_percent_signs_not_matching_pattern() {
	input := "Discount is 50% off"
	expected := "Discount is 50% off"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_consecutive_unix_style_variables() {
	input := "${HOME}${PATH}"
	expected := "/home/user/usr/bin:/usr/local/bin"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func (s *ExpandEnvTestSuite) Test_expands_consecutive_windows_style_variables() {
	input := "%HOME%%PATH%"
	expected := "/home/user/usr/bin:/usr/local/bin"

	result := ExpandEnv(input)
	s.Assert().Equal(expected, result)
}

func TestExpandEnvTestSuite(t *testing.T) {
	suite.Run(t, new(ExpandEnvTestSuite))
}
