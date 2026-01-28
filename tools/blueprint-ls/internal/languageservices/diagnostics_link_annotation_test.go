package languageservices

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// LinkAnnotationDiagnosticsSuite tests that link annotation validation
// produces the same diagnostics for YAML and JSONC documents.
type LinkAnnotationDiagnosticsSuite struct {
	suite.Suite
	loader container.Loader
}

func (s *LinkAnnotationDiagnosticsSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	// Create mock provider with link annotation definitions
	providers := map[string]provider.Provider{
		"aws": &mockAWSProviderForDiagnostics{},
	}

	s.loader = container.NewDefaultLoader(
		providers,
		nil,
		nil,
		nil,
		container.WithLoaderLogger(core.NewLoggerFromZap(logger)),
	)
}

func (s *LinkAnnotationDiagnosticsSuite) Test_produces_same_diagnostics_for_yaml_and_jsonc() {
	// Blueprint with annotation on the CORRECT resource (Lambda function)
	// The annotation definition is keyed by "aws/lambda/function::"
	yamlBlueprint := `
version: "2025-11-02"
resources:
  lambdaFunction:
    type: aws/lambda/function
    metadata:
      labels:
        function: "true"
      annotations:
        aws.dynamodb.lambda.stream.startingPosition: INVALID_VALUE
    linkSelector:
      byLabel:
        function: "true"
    spec:
      functionName: testFunction
  dynamoDBTable:
    type: aws/dynamodb/table
    metadata:
      labels:
        function: "true"
    spec:
      tableName: testTable
`

	jsoncBlueprint := `{
  "version": "2025-11-02",
  "resources": {
    "lambdaFunction": {
      "type": "aws/lambda/function",
      "metadata": {
        "labels": {
          "function": "true"
        },
        "annotations": {
          "aws.dynamodb.lambda.stream.startingPosition": "INVALID_VALUE"
        }
      },
      "linkSelector": {
        "byLabel": {
          "function": "true"
        }
      },
      "spec": {
        "functionName": "testFunction"
      }
    },
    "dynamoDBTable": {
      "type": "aws/dynamodb/table",
      "metadata": {
        "labels": {
          "function": "true"
        }
      },
      "spec": {
        "tableName": "testTable"
      }
    }
  }
}`

	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)

	// Test YAML
	_, yamlErr := s.loader.ValidateString(
		context.Background(),
		yamlBlueprint,
		schema.YAMLSpecFormat,
		params,
	)

	// Test JSONC
	_, jsoncErr := s.loader.ValidateString(
		context.Background(),
		jsoncBlueprint,
		schema.JWCCSpecFormat,
		params,
	)

	// Both should produce the same error (or lack thereof)
	yamlHasError := yamlErr != nil
	jsoncHasError := jsoncErr != nil
	s.Equal(yamlHasError, jsoncHasError, "Both formats should produce the same error state")

	// If both have errors, verify the child errors contain the expected message
	if yamlHasError && jsoncHasError {
		yamlLoadErr, yamlIsLoadErr := yamlErr.(*errors.LoadError)
		jsoncLoadErr, jsoncIsLoadErr := jsoncErr.(*errors.LoadError)

		s.True(yamlIsLoadErr, "YAML error should be a LoadError")
		s.True(jsoncIsLoadErr, "JSONC error should be a LoadError")

		if yamlIsLoadErr && jsoncIsLoadErr {
			s.Equal(len(yamlLoadErr.ChildErrors), len(jsoncLoadErr.ChildErrors),
				"Both formats should have the same number of child errors")

			// Check that at least one child error mentions INVALID_VALUE
			yamlHasInvalidValue := false
			jsoncHasInvalidValue := false

			for _, child := range yamlLoadErr.ChildErrors {
				if childLoadErr, ok := child.(*errors.LoadError); ok {
					if childLoadErr.Err != nil && containsString(childLoadErr.Err.Error(), "INVALID_VALUE") {
						yamlHasInvalidValue = true
					}
				}
			}

			for _, child := range jsoncLoadErr.ChildErrors {
				if childLoadErr, ok := child.(*errors.LoadError); ok {
					if childLoadErr.Err != nil && containsString(childLoadErr.Err.Error(), "INVALID_VALUE") {
						jsoncHasInvalidValue = true
					}
				}
			}

			s.True(yamlHasInvalidValue, "YAML should have a child error mentioning INVALID_VALUE")
			s.True(jsoncHasInvalidValue, "JSONC should have a child error mentioning INVALID_VALUE")
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLinkAnnotationDiagnosticsSuite(t *testing.T) {
	suite.Run(t, new(LinkAnnotationDiagnosticsSuite))
}

// Mock AWS provider for testing
type mockAWSProviderForDiagnostics struct{}

func (p *mockAWSProviderForDiagnostics) Namespace(ctx context.Context) (string, error) {
	return "aws", nil
}

func (p *mockAWSProviderForDiagnostics) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (p *mockAWSProviderForDiagnostics) Resource(ctx context.Context, resourceType string) (provider.Resource, error) {
	switch resourceType {
	case "aws/lambda/function":
		return &testutils.LambdaFunctionResource{}, nil
	case "aws/dynamodb/table":
		return &testutils.DynamoDBTableResource{}, nil
	}
	return nil, nil
}

func (p *mockAWSProviderForDiagnostics) DataSource(ctx context.Context, dataSourceType string) (provider.DataSource, error) {
	return nil, nil
}

func (p *mockAWSProviderForDiagnostics) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (provider.Link, error) {
	if resourceTypeA == "aws/lambda/function" && resourceTypeB == "aws/dynamodb/table" {
		return &mockLambdaDynamoDBLinkForDiagnostics{}, nil
	}
	return nil, nil
}

func (p *mockAWSProviderForDiagnostics) CustomVariableType(ctx context.Context, customVariableType string) (provider.CustomVariableType, error) {
	return nil, nil
}

func (p *mockAWSProviderForDiagnostics) Function(ctx context.Context, functionName string) (provider.Function, error) {
	return nil, nil
}

func (p *mockAWSProviderForDiagnostics) ListResourceTypes(ctx context.Context) ([]string, error) {
	return []string{"aws/lambda/function", "aws/dynamodb/table"}, nil
}

func (p *mockAWSProviderForDiagnostics) ListLinkTypes(ctx context.Context) ([]string, error) {
	return []string{"aws/lambda/function::aws/dynamodb/table"}, nil
}

func (p *mockAWSProviderForDiagnostics) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *mockAWSProviderForDiagnostics) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *mockAWSProviderForDiagnostics) ListFunctions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *mockAWSProviderForDiagnostics) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}

