package transformertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func handlerAbstractResource() transform.AbstractResource {
	return &transformerv1.AbstractResourceDefinition{
		Type:                 "test/celerity/handler",
		Label:                "Celerity Handler",
		PlainTextSummary:     "An abstract resource for managing a Celerity application handler.",
		FormattedDescription: "The abstract resource type used to define a [Celerity Handler](https://www.celerityframework.io/docs/applications/resources/celerity-handler).",
		Schema:               handlerAbstractResourceSchema(),
		IDField:              "id",
		FormattedExamples: []string{
			"```yaml\nresources:\n - type: test/celerity/handler\n   name: ProcessOrders\n   properties:\n     functionName: ProcessOrders\n```",
			"```yaml\nresources:\n - type: test/celerity/handler\n   name: ProcessOrders\n   properties:\n     functionName: ProcessOrders\n     id: arn:aws:lambda:us-west-2:123456789012:function:ProcessOrders\n```",
			"Some example with `inline code`.",
		},
		ResourceCanLinkTo: []string{
			"test/celerity/datastore",
			"test/celerity/bucket",
			"test/celerity/queue",
		},
	}
}

func handlerAbstractResourceSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type:        provider.ResourceDefinitionsSchemaTypeObject,
		Label:       "CelerityHandlerDefinition",
		Description: "The definition of a Celerity application handler.",
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"functionName": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Description: "The name of the handler in the target environment.",
				Computed:    false,
				Nullable:    false,
				Examples: []*core.MappingNode{
					core.MappingNodeFromString("ProcessOrders"),
				},
			},
			"id": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Description: "The unique ID of the handler in the target environment.",
				Computed:    true,
				Nullable:    false,
			},
			"nestedObject": {
				Type:        provider.ResourceDefinitionsSchemaTypeObject,
				Description: "A nested object definition to test out rendering.",
				Label:       "NestedObjectDefinition",
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"nestedField": {
						Type:        provider.ResourceDefinitionsSchemaTypeString,
						Description: "A nested field.",
						Computed:    false,
						Nullable:    false,
					},
					"nestedField2": {
						Type:        provider.ResourceDefinitionsSchemaTypeObject,
						Description: "A deeply nested object.",
						Label:       "DeeplyNestedObjectDefinition",
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"deeplyNestedField": {
								Type:        provider.ResourceDefinitionsSchemaTypeString,
								Description: "A deeply nested field.",
								Computed:    false,
								Nullable:    false,
							},
						},
						Required: []string{"deeplyNestedField"},
						Nullable: false,
						Computed: false,
					},
				},
				Required: []string{"nestedField"},
				Nullable: false,
				Computed: false,
			},
			"unionField": {
				Type:        provider.ResourceDefinitionsSchemaTypeUnion,
				Description: "A union field definition to test out rendering.",
				OneOf: []*provider.ResourceDefinitionsSchema{
					{
						Type:        provider.ResourceDefinitionsSchemaTypeString,
						Description: "A string value.",
					},
					{
						Type:        provider.ResourceDefinitionsSchemaTypeInteger,
						Description: "An integer value.",
					},
					{
						Type:        provider.ResourceDefinitionsSchemaTypeArray,
						Description: "An array value.",
						Items: &provider.ResourceDefinitionsSchema{
							Type:        provider.ResourceDefinitionsSchemaTypeObject,
							Label:       "UnionNestedDefinition",
							Description: "A definition nested in a union field.",
							Attributes: map[string]*provider.ResourceDefinitionsSchema{
								"unionNestedField": {
									Type:        provider.ResourceDefinitionsSchemaTypeString,
									Description: "A union nested field.",
									Computed:    false,
									Nullable:    false,
								},
							},
							Required: []string{"unionNestedField"},
						},
					},
				},
				Nullable: false,
				Computed: false,
				Examples: []*core.MappingNode{
					core.MappingNodeFromString("string"),
					core.MappingNodeFromInt(123),
					&core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"unionNestedField": core.MappingNodeFromString("value"),
						},
					},
				},
			},
			"arrayField": {
				Type:        provider.ResourceDefinitionsSchemaTypeArray,
				Description: "An array field definition to test out rendering.",
				Items: &provider.ResourceDefinitionsSchema{
					Type:        provider.ResourceDefinitionsSchemaTypeString,
					Description: "An array item.",
				},
				Nullable: false,
				Computed: false,
				Examples: []*core.MappingNode{
					&core.MappingNode{
						Items: []*core.MappingNode{
							core.MappingNodeFromString("item1"),
							core.MappingNodeFromString("item2"),
						},
					},
				},
			},
			"mapField": {
				Type:        provider.ResourceDefinitionsSchemaTypeMap,
				Description: "A map field definition to test out rendering.",
				MapValues: &provider.ResourceDefinitionsSchema{
					Type:        provider.ResourceDefinitionsSchemaTypeString,
					Description: "A map value.",
				},
				Nullable: false,
				Computed: false,
				Examples: []*core.MappingNode{
					&core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"key1": core.MappingNodeFromString("value1"),
							"key2": core.MappingNodeFromString("value2"),
						},
					},
				},
			},
		},
		Required: []string{"functionName"},
		Nullable: false,
		Computed: false,
	}
}
