package version

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConstraintTestSuite struct {
	suite.Suite
}

func (s *ConstraintTestSuite) Test_ParseConstraint_exact() {
	c, err := ParseConstraint("1.2.3")
	s.NoError(err)
	s.Equal(ConstraintExact, c.Type)
	s.Equal(1, c.Version.Major)
	s.Equal(2, c.Version.Minor)
	s.Equal(3, c.Version.Patch)
}

func (s *ConstraintTestSuite) Test_ParseConstraint_caret() {
	c, err := ParseConstraint("^1.2.3")
	s.NoError(err)
	s.Equal(ConstraintCaret, c.Type)
	s.Equal(1, c.Version.Major)
	s.Equal(2, c.Version.Minor)
	s.Equal(3, c.Version.Patch)
}

func (s *ConstraintTestSuite) Test_ParseConstraint_tilde() {
	c, err := ParseConstraint("~1.2.3")
	s.NoError(err)
	s.Equal(ConstraintTilde, c.Type)
	s.Equal(1, c.Version.Major)
	s.Equal(2, c.Version.Minor)
	s.Equal(3, c.Version.Patch)
}

func (s *ConstraintTestSuite) Test_ParseConstraint_with_prerelease() {
	c, err := ParseConstraint("^1.2.3-beta.1")
	s.NoError(err)
	s.Equal(ConstraintCaret, c.Type)
	s.Equal("beta.1", c.Version.Prerelease)
}

func (s *ConstraintTestSuite) Test_ParseConstraint_invalid() {
	tests := []struct {
		input       string
		errContains string
	}{
		{"", "cannot be empty"},
		{"^", "version cannot be empty after constraint prefix"},
		{"~", "version cannot be empty after constraint prefix"},
		{"^1.0", "must have exactly 3 parts"},
		{"~a.b.c", "invalid major version"},
		{"1.x.0", "invalid minor version"},
	}
	for _, tt := range tests {
		_, err := ParseConstraint(tt.input)
		s.Error(err, "expected error for: %s", tt.input)
		s.Contains(err.Error(), tt.errContains, "error message for: %s", tt.input)
	}
}

func (s *ConstraintTestSuite) Test_String() {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3", "1.2.3"},
		{"^1.2.3", "^1.2.3"},
		{"~1.2.3", "~1.2.3"},
		{"^1.0.0-alpha", "^1.0.0-alpha"},
	}
	for _, tt := range tests {
		c, err := ParseConstraint(tt.input)
		s.NoError(err)
		s.Equal(tt.expected, c.String())
	}
}

func (s *ConstraintTestSuite) Test_IsExact() {
	exactConstraint, _ := ParseConstraint("1.2.3")
	s.True(exactConstraint.IsExact())

	caretConstraint, _ := ParseConstraint("^1.2.3")
	s.False(caretConstraint.IsExact())

	tildeConstraint, _ := ParseConstraint("~1.2.3")
	s.False(tildeConstraint.IsExact())
}

func (s *ConstraintTestSuite) Test_Matches_exact() {
	c, _ := ParseConstraint("1.2.3")

	s.True(c.Matches(mustParse("1.2.3")))
	s.False(c.Matches(mustParse("1.2.4")))
	s.False(c.Matches(mustParse("1.2.2")))
	s.False(c.Matches(mustParse("1.3.0")))
	s.False(c.Matches(mustParse("2.0.0")))
	s.False(c.Matches(mustParse("0.0.0")))
}

func (s *ConstraintTestSuite) Test_Matches_caret_same_major() {
	c, _ := ParseConstraint("^1.2.3")

	// Matches: >=1.2.3, <2.0.0
	s.True(c.Matches(mustParse("1.2.3")), "exact match")
	s.True(c.Matches(mustParse("1.2.4")), "higher patch")
	s.True(c.Matches(mustParse("1.3.0")), "higher minor")
	s.True(c.Matches(mustParse("1.9.9")), "high minor and patch")
	s.True(c.Matches(mustParse("1.99.99")), "very high minor and patch")
}

func (s *ConstraintTestSuite) Test_Matches_caret_excludes() {
	c, _ := ParseConstraint("^1.2.3")

	s.False(c.Matches(mustParse("1.2.2")), "lower patch")
	s.False(c.Matches(mustParse("1.1.9")), "lower minor")
	s.False(c.Matches(mustParse("2.0.0")), "major bump")
	s.False(c.Matches(mustParse("0.9.9")), "different major")
	s.False(c.Matches(mustParse("3.0.0")), "higher major")
}