// Mock Link for Lambda -> DynamoDB
type mockLambdaDynamoDBLinkForDiagnostics struct{}

func (l *mockLambdaDynamoDBLinkForDiagnostics) StageChanges(ctx context.Context, input *provider.LinkStageChangesInput) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) UpdateResourceA(ctx context.Context, input *provider.LinkUpdateResourceInput) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) UpdateResourceB(ctx context.Context, input *provider.LinkUpdateResourceInput) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) UpdateIntermediaryResources(ctx context.Context, input *provider.LinkUpdateIntermediaryResourcesInput) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) GetPriorityResource(ctx context.Context, input *provider.LinkGetPriorityResourceInput) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) GetType(ctx context.Context, input *provider.LinkGetTypeInput) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) GetTypeDescription(ctx context.Context, input *provider.LinkGetTypeDescriptionInput) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) GetAnnotationDefinitions(ctx context.Context, input *provider.LinkGetAnnotationDefinitionsInput) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	trimHorizon := "TRIM_HORIZON"
	latest := "LATEST"
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{
			// Annotation is for the Lambda function (Resource A - the selector)
			"aws/lambda/function::aws.dynamodb.lambda.stream.startingPosition": {
				Name:     "aws.dynamodb.lambda.stream.startingPosition",
				Label:    "Starting Position",
				Type:     core.ScalarTypeString,
				Required: false,
				AllowedValues: []*core.ScalarValue{
					{StringValue: &trimHorizon},
					{StringValue: &latest},
				},
			},
		},
	}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) GetKind(ctx context.Context, input *provider.LinkGetKindInput) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{}, nil
}

func (l *mockLambdaDynamoDBLinkForDiagnostics) GetIntermediaryExternalState(ctx context.Context, input *provider.LinkGetIntermediaryExternalStateInput) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}
