package docgen

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

func getProviderLinkDocs(
	ctx context.Context,
	providerPlugin provider.Provider,
	linkType string,
	params core.BlueprintParams,
) (*PluginDocsLink, error) {
	linkTypeParts, err := extractLinkTypeInfo(linkType)
	if err != nil {
		return nil, err
	}

	link, err := providerPlugin.Link(
		ctx,
		linkTypeParts.resourceTypeA,
		linkTypeParts.resourceTypeB,
	)
	if err != nil {
		return nil, err
	}

	typeInfo, err := link.GetType(
		ctx,
		&provider.LinkGetTypeInput{
			LinkContext: createLinkContext(params),
		},
	)
	if err != nil {
		return nil, err
	}

	typeDescriptionOutput, err := link.GetTypeDescription(
		ctx,
		&provider.LinkGetTypeDescriptionInput{
			LinkContext: createLinkContext(params),
		},
	)
	if err != nil {
		return nil, err
	}

	annotationDefinitionDocs, err := getAnnotationDefinitionDocs(ctx, link, params)
	if err != nil {
		return nil, err
	}

	return &PluginDocsLink{
		Type:                  typeInfo.Type,
		Description:           getProviderLinkDescription(typeDescriptionOutput),
		Summary:               getProviderLinkSummary(typeDescriptionOutput),
		AnnotationDefinitions: annotationDefinitionDocs,
	}, nil
}

func getAnnotationDefinitionDocs(
	ctx context.Context,
	link provider.Link,
	params core.BlueprintParams,
) (map[string]*PluginDocsLinkAnnotationDefinition, error) {
	annotationDefinitionsOutput, err := link.GetAnnotationDefinitions(
		ctx,
		&provider.LinkGetAnnotationDefinitionsInput{
			LinkContext: createLinkContext(params),
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

func toDocsLinkAnnotationDefinition(
	annotationDefinition *provider.LinkAnnotationDefinition,
) *PluginDocsLinkAnnotationDefinition {
	return &PluginDocsLinkAnnotationDefinition{
		Name:          annotationDefinition.Name,
		Label:         annotationDefinition.Label,
		Type:          string(annotationDefinition.Type),
		Description:   annotationDefinition.Description,
		Default:       annotationDefinition.DefaultValue,
		AllowedValues: annotationDefinition.AllowedValues,
		Examples:      annotationDefinition.Examples,
		Required:      annotationDefinition.Required,
	}
}

func getProviderLinkDescription(
	typeDescriptionOutput *provider.LinkGetTypeDescriptionOutput,
) string {
	if typeDescriptionOutput.MarkdownDescription != "" {
		return typeDescriptionOutput.MarkdownDescription
	}
	return typeDescriptionOutput.PlainTextDescription
}

func getProviderLinkSummary(
	output *provider.LinkGetTypeDescriptionOutput,
) string {
	if strings.TrimSpace(output.MarkdownSummary) != "" {
		return output.MarkdownSummary
	}

	if strings.TrimSpace(output.PlainTextSummary) != "" {
		return output.PlainTextSummary
	}

	return truncateDescription(getProviderLinkDescription(output), 120)
}
