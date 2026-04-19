package docgen

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

func getTransformerAbstractLinkDocs(
	ctx context.Context,
	namespace string,
	transformerPlugin transform.SpecTransformer,
	linkType string,
	params core.BlueprintParams,
) (*PluginDocsLink, error) {
	link, err := transformerPlugin.AbstractLink(
		ctx,
		linkType,
	)
	if err != nil {
		return nil, err
	}

	typeInfo, err := link.GetType(
		ctx,
		&transform.AbstractLinkGetTypeInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	typeDescriptionOutput, err := link.GetTypeDescription(
		ctx,
		&transform.AbstractLinkGetTypeDescriptionInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	annotationDefinitionDocs, err := getAbstractLinkAnnotationDefinitionDocs(ctx, link, namespace, params)
	if err != nil {
		return nil, err
	}

	cardinalityOutput, err := link.GetCardinality(
		ctx,
		&transform.AbstractLinkGetCardinalityInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	return &PluginDocsLink{
		Type:                  typeInfo.Type,
		Description:           getTransformerLinkDescription(typeDescriptionOutput),
		Summary:               getTransformerLinkSummary(typeDescriptionOutput),
		AnnotationDefinitions: annotationDefinitionDocs,
		CardinalityA:          toDocsLinkCardinality(cardinalityOutput.CardinalityA),
		CardinalityB:          toDocsLinkCardinality(cardinalityOutput.CardinalityB),
	}, nil
}

func getAbstractLinkAnnotationDefinitionDocs(
	ctx context.Context,
	link transform.AbstractLink,
	namespace string,
	params core.BlueprintParams,
) (map[string]*PluginDocsLinkAnnotationDefinition, error) {
	annotationDefinitionsOutput, err := link.GetAnnotationDefinitions(
		ctx,
		&transform.AbstractLinkGetAnnotationDefinitionsInput{
			TransformerContext: createTransformerContext(namespace, params),
		},
	)
	if err != nil {
		return nil, err
	}

	annotationDefinitionDocs := make(
		map[string]*PluginDocsLinkAnnotationDefinition,
		len(annotationDefinitionsOutput.AnnotationDefinitions),
	)
	for name, annotationDefinition := range annotationDefinitionsOutput.AnnotationDefinitions {
		annotationDefinitionDocs[name] = toDocsLinkAnnotationDefinition(
			annotationDefinition,
		)
	}

	return annotationDefinitionDocs, nil
}

func getTransformerLinkDescription(
	typeDescriptionOutput *transform.AbstractLinkGetTypeDescriptionOutput,
) string {
	if typeDescriptionOutput.MarkdownDescription != "" {
		return typeDescriptionOutput.MarkdownDescription
	}
	return typeDescriptionOutput.PlainTextDescription
}

func getTransformerLinkSummary(
	typeDescriptionOutput *transform.AbstractLinkGetTypeDescriptionOutput,
) string {
	if strings.TrimSpace(typeDescriptionOutput.MarkdownSummary) != "" {
		return typeDescriptionOutput.MarkdownSummary
	}

	if strings.TrimSpace(typeDescriptionOutput.PlainTextSummary) != "" {
		return typeDescriptionOutput.PlainTextSummary
	}

	return truncateDescription(getTransformerLinkDescription(typeDescriptionOutput), 120)
}
