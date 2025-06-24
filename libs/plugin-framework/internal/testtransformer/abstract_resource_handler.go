package testtransformer

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/transformerv1"
)

func abstractResourceHandler() transform.AbstractResource {
	descriptionInfo := AbstractResourceHandlerTypeDescription()
	examples := AbstractResourceHandlerExamples()
	return &transformerv1.AbstractResourceDefinition{
		Type:                 "celerity/handler",
		Label:                "Celerity Handler",
		Schema:               AbstractResourceHandlerSchema(),
		PlainTextDescription: descriptionInfo.PlainTextDescription,
		FormattedDescription: descriptionInfo.MarkdownDescription,
		PlainTextSummary:     descriptionInfo.PlainTextSummary,
		FormattedSummary:     descriptionInfo.MarkdownSummary,
		PlainTextExamples:    examples.PlainTextExamples,
		FormattedExamples:    examples.MarkdownExamples,
		IDField:              "id",
		ResourceCanLinkTo:    []string{"celerity/datastore"},
		CustomValidateFunc:   customValidateHandler,
		CommonTerminal:       true,
	}
}

func AbstractResourceHandlerTypeDescription() *transform.AbstractResourceGetTypeDescriptionOutput {
	return &transform.AbstractResourceGetTypeDescriptionOutput{
		PlainTextDescription: "A Celerity Handler running code in the cloud",
		MarkdownDescription:  "A **Celerity** Handler for running code in the cloud",
		PlainTextSummary:     "An Celerity Handler",
		MarkdownSummary:      "A **Celerity** Handler",
	}
}

func AbstractResourceHandlerExamples() *transform.AbstractResourceGetExamplesOutput {
	return &transform.AbstractResourceGetExamplesOutput{
		MarkdownExamples: []string{
			"```yaml\nresources:\n  - type: celerity/handler\n    name: example-handler\n```",
		},
		PlainTextExamples: []string{
			"resources:\n  - type: celerity/handler\n    name: example-handler\n",
		},
	}
}

// AbstractResourceHandlerSchema returns a stub spec definition
// for the Handler resource.
func AbstractResourceHandlerSchema() *provider.ResourceDefinitionsSchema {
	return &provider.ResourceDefinitionsSchema{
		Type: provider.ResourceDefinitionsSchemaTypeObject,
		Attributes: map[string]*provider.ResourceDefinitionsSchema{
			"handlerName": {
				Type:        provider.ResourceDefinitionsSchemaTypeString,
				Label:       "Handler Name",
				Description: "The name of the Celerity handler",
				Examples: []*core.MappingNode{
					core.MappingNodeFromString("example-handler"),
				},
			},
			"id": {
				Type:     provider.ResourceDefinitionsSchemaTypeString,
				Computed: true,
			},
		},
	}
}

func customValidateHandler(
	ctx context.Context,
	input *transform.AbstractResourceValidateInput,
) (*transform.AbstractResourceValidateOutput, error) {
	return AbstractResourceHandlerValidateOutput(), nil
}

// AbstractResourceHandlerValidateOutput returns a stub validation output
// for the Handler abstract resource.
func AbstractResourceHandlerValidateOutput() *transform.AbstractResourceValidateOutput {
	colAccuracy := substitutions.ColumnAccuracyExact
	return &transform.AbstractResourceValidateOutput{
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "This is a warning about an invalid celerity handler spec",
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
