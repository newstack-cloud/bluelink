package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type AnnotationsTestSuite struct {
	suite.Suite
}

func (s *AnnotationsTestSuite) Test_get_bool_annotation() {
	testCases := []struct {
		name     string
		resource *provider.ResourceInfo
		query    *AnnotationQuery[bool]
		expected bool
		found    bool
	}{
		{
			name: "annotation exists",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.populateEnvVars": core.MappingNodeFromBool(true),
				},
			),
			query:    &AnnotationQuery[bool]{Key: "aws.lambda.function.myFunction.populateEnvVars", Default: false},
			expected: true,
			found:    true,
		},
		{
			name: "annotation does not exist",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.populateEnvVars": core.MappingNodeFromBool(false),
				},
			),
			query:    &AnnotationQuery[bool]{Key: "aws.lambda.function.myFunction.nonExistent", Default: true},
			expected: true,
		},
		{
			name:     "resource has no annotations",
			resource: &provider.ResourceInfo{},
			query:    &AnnotationQuery[bool]{Key: "aws.lambda.function.myFunction.populateEnvVars", Default: false},
			expected: false,
			found:    false,
		},
		{
			name: "fallback key used",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.populateEnvVars": core.MappingNodeFromBool(true),
				},
			),
			query: &AnnotationQuery[bool]{
				Key:         "aws.lambda.function.myFunction.populateEnvVars",
				FallbackKey: "aws.lambda.function.populateEnvVars",
				Default:     false,
			},
			expected: true,
			found:    true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, found := GetBoolAnnotation(tc.resource, tc.query)
			s.Assert().Equal(tc.expected, value, "Expected value to match")
			s.Assert().Equal(tc.found, found, "Expected found to match")
		})
	}
}

func (s *AnnotationsTestSuite) Test_get_string_annotation() {
	testCases := []struct {
		name     string
		resource *provider.ResourceInfo
		query    *AnnotationQuery[string]
		expected string
		found    bool
	}{
		{
			name: "annotation exists",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.envVarName": core.MappingNodeFromString("AWS_LAMBDA_FUNCTION_TEST"),
				},
			),
			query:    &AnnotationQuery[string]{Key: "aws.lambda.function.myFunction.envVarName", Default: ""},
			expected: "AWS_LAMBDA_FUNCTION_TEST",
			found:    true,
		},
		{
			name: "annotation does not exist",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.envVarName": core.MappingNodeFromString("AWS_LAMBDA_FUNCTION_TEST"),
				},
			),
			query:    &AnnotationQuery[string]{Key: "aws.lambda.function.myFunction.nonExistent", Default: "defaultValue"},
			expected: "defaultValue",
		},
		{
			name: "fallback key used",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.envVarName": core.MappingNodeFromString("AWS_LAMBDA_FUNCTION_TEST"),
				},
			),
			query: &AnnotationQuery[string]{
				Key:         "aws.lambda.function.myFunction.envVarName",
				FallbackKey: "aws.lambda.function.envVarName",
				Default:     "",
			},
			expected: "AWS_LAMBDA_FUNCTION_TEST",
			found:    true,
		},
		{
			name:     "resource has no annotations",
			resource: &provider.ResourceInfo{},
			query:    &AnnotationQuery[string]{Key: "aws.lambda.function.myFunction.envVarName", Default: ""},
			expected: "",
			found:    false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, found := GetStringAnnotation(tc.resource, tc.query)
			s.Assert().Equal(tc.expected, value, "Expected value to match")
			s.Assert().Equal(tc.found, found, "Expected found to match")
		})
	}
}

func (s *AnnotationsTestSuite) Test_get_int_annotation() {
	testCases := []struct {
		name     string
		resource *provider.ResourceInfo
		query    *AnnotationQuery[int]
		expected int
		found    bool
	}{
		{
			name: "annotation exists",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.timeout": core.MappingNodeFromInt(30),
				},
			),
			query:    &AnnotationQuery[int]{Key: "aws.lambda.function.myFunction.timeout", Default: 60},
			expected: 30,
			found:    true,
		},
		{
			name: "annotation does not exist",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.timeout": core.MappingNodeFromInt(30),
				},
			),
			query:    &AnnotationQuery[int]{Key: "aws.lambda.function.myFunction.nonExistent", Default: 60},
			expected: 60,
		},
		{
			name: "fallback key used",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.timeout": core.MappingNodeFromInt(45),
				},
			),
			query: &AnnotationQuery[int]{
				Key:         "aws.lambda.function.myFunction.timeout",
				FallbackKey: "aws.lambda.function.timeout",
				Default:     60,
			},
			expected: 45,
			found:    true,
		},
		{
			name:     "resource has no annotations",
			resource: &provider.ResourceInfo{},
			query:    &AnnotationQuery[int]{Key: "aws.lambda.function.myFunction.timeout", Default: 60},
			expected: 60,
			found:    false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, found := GetIntAnnotation(tc.resource, tc.query)
			s.Assert().Equal(tc.expected, value, "Expected value to match")
			s.Assert().Equal(tc.found, found, "Expected found to match")
		})
	}
}

func (s *AnnotationsTestSuite) Test_get_float_annotation() {
	testCases := []struct {
		name     string
		resource *provider.ResourceInfo
		query    *AnnotationQuery[float64]
		expected float64
		found    bool
	}{
		{
			name: "annotation exists",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.memorySize": core.MappingNodeFromFloat(128.5),
				},
			),
			query:    &AnnotationQuery[float64]{Key: "aws.lambda.function.myFunction.memorySize", Default: 256.0},
			expected: 128.5,
			found:    true,
		},
		{
			name: "annotation does not exist",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.myFunction.memorySize": core.MappingNodeFromFloat(128.5),
				},
			),
			query:    &AnnotationQuery[float64]{Key: "aws.lambda.function.myFunction.nonExistent", Default: 256.0},
			expected: 256.0,
		},
		{
			name: "fallback key used",
			resource: createResourceInfoForAnnotations(
				map[string]*core.MappingNode{
					"aws.lambda.function.memorySize": core.MappingNodeFromFloat(192.0),
				},
			),
			query: &AnnotationQuery[float64]{
				Key:         "aws.lambda.function.myFunction.memorySize",
				FallbackKey: "aws.lambda.function.memorySize",
				Default:     256.0,
			},
			expected: 192.0,
			found:    true,
		},
		{
			name:     "resource has no annotations",
			resource: &provider.ResourceInfo{},
			query:    &AnnotationQuery[float64]{Key: "aws.lambda.function.myFunction.memorySize", Default: 256.0},
			expected: 256.0,
			found:    false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, found := GetFloatAnnotation(tc.resource, tc.query)
			s.Assert().Equal(tc.expected, value, "Expected value to match")
			s.Assert().Equal(tc.found, found, "Expected found to match")
		})
	}
}

func createResourceInfoForAnnotations(annotations map[string]*core.MappingNode) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Metadata: &provider.ResolvedResourceMetadata{
				Annotations: &core.MappingNode{
					Fields: annotations,
				},
			},
		},
	}
}

func TestAnnotationsTestSuite(t *testing.T) {
	suite.Run(t, new(AnnotationsTestSuite))
}
