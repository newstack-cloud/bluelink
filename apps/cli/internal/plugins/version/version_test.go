package version

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type VersionTestSuite struct {
	suite.Suite
}

func (s *VersionTestSuite) Test_Parse_valid_versions() {
	tests := []struct {
		input    string
		expected *Version
	}{
		{"1.0.0", &Version{Major: 1, Minor: 0, Patch: 0}},
		{"2.3.4", &Version{Major: 2, Minor: 3, Patch: 4}},
		{"0.1.0", &Version{Major: 0, Minor: 1, Patch: 0}},
		{"10.20.30", &Version{Major: 10, Minor: 20, Patch: 30}},
		{"1.2.3-alpha", &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha"}},
		{"1.2.3-beta.1", &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta.1"}},
		{"0.0.0", &Version{Major: 0, Minor: 0, Patch: 0}},
		{"1.0.0-rc.1", &Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1"}},
	}
	for _, tt := range tests {
		v, err := Parse(tt.input)
		s.NoError(err, "input: %s", tt.input)
		s.Equal(tt.expected.Major, v.Major, "major for %s", tt.input)
		s.Equal(tt.expected.Minor, v.Minor, "minor for %s", tt.input)
		s.Equal(tt.expected.Patch, v.Patch, "patch for %s", tt.input)
		s.Equal(tt.expected.Prerelease, v.Prerelease, "prerelease for %s", tt.input)
	}
}

func (s *VersionTestSuite) Test_Parse_invalid_versions() {
	tests := []struct {
		input       string
		errContains string
	}{
		{"", "cannot be empty"},
		{"1", "must have exactly 3 parts"},
		{"1.0", "must have exactly 3 parts"},
		{"1.0.0.0", "must have exactly 3 parts"},
		{"a.b.c", "invalid major version"},
		{"1.b.c", "invalid minor version"},
		{"1.0.c", "invalid patch version"},
		{"1.0.0-", "prerelease suffix cannot be empty"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.input)
		s.Error(err, "expected error for: %s", tt.input)
		s.Contains(err.Error(), tt.errContains, "error message for: %s", tt.input)
	}
}

func (s *VersionTestSuite) Test_String() {
	tests := []struct {
		version  *Version
		expected string
	}{
		{&Version{Major: 1, Minor: 0, Patch: 0}, "1.0.0"},
		{&Version{Major: 2, Minor: 3, Patch: 4}, "2.3.4"},
		{&Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha"}, "1.2.3-alpha"},
		{&Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "beta.1"}, "1.0.0-beta.1"},
	}
	for _, tt := range tests {
		s.Equal(tt.expected, tt.version.String())
	}
}

func (s *VersionTestSuite) Test_Compare_major_differences() {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"3.0.0", "3.0.0", 0},
	}
	for _, tt := range tests {
		a, _ := Parse(tt.a)
		b, _ := Parse(tt.b)
		s.Equal(tt.expected, a.Compare(b), "%s vs %s", tt.a, tt.b)
	}
}

func (s *VersionTestSuite) Test_Compare_minor_differences() {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "1.1.0", -1},
		{"1.2.0", "1.1.0", 1},
		{"1.5.0", "1.5.0", 0},
	}
	for _, tt := range tests {
		a, _ := Parse(tt.a)
		b, _ := Parse(tt.b)
		s.Equal(tt.expected, a.Compare(b), "%s vs %s", tt.a, tt.b)
	}
}

func (s *VersionTestSuite) Test_Compare_patch_differences() {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "1.0.1", -1},
		{"1.0.2", "1.0.1", 1},
		{"1.0.5", "1.0.5", 0},
	}
	for _, tt := range tests {
		a, _ := Parse(tt.a)
		b, _ := Parse(tt.b)
		s.Equal(tt.expected, a.Compare(b), "%s vs %s", tt.a, tt.b)
	}
}

func (s *VersionTestSuite) Test_Compare_prerelease() {
	tests := []struct {
		a, b     string
		expected int
	}{
		// Prerelease < release
		{"1.2.3-alpha", "1.2.3", -1},
		{"1.2.3", "1.2.3-alpha", 1},
		// Lexical comparison of prereleases
		{"1.2.3-alpha", "1.2.3-beta", -1},
		{"1.2.3-beta", "1.2.3-alpha", 1},
		{"1.2.3-alpha", "1.2.3-alpha", 0},
		// Both without prerelease
		{"1.2.3", "1.2.3", 0},
	}
	for _, tt := range tests {
		a, _ := Parse(tt.a)
		b, _ := Parse(tt.b)
		s.Equal(tt.expected, a.Compare(b), "%s vs %s", tt.a, tt.b)
	}
}

func (s *VersionTestSuite) Test_LessThan() {
	a, _ := Parse("1.0.0")
	b, _ := Parse("2.0.0")
	c, _ := Parse("1.0.0")

	s.True(a.LessThan(b))
	s.False(b.LessThan(a))
	s.False(a.LessThan(c))
}

func (s *VersionTestSuite) Test_GreaterThanOrEqual() {
	a, _ := Parse("1.0.0")
	b, _ := Parse("2.0.0")
	c, _ := Parse("1.0.0")

	s.False(a.GreaterThanOrEqual(b))
	s.True(b.GreaterThanOrEqual(a))
	s.True(a.GreaterThanOrEqual(c))
}

func TestVersionTestSuite(t *testing.T) {
	suite.Run(t, new(VersionTestSuite))
}
