package headless

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HeadlessSuite struct {
	suite.Suite
}

func (s *HeadlessSuite) Test_required_with_explicit_value_passes() {
	rule := Required(Flag{Name: "test-flag", Value: "some-value", IsDefault: false})
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_required_with_default_value_returns_error() {
	rule := Required(Flag{Name: "test-flag", Value: "default", IsDefault: true})
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--test-flag")
	s.Contains(err.Error(), "non-interactive")
}

func (s *HeadlessSuite) Test_required_with_empty_value_returns_error() {
	rule := Required(Flag{Name: "test-flag", Value: "", IsDefault: false})
	err := rule.Validate()
	s.Error(err)
}

func (s *HeadlessSuite) Test_one_of_with_one_set_passes() {
	rule := OneOf(
		Flag{Name: "flag-a", Value: "", IsDefault: true},
		Flag{Name: "flag-b", Value: "value", IsDefault: false},
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_one_of_with_none_set_returns_error() {
	rule := OneOf(
		Flag{Name: "flag-a", Value: "", IsDefault: true},
		Flag{Name: "flag-b", Value: "default", IsDefault: true},
	)
	err := rule.Validate()
	s.Error(err)
	s.Contains(err.Error(), "--flag-a")
	s.Contains(err.Error(), "--flag-b")
}

func (s *HeadlessSuite) Test_one_of_with_both_set_passes() {
	rule := OneOf(
		Flag{Name: "flag-a", Value: "value-a", IsDefault: false},
		Flag{Name: "flag-b", Value: "value-b", IsDefault: false},
	)
	err := rule.Validate()
	s.NoError(err)
}

func (s *HeadlessSuite) Test_validate_in_interactive_mode_skips_validation() {
	cleanup := SetHeadlessForTesting(false)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "missing", Value: "", IsDefault: true}),
	)
	s.NoError(err)
}

func (s *HeadlessSuite) Test_validate_in_headless_mode_validates_rules() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "missing", Value: "", IsDefault: true}),
	)
	s.Error(err)
}

func (s *HeadlessSuite) Test_validate_aggregates_multiple_errors() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "flag-a", Value: "", IsDefault: true}),
		Required(Flag{Name: "flag-b", Value: "", IsDefault: true}),
	)
	s.Error(err)
	s.Contains(err.Error(), "--flag-a")
	s.Contains(err.Error(), "--flag-b")
}

func (s *HeadlessSuite) Test_validate_with_all_valid_returns_nil() {
	cleanup := SetHeadlessForTesting(true)
	defer cleanup()

	err := Validate(
		Required(Flag{Name: "flag-a", Value: "value", IsDefault: false}),
	)
	s.NoError(err)
}

func TestHeadlessSuite(t *testing.T) {
	suite.Run(t, new(HeadlessSuite))
}