func (s *ConstraintTestSuite) Test_Matches_caret_zero_major() {
	// For major version 0, caret still allows minor bumps within 0.x
	c, _ := ParseConstraint("^0.2.3")

	s.True(c.Matches(mustParse("0.2.3")))
	s.True(c.Matches(mustParse("0.2.4")))
	s.True(c.Matches(mustParse("0.3.0")))
	s.False(c.Matches(mustParse("0.2.2")))
	s.False(c.Matches(mustParse("1.0.0")))
}

func (s *ConstraintTestSuite) Test_Matches_tilde_same_minor() {
	c, _ := ParseConstraint("~1.2.3")

	// Matches: >=1.2.3, <1.3.0
	s.True(c.Matches(mustParse("1.2.3")), "exact match")
	s.True(c.Matches(mustParse("1.2.4")), "higher patch")
	s.True(c.Matches(mustParse("1.2.99")), "high patch")
}

func (s *ConstraintTestSuite) Test_Matches_tilde_excludes() {
	c, _ := ParseConstraint("~1.2.3")

	s.False(c.Matches(mustParse("1.2.2")), "lower patch")
	s.False(c.Matches(mustParse("1.3.0")), "minor bump")
	s.False(c.Matches(mustParse("1.1.9")), "lower minor")
	s.False(c.Matches(mustParse("2.0.0")), "major bump")
	s.False(c.Matches(mustParse("0.2.3")), "different major")
}

func (s *ConstraintTestSuite) Test_Matches_tilde_zero_versions() {
	c, _ := ParseConstraint("~0.1.0")

	s.True(c.Matches(mustParse("0.1.0")))
	s.True(c.Matches(mustParse("0.1.5")))
	s.False(c.Matches(mustParse("0.2.0")))
	s.False(c.Matches(mustParse("1.0.0")))
}

func (s *ConstraintTestSuite) Test_FindBestMatch_caret() {
	available := []*Version{
		mustParse("1.0.0"),
		mustParse("1.1.0"),
		mustParse("1.2.0"),
		mustParse("1.2.5"),
		mustParse("2.0.0"),
	}

	c, _ := ParseConstraint("^1.0.0")
	best := c.FindBestMatch(available)
	s.NotNil(best)
	s.Equal("1.2.5", best.String())
}

func (s *ConstraintTestSuite) Test_FindBestMatch_tilde() {
	available := []*Version{
		mustParse("1.0.0"),
		mustParse("1.1.0"),
		mustParse("1.1.5"),
		mustParse("1.2.0"),
		mustParse("2.0.0"),
	}

	c, _ := ParseConstraint("~1.1.0")
	best := c.FindBestMatch(available)
	s.NotNil(best)
	s.Equal("1.1.5", best.String())
}

func (s *ConstraintTestSuite) Test_FindBestMatch_exact() {
	available := []*Version{
		mustParse("1.0.0"),
		mustParse("1.1.0"),
		mustParse("1.2.0"),
	}

	c, _ := ParseConstraint("1.1.0")
	best := c.FindBestMatch(available)
	s.NotNil(best)
	s.Equal("1.1.0", best.String())
}

func (s *ConstraintTestSuite) Test_FindBestMatch_no_match() {
	available := []*Version{
		mustParse("1.0.0"),
		mustParse("1.1.0"),
	}

	c, _ := ParseConstraint("^3.0.0")
	best := c.FindBestMatch(available)
	s.Nil(best)
}

func (s *ConstraintTestSuite) Test_FindBestMatch_empty_list() {
	c, _ := ParseConstraint("^1.0.0")
	best := c.FindBestMatch([]*Version{})
	s.Nil(best)
}

func (s *ConstraintTestSuite) Test_FindBestMatch_prefers_release_over_prerelease() {
	available := []*Version{
		mustParse("1.0.0-alpha"),
		mustParse("1.0.0-beta"),
		mustParse("1.0.0"),
		mustParse("1.0.1-rc.1"),
	}

	c, _ := ParseConstraint("^1.0.0-alpha")
	best := c.FindBestMatch(available)
	s.NotNil(best)
	// 1.0.1-rc.1 is highest because 1.0.1 > 1.0.0, even with prerelease
	s.Equal("1.0.1-rc.1", best.String())
}

func mustParse(s string) *Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestConstraintTestSuite(t *testing.T) {
	suite.Run(t, new(ConstraintTestSuite))
}
