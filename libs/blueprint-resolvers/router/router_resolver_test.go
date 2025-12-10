package router

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

type RouterChildResolverSuite struct {
	suite.Suite
	defaultResolver *mockChildResolver
	s3Resolver      *mockChildResolver
	router          includes.ChildResolver
}

func (s *RouterChildResolverSuite) SetupTest() {
	s.defaultResolver = &mockChildResolver{
		resolveResult: &includes.ChildBlueprintInfo{
			BlueprintSource: strPtr("default blueprint content"),
		},
	}
	s.s3Resolver = &mockChildResolver{
		resolveResult: &includes.ChildBlueprintInfo{
			BlueprintSource: strPtr("s3 blueprint content"),
		},
	}
	s.router = NewResolver(
		s.defaultResolver,
		WithRoute("aws/s3", s.s3Resolver),
	)
}

func (s *RouterChildResolverSuite) Test_routes_to_default_resolver_when_metadata_is_nil() {
	// This test verifies the fix for the nil pointer dereference regression
	// when include.Metadata is nil
	path := "child.blueprint.yaml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: nil, // Nil metadata should not cause a panic
	}

	result, err := s.router.Resolve(context.TODO(), "test", include, nil)

	s.Require().NoError(err)
	s.Assert().NotNil(result)
	s.Assert().Equal("default blueprint content", *result.BlueprintSource)
	s.Assert().True(s.defaultResolver.resolveCalled, "Default resolver should have been called")
	s.Assert().False(s.s3Resolver.resolveCalled, "S3 resolver should not have been called")
}

func (s *RouterChildResolverSuite) Test_routes_to_default_resolver_when_sourceType_is_empty() {
	path := "child.blueprint.yaml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				// No sourceType field
			},
		},
	}

	result, err := s.router.Resolve(context.TODO(), "test", include, nil)

	s.Require().NoError(err)
	s.Assert().NotNil(result)
	s.Assert().Equal("default blueprint content", *result.BlueprintSource)
	s.Assert().True(s.defaultResolver.resolveCalled, "Default resolver should have been called")
	s.Assert().False(s.s3Resolver.resolveCalled, "S3 resolver should not have been called")
}

func (s *RouterChildResolverSuite) Test_routes_to_default_resolver_when_sourceType_is_empty_string() {
	path := "child.blueprint.yaml"
	emptySourceType := ""
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
						StringValue: &emptySourceType,
					},
				},
			},
		},
	}

	result, err := s.router.Resolve(context.TODO(), "test", include, nil)

	s.Require().NoError(err)
	s.Assert().NotNil(result)
	s.Assert().Equal("default blueprint content", *result.BlueprintSource)
	s.Assert().True(s.defaultResolver.resolveCalled, "Default resolver should have been called")
}

func (s *RouterChildResolverSuite) Test_routes_to_specific_resolver_based_on_sourceType() {
	path := "s3://bucket/child.blueprint.yaml"
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

	result, err := s.router.Resolve(context.TODO(), "test", include, nil)

	s.Require().NoError(err)
	s.Assert().NotNil(result)
	s.Assert().Equal("s3 blueprint content", *result.BlueprintSource)
	s.Assert().False(s.defaultResolver.resolveCalled, "Default resolver should not have been called")
	s.Assert().True(s.s3Resolver.resolveCalled, "S3 resolver should have been called")
}

func (s *RouterChildResolverSuite) Test_returns_error_for_unknown_sourceType() {
	path := "child.blueprint.yaml"
	sourceType := "unknown/source"
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

	_, err := s.router.Resolve(context.TODO(), "test", include, nil)

	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "no resolver found for sourceType: unknown/source")
}

// mockChildResolver is a test double for includes.ChildResolver
type mockChildResolver struct {
	resolveResult *includes.ChildBlueprintInfo
	resolveError  error
	resolveCalled bool
}

func (m *mockChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {
	m.resolveCalled = true
	if m.resolveError != nil {
		return nil, m.resolveError
	}
	return m.resolveResult, nil
}

func strPtr(s string) *string {
	return &s
}

func TestRouterChildResolverSuite(t *testing.T) {
	suite.Run(t, new(RouterChildResolverSuite))
}
