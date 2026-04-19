package testtransformer

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func abstractLinkHandlerAPI() transform.AbstractLink {
	return &transformerv1.AbstractLinkDefinition{
		ResourceTypeA:         "celerity/handler",
		ResourceTypeB:         "celerity/api",
		PlainTextSummary:      "Celerity Handler to API link",
		FormattedSummary:      "**Celerity** Handler to **Celerity** API link",
		PlainTextDescription:  "A link between a Celerity handler and a Celerity API",
		FormattedDescription:  "A link between a **Celerity** handler and a **Celerity** API",
		AnnotationDefinitions: AbstractLinkHandlerAPIAnnotations(),
		CardinalityA: provider.LinkCardinality{
			Min: 0,
			Max: 1,
		},
		CardinalityB: provider.LinkCardinality{
			Min: 1,
			Max: 0,
		},
	}
}

// AbstractLinkHandlerAPITypeDescription returns the expected type description
// for the handler -> api abstract link.
func AbstractLinkHandlerAPITypeDescription() *transform.AbstractLinkGetTypeDescriptionOutput {
	return &transform.AbstractLinkGetTypeDescriptionOutput{
		PlainTextDescription: "A link between a Celerity handler and a Celerity API",
		MarkdownDescription:  "A link between a **Celerity** handler and a **Celerity** API",
		PlainTextSummary:     "Celerity Handler to API link",
		MarkdownSummary:      "**Celerity** Handler to **Celerity** API link",
	}
}

// AbstractLinkHandlerAPIAnnotations returns the annotation definitions
// for the handler -> api abstract link.
func AbstractLinkHandlerAPIAnnotations() map[string]*provider.LinkAnnotationDefinition {
	return map[string]*provider.LinkAnnotationDefinition{
		"celerity/handler::celerity.handler.http.method": {
			Name:         "celerity.handler.http.method",
			Label:        "HTTP Method",
			Type:         core.ScalarTypeString,
			Description:  "The HTTP method for the handler endpoint.",
			DefaultValue: core.ScalarFromString("GET"),
			AllowedValues: []*core.ScalarValue{
				core.ScalarFromString("GET"),
				core.ScalarFromString("POST"),
				core.ScalarFromString("PUT"),
				core.ScalarFromString("DELETE"),
			},
			Examples: []*core.ScalarValue{
				core.ScalarFromString("GET"),
				core.ScalarFromString("POST"),
			},
			Required:  true,
			AppliesTo: provider.LinkAnnotationResourceA,
		},
	}
}
