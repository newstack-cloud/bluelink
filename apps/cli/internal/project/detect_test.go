package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DetectSuite struct {
	suite.Suite
	tempDir string
}

func (s *DetectSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "detect-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *DetectSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *DetectSuite) writeFile(name string) {
	err := os.WriteFile(filepath.Join(s.tempDir, name), []byte("test"), 0644)
	s.Require().NoError(err)
}

func (s *DetectSuite) Test_DetectBlueprintFile_returns_yaml_when_present() {
	s.writeFile("project.blueprint.yaml")

	s.Equal("project.blueprint.yaml", DetectBlueprintFile(s.tempDir))
}

func (s *DetectSuite) Test_DetectBlueprintFile_returns_jsonc_when_present() {
	s.writeFile("project.blueprint.jsonc")

	s.Equal("project.blueprint.jsonc", DetectBlueprintFile(s.tempDir))
}

func (s *DetectSuite) Test_DetectBlueprintFile_returns_bp_when_present() {
	s.writeFile("project.bp")

	s.Equal("project.bp", DetectBlueprintFile(s.tempDir))
}

func (s *DetectSuite) Test_DetectBlueprintFile_prefers_yaml_over_bp() {
	s.writeFile("project.bp")
	s.writeFile("project.blueprint.yaml")

	s.Equal("project.blueprint.yaml", DetectBlueprintFile(s.tempDir))
}

func (s *DetectSuite) Test_DetectBlueprintFile_falls_back_to_default_when_none_present() {
	s.Equal(DefaultBlueprintFile, DetectBlueprintFile(s.tempDir))
}

func TestDetectSuite(t *testing.T) {
	suite.Run(t, new(DetectSuite))
}
