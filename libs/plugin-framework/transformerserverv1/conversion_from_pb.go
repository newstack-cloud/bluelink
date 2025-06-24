package transformerserverv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
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
