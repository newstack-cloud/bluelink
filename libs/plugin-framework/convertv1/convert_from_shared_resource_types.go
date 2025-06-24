package convertv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

// FromPBDeployResourceCompleteResponse converts a DeployResourceCompleteResponse
// from a protobuf message to a type comptabile with the blueprint framework.
func FromPBDeployResourceCompleteResponse(
	response *sharedtypesv1.DeployResourceCompleteResponse,
) (*provider.ResourceDeployOutput, error) {
	if response == nil {
		return nil, nil
	}

	computedFieldValues, err := FromPBMappingNodeMap(response.ComputedFieldValues)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceDeployOutput{
		ComputedFieldValues: computedFieldValues,
	}, nil
}

// FromPBTypeDescriptionForResource converts a TypeDescription from a protobuf message to a core type
// compatible with the blueprint framework specifically for resources.
func FromPBTypeDescriptionForResource(
	req *sharedtypesv1.TypeDescription,
) *provider.ResourceGetTypeDescriptionOutput {
	if req == nil {
		return nil
	}

	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: req.PlainTextDescription,
		MarkdownDescription:  req.MarkdownDescription,
		PlainTextSummary:     req.PlainTextSummary,
		MarkdownSummary:      req.MarkdownSummary,
	}
}

// FromPBExamplesForResource converts examples from a protobuf message to a core type
// compatible with the blueprint framework specifically for resources.
func FromPBExamplesForResource(
	req *sharedtypesv1.Examples,
) *provider.ResourceGetExamplesOutput {
	if req == nil {
		return nil
	}

	return &provider.ResourceGetExamplesOutput{
		PlainTextExamples: req.Examples,
		MarkdownExamples:  req.FormattedExamples,
	}
}
