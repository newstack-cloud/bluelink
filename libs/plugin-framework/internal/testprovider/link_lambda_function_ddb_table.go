package testprovider

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func linkLambdaFunctionDynamoDBTable() provider.Link {
	descriptionInfo := LinkLambdaFunctionDDBTableTypeDescriptionOutput()
	return &providerv1.LinkDefinition{
		ResourceTypeA:         "aws/lambda/function",
		ResourceTypeB:         "aws/dynamodb/table",
		Kind:                  provider.LinkKindHard,
		PriorityResource:      provider.LinkPriorityResourceB,
		PlainTextDescription:  descriptionInfo.PlainTextDescription,
		FormattedDescription:  descriptionInfo.MarkdownDescription,
		PlainTextSummary:      descriptionInfo.PlainTextSummary,
		FormattedSummary:      descriptionInfo.MarkdownSummary,
		AnnotationDefinitions: LinkLambdaFunctionDDBTableAnnotations(),
		CardinalityA:          LinkLambdaFunctionDDBTableCardinalityA(),
		CardinalityB:          LinkLambdaFunctionDDBTableCardinalityB(),
		Provides:              LinkLambdaFunctionDDBTableProvides(),
		Requires:              LinkLambdaFunctionDDBTableRequires(),
		// The function is written, the table is read, which is the shape of every
		// access link and the case the declaration exists for.
		Modifies:                         provider.LinkModifiesResourceA,
		ValidateFunc:                     linkLambdaFunctionDDBTableValidate,
		StageChangesFunc:                 linkLambdaFunctionDDBTableStageChanges,
		UpdateLinkedResourcesFunc:        linkLambdaFunctionDDBTableUpdateLinkedResources,
		UpdateIntermediaryResourcesFunc:  linkLambdaFunctionDDBTableUpdateIntermediaryResources,
		GetIntermediaryExternalStateFunc: linkLambdaFunctionDDBTableGetIntermediaryExternalState,
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
			AppliesTo:     provider.LinkAnnotationResourceA,
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

func linkLambdaFunctionDDBTableUpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	// A link often has to modify a shared intermediary resource, under a lock, before it
	// can touch resource A at all, so the resource service and the link ID that scopes
	// its locks have to reach the plugin here and not only in the intermediaries phase.
	err := checkLinkResourceServiceAccess(ctx, input, input.ResourceAInfo)
	if err != nil {
		return nil, err
	}

	return LinkLambdaDynamoDBUpdateLinkedResourcesOutput(input.LinkID), nil
}

// Exercises the resource service the way a real link would, so the test fails if the
// service is not reachable from a resource update rather than only from the
// intermediaries update.
// lookupTarget is always the lambda function of the pair, as the test host has no
// implementation registered for the dynamodb table.
func checkLinkResourceServiceAccess(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
	lookupTarget *provider.ResourceInfo,
) error {
	if input.ResourceService == nil {
		return errors.New("no resource service was provided to the link resource update")
	}

	_, err := input.ResourceService.LookupResourceInState(
		ctx,
		&provider.ResourceLookupInput{
			InstanceID: lookupTarget.InstanceID,
			ExternalID: core.StringValue(
				lookupTarget.CurrentResourceState.SpecData.Fields["arn"],
			),
			ResourceType: lookupTarget.CurrentResourceState.Type,
			ProviderContext: provider.NewProviderContextFromLinkContext(
				input.LinkContext,
				"aws",
			),
		},
	)

	return err
}

func LinkLambdaDynamoDBUpdateLinkedResourcesOutput(linkID string) *provider.LinkUpdateLinkedResourcesOutput {
	return &provider.LinkUpdateLinkedResourcesOutput{
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"environmentVariables.TABLE_NAME_ordersTable": core.MappingNodeFromString("orders-updated"),
				// Echoed back so the test can assert the link ID survived the trip to the
				// plugin. Locks acquired here are released by the container against this
				// same ID, so an empty one would leak every lock the link takes.
				"observedLinkId": core.MappingNodeFromString(linkID),
			},
		},
		ResourceDataMappings: map[string]string{
			"saveOrderFunction::spec.environment.variables.TABLE_NAME_ordersTable": "saveOrderFunction.environmentVariables.TABLE_NAME_ordersTable",
		},
	}
}

