package transformertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func datastoreAbstractResource() transform.AbstractResource {
	return &transformerv1.AbstractResourceDefinition{
		Type:                 "test/celerity/datastore",
		Label:                "Celerity Datastore",
		PlainTextSummary:     "A resource for managing a NoSQL data store.",
		FormattedDescription: "The resource type used to define a NoSQL data store used by a Celerity application.",
		Schema:               datastoreAbstractResourceSchema(),
		IDField:              "id",
		FormattedExamples: []string{
			"```yaml\nresources:\n - type: test/celerity/datastore\n   name: ProcessOrders\n   properties:\n     tableName: ProcessOrders\n```",
			"```yaml\nresources:\n - type: test/celerity/datastore\n   name: ProcessOrders\n   properties:\n     tableName: ProcessOrders\n     id: arn:aws:dynamodb:us-west-2:123456789012:table/ProcessOrders\n```",
			"Some example with `inline code`.",
		},
		ResourceCanLinkTo: []string{},
	}
}

func datastoreAbstractResourceSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type:        provider.ResourceDefinitionsSchemaTypeObject,
		Label:       "CelerityDatastoreDefinition",
		Description: "The definition of a NOSQL data store.",
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"tableName": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Description: "The name of the NoSQL data store/table for the target environment.",
				Computed:    false,
				Nullable:    false,
				Examples: []*core.MappingNode{
					core.MappingNodeFromString("ProcessOrders"),
				},
			},
			"id": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Description: "The ID for the NoSQL data store in the target environment.",
				Computed:    true,
				Nullable:    false,
			},
		},
		Required: []string{"tableName"},
		Nullable: false,
		Computed: false,
	}
}
