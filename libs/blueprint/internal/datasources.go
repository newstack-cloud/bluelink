// Data source implementations for tests.

package internal

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

type VPCDataSource struct{}

func (d *VPCDataSource) GetSpecDefinition(
	ctx context.Context,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	return &provider.DataSourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.DataSourceSpecDefinition{
			Fields: map[string]*provider.DataSourceSpecSchema{
				"vpcId": {
					Type: provider.DataSourceSpecTypeString,
				},
				"subnetIds": {
					Type: provider.DataSourceSpecTypeArray,
					Items: &provider.DataSourceSpecSchema{
						Type: provider.DataSourceSpecTypeString,
					},
				},
			},
		},
	}, nil
}

func (d *VPCDataSource) Fetch(
	ctx context.Context,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	vpc := "vpc-12345678"
	subnet1 := "subnet-12345678"
	subnet2 := "subnet-87654321"
	return &provider.DataSourceFetchOutput{
		Data: map[string]*core.MappingNode{
			"vpcId": {
				Scalar: &core.ScalarValue{
					StringValue: &vpc,
				},
			},
			"subnetIds": {
				Items: []*core.MappingNode{
					{
						Scalar: &core.ScalarValue{
							StringValue: &subnet1,
						},
					},
					{
						Scalar: &core.ScalarValue{
							StringValue: &subnet2,
						},
					},
				},
			},
		},
	}, nil
}

func (d *VPCDataSource) GetType(
	ctx context.Context,
	input *provider.DataSourceGetTypeInput,
) (*provider.DataSourceGetTypeOutput, error) {
	return &provider.DataSourceGetTypeOutput{
		Type: "aws/vpc",
	}, nil
}

func (d *VPCDataSource) GetTypeDescription(
	ctx context.Context,
	input *provider.DataSourceGetTypeDescriptionInput,
) (*provider.DataSourceGetTypeDescriptionOutput, error) {
	return &provider.DataSourceGetTypeDescriptionOutput{
		MarkdownDescription:  "# VPC\n\n A Virtual Private Cloud (VPC) in AWS.",
		PlainTextDescription: "",
	}, nil
}

func (d *VPCDataSource) GetFilterFields(
	ctx context.Context,
	input *provider.DataSourceGetFilterFieldsInput,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	return &provider.DataSourceGetFilterFieldsOutput{
		FilterFields: map[string]*provider.DataSourceFilterSchema{
			"vpcId": {
				Type: provider.DataSourceFilterSearchValueTypeString,
				SupportedOperators: []schema.DataSourceFilterOperator{
					schema.DataSourceFilterOperatorEquals,
					schema.DataSourceFilterOperatorNotEquals,
				},
			},
			"tags": {
				Type: provider.DataSourceFilterSearchValueTypeString,
				SupportedOperators: []schema.DataSourceFilterOperator{
					schema.DataSourceFilterOperatorContains,
					schema.DataSourceFilterOperatorNotContains,
					schema.DataSourceFilterOperatorHasKey,
					schema.DataSourceFilterOperatorNotHasKey,
				},
			},
		},
	}, nil
}

func (d *VPCDataSource) CustomValidate(
	ctx context.Context,
	input *provider.DataSourceValidateInput,
) (*provider.DataSourceValidateOutput, error) {
	return &provider.DataSourceValidateOutput{}, nil
}

func (d *VPCDataSource) GetExamples(
	ctx context.Context,
	input *provider.DataSourceGetExamplesInput,
) (*provider.DataSourceGetExamplesOutput, error) {
	return &provider.DataSourceGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}
