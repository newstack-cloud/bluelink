package testutils

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type VPCDataSource struct{}

func (d *VPCDataSource) GetSpecDefinition(
	ctx context.Context,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	return &provider.DataSourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.DataSourceSpecDefinition{
			Fields: map[string]*provider.DataSourceSpecSchema{
				"instanceConfigId": {
					Type:                 provider.DataSourceSpecTypeString,
					Description:          "The unique identifier for the instance configuration.",
					FormattedDescription: "The unique identifier for the **instance configuration**.",
				},
				"vpcId": {
					Type:                 provider.DataSourceSpecTypeString,
					Description:          "The unique identifier of the VPC.",
					FormattedDescription: "The unique identifier of the **VPC**.",
				},
				"subnetIds": {
					Type:                 provider.DataSourceSpecTypeArray,
					Description:          "The list of subnet identifiers associated with the VPC.",
					FormattedDescription: "The list of **subnet identifiers** associated with the VPC.",
				},
				"tags": {
					Type:                 provider.DataSourceSpecTypeString,
					Description:          "A map of tags associated with the VPC.",
					FormattedDescription: "A map of **tags** associated with the VPC.",
				},
			},
		},
	}, nil
}

func (d *VPCDataSource) Fetch(
	ctx context.Context,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	return &provider.DataSourceFetchOutput{
		Data: map[string]*core.MappingNode{},
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
			"instanceConfigId": {
				Type:                 provider.DataSourceFilterSearchValueTypeString,
				Description:          "The ID of the instance configuration.",
				FormattedDescription: "The ID of the **instance configuration**.",
			},
			"tags": {
				Type:                 provider.DataSourceFilterSearchValueTypeString,
				Description:          "A map of tags to filter the VPCs.",
				FormattedDescription: "A map of **tags** to filter the VPCs.",
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
