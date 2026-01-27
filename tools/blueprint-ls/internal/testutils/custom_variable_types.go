package testutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type InstanceTypeCustomVariableType struct{}

func (t *InstanceTypeCustomVariableType) Options(
	ctx context.Context,
	input *provider.CustomVariableTypeOptionsInput,
) (*provider.CustomVariableTypeOptionsOutput, error) {
	t2nano := "t2.nano"
	t2micro := "t2.micro"
	t2small := "t2.small"
	t2medium := "t2.medium"
	t2large := "t2.large"
	t2xlarge := "t2.xlarge"
	t22xlarge := "t2.2xlarge"
	return &provider.CustomVariableTypeOptionsOutput{
		Options: map[string]*provider.CustomVariableTypeOption{
			"T2 Nano": {
				Value: &core.ScalarValue{
					StringValue: &t2nano,
				},
				Label:               "T2 Nano",
				Description:         "Burstable instance with 1 vCPU and 0.5 GiB memory.",
				MarkdownDescription: "Burstable instance with **1 vCPU** and **0.5 GiB** memory.",
			},
			"T2 Micro": {
				Value: &core.ScalarValue{
					StringValue: &t2micro,
				},
				Label:               "T2 Micro",
				Description:         "Burstable instance with 1 vCPU and 1 GiB memory.",
				MarkdownDescription: "Burstable instance with **1 vCPU** and **1 GiB** memory.",
			},
			"T2 Small": {
				Value: &core.ScalarValue{
					StringValue: &t2small,
				},
				Label:               "T2 Small",
				Description:         "Burstable instance with 1 vCPU and 2 GiB memory.",
				MarkdownDescription: "Burstable instance with **1 vCPU** and **2 GiB** memory.",
			},
			"T2 Medium": {
				Value: &core.ScalarValue{
					StringValue: &t2medium,
				},
				Label:               "T2 Medium",
				Description:         "Burstable instance with 2 vCPUs and 4 GiB memory.",
				MarkdownDescription: "Burstable instance with **2 vCPUs** and **4 GiB** memory.",
			},
			"T2 Large": {
				Value: &core.ScalarValue{
					StringValue: &t2large,
				},
				Label:               "T2 Large",
				Description:         "Burstable instance with 2 vCPUs and 8 GiB memory.",
				MarkdownDescription: "Burstable instance with **2 vCPUs** and **8 GiB** memory.",
			},
			"T2 XLarge": {
				Value: &core.ScalarValue{
					StringValue: &t2xlarge,
				},
				Label:               "T2 XLarge",
				Description:         "Burstable instance with 4 vCPUs and 16 GiB memory.",
				MarkdownDescription: "Burstable instance with **4 vCPUs** and **16 GiB** memory.",
			},
			"T2 2XLarge": {
				Value: &core.ScalarValue{
					StringValue: &t22xlarge,
				},
				Label:               "T2 2XLarge",
				Description:         "Burstable instance with 8 vCPUs and 32 GiB memory.",
				MarkdownDescription: "Burstable instance with **8 vCPUs** and **32 GiB** memory.",
			},
		},
	}, nil
}

func (t *InstanceTypeCustomVariableType) GetType(
	ctx context.Context,
	input *provider.CustomVariableTypeGetTypeInput,
) (*provider.CustomVariableTypeGetTypeOutput, error) {
	return &provider.CustomVariableTypeGetTypeOutput{
		Type: "aws/ec2/instanceType",
	}, nil
}

func (t *InstanceTypeCustomVariableType) GetDescription(
	ctx context.Context,
	input *provider.CustomVariableTypeGetDescriptionInput,
) (*provider.CustomVariableTypeGetDescriptionOutput, error) {
	return &provider.CustomVariableTypeGetDescriptionOutput{
		MarkdownDescription:  "# EC2 Instance Type\n\nAn EC2 instance type.",
		PlainTextDescription: "",
	}, nil
}

func (t *InstanceTypeCustomVariableType) GetExamples(
	ctx context.Context,
	input *provider.CustomVariableTypeGetExamplesInput,
) (*provider.CustomVariableTypeGetExamplesOutput, error) {
	return &provider.CustomVariableTypeGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}
