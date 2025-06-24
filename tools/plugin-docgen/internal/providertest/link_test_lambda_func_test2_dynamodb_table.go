package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func lambdaFunctionTest2DynamoDBTableLink() provider.Link {
	return &providerv1.LinkDefinition{
		ResourceTypeA:        "test/lambda/function",
		ResourceTypeB:        "test2/dynamodb/table",
		Kind:                 provider.LinkKindSoft,
		PlainTextDescription: "A link between an AWS Lambda function and an AWS DynamoDB table.",
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{
			"test/lambda/function::aws.lambda.dynamodb.accessType": {
				Name:         "aws.dynamodb.lambda.accessType",
				Label:        "Lambda Access Type",
				Type:         core.ScalarTypeString,
				Description:  "The type of access the Lambda function has to linked DynamoDB tables.",
				DefaultValue: core.ScalarFromString("read"),
				AllowedValues: []*core.ScalarValue{
					core.ScalarFromString("read"),
					core.ScalarFromString("write"),
				},
				Required: false,
			},
			"test/lambda/function::aws.lambda.dynamodb.accessTables": {
				Name:        "aws.lambda.dynamodb.accessTables",
				Label:       "Access Tables",
				Type:        core.ScalarTypeString,
				Description: "A comma-separated list of table names to apply the access type annotation value to.",
				Examples: []*core.ScalarValue{
					core.ScalarFromString("Orders,Customers"),
				},
				Required: false,
			},
		},
	}
}
