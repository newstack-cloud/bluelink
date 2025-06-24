package utils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/suite"
)

type BlueprintFormatFromExtensionTestSuite struct {
	suite.Suite
}

func (s *BlueprintFormatFromExtensionTestSuite) Test_returns_correct_format_for_known_extensions() {
	filePathFormatMaps := map[string]schema.SpecFormat{
		"blueprint.jsonc":  schema.JWCCSpecFormat,
		"blueprint.json":   schema.JWCCSpecFormat,
		"blueprint.hujson": schema.JWCCSpecFormat,
		"blueprint.yaml":   schema.YAMLSpecFormat,
		"blueprint.yml":    schema.YAMLSpecFormat,
	}

	for filePath, expectedFormat := range filePathFormatMaps {
		format, err := BlueprintFormatFromExtension(filePath)
		s.Require().NoError(err)
		s.Assert().Equal(expectedFormat, format)
	}
}

func (s *BlueprintFormatFromExtensionTestSuite) Test_returns_error_for_unknown_extension() {
	filePath := "blueprint.txt"
	expectedErr := "deploy engine error: unsupported blueprint format file \"blueprint.txt\", " +
		"only json or yaml files with extensions are supported"

	_, err := BlueprintFormatFromExtension(filePath)
	s.Require().Error(err)
	s.Assert().Equal(expectedErr, err.Error())
}

func TestBlueprintFormatFromExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(BlueprintFormatFromExtensionTestSuite))
}
