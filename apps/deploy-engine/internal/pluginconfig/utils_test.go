package pluginconfig

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type UtilsSuite struct {
	suite.Suite
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_exact_match() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"1.2.3",
	)
	s.Require().NoError(err)
	s.Assert().True(isCompatible)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_minor_range_match() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"^1.2.3",
	)
	s.Require().NoError(err)
	s.Assert().True(isCompatible)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_patch_range_match() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"~1.2.3",
	)
	s.Require().NoError(err)
	s.Assert().True(isCompatible)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_prerelease_version() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3-alpha.2",
		"^1.2.3-alpha.1",
	)
	s.Require().NoError(err)
	s.Assert().True(isCompatible)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_invalid_version_constraint_format() {
	_, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"acf.b.4-alpha.1",
	)
	s.Assert().Error(err)
	s.Assert().Equal(
		"invalid version or constraint format: acf.b.4-alpha.1",
		err.Error(),
	)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_major_mismatch() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"^2.2.3",
	)
	s.Require().NoError(err)
	s.Assert().False(isCompatible)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_minor_mismatch() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"^1.3.3",
	)
	s.Require().NoError(err)
	s.Assert().False(isCompatible)
}

func (s *UtilsSuite) Test_check_plugin_version_compatibility_patch_mismatch() {
	isCompatible, err := CheckPluginVersionCompatibility(
		"1.2.3",
		"~1.2.4",
	)
	s.Require().NoError(err)
	s.Assert().False(isCompatible)
}

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}
