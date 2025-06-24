package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func lambdaFunctionDataSource() provider.DataSource {
	return &providerv1.DataSourceDefinition{
		Type:                 "test/lambda/function",
		Label:                "AWS Lambda Function",
		PlainTextSummary:     "An external data source for an AWS Lambda function.",
		FormattedDescription: "An external data source that can be used to retrieve information about an AWS Lambda function.",
		Fields: map[string]*provider.DataSourceSpecSchema{
			"functionName": {
				Type:        provider.DataSourceSpecTypeString,
				Description: "The name of the Lambda function stored in the AWS system.",
				Nullable:    false,
			},
			"arn": {
				Type:        provider.DataSourceSpecTypeString,
				Description: "The Amazon Resource Name (ARN) of the Lambda function.",
				Nullable:    false,
			},
			"layers": {
				Type:        provider.DataSourceSpecTypeArray,
				Description: "The layers attached to the Lambda function.",
				Items: &provider.DataSourceSpecSchema{
					Type:        provider.DataSourceSpecTypeString,
					Description: "A layer ARN.",
				},
				Nullable: true,
			},
		},
		FilterFields: map[string]*provider.DataSourceFilterSchema{
			"functionName": {
				Type:        provider.DataSourceFilterSearchValueTypeString,
				Description: "Filter by the name of the Lambda function.",
			},
			"arn": {
				Type:        provider.DataSourceFilterSearchValueTypeString,
				Description: "Filter by the Amazon Resource Name (ARN) of the Lambda function.",
			},
		},
		MarkdownExamples: []string{
			"```yaml\ndataSources:\n  - type: test/lambda/function\n    name: ProcessOrders\n    properties:\n      functionName: ProcessOrders\n```",
			"```yaml\ndataSources:\n  - type: test/lambda/function\n    name: ProcessOrders\n    properties:\n      functionName: ProcessOrders\n      arn: arn:aws:lambda:us-west-2:123456789012:function:ProcessOrders\n```",
		},
	}
}
