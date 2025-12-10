// Package main provides a minimal test provider for e2e testing.
// This provider only supports aws/lambda/function for staging tests.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/plugin"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func main() {
	serviceClient, closeService, err := pluginservicev1.NewEnvServiceClient()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer closeService()

	hostInfoContainer := pluginutils.NewHostInfoContainer()
	providerServer := providerv1.NewProviderPlugin(
		newTestProvider(),
		hostInfoContainer,
		serviceClient,
	)

	config := plugin.ServePluginConfiguration{
		ID: "bluelink-test/aws",
		PluginMetadata: &pluginservicev1.PluginMetadata{
			PluginVersion:        "1.0.0",
			DisplayName:          "Test AWS Provider",
			FormattedDescription: "Stub AWS provider for e2e testing",
			RepositoryUrl:        "https://github.com/newstack-cloud/bluelink",
			Author:               "Bluelink",
		},
		ProtocolVersion: "1.0",
	}

	fmt.Println("Starting test provider plugin server...")
	close, err := plugin.ServeProviderV1(
		context.Background(),
		providerServer,
		serviceClient,
		hostInfoContainer,
		config,
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	pluginutils.WaitForShutdown(close)
}

// newTestProvider creates a minimal test AWS provider for e2e testing.
// It only supports the aws/lambda/function resource type.
func newTestProvider() provider.Provider {
	return &providerv1.ProviderPluginDefinition{
		ProviderNamespace:        "aws",
		ProviderConfigDefinition: testProviderConfigDefinition(),
		Resources: map[string]provider.Resource{
			"aws/lambda/function": resourceLambdaFunction(),
		},
		DataSources:         map[string]provider.DataSource{},
		Links:               map[string]provider.Link{},
		CustomVariableTypes: map[string]provider.CustomVariableType{},
		Functions:           map[string]provider.Function{},
		ProviderRetryPolicy: &provider.RetryPolicy{
			MaxRetries:      3,
			FirstRetryDelay: 4,
			MaxDelay:        200,
			BackoffFactor:   1.5,
			Jitter:          true,
		},
	}
}

func testProviderConfigDefinition() *core.ConfigDefinition {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{
			"accessKeyId": {
				Type:        core.ScalarTypeString,
				Label:       "Access Key ID",
				Description: "The access key ID for the AWS account.",
				Required:    false,
			},
			"secretAccessKey": {
				Type:        core.ScalarTypeString,
				Label:       "Secret Access Key",
				Description: "The secret access key for the AWS account.",
				Required:    false,
			},
			"region": {
				Type:        core.ScalarTypeString,
				Label:       "Region",
				Description: "The AWS region.",
				Required:    false,
			},
		},
	}
}

// resourceLambdaFunction creates a minimal lambda function resource definition.
func resourceLambdaFunction() provider.Resource {
	return &providerv1.ResourceDefinition{
		Type:                 "aws/lambda/function",
		Label:                "AWS Lambda Function",
		Schema:               lambdaFunctionSchema(),
		PlainTextDescription: "A stub AWS Lambda Function for e2e testing",
		FormattedDescription: "A stub **AWS Lambda Function** for e2e testing",
		PlainTextSummary:     "AWS Lambda Function",
		FormattedSummary:     "**AWS Lambda Function**",
		IDField:              "arn",
		CreateFunc: providerv1.RetryableReturnValue(
			deployLambdaFunction,
			func(err error) bool { return true },
		),
		UpdateFunc: providerv1.RetryableReturnValue(
			deployLambdaFunction,
			func(err error) bool { return true },
		),
		DestroyFunc:          destroyLambdaFunction,
		GetExternalStateFunc: getLambdaFunctionExternalState,
	}
}

func lambdaFunctionSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"functionName": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Label:       "Function Name",
				Description: "The name of the Lambda function",
			},
			"arn": {
				Type:     provider.ResourceDefinitionsSchemaTypeString,
				Computed: true,
			},
		},
	}
}

func deployLambdaFunction(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	// Return a deterministic mock ARN for testing
	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.arn": core.MappingNodeFromString(
				"arn:aws:lambda:us-east-1:123456789012:function:test-function",
			),
		},
	}, nil
}

func destroyLambdaFunction(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

func getLambdaFunctionExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	// Return empty state to indicate resource doesn't exist yet
	// This triggers a CREATE action during staging
	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: nil,
	}, nil
}
