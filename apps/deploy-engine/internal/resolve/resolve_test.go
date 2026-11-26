package resolve

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

// Which object a remote blueprint location resolves to.
type RemoteIncludeSuite struct {
	suite.Suite
}

func (s *RemoteIncludeSuite) Test_a_blueprint_at_the_root_of_a_bucket_has_no_leading_slash() {
	include, err := BlueprintDocumentInfoToInclude(&BlueprintDocumentInfo{
		FileSourceScheme: "s3",
		Directory:        "test-blueprints",
		BlueprintFile:    "valid.blueprint.yaml",
	})
	s.Require().NoError(err)
	s.Equal("valid.blueprint.yaml", core.StringValue(include.Path))
	s.Equal("test-blueprints", core.StringValue(include.Metadata.Fields["bucket"]))
}

func (s *RemoteIncludeSuite) Test_a_bucket_given_with_a_trailing_slash_resolves_to_its_root() {
	include, err := BlueprintDocumentInfoToInclude(&BlueprintDocumentInfo{
		FileSourceScheme: "gcs",
		Directory:        "test-blueprints/",
		BlueprintFile:    "valid.blueprint.yaml",
	})
	s.Require().NoError(err)
	s.Equal("valid.blueprint.yaml", core.StringValue(include.Path))
}

func (s *RemoteIncludeSuite) Test_a_blueprint_under_a_prefix_is_joined_with_one_slash() {
	include, err := BlueprintDocumentInfoToInclude(&BlueprintDocumentInfo{
		FileSourceScheme: "azureblob",
		Directory:        "test-blueprints/app/prod/",
		BlueprintFile:    "valid.blueprint.yaml",
	})
	s.Require().NoError(err)
	s.Equal("app/prod/valid.blueprint.yaml", core.StringValue(include.Path))
	s.Equal("test-blueprints", core.StringValue(include.Metadata.Fields["container"]))
}

func (s *RemoteIncludeSuite) Test_a_blueprint_file_given_with_a_leading_slash_is_joined_with_one_slash() {
	include, err := BlueprintDocumentInfoToInclude(&BlueprintDocumentInfo{
		FileSourceScheme: "s3",
		Directory:        "test-blueprints/app",
		BlueprintFile:    "/valid.blueprint.yaml",
	})
	s.Require().NoError(err)
	s.Equal("app/valid.blueprint.yaml", core.StringValue(include.Path))
}

func TestRemoteIncludeSuite(t *testing.T) {
	suite.Run(t, new(RemoteIncludeSuite))
}