func LinkLambdaDynamoDBUpdateResourceBOutput() *provider.LinkUpdateLinkedResourcesOutput {
	return &provider.LinkUpdateLinkedResourcesOutput{
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

	// Check existing resource state, typically this would only be called
	// when the intermediary resource already exists in the blueprint and is being updated.
	_, err := input.ResourceService.LookupResourceInState(
		ctx,
		&provider.ResourceLookupInput{
			InstanceID: input.ResourceAInfo.InstanceID,
			ExternalID: core.StringValue(
				input.ResourceAInfo.CurrentResourceState.SpecData.Fields["arn"],
			),
			ResourceType: input.ResourceAInfo.CurrentResourceState.Type,
			ProviderContext: provider.NewProviderContextFromLinkContext(
				input.LinkContext,
				"aws",
			),
		},
	)
	if err != nil {
		return nil, err
	}

	if input.LinkUpdateType == provider.LinkUpdateTypeUpdate ||
		input.LinkUpdateType == provider.LinkUpdateTypeCreate {
		_, err := input.ResourceService.Deploy(
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

		err = input.ResourceService.AcquireResourceLock(
			ctx,
			&provider.AcquireResourceLockInput{
				InstanceID:   changes.AppliedResourceInfo.InstanceID,
				ResourceName: "exampleResourceToLock",
				ProviderContext: provider.NewProviderContextFromLinkContext(
					input.LinkContext,
					"aws",
				),
				AcquiredBy: input.LinkID,
			},
		)
		if err != nil {
			return nil, err
		}
	} else {
		// Destroy the mock resource to test the link interacting
		// with the plugin service to destroy resources.
		err := input.ResourceService.Destroy(
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

func linkLambdaFunctionDDBTableGetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return LinkLambdaDynamoDBGetIntermediaryExternalStateOutput(), nil
}

// LinkLambdaFunctionDDBTableCardinalityA returns the cardinality for the A side
// of the lambda function -> dynamodb table link.
func LinkLambdaFunctionDDBTableCardinalityA() provider.LinkCardinality {
	return provider.LinkCardinality{
		Min: 0,
		Max: 0,
	}
}

// LinkLambdaFunctionDDBTableCardinalityB returns the cardinality for the B side
// of the lambda function -> dynamodb table link.
func LinkLambdaFunctionDDBTableCardinalityB() provider.LinkCardinality {
	return provider.LinkCardinality{
		Min: 1,
		Max: 5,
	}
}

func linkLambdaFunctionDDBTableValidate(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return LinkLambdaFunctionDDBTableValidateOutput(), nil
}

// LinkLambdaFunctionDDBTableValidateOutput returns the expected output for
// validating the lambda function -> dynamodb table link.
func LinkLambdaFunctionDDBTableValidateOutput() *provider.LinkValidateOutput {
	return &provider.LinkValidateOutput{
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "Lambda function has broad access to DynamoDB table",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   10,
							Column: 5,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   15,
							Column: 10,
						},
					},
				},
			},
		},
	}
}

// LinkLambdaDynamoDBGetIntermediaryExternalStateOutput returns test output for
// the GetIntermediaryExternalState method.
func LinkLambdaDynamoDBGetIntermediaryExternalStateOutput() *provider.LinkGetIntermediaryExternalStateOutput {
	return &provider.LinkGetIntermediaryExternalStateOutput{
		IntermediaryStates: map[string]*provider.IntermediaryExternalState{
			"iam-role-1": {
				ResourceID:   "iam-role-1",
				ResourceType: "aws/iam/role",
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"arn":      core.MappingNodeFromString("arn:aws:iam::123456789012:role/lambda-dynamodb-access"),
						"roleName": core.MappingNodeFromString("lambda-dynamodb-access"),
					},
				},
				Exists: true,
			},
			"iam-policy-1": {
				ResourceID:   "iam-policy-1",
				ResourceType: "aws/iam/policy",
				SpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"arn":        core.MappingNodeFromString("arn:aws:iam::123456789012:policy/dynamodb-read-write"),
						"policyName": core.MappingNodeFromString("dynamodb-read-write"),
					},
				},
				Exists: true,
			},
		},
	}
}

// LinkLambdaFunctionDDBTableProvides returns the capabilities the link establishes,
// used to verify that capability declarations survive the plugin protocol.
func LinkLambdaFunctionDDBTableProvides() []provider.LinkCapability {
	return []provider.LinkCapability{
		{
			Name:     "test.provider/table-access-granted",
			Resource: provider.LinkPriorityResourceB,
		},
	}
}

// LinkLambdaFunctionDDBTableRequires returns the capabilities the link depends on,
// used to verify that capability declarations survive the plugin protocol.
func LinkLambdaFunctionDDBTableRequires() []provider.LinkCapability {
	return []provider.LinkCapability{
		{
			Name:      "test.provider/network-attached",
			Resource:  provider.LinkPriorityResourceA,
			MustExist: true,
		},
	}
}
