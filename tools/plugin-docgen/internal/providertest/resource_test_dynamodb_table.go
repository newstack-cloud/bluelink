package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func dynamodbTableResource() provider.Resource {
	return &providerv1.ResourceDefinition{
		Type:                 "test/dynamodb/table",
		Label:                "AWS DynamoDB Table",
		PlainTextSummary:     "A resource for managing an AWS DynamoDB table.",
		FormattedDescription: "The resource type used to define a DynamoDB table that is deployed to AWS.",
		Schema: &provider.ResourceDefinitionsSchema{
			Type:        provider.ResourceDefinitionsSchemaTypeObject,
			Label:       "DynamoDBTableDefinition",
			Description: "The definition of an AWS DynamoDB table.",
			Attributes: map[string]*provider.ResourceDefinitionsSchema{
				"tableName": {
					Type:        provider.ResourceDefinitionsSchemaTypeString,
					Description: "The name of the DynamoDB table in the AWS system.",
					Computed:    false,
					Nullable:    false,
					Examples: []*core.MappingNode{
						core.MappingNodeFromString("Orders"),
					},
				},
				"arn": {
					Type:        provider.ResourceDefinitionsSchemaTypeString,
					Description: "The Amazon Resource Name (ARN) of the DynamoDB table.",
					Computed:    true,
					Nullable:    false,
				},
			},
			Required: []string{"tableName"},
			Nullable: false,
			Computed: false,
		},
		IDField: "arn",
		FormattedExamples: []string{
			"```yaml\nresources:\n  - type: test/dynamodb/table\n    name: Orders\n    properties:\n      tableName: Orders\n```",
			"```yaml\nresources:\n  - type: test/dynamodb/table\n    name: Orders\n    properties:\n      tableName: Orders\n      arn: arn:aws:dynamodb:us-west-2:123456789012:table/Orders\n```",
		},
		ResourceCanLinkTo: []string{},
	}
}
