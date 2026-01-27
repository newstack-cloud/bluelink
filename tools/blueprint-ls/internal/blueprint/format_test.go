package blueprint

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
)

type FormatSuite struct {
	suite.Suite
}

func (s *FormatSuite) TestDetermineDocFormat_YAMLExtension() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.yaml"))
	s.Equal(schema.YAMLSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_YMLExtension() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.yml"))
	s.Equal(schema.YAMLSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_JSONExtension() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.json"))
	s.Equal(schema.JWCCSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_JSONCExtension() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.jsonc"))
	s.Equal(schema.JWCCSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_HuJSONExtension() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.hujson"))
	s.Equal(schema.JWCCSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_UnknownExtensionDefaultsToYAML() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.txt"))
	s.Equal(schema.YAMLSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_NoExtension() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file"))
	s.Equal(schema.YAMLSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_PathWithMultipleDots() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.blueprint.yaml"))
	s.Equal(schema.YAMLSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_PathWithMultipleDotsJSON() {
	result := DetermineDocFormat(lsp.URI("file:///path/to/file.blueprint.json"))
	s.Equal(schema.JWCCSpecFormat, result)
}

func (s *FormatSuite) TestDetermineDocFormat_EmptyURI() {
	result := DetermineDocFormat(lsp.URI(""))
	s.Equal(schema.YAMLSpecFormat, result)
}

func TestFormatSuite(t *testing.T) {
	suite.Run(t, new(FormatSuite))
}
