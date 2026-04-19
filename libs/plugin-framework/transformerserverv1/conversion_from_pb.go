package transformerserverv1

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/convertv1"
	sharedtypesv1 "github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

func fromPBTypeDescription(
	typeDescripion *sharedtypesv1.TypeDescription,
) *transform.AbstractResourceGetTypeDescriptionOutput {
	if typeDescripion == nil {
		return nil
	}

	return &transform.AbstractResourceGetTypeDescriptionOutput{
		MarkdownDescription:  typeDescripion.MarkdownDescription,
		PlainTextDescription: typeDescripion.PlainTextDescription,
		MarkdownSummary:      typeDescripion.MarkdownSummary,
		PlainTextSummary:     typeDescripion.PlainTextSummary,
	}
}

func fromPBExamplesForAbstractResource(
	examples *sharedtypesv1.Examples,
) *transform.AbstractResourceGetExamplesOutput {
	if examples == nil {
		return nil
	}

	return &transform.AbstractResourceGetExamplesOutput{
		MarkdownExamples:  examples.FormattedExamples,
		PlainTextExamples: examples.Examples,
	}
}

func fromPBAbstractLinkTypes(linkTypes *AbstractLinkTypes) []string {
	if linkTypes == nil {
		return nil
	}

	return linkTypes.LinkTypes
}

func fromPBAbstractLinkType(
	abstractLinkType *AbstractLinkType,
) *transform.AbstractLinkGetTypeOutput {
	if abstractLinkType == nil {
		return nil
	}

	linkType := abstractLinkType.GetLinkType()
	resourceTypeA, resourceTypeB := splitLinkType(linkType)

	return &transform.AbstractLinkGetTypeOutput{
		Type:          linkType,
		ResourceTypeA: resourceTypeA,
		ResourceTypeB: resourceTypeB,
	}
}

func fromPBAbstractLinkTypeDescription(
	typeDescription *sharedtypesv1.TypeDescription,
) *transform.AbstractLinkGetTypeDescriptionOutput {
	if typeDescription == nil {
		return nil
	}

	return &transform.AbstractLinkGetTypeDescriptionOutput{
		MarkdownDescription:  typeDescription.MarkdownDescription,
		PlainTextDescription: typeDescription.PlainTextDescription,
		MarkdownSummary:      typeDescription.MarkdownSummary,
		PlainTextSummary:     typeDescription.PlainTextSummary,
	}
}

func fromPBAbstractLinkAnnotationDefinitions(
	pbDefinitions *sharedtypesv1.LinkAnnotationDefinitions,
) (*transform.AbstractLinkGetAnnotationDefinitionsOutput, error) {
	if pbDefinitions == nil {
		return nil, nil
	}

	annotations := make(map[string]*provider.LinkAnnotationDefinition)
	for key, pbAnnotation := range pbDefinitions.Definitions {
		annotation, err := fromPBAbstractLinkAnnotationDefinition(pbAnnotation)
		if err != nil {
			return nil, err
		}
		annotations[key] = annotation
	}

	return &transform.AbstractLinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: annotations,
	}, nil
}

func fromPBAbstractLinkAnnotationDefinition(
	pbDefinition *sharedtypesv1.LinkAnnotationDefinition,
) (*provider.LinkAnnotationDefinition, error) {
	if pbDefinition == nil {
		return nil, nil
	}

	defaultValue, err := serialisation.FromScalarValuePB(
		pbDefinition.DefaultValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	allowedValues, err := convertv1.FromPBScalarSlice(pbDefinition.AllowedValues)
	if err != nil {
		return nil, err
	}

	examples, err := convertv1.FromPBScalarSlice(pbDefinition.Examples)
	if err != nil {
		return nil, err
	}

	return &provider.LinkAnnotationDefinition{
		Name:          pbDefinition.Name,
		Label:         pbDefinition.Label,
		Type:          convertv1.FromPBScalarType(pbDefinition.Type),
		Description:   pbDefinition.Description,
		DefaultValue:  defaultValue,
		AllowedValues: allowedValues,
		Examples:      examples,
		Required:      pbDefinition.Required,
		AppliesTo:     provider.LinkAnnotationResource(pbDefinition.AppliesTo),
	}, nil
}

func splitLinkType(linkType string) (string, string) {
	parts := strings.SplitN(linkType, "::", 2)
	if len(parts) != 2 {
		return linkType, ""
	}
	return parts[0], parts[1]
}
