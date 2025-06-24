// Custom variable type implementations for tests.

package internal

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
			t2nano: {
				Value: &core.ScalarValue{
					StringValue: &t2nano,
				},
			},
			t2micro: {
				Value: &core.ScalarValue{
					StringValue: &t2micro,
				},
			},
			t2small: {
				Value: &core.ScalarValue{
					StringValue: &t2small,
				},
			},
			t2medium: {
				Value: &core.ScalarValue{
					StringValue: &t2medium,
				},
			},
			t2large: {
				Value: &core.ScalarValue{
					StringValue: &t2large,
				},
			},
			t2xlarge: {
				Value: &core.ScalarValue{
					StringValue: &t2xlarge,
				},
			},
			t22xlarge: {
				Value: &core.ScalarValue{
					StringValue: &t22xlarge,
				},
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
