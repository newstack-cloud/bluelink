package corefunctions

import (
	"context"
	"fmt"
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// EqFunction provides the implementation of
// a function that checks the equality of two values.
type EqFunction struct {
	definition *function.Definition
}

// NewEqFunction creates a new instance of the EqFunction with
// a complete function definition.
func NewEqFunction() provider.Function {
	return &EqFunction{
		definition: &function.Definition{
			Description: "A function that determines whether two values of the same type are equal.",
			FormattedDescription: "A function that determines whether two values of the same type are equal.\n\n" +
				"**Examples:**\n\n" +
				"```\n${eq(variables.environment, \"prod\")}\n```",
			Parameters: []function.Parameter{
				&function.AnyParameter{
					Label:       "a",
					Description: "The left-hand side of the equality comparison.",
				},
				&function.AnyParameter{
					Label:       "b",
					Description: "The right-hand side of the equality comparison.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "boolean",
					Type:  function.ValueTypeBool,
				},
				Description: "True, if the two values are equal, false otherwise.",
			},
		},
	}
}

func (f *EqFunction) GetDefinition(
	ctx context.Context,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	return &provider.FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *EqFunction) Call(
	ctx context.Context,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	var lhs interface{}
	var rhs interface{}
	if err := input.Arguments.GetMultipleVars(ctx, &lhs, &rhs); err != nil {
		return nil, err
	}

	lhsVal := reflect.ValueOf(lhs)
	rhsVal := reflect.ValueOf(rhs)

	if lhsVal.Kind() == reflect.Pointer {
		lhsVal = lhsVal.Elem()
	}

	if rhsVal.Kind() == reflect.Pointer {
		rhsVal = rhsVal.Elem()
	}

	if lhsVal.Kind() != rhsVal.Kind() {
		return nil, function.NewFuncCallError(
			fmt.Sprintf(
				"expected both values to be of the same type, got %s and %s",
				lhsVal.Kind(),
				rhsVal.Kind(),
			),
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	lhsValComparable, isLHSComparable := lhsVal.Interface().(core.Comparable[any])
	rhsValComparable, isRHSComparable := rhsVal.Interface().(core.Comparable[any])

	if isLHSComparable && isRHSComparable {
		return &provider.FunctionCallOutput{
			ResponseData: lhsValComparable.Equal(rhsValComparable),
		}, nil
	}

	if lhsVal.Kind() == reflect.Map && rhsVal.Kind() == reflect.Map {
		return &provider.FunctionCallOutput{
			ResponseData: reflect.DeepEqual(lhsVal.Interface(), rhsVal.Interface()),
		}, nil
	}

	return &provider.FunctionCallOutput{
		ResponseData: lhsVal.Equal(rhsVal),
	}, nil
}
