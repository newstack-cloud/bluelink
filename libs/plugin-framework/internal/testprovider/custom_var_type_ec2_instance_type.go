package testprovider

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

const (
	optionT2Nano    = "t2.nano"
	optionT2Micro   = "t2.micro"
	optionT2Small   = "t2.small"
	optionT2Medium  = "t2.medium"
	optionT2Large   = "t2.large"
	optionT2XLarge  = "t2.xlarge"
	optionT2X2Large = "t2.2xlarge"
)

func customVarTypeEC2InstanceType() provider.CustomVariableType {
	descriptionOutput := CustomVarTypeInstanceTypeDescriptionOutput()
	optionsOutput := CustomVarTypeInstanceTypeOptionsOutput()
	examplesOutput := CustomVarTypeInstanceTypeExamplesOutput()

	return &providerv1.CustomVariableTypeDefinition{
		Type:                 "aws/ec2/instanceType",
		Label:                "AWS EC2 Instance Type",
		FormattedDescription: descriptionOutput.MarkdownDescription,
		PlainTextDescription: descriptionOutput.PlainTextDescription,
		FormattedSummary:     descriptionOutput.MarkdownSummary,
		PlainTextSummary:     descriptionOutput.PlainTextSummary,
		CustomVarTypeOptions: optionsOutput.Options,
		PlainTextExamples:    examplesOutput.PlainTextExamples,
		FormattedExamples:    examplesOutput.MarkdownExamples,
	}
}

func CustomVarTypeInstanceTypeOptionsOutput() *provider.CustomVariableTypeOptionsOutput {
	return &provider.CustomVariableTypeOptionsOutput{
		Options: map[string]*provider.CustomVariableTypeOption{
			optionT2Nano: {
				Label: optionT2Nano,
				Value: core.ScalarFromString(optionT2Nano),
			},
			optionT2Micro: {
				Label: optionT2Micro,
				Value: core.ScalarFromString(optionT2Micro),
			},
			optionT2Small: {
				Label: optionT2Small,
				Value: core.ScalarFromString(optionT2Small),
			},
			optionT2Medium: {
				Label: optionT2Medium,
				Value: core.ScalarFromString(optionT2Medium),
			},
			optionT2Large: {
				Label: optionT2Large,
				Value: core.ScalarFromString(optionT2Large),
			},
			optionT2XLarge: {
				Label: optionT2XLarge,
				Value: core.ScalarFromString(optionT2XLarge),
			},
			optionT2X2Large: {
				Label: optionT2X2Large,
				Value: core.ScalarFromString(optionT2X2Large),
			},
		},
	}
}

func CustomVarTypeInstanceTypeDescriptionOutput() *provider.CustomVariableTypeGetDescriptionOutput {
	return &provider.CustomVariableTypeGetDescriptionOutput{
		PlainTextSummary:     "This is a plain text summary of the instance type custom variable type",
		MarkdownSummary:      "This is a **formatted** summary of the instance type custom variable type",
		PlainTextDescription: "This is a plain text description of the instance type custom variable type",
		MarkdownDescription:  "This is a **formatted** description of the instance type custom variable type",
	}
}

func CustomVarTypeInstanceTypeExamplesOutput() *provider.CustomVariableTypeGetExamplesOutput {
	return &provider.CustomVariableTypeGetExamplesOutput{
		PlainTextExamples: []string{
			"This is a plain text example of the instance type custom variable type",
		},
		MarkdownExamples: []string{
			"This is a **formatted** example of the instance type custom variable type",
		},
	}
}
