package gcs

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

type GCSChildResolverSuite struct {
	resolver                includes.ChildResolver
	expectedBlueprintSource string
	suite.Suite
}

func (s *GCSChildResolverSuite) SetupTest() {
	expectedBytes, err := os.ReadFile("../__testdata/gcs/data/test-bucket/gcs.test.blueprint.yml")
	s.Require().NoError(err)
	s.expectedBlueprintSource = string(expectedBytes)
	s.resolver = NewResolver("http://localhost:8184/storage/v1/")
}

func (s *GCSChildResolverSuite) Test_resolves_blueprint_file() {
	path := "gcs.test.blueprint.yml"
	bucket := "test-bucket"
	sourceType := "gcloud/storage"
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
			},
		},
	}
	resolvedInfo, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().NoError(err)
	s.Assert().NotNil(resolvedInfo)
	s.Assert().NotNil(resolvedInfo.BlueprintSource)
	s.Assert().Equal(s.expectedBlueprintSource, *resolvedInfo.BlueprintSource)
}

func (s *GCSChildResolverSuite) Test_returns_error_when_path_is_empty() {
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
			"for the google cloud storage child resolver, the provided value is either empty or not a string",
		runErr.Err.Error(),
	)
}

func (s *GCSChildResolverSuite) Test_returns_error_when_metadata_is_not_set() {
	path := "gcs.test.blueprint.yml"
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
		"[include.test]: invalid metadata provided for the Google Cloud Storage include",
		runErr.Err.Error(),
	)
}

func (s *GCSChildResolverSuite) Test_returns_error_when_bucket_is_missing_from_metadata() {
	path := "gcs.test.blueprint.yml"
	sourceType := "gcloud/storage"
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
		"[include.test]: missing bucket field in metadata for the Google Cloud Storage include",
		runErr.Err.Error(),
	)
}

func (s *GCSChildResolverSuite) Test_returns_error_when_file_does_not_exist() {
	path := "gcs.missing.blueprint.yml"
	bucket := "test-bucket"
	sourceType := "gcloud/storage"
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
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeBlueprintNotFound, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: blueprint not found at path: gcs://test-bucket/gcs.missing.blueprint.yml",
		runErr.Err.Error(),
	)
}

func TestGCSChildResolverSuite(t *testing.T) {
	suite.Run(t, new(GCSChildResolverSuite))
}
