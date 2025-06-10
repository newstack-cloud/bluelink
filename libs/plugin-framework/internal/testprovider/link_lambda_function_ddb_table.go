package testprovider

import (
	"context"

	"github.com/newstack-cloud/celerity/libs/blueprint/core"
	"github.com/newstack-cloud/celerity/libs/blueprint/provider"
	"github.com/newstack-cloud/celerity/libs/plugin-framework/sdk/providerv1"
)

func linkLambdaFunctionDynamoDBTable() provider.Link {
	descriptionInfo := LinkLambdaFunctionDDBTableTypeDescriptionOutput()
	return &providerv1.LinkDefinition{
		ResourceTypeA:                   "aws/lambda/function",
		ResourceTypeB:                   "aws/dynamodb/table",
		Kind:                            provider.LinkKindHard,
		PriorityResource:                provider.LinkPriorityResourceB,
		PlainTextDescription:            descriptionInfo.PlainTextDescription,
		FormattedDescription:            descriptionInfo.MarkdownDescription,
		PlainTextSummary:                descriptionInfo.PlainTextSummary,
		FormattedSummary:                descriptionInfo.MarkdownSummary,
		AnnotationDefinitions:           LinkLambdaFunctionDDBTableAnnotations(),
		StageChangesFunc:                linkLambdaFunctionDDBTableStageChanges,
		UpdateResourceAFunc:             linkLambdaFunctionDDBTableUpdateResourceA,
		UpdateResourceBFunc:             linkLambdaFunctionDDBTableUpdateResourceB,
		UpdateIntermediaryResourcesFunc: linkLambdaFunctionDDBTableUpdateIntermediaryResources,
	}
}

func LinkLambdaFunctionDDBTableTypeDescriptionOutput() *provider.LinkGetTypeDescriptionOutput {
	return &provider.LinkGetTypeDescriptionOutput{
		PlainTextDescription: "A link between an AWS Lambda function and an AWS DynamoDB table",
		MarkdownDescription:  "A link between an **AWS** Lambda function and an **AWS** DynamoDB table",
		PlainTextSummary:     "AWS Lambda Function to DynamoDB Table link",
		MarkdownSummary:      "**AWS** Lambda Function to **AWS** DynamoDB Table link",
	}
}

func LinkLambdaFunctionDDBTableAnnotations() map[string]*provider.LinkAnnotationDefinition {
	allowedValues := []*core.ScalarValue{
		core.ScalarFromString("read"),
		core.ScalarFromString("write"),
	}

	return map[string]*provider.LinkAnnotationDefinition{
		"aws/lambda/function::aws.lambda.dynamodb.accessType": {
			Name:  "aws.lambda.dynamodb.accessType",
			Label: "Access Type",
			Type:  core.ScalarTypeString,
			Description: "The type of access the Lambda function has to the DynamoDB table. " +
				"Valid values are `read` and `write`.",
			DefaultValue:  core.ScalarFromString("read"),
			AllowedValues: allowedValues,
			Examples:      allowedValues,
			Required:      true,
		},
	}
}

func linkLambdaFunctionDDBTableStageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return LinkLambdaDynamoDBChangesOutput(), nil
}

func LinkLambdaDynamoDBChangesOutput() *provider.LinkStageChangesOutput {
	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			ModifiedFields: []*provider.FieldChange{
				{
					FieldPath: "saveOrderFunction.environmentVariables.TABLE_NAME_ordersTable",
					NewValue:  core.MappingNodeFromString("orders-updated"),
					PrevValue: core.MappingNodeFromString("orders"),
				},
			},
			NewFields:                 []*provider.FieldChange{},
			RemovedFields:             []string{},
			UnchangedFields:           []string{},
			FieldChangesKnownOnDeploy: []string{},
		},
	}
}

func linkLambdaFunctionDDBTableUpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return LinkLambdaDynamoDBUpdateResourceAOutput(), nil
}

func LinkLambdaDynamoDBUpdateResourceAOutput() *provider.LinkUpdateResourceOutput {
	return &provider.LinkUpdateResourceOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"environmentVariables.TABLE_NAME_ordersTable": core.MappingNodeFromString("orders-updated"),
			},
		},
	}
}

func linkLambdaFunctionDDBTableUpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return LinkLambdaDynamoDBUpdateResourceBOutput(), nil
}

func LinkLambdaDynamoDBUpdateResourceBOutput() *provider.LinkUpdateResourceOutput {
	return &provider.LinkUpdateResourceOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}
}

func linkLambdaFunctionDDBTableUpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	// Deploy a mock resource to test the link interacting
	// with the plugin service to deploy resources.
	changes := createDeployIntermediaryResourceChanges()

	if input.LinkUpdateType == provider.LinkUpdateTypeUpdate ||
		input.LinkUpdateType == provider.LinkUpdateTypeCreate {
		_, err := input.ResourceDeployService.Deploy(
			ctx,
			"aws/lambda/function",
			&provider.ResourceDeployServiceInput{
				DeployInput: &provider.ResourceDeployInput{
					InstanceID: changes.AppliedResourceInfo.InstanceID,
					ResourceID: changes.AppliedResourceInfo.ResourceID,
					Changes:    changes,
					ProviderContext: provider.NewProviderContextFromLinkContext(
						input.LinkContext,
						"aws",
					),
				},
				WaitUntilStable: true,
			},
		)
		if err != nil {
			return nil, err
		}
	} else {
		// Destroy the mock resource to test the link interacting
		// with the plugin service to destroy resources.
		err := input.ResourceDeployService.Destroy(
			ctx,
			"aws/lambda/function",
			&provider.ResourceDestroyInput{
				InstanceID:    changes.AppliedResourceInfo.InstanceID,
				ResourceID:    changes.AppliedResourceInfo.ResourceID,
				ResourceState: changes.AppliedResourceInfo.CurrentResourceState,
				ProviderContext: provider.NewProviderContextFromLinkContext(
					input.LinkContext,
					"aws",
				),
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return LinkLambdaDynamoDBUpdateIntermediaryResourcesOutput(), nil
}

func LinkLambdaDynamoDBUpdateIntermediaryResourcesOutput() *provider.LinkUpdateIntermediaryResourcesOutput {
	return &provider.LinkUpdateIntermediaryResourcesOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}
}
