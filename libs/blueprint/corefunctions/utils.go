package corefunctions

import (
	"fmt"
	"reflect"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// The function environment expects lists to be passed around as []interface{}
// where concrete types are asserted by functions that consume them.
func intoInterfaceSlice[Type any](slice []Type) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

type numberTypeInfo struct {
	lhs numberTypeElemInfo
	rhs numberTypeElemInfo
}

type numberTypeElemInfo struct {
	intVal   *int64
	floatVal *float64
}

func checkNumberTypes(input *provider.FunctionCallInput, lhs interface{}, rhs interface{}) (numberTypeInfo, error) {
	lhsInt, isLHSInt := lhs.(int64)
	rhsInt, isRHSInt := rhs.(int64)

	lhsFloat, isLHSFloat := lhs.(float64)
	rhsFloat, isRHSFloat := rhs.(float64)

	if !isLHSInt && !isLHSFloat {
		return numberTypeInfo{}, function.NewFuncCallError(
			fmt.Sprintf(
				"expected the left-hand side of the comparison to be a number, got %s",
				reflect.TypeOf(lhs),
			),
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if !isRHSInt && !isRHSFloat {
		return numberTypeInfo{}, function.NewFuncCallError(
			fmt.Sprintf(
				"expected the right-hand side of the comparison to be a number, got %s",
				reflect.TypeOf(rhs),
			),
			function.FuncCallErrorCodeInvalidArgumentType,
			input.CallContext.CallStackSnapshot(),
		)
	}

	lhsIntPtr := &lhsInt
	if !isLHSInt {
		lhsIntPtr = nil
	}

	lhsFloatPtr := &lhsFloat
	if !isLHSFloat {
		lhsFloatPtr = nil
	}

	rhsIntPtr := &rhsInt
	if !isRHSInt {
		rhsIntPtr = nil
	}

	rhsFloatPtr := &rhsFloat
	if !isRHSFloat {
		rhsFloatPtr = nil
	}

	return numberTypeInfo{
		lhs: numberTypeElemInfo{
			intVal:   lhsIntPtr,
			floatVal: lhsFloatPtr,
		},
		rhs: numberTypeElemInfo{
			intVal:   rhsIntPtr,
			floatVal: rhsFloatPtr,
		},
	}, nil
}
