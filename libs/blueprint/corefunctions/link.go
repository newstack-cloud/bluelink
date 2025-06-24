package corefunctions

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// LinkFunction provides the implementation of the
// core "link" function defined in the blueprint specification.
type LinkFunction struct {
	definition                   *function.Definition
	linkStateRetriever           LinkStateRetriever
	blueprintInstanceIDRetriever BlueprintInstanceIDRetriever
}

// BlueprintInstanceIDRetriever is a function that can be used to
// retrieve the instance ID for the current blueprint execution environment.
type BlueprintInstanceIDRetriever func(ctx context.Context) (string, error)

// NewLinkFunction creates a new instance of the LinkFunction with
// a complete function definition.
// This function takes a state container for the current blueprint execution
// environment as a parameter to allow the link function to retrieve
// and return the state of a link between two resources.
// This also requires a function that resolves a blueprint instance ID
// for the current blueprint execution environment for the blueprint in
// which the link function is called.
func NewLinkFunction(
	linkStateRetriever LinkStateRetriever,
	blueprintInstanceIDRetriever BlueprintInstanceIDRetriever,
) provider.Function {
	return &LinkFunction{
		linkStateRetriever:           linkStateRetriever,
		blueprintInstanceIDRetriever: blueprintInstanceIDRetriever,
		definition: &function.Definition{
			Description: "A function to retrieve the state of a link between two resources.",
			FormattedDescription: "A function to retrieve the state of a link between two resources.\n\n" +
				"**Examples:**\n\n" +
				"Using resource references:\n" +
				"```${link(resources.orderApi, resources.createOrderFunction)}```\n" +
				"Using implicit resource references (identifiers without a namespace are either resources or functions):\n" +
				"```${link(orderApi, listOrdersFunction)}```\n" +
				"Using string literals:\n" +
				"```${link(\"orderApi\", \"deleteOrderFunction\")}```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label: "resourceA",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Type: function.ValueTypeString,
						},
						&function.ValueTypeDefinitionObject{
							Label: "ResourceRef",
							AttributeTypes: map[string]function.AttributeType{
								"name": {
									Type: &function.ValueTypeDefinitionScalar{
										Type: function.ValueTypeString,
									},
								},
							},
						},
					},
					Description: "Resource A in the relationship.",
				},
				&function.AnyParameter{
					Label: "resourceB",
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Type: function.ValueTypeString,
						},
						&function.ValueTypeDefinitionObject{
							Label: "ResourceRef",
							AttributeTypes: map[string]function.AttributeType{
								"name": {
									Type: &function.ValueTypeDefinitionScalar{
										Type: function.ValueTypeString,
									},
								},
							},
						},
					},
					Description: "Resource A in the relationship.",
				},
			},
			Return: &function.ObjectReturn{
				Description: "An object containing all the information about the link between " +
					"the two resources made available by the provider that powers the link.",
			},
		},
	}
}

func (f *LinkFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *LinkFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var resourceA interface{}
	var resourceB interface{}
	err := input.Arguments.GetMultipleVars(ctx, &resourceA, &resourceB)
	if err != nil {
		return nil, err
	}

	resourceAStr, err := extractResourceName(input, 0, resourceA)
	if err != nil {
		return nil, err
	}

	resourceBStr, err := extractResourceName(input, 1, resourceB)
	if err != nil {
		return nil, err
	}

	instanceId, err := f.blueprintInstanceIDRetriever(ctx)
	if err != nil {
		return nil, function.NewFuncCallError(
			"failed to retrieve current blueprint instance ID",
			function.FuncCallErrorCodeFunctionCall,
			input.CallContext.CallStackSnapshot(),
		)
	}

	linkState, err := f.linkStateRetriever.GetByName(
		ctx,
		instanceId,
		fmt.Sprintf("%s::%s", resourceAStr, resourceBStr),
	)

	if err != nil {
		return nil, function.NewFuncCallError(
			fmt.Sprintf(
				"failed to retrieve link state for %q and %q: %v",
				resourceAStr,
				resourceBStr,
				err,
			),
			function.FuncCallErrorCodeFunctionCall,
			input.CallContext.CallStackSnapshot(),
		)
	}

	linkStateMap := linkStateToInterfaceMap(linkState)

	return &provider.FunctionCallOutput{
		ResponseData: linkStateMap,
	}, nil
}

func extractResourceName(input *provider.FunctionCallInput, index int, resourceArg interface{}) (string, error) {
	resourceStr, isResourceStr := resourceArg.(string)
	if !isResourceStr {
		resourceName, isMapAttr := resourceArg.(map[string]interface{})["name"].(string)
		if !isMapAttr {
			return "", function.NewFuncCallError(
				fmt.Sprintf("argument %d must be a string or a resource reference.", index),
				function.FuncCallErrorCodeInvalidInput,
				input.CallContext.CallStackSnapshot(),
			)
		}
		return resourceName, nil
	}

	return resourceStr, nil
}

// Converts a link state struct to a map of string to interface
// to be compatible with any function that the value can be passed
// to.
func linkStateToInterfaceMap(linkState state.LinkState) map[string]interface{} {
	intermediaryResourceStates := make([]interface{}, 0, len(linkState.IntermediaryResourceStates))
	for _, state := range linkState.IntermediaryResourceStates {
		intermediaryResourceStates = append(intermediaryResourceStates, linkResourceStateToInterfaceMap(state))
	}

	return map[string]interface{}{
		"intermediaryResourceStates": intermediaryResourceStates,
		"linkData":                   linkState.Data,
	}
}

func linkResourceStateToInterfaceMap(resourceState *state.LinkIntermediaryResourceState) map[string]interface{} {
	return map[string]interface{}{
		"resourceID":       resourceState.ResourceID,
		"resourceSpecData": resourceState.ResourceSpecData,
	}
}

// LinkStateRetriever is an interface that defines a service
// that can be used to retrieve the state of a link between
// two resources in a blueprint instance.
type LinkStateRetriever interface {
	// Get deals with retrieving the state for a given link
	// in the provided blueprint instance.
	GetByName(ctx context.Context, instanceID string, linkName string) (state.LinkState, error)
}
