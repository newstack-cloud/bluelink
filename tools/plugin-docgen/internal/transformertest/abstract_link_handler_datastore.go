package transformertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func handlerDatastoreAbstractLink() *transformerv1.AbstractLinkDefinition {
	return &transformerv1.AbstractLinkDefinition{
		ResourceTypeA:        "test/celerity/handler",
		ResourceTypeB:        "test/celerity/datastore",
		PlainTextSummary:     "A link between a Celerity handler and a NoSQL data store.",
		FormattedDescription: "Links a [Celerity Handler](https://www.celerityframework.io/docs/applications/resources/celerity-handler) to a Celerity datastore.",
		CardinalityA: provider.LinkCardinality{
			Min: 0,
			Max: 3,
		},
		CardinalityB: provider.LinkCardinality{
			Min: 1,
			Max: 0,
		},
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{
			"test/celerity/handler::celerity.handler.datastore.accessType": {
				Name:         "celerity.handler.datastore.accessType",
				Label:        "Datastore Access Type",
				Type:         core.ScalarTypeString,
				Description:  "The type of access the handler has to linked datastores.",
				DefaultValue: core.ScalarFromString("read"),
				AllowedValues: []*core.ScalarValue{
					core.ScalarFromString("read"),
					core.ScalarFromString("write"),
				},
				Required: false,
			},
		},
	}
}
