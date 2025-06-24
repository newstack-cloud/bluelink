package docgen

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

func getTransformerAbstractResourceDocs(
	ctx context.Context,
	namespace string,
	transformerPlugin transform.SpecTransformer,
	resourceType string,
	params core.BlueprintParams,
) (*PluginDocsResource, error) {
	abstractResource, err := transformerPlugin.AbstractResource(ctx, resourceType)
	if err != nil {
		return nil, err
	}

	typeInfo, err := abstractResource.GetType(
		ctx,
		&transform.AbstractResourceGetTypeInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	typeDescriptionOutput, err := abstractResource.GetTypeDescription(
		ctx,
		&transform.AbstractResourceGetTypeDescriptionInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	examplesOutput, err := abstractResource.GetExamples(
		ctx,
		&transform.AbstractResourceGetExamplesInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	canLinkToOutput, err := abstractResource.CanLinkTo(
		ctx,
		&transform.AbstractResourceCanLinkToInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	resourceSpec, err := getTransformerAbstractResourceSpecDocs(
		ctx,
		namespace,
		abstractResource,
		params,
	)
	if err != nil {
		return nil, err
	}

	return &PluginDocsResource{
		Type:    typeInfo.Type,
		Label:   typeInfo.Label,
		Summary: getTransformerAbstractResourceSummary(typeDescriptionOutput),
		Description: getTransformerAbstractResourceDescription(
			typeDescriptionOutput,
		),
		Specification: resourceSpec,
		Examples: getTransformerAbstractResourceExamples(
			examplesOutput,
		),
		CanLinkTo: canLinkToOutput.CanLinkTo,
	}, nil
}

func getTransformerAbstractResourceSummary(
	output *transform.AbstractResourceGetTypeDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownSummary) != "" {
		return output.MarkdownSummary
	}

	if strings.TrimSpace(output.PlainTextSummary) != "" {
		return output.PlainTextSummary
	}

	return truncateDescription(getTransformerAbstractResourceDescription(output), 120)
}

func getTransformerAbstractResourceDescription(
	output *transform.AbstractResourceGetTypeDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownDescription) != "" {
		return output.MarkdownDescription
	}

	return output.PlainTextDescription
}

func getTransformerAbstractResourceExamples(
	output *transform.AbstractResourceGetExamplesOutput,
) []string {
	if len(output.MarkdownExamples) > 0 {
		return output.MarkdownExamples
	}

	return output.PlainTextExamples
}

func getTransformerAbstractResourceSpecDocs(
	ctx context.Context,
	namespace string,
	abstractResource transform.AbstractResource,
	params core.BlueprintParams,
) (*PluginDocResourceSpec, error) {
	specDefinitionOutput, err := abstractResource.GetSpecDefinition(
		ctx,
		&transform.AbstractResourceGetSpecDefinitionInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	spec := &PluginDocResourceSpec{
		Schema:  convertSpecSchema(specDefinitionOutput.SpecDefinition.Schema),
		IDField: specDefinitionOutput.SpecDefinition.IDField,
	}

	return spec, nil
}
