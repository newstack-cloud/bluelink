package resolverfs

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type FileSystemChildResolverSuite struct {
	fs                      afero.Fs
	resolver                includes.ChildResolver
	expectedBlueprintSource string
	suite.Suite
}

func (s *FileSystemChildResolverSuite) SetupTest() {
	s.fs = afero.NewOsFs()
	expectedBytes, err := os.ReadFile("__testdata/fs.test.blueprint.yml")
	s.Require().NoError(err)
	s.expectedBlueprintSource = string(expectedBytes)
	s.resolver = NewResolver(s.fs)
}

func (s *FileSystemChildResolverSuite) Test_resolves_blueprint_file() {
	workingDir, err := os.Getwd()
	s.Require().NoError(err)
	absPath := path.Join(workingDir, "__testdata/fs.test.blueprint.yml")
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &absPath,
			},
		},
	}
	resolvedInfo, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().NoError(err)
	s.Assert().NotNil(resolvedInfo)
	s.Assert().NotNil(resolvedInfo.BlueprintSource)
	s.Assert().Equal(s.expectedBlueprintSource, *resolvedInfo.BlueprintSource)
}

func (s *FileSystemChildResolverSuite) Test_returns_error_when_path_is_empty() {
	path := ""
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidPath, runErr.ReasonCode)
	s.Assert().Equal(
		runErr.Err.Error(),
		"[include.test]: invalid path found, path value must be a string "+
			"for the file system child resolver, the provided value is either empty or not a string",
	)
}

func (s *FileSystemChildResolverSuite) Test_returns_error_when_file_does_not_exist() {
	workingDir, err := os.Getwd()
	s.Require().NoError(err)
	absPath := path.Join(workingDir, "__testdata/fs.missing.test.blueprint.yml")
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &absPath,
			},
		},
	}
	_, err = s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeBlueprintNotFound, runErr.ReasonCode)
	s.Assert().Equal(
		runErr.Err.Error(),
		"[include.test]: blueprint not found at path: "+absPath,
	)
}

func TestFileSystemChildResolverSuite(t *testing.T) {
	suite.Run(t, new(FileSystemChildResolverSuite))
}
