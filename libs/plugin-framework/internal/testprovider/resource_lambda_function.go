package testprovider

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func resourceLambdaFunction() provider.Resource {
	descriptionInfo := ResourceLambdaFunctionTypeDescription()
	examples := ResourceLambdaFunctionExamples()
	return &providerv1.ResourceDefinition{
		Type:                   "aws/lambda/function",
		Label:                  "AWS Lambda Function",
		Schema:                 ResourceLambdaFunctionSchema(),
		PlainTextDescription:   descriptionInfo.PlainTextDescription,
		FormattedDescription:   descriptionInfo.MarkdownDescription,
		PlainTextSummary:       descriptionInfo.PlainTextSummary,
		FormattedSummary:       descriptionInfo.MarkdownSummary,
		PlainTextExamples:      examples.PlainTextExamples,
		FormattedExamples:      examples.MarkdownExamples,
		IDField:                "arn",
		StabilisedDependencies: []string{"aws/sqs/queue"},
		CreateFunc: providerv1.RetryableReturnValue(
			deployLambdaFunction,
			func(err error) bool {
				return true
			},
		),
		UpdateFunc: providerv1.RetryableReturnValue(
			deployLambdaFunction,
			func(err error) bool {
				return true
			},
		),
		DestroyFunc:          destroyLambdaFunction,
		CustomValidateFunc:   customValidateLambdaFunction,
		GetExternalStateFunc: getLambdaFunctionExternalState,
	}
}

func ResourceLambdaFunctionTypeDescription() *provider.ResourceGetTypeDescriptionOutput {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: "An AWS Lambda Function for running code in the cloud",
		MarkdownDescription:  "An **AWS** Lambda Function for running code in the cloud",
		PlainTextSummary:     "An AWS Lambda Function",
		MarkdownSummary:      "An **AWS** Lambda Function",
	}
}

func ResourceLambdaFunctionExamples() *provider.ResourceGetExamplesOutput {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples: []string{
			"```yaml\nresources:\n  - type: aws/lambda/function\n    name: example-function\n```",
		},
		PlainTextExamples: []string{
			"resources:\n  - type: aws/lambda/function\n    name: example-function\n",
		},
	}
}

func ResourceLambdaDeployOutput() *provider.ResourceDeployOutput {
	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.arn": core.MappingNodeFromString(
				"arn:aws:lambda:us-west-2:123456789012:function:processOrderFunction_0",
			),
		},
	}
}

func deployLambdaFunction(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	output := ResourceLambdaDeployOutput()

	// Capture tagging config info in output for testing gRPC round-trip.
	// This allows tests to verify that TaggingConfig flows correctly
	// through the gRPC layer.
	if input.ProviderContext != nil {
		taggingConfig := input.ProviderContext.TaggingConfig()
		if taggingConfig != nil {
			output.ComputedFieldValues["__test_tagging_enabled"] = core.MappingNodeFromBool(taggingConfig.Enabled)
			output.ComputedFieldValues["__test_tagging_prefix"] = core.MappingNodeFromString(taggingConfig.Prefix)
			output.ComputedFieldValues["__test_tagging_deploy_engine_version"] = core.MappingNodeFromString(taggingConfig.DeployEngineVersion)
			output.ComputedFieldValues["__test_tagging_provider_plugin_id"] = core.MappingNodeFromString(taggingConfig.ProviderPluginID)
			output.ComputedFieldValues["__test_tagging_provider_plugin_version"] = core.MappingNodeFromString(taggingConfig.ProviderPluginVersion)
		}
	}

	return output, nil
}

func destroyLambdaFunction(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}

func customValidateLambdaFunction(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	return ResourceLambdaFunctionValidateOutput(), nil
}

// ResourceLambdaFunctionValidateOutput returns a stub validation output
// for the LambdaFunction resource.
func ResourceLambdaFunctionValidateOutput() *provider.ResourceValidateOutput {
	colAccuracy := substitutions.ColumnAccuracyExact
	return &provider.ResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "This is a warning about an invalid lambda function spec",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   110,
							Column: 40,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   110,
							Column: 80,
						},
					},
					ColumnAccuracy: &colAccuracy,
				},
			},
		},
	}
}

// ResourceLambdaFunctionSchema returns a stub spec definition
// for the LambdaFunction resource.
func ResourceLambdaFunctionSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"functionName": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Label:       "Function Name",
				Description: "The name of the Lambda function",
				Examples: []*core.MappingNode{
					core.MappingNodeFromString("example-function"),
				},
			},
			"otherConfigurationValue": {
				Type:        provider.ResourceDefinitionsSchemaTypeUnion,
				Label:       "Other Configuration Value",
				Description: "An example of a union type",
				OneOf: []*provider.ResourceDefinitionsSchema{
					{
						Type:  provider.ResourceDefinitionsSchemaTypeString,
						Label: "String Value",
					},
					{
						Type:  provider.ResourceDefinitionsSchemaTypeInteger,
						Label: "Integer Value",
					},
				},
			},
			"arn": {
				Type:     provider.ResourceDefinitionsSchemaTypeString,
				Computed: true,
			},
		},
	}
}

func getLambdaFunctionExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: ResourceLambdaFunctionExternalState(),
	}, nil
}

func ResourceLambdaFunctionExternalState() *core.MappingNode {
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"arn": core.MappingNodeFromString(
				"arn:aws:lambda:us-west-2:123456789012:function:processOrderFunction_0",
			),
			"functionName": core.MappingNodeFromString("Process-Order-Function-0"),
		},
	}
}
