package corefunctions

import (
	"context"
	"fmt"
	"strings"

	"github.com/newstack-cloud/celerity/libs/blueprint/function"
	"github.com/newstack-cloud/celerity/libs/blueprint/provider"
	"github.com/newstack-cloud/celerity/libs/common/core"
)

// ContainsFunction provides the implementation of
// a function that checks if a string has a suffix.
type ContainsFunction struct {
	definition *function.Definition
}

// NewContainsFunction creates a new instance of the ContainsFunction with
// a complete function definition.
func NewContainsFunction() provider.Function {
	return &ContainsFunction{
		definition: &function.Definition{
			Description: "Checks if a string contains a given substring or if an array contains a given value.",
			FormattedDescription: "Checks if a string contains a given substring or if an array contains a given value.\n\n" +
				"**Examples:**\n\n" +
				"```\n${contains(values.cacheClusterConfig.host, \"celerityframework.com\")}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					UnionTypes: []function.ValueTypeDefinition{
						&function.ValueTypeDefinitionScalar{
							Label: "string",
							Type:  function.ValueTypeString,
						},
						&function.ValueTypeDefinitionList{
							Label: "substring",
							ElementType: &function.ValueTypeDefinitionAny{
								Label: "any",
								Type:  function.ValueTypeAny,
							},
						},
					},
					Description: "A valid string literal, reference or function call yielding a return value " +
						"representing a string or array to search.",
				},
				&function.AnyParameter{
					Label:       "any",
					Description: "The substring or value to search for in the string or array.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "True, if the substring or value is found in the string or array, false otherwise.",
			},
		},
	}
}

func (f *ContainsFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *ContainsFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var haystack interface{}
	var needle interface{}
	if err := input.Arguments.GetMultipleVars(ctx, &haystack, &needle); err != nil {
		return nil, err
	}

	strHaystack, isHaystackStr := haystack.(string)
	strNeedle, isNeedleStr := needle.(string)
	if isHaystackStr && isNeedleStr {
		hasSubStr := strings.Contains(strHaystack, strNeedle)
		return &provider.FunctionCallOutput{
			ResponseData: hasSubStr,
		}, nil
	}

	if isHaystackStr && !isNeedleStr {
		return nil, function.NewFuncCallError(
			fmt.Sprintf(
				"Invalid input type. Expected string for item to search"+
					" for in a string search space, received %T",
				needle,
			),
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	sliceHaystack, isHaystackSlice := haystack.([]interface{})
	if isHaystackSlice {
		found := false
		i := 0
		for !found && i < len(sliceHaystack) {
			comparable, isComparable := sliceHaystack[i].(core.Comparable[any])
			comparableNeedle, isComparableNeedle := needle.(core.Comparable[any])
			if isComparable && isComparableNeedle {
				found = comparable.Equal(comparableNeedle)
			} else {
				found = sliceHaystack[i] == needle
			}
			i += 1
		}
		return &provider.FunctionCallOutput{
			ResponseData: found,
		}, nil
	}

	return nil, function.NewFuncCallError(
		fmt.Sprintf(
			"Invalid input type. Expected string or array for search space, received %T",
			haystack,
		),
		function.FuncCallErrorCodeInvalidInput,
		input.CallContext.CallStackSnapshot(),
	)
}
