package schema

import (
	"os"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type LoadTestSuite struct {
	specFixtures map[string]fixture
	suite.Suite
}

type fixture struct {
	filePath  string
	stringVal string
}

func (s *LoadTestSuite) SetupSuite() {
	s.specFixtures = make(map[string]fixture)
	fixturesToLoad := map[string]string{
		"yaml":                   "__testdata/load/blueprint.yml",
		"jwcc":                   "__testdata/load/blueprint.jsonc",
		"yamlWithInclude":        "__testdata/load/blueprint-with-include.yml",
		"jwccWithInclude":        "__testdata/load/blueprint-with-include.jsonc",
		"yamlWithMultipleValues": "__testdata/load/blueprint-with-multiple-values.yml",
		"jwccWithMultipleValues": "__testdata/load/blueprint-with-multiple-values.jsonc",
	}

	for name, filePath := range fixturesToLoad {
		specBytes, err := os.ReadFile(filePath)
		s.Require().NoError(err)
		s.specFixtures[name] = fixture{
			filePath:  filePath,
			stringVal: string(specBytes),
		}
	}
}

func (s *LoadTestSuite) Test_loads_blueprint_from_yaml_file() {
	blueprint, err := Load(s.specFixtures["yaml"].filePath, YAMLSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_json_file() {
	blueprint, err := Load(s.specFixtures["jwcc"].filePath, JWCCSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_yaml_file_with_includes() {
	blueprint, err := Load(s.specFixtures["yamlWithInclude"].filePath, YAMLSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_json_file_with_include() {
	blueprint, err := Load(s.specFixtures["jwccWithInclude"].filePath, JWCCSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_yaml_file_with_multiple_values() {
	blueprint, err := Load(s.specFixtures["yamlWithMultipleValues"].filePath, YAMLSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_json_file_with_multiple_values() {
	blueprint, err := Load(s.specFixtures["jwccWithMultipleValues"].filePath, JWCCSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_yaml_string() {
	blueprint, err := LoadString(s.specFixtures["yaml"].stringVal, YAMLSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func (s *LoadTestSuite) Test_loads_blueprint_from_json_string() {
	blueprint, err := LoadString(s.specFixtures["jwcc"].stringVal, JWCCSpecFormat)
	s.Require().NoError(err)
	err = testhelpers.Snapshot(blueprint)
	s.Require().NoError(err)
}

func TestLoadTestSuite(t *testing.T) {
	suite.Run(t, new(LoadTestSuite))
}
