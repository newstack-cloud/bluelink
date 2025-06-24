package s3

import (
	"context"
	"os"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

type S3ChildResolverSuite struct {
	resolver                includes.ChildResolver
	expectedBlueprintSource string
	suite.Suite
}

func (s *S3ChildResolverSuite) SetupTest() {
	expectedBytes, err := os.ReadFile("../__testdata/s3/data/test-bucket/s3.test.blueprint.yml")
	s.Require().NoError(err)
	s.expectedBlueprintSource = string(expectedBytes)
	s.resolver = NewResolver("http://localhost:4579", true /* usePathStyle */)
}

func (s *S3ChildResolverSuite) Test_resolves_blueprint_file() {
	path := "s3.test.blueprint.yml"
	bucket := "test-bucket"
	sourceType := "aws/s3"
	region := "eu-west-2"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
				"bucket": {
					Scalar: &core.ScalarValue{
						StringValue: &bucket,
					},
				},
				"region": {
					Scalar: &core.ScalarValue{
						StringValue: &region,
					},
				},
			},
		},
	}
	resolvedInfo, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().NoError(err)
	s.Assert().NotNil(resolvedInfo)
	s.Assert().NotNil(resolvedInfo.BlueprintSource)
	s.Assert().Equal(s.expectedBlueprintSource, *resolvedInfo.BlueprintSource)
}

func (s *S3ChildResolverSuite) Test_returns_error_when_path_is_empty() {
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
		"[include.test]: invalid path found, path value must be a string "+
			"for the s3 child resolver, the provided value is either empty or not a string",
		runErr.Err.Error(),
	)
}

func (s *S3ChildResolverSuite) Test_returns_error_when_metadata_is_not_set() {
	path := "s3.test.blueprint.yml"
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
	s.Assert().Equal(includes.ErrorReasonCodeInvalidMetadata, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: invalid metadata provided for the S3 include",
		runErr.Err.Error(),
	)
}

func (s *S3ChildResolverSuite) Test_returns_error_when_bucket_is_missing_from_metadata() {
	path := "s3.test.blueprint.yml"
	sourceType := "aws/s3"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidMetadata, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: missing bucket field in metadata for the S3 include",
		runErr.Err.Error(),
	)
}

func (s *S3ChildResolverSuite) Test_returns_error_when_file_does_not_exist() {
	path := "s3.missing.blueprint.yml"
	bucket := "test-bucket"
	sourceType := "aws/s3"
	region := "eu-west-2"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
				"bucket": {
					Scalar: &core.ScalarValue{
						StringValue: &bucket,
					},
				},
				"region": {
					Scalar: &core.ScalarValue{
						StringValue: &region,
					},
				},
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeBlueprintNotFound, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: blueprint not found at path: s3://test-bucket/s3.missing.blueprint.yml",
		runErr.Err.Error(),
	)
}

func TestS3ChildResolverSuite(t *testing.T) {
	suite.Run(t, new(S3ChildResolverSuite))
}
