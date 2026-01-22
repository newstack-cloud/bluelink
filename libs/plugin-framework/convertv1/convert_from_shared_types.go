package convertv1

import (
	"errors"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schemapb"
	"github.com/newstack-cloud/bluelink/libs/blueprint/serialisation"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pbutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// FromPBScalarMap converts a map of protobuf ScalarValues to a map of core ScalarValues
// compatible with the blueprint framework.
func FromPBScalarMap(m map[string]*schemapb.ScalarValue) (map[string]*core.ScalarValue, error) {
	coreMap := make(map[string]*core.ScalarValue)
	for k, scalar := range m {
		coreScalar, err := serialisation.FromScalarValuePB(scalar, true /* optional */)
		if err != nil {
			return nil, err
		}
		coreMap[k] = coreScalar
	}
	return coreMap, nil
}

// FromPBScalarSlice converts a slice of protobuf ScalarValues to a slice of core ScalarValues
// compatible with the blueprint framework.
func FromPBScalarSlice(s []*schemapb.ScalarValue) ([]*core.ScalarValue, error) {
	coreSlice := make([]*core.ScalarValue, len(s))
	for i, scalar := range s {
		coreScalar, err := serialisation.FromScalarValuePB(scalar, true /* optional */)
		if err != nil {
			return nil, err
		}
		coreSlice[i] = coreScalar
	}
	return coreSlice, nil
}

// FromPBConfigDefinitionResponse converts a ConfigDefinitionResponse from a protobuf message
// to a core type compatible with the blueprint framework.
func FromPBConfigDefinitionResponse(
	resp *sharedtypesv1.ConfigDefinitionResponse,
	pluginAction errorsv1.PluginAction,
) (*core.ConfigDefinition, error) {
	if resp == nil {
		return nil, nil
	}

	switch result := resp.Response.(type) {
	case *sharedtypesv1.ConfigDefinitionResponse_ConfigDefinition:
		return fromPBConfigDefinition(result.ConfigDefinition)
	case *sharedtypesv1.ConfigDefinitionResponse_ErrorResponse:
		return nil, errorsv1.CreateErrorFromResponse(
			result.ErrorResponse,
			pluginAction,
		)
	}

	return nil, errorsv1.CreateGeneralError(
		errorsv1.ErrUnexpectedResponseType(
			pluginAction,
		),
		pluginAction,
	)
}

func fromPBConfigDefinition(
	pbConfigDef *sharedtypesv1.ConfigDefinition,
) (*core.ConfigDefinition, error) {
	if pbConfigDef == nil {
		return nil, nil
	}

	coreFields := map[string]*core.ConfigFieldDefinition{}
	for fieldName, pbFieldDef := range pbConfigDef.Fields {
		coreFieldDef, err := fromPBConfigFieldDefinition(pbFieldDef)
		if err != nil {
			return nil, err
		}
		coreFields[fieldName] = coreFieldDef
	}

	return &core.ConfigDefinition{
		Fields:                coreFields,
		AllowAdditionalFields: pbConfigDef.AllowAdditionalFields,
	}, nil
}

func fromPBConfigFieldDefinition(
	pbFieldDefinition *sharedtypesv1.ConfigFieldDefinition,
) (*core.ConfigFieldDefinition, error) {
	if pbFieldDefinition == nil {
		return nil, nil
	}

	defaultValue, err := serialisation.FromScalarValuePB(
		pbFieldDefinition.DefaultValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	allowedValues, err := FromPBScalarSlice(pbFieldDefinition.AllowedValues)
	if err != nil {
		return nil, err
	}

	examples, err := FromPBScalarSlice(pbFieldDefinition.Examples)
	if err != nil {
		return nil, err
	}

	return &core.ConfigFieldDefinition{
		Type:          FromPBScalarType(pbFieldDefinition.Type),
		Label:         pbFieldDefinition.Label,
		Description:   pbFieldDefinition.Description,
		DefaultValue:  defaultValue,
		AllowedValues: allowedValues,
		Examples:      examples,
		Secret:        pbFieldDefinition.Secret,
		Required:      pbFieldDefinition.Required,
	}, nil
}

// FromPBScalarType converts a ScalarType from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBScalarType(
	scalarType sharedtypesv1.ScalarType,
) core.ScalarType {
	switch scalarType {
	case sharedtypesv1.ScalarType_SCALAR_TYPE_INTEGER:
		return core.ScalarTypeInteger
	case sharedtypesv1.ScalarType_SCALAR_TYPE_FLOAT:
		return core.ScalarTypeFloat
	case sharedtypesv1.ScalarType_SCALAR_TYPE_BOOLEAN:
		return core.ScalarTypeBool
	}

	return core.ScalarTypeString
}

// FromPBFunctionDefinitionRequest converts a FunctionDefinitionRequest from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBFunctionDefinitionRequest(
	req *sharedtypesv1.FunctionDefinitionRequest,
) (*provider.FunctionGetDefinitionInput, error) {
	if req == nil {
		return nil, errors.New("function definition request cannot be nil")
	}

	params, err := fromPBBlueprintParams(req.Params)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionGetDefinitionInput{
		Params: params,
	}, nil
}

// FromPBFunctionCallRequest converts a FunctionCallRequest from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBFunctionCallRequest(
	req *sharedtypesv1.FunctionCallRequest,
	functionRegistry provider.FunctionRegistry,
) (*provider.FunctionCallInput, error) {
	if req == nil {
		return nil, errors.New("function call request cannot be nil")
	}

	deserialisedArgs, err := fromPBFunctionCallArgs(req.Args)
	if err != nil {
		return nil, err
	}

	callCtxInfo, err := fromPBFunctionCallContext(req.CallContext)
	if err != nil {
		return nil, err
	}
	// Ensure the function registry is scoped to the current call context
	// based on the stack received over the "wire".
	scopedFunctionRegistry := functionRegistry.ForCallContext(callCtxInfo.stack)
	callCtx := subengine.NewFunctionCallContext(
		callCtxInfo.stack,
		scopedFunctionRegistry,
		callCtxInfo.params,
		callCtxInfo.location,
	)
	args := callCtx.NewCallArgs(deserialisedArgs...)
	callCtx.NewCallArgs()

	return &provider.FunctionCallInput{
		Arguments:   args,
		CallContext: callCtx,
	}, nil
}

// FromPBFunctionCallResult converts a FunctionCallResult from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBFunctionCallResult(
	res *sharedtypesv1.FunctionCallResult,
) (*provider.FunctionCallOutput, error) {
	if res == nil {
		return nil, nil
	}

	responseData, err := pbutils.ConvertPBAnyToInterface(res.ResponseData)
	if err != nil {
		return nil, err
	}

	funcInfo, err := fromPBFunctionRuntimeInfo(res.FunctionInfo)
	if err != nil {
		return nil, err
	}

	return &provider.FunctionCallOutput{
		ResponseData: responseData,
		FunctionInfo: funcInfo,
	}, nil
}

func fromPBFunctionRuntimeInfo(
	pbFuncInfo *sharedtypesv1.FunctionRuntimeInfo,
) (provider.FunctionRuntimeInfo, error) {
	if pbFuncInfo == nil || pbFuncInfo.FunctionName == "" {
		return provider.FunctionRuntimeInfo{}, nil
	}

	partialArgs, err := pbutils.ConvertPBAnyToInterface(pbFuncInfo.PartialArgs)
	if err != nil {
		return provider.FunctionRuntimeInfo{}, err
	}

	partialArgsSlice, ok := partialArgs.([]any)
	if !ok {
		return provider.FunctionRuntimeInfo{}, fmt.Errorf(
			"expected partial args to be a []any, got %T",
			partialArgs,
		)
	}

	return provider.FunctionRuntimeInfo{
		FunctionName: pbFuncInfo.FunctionName,
		PartialArgs:  partialArgsSlice,
		ArgsOffset:   int(pbFuncInfo.ArgsOffset),
	}, nil
}

// FromPBFunctionDefinition converts a FunctionDefinition from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBFunctionDefinition(
	pbFuncDef *sharedtypesv1.FunctionDefinition,
) (*function.Definition, error) {
	if pbFuncDef == nil {
		return nil, nil
	}

	params, err := fromPBFunctionParams(pbFuncDef.Parameters)
	if err != nil {
		return nil, err
	}

	returnDef, err := fromPBFunctionReturn(pbFuncDef.Return)
	if err != nil {
		return nil, err
	}

	return &function.Definition{
		Name:                 pbFuncDef.Name,
		Summary:              pbFuncDef.Summary,
		FormattedSummary:     pbFuncDef.FormattedSummary,
		Description:          pbFuncDef.Description,
		FormattedDescription: pbFuncDef.FormattedDescription,
		Parameters:           params,
		Return:               returnDef,
		Internal:             pbFuncDef.Internal,
	}, nil
}

func fromPBFunctionParams(
	pbParams []*sharedtypesv1.FunctionParameter,
) ([]function.Parameter, error) {
	params := make([]function.Parameter, len(pbParams))
	for i, pbParam := range pbParams {
		param, err := fromPBFunctionParameter(pbParam)
		if err != nil {
			return nil, err
		}
		params[i] = param
	}
	return params, nil
}

func fromPBFunctionParameter(
	paramPB *sharedtypesv1.FunctionParameter,
) (function.Parameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	switch concreteParamType := paramPB.Parameter.(type) {
	case *sharedtypesv1.FunctionParameter_ScalarParameter:
		return fromPBScalarParameter(concreteParamType.ScalarParameter)
	case *sharedtypesv1.FunctionParameter_ListParameter:
		return fromPBListParameter(concreteParamType.ListParameter)
	case *sharedtypesv1.FunctionParameter_MapParameter:
		return fromPBMapParameter(concreteParamType.MapParameter)
	case *sharedtypesv1.FunctionParameter_ObjectParameter:
		return fromPBObjectParameter(concreteParamType.ObjectParameter)
	case *sharedtypesv1.FunctionParameter_FunctionTypeParameter:
		return fromPBFunctionTypeParameter(
			concreteParamType.FunctionTypeParameter,
		)
	case *sharedtypesv1.FunctionParameter_VariadicParameter:
		return fromPBVariadicParameter(
			concreteParamType.VariadicParameter,
		)
	case *sharedtypesv1.FunctionParameter_AnyParameter:
		return fromPBAnyParameter(
			concreteParamType.AnyParameter,
		)
	}

	return nil, fmt.Errorf(
		"unknown parameter type: %T",
		paramPB.Parameter,
	)
}

func fromPBScalarParameter(
	paramPB *sharedtypesv1.FunctionScalarParameter,
) (*function.ScalarParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	valueTypeDef, err := fromPBFunctionValueTypeDefinition(paramPB.Type)
	if err != nil {
		return nil, err
	}

	return &function.ScalarParameter{
		Name:                 paramPB.Name,
		Label:                paramPB.Label,
		Type:                 valueTypeDef,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Optional:             paramPB.Optional,
	}, nil
}

func fromPBListParameter(
	paramPB *sharedtypesv1.FunctionListParameter,
) (*function.ListParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	elementTypeDef, err := fromPBFunctionValueTypeDefinition(paramPB.ElementType)
	if err != nil {
		return nil, err
	}

	return &function.ListParameter{
		Name:                 paramPB.Name,
		Label:                paramPB.Label,
		ElementType:          elementTypeDef,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Optional:             paramPB.Optional,
	}, nil
}

func fromPBMapParameter(
	paramPB *sharedtypesv1.FunctionMapParameter,
) (*function.MapParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	valueTypeDef, err := fromPBFunctionValueTypeDefinition(paramPB.ElementType)
	if err != nil {
		return nil, err
	}

	return &function.MapParameter{
		Name:                 paramPB.Name,
		Label:                paramPB.Label,
		ElementType:          valueTypeDef,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Optional:             paramPB.Optional,
	}, nil
}

func fromPBObjectParameter(
	paramPB *sharedtypesv1.FunctionObjectParameter,
) (*function.ObjectParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	objectValueType, err := fromPBFunctionValueTypeDefinition(paramPB.ObjectValueType)
	if err != nil {
		return nil, err
	}

	return &function.ObjectParameter{
		Name:                 paramPB.Name,
		Label:                paramPB.Label,
		ObjectValueType:      objectValueType,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Optional:             paramPB.Optional,
	}, nil
}

func fromPBFunctionTypeParameter(
	paramPB *sharedtypesv1.FunctionTypeParameter,
) (*function.FunctionParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	functionTypeDef, err := fromPBFunctionValueTypeDefinition(paramPB.FunctionType)
	if err != nil {
		return nil, err
	}

	return &function.FunctionParameter{
		Name:                 paramPB.Name,
		Label:                paramPB.Label,
		FunctionType:         functionTypeDef,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Optional:             paramPB.Optional,
	}, nil
}

func fromPBVariadicParameter(
	paramPB *sharedtypesv1.FunctionVariadicParameter,
) (*function.VariadicParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	valueTypeDef, err := fromPBFunctionValueTypeDefinition(paramPB.Type)
	if err != nil {
		return nil, err
	}

	return &function.VariadicParameter{
		Label:                paramPB.Label,
		Type:                 valueTypeDef,
		SingleType:           paramPB.SingleType,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Named:                paramPB.Named,
	}, nil
}

func fromPBAnyParameter(
	paramPB *sharedtypesv1.FunctionAnyParameter,
) (*function.AnyParameter, error) {
	if paramPB == nil {
		return nil, nil
	}

	unionTypes, err := fromPBUnionValueTypeDefinitions(paramPB.UnionTypes)
	if err != nil {
		return nil, err
	}

	return &function.AnyParameter{
		Name:                 paramPB.Name,
		Label:                paramPB.Label,
		UnionTypes:           unionTypes,
		Description:          paramPB.Description,
		FormattedDescription: paramPB.FormattedDescription,
		AllowNullValue:       paramPB.AllowNullValue,
		Optional:             paramPB.Optional,
	}, nil
}

func fromPBFunctionReturn(
	returnTypePB *sharedtypesv1.FunctionReturn,
) (function.Return, error) {
	if returnTypePB == nil {
		return nil, nil
	}

	switch concreteReturnType := returnTypePB.Return.(type) {
	case *sharedtypesv1.FunctionReturn_ScalarReturn:
		return fromPBFunctionScalarReturn(
			concreteReturnType.ScalarReturn,
		)
	case *sharedtypesv1.FunctionReturn_ListReturn:
		return fromPBFunctionListReturn(
			concreteReturnType.ListReturn,
		)
	case *sharedtypesv1.FunctionReturn_MapReturn:
		return fromPBFunctionMapReturn(
			concreteReturnType.MapReturn,
		)
	case *sharedtypesv1.FunctionReturn_ObjectReturn:
		return fromPBFunctionObjectReturn(
			concreteReturnType.ObjectReturn,
		)
	case *sharedtypesv1.FunctionReturn_FunctionTypeReturn:
		return fromPBFunctionTypeReturn(
			concreteReturnType.FunctionTypeReturn,
		)
	case *sharedtypesv1.FunctionReturn_AnyReturn:
		return fromPBAnyReturn(
			concreteReturnType.AnyReturn,
		)
	}

	return nil, fmt.Errorf(
		"unknown return type: %T",
		returnTypePB,
	)
}

func fromPBFunctionScalarReturn(
	returnPB *sharedtypesv1.FunctionScalarReturn,
) (*function.ScalarReturn, error) {
	if returnPB == nil {
		return nil, nil
	}

	valueTypeDef, err := fromPBFunctionValueTypeDefinition(returnPB.Type)
	if err != nil {
		return nil, err
	}

	return &function.ScalarReturn{
		Type:                 valueTypeDef,
		Description:          returnPB.Description,
		FormattedDescription: returnPB.FormattedDescription,
	}, nil
}

func fromPBFunctionListReturn(
	returnPB *sharedtypesv1.FunctionListReturn,
) (*function.ListReturn, error) {
	if returnPB == nil {
		return nil, nil
	}

	elementTypeDef, err := fromPBFunctionValueTypeDefinition(returnPB.ElementType)
	if err != nil {
		return nil, err
	}

	return &function.ListReturn{
		ElementType:          elementTypeDef,
		Description:          returnPB.Description,
		FormattedDescription: returnPB.FormattedDescription,
	}, nil
}

func fromPBFunctionMapReturn(
	returnPB *sharedtypesv1.FunctionMapReturn,
) (*function.MapReturn, error) {
	if returnPB == nil {
		return nil, nil
	}

	valueTypeDef, err := fromPBFunctionValueTypeDefinition(returnPB.ElementType)
	if err != nil {
		return nil, err
	}

	return &function.MapReturn{
		ElementType:          valueTypeDef,
		Description:          returnPB.Description,
		FormattedDescription: returnPB.FormattedDescription,
	}, nil
}

func fromPBFunctionObjectReturn(
	returnPB *sharedtypesv1.FunctionObjectReturn,
) (*function.ObjectReturn, error) {
	if returnPB == nil {
		return nil, nil
	}

	objectValueType, err := fromPBFunctionValueTypeDefinition(
		returnPB.ObjectValueType,
	)
	if err != nil {
		return nil, err
	}

	return &function.ObjectReturn{
		ObjectValueType:      objectValueType,
		Description:          returnPB.Description,
		FormattedDescription: returnPB.FormattedDescription,
	}, nil
}

func fromPBFunctionTypeReturn(
	returnPB *sharedtypesv1.FunctionTypeReturn,
) (*function.FunctionReturn, error) {
	if returnPB == nil {
		return nil, nil
	}

	functionTypeDef, err := fromPBFunctionValueTypeDefinition(
		returnPB.FunctionType,
	)
	if err != nil {
		return nil, err
	}

	return &function.FunctionReturn{
		FunctionType:         functionTypeDef,
		Description:          returnPB.Description,
		FormattedDescription: returnPB.FormattedDescription,
	}, nil
}

func fromPBAnyReturn(
	returnPB *sharedtypesv1.FunctionAnyReturn,
) (*function.AnyReturn, error) {
	if returnPB == nil {
		return nil, nil
	}

	unionTypes, err := fromPBUnionValueTypeDefinitions(returnPB.UnionTypes)
	if err != nil {
		return nil, err
	}

	return &function.AnyReturn{
		Type:                 fromPBFunctionValueType(returnPB.Type),
		UnionTypes:           unionTypes,
		Description:          returnPB.Description,
		FormattedDescription: returnPB.FormattedDescription,
	}, nil
}

func fromPBFunctionValueTypeDefinition(
	valueTypePB *sharedtypesv1.FunctionValueTypeDefinition,
) (function.ValueTypeDefinition, error) {
	if valueTypePB == nil {
		return nil, nil
	}

	switch concreteValueType := valueTypePB.ValueTypeDefinition.(type) {
	case *sharedtypesv1.FunctionValueTypeDefinition_ScalarValueType:
		return fromPBFunctionScalarValueTypeDefinition(
			concreteValueType.ScalarValueType,
		)
	case *sharedtypesv1.FunctionValueTypeDefinition_ListValueType:
		return fromPBFunctionListValueTypeDefinition(
			concreteValueType.ListValueType,
		)
	case *sharedtypesv1.FunctionValueTypeDefinition_MapValueType:
		return fromPBFunctionMapValueTypeDefinition(
			concreteValueType.MapValueType,
		)
	case *sharedtypesv1.FunctionValueTypeDefinition_ObjectValueType:
		return fromPBFunctionObjectValueTypeDefinition(
			concreteValueType.ObjectValueType,
		)
	case *sharedtypesv1.FunctionValueTypeDefinition_FunctionValueType:
		return fromPBFunctionTypeValueTypeDefinition(
			concreteValueType.FunctionValueType,
		)
	case *sharedtypesv1.FunctionValueTypeDefinition_AnyValueType:
		return fromPBAnyValueTypeDefinition(
			concreteValueType.AnyValueType,
		)
	}

	return nil, fmt.Errorf(
		"unknown value type: %T",
		valueTypePB,
	)
}

func fromPBFunctionScalarValueTypeDefinition(
	valueTypeDefPB *sharedtypesv1.FunctionScalarValueTypeDefinition,
) (*function.ValueTypeDefinitionScalar, error) {
	if valueTypeDefPB == nil {
		return nil, nil
	}

	return &function.ValueTypeDefinitionScalar{
		Label:                valueTypeDefPB.Label,
		Type:                 fromPBFunctionValueType(valueTypeDefPB.Type),
		Description:          valueTypeDefPB.Description,
		FormattedDescription: valueTypeDefPB.FormattedDescription,
		StringChoices:        valueTypeDefPB.StringChoices,
	}, nil
}

func fromPBFunctionListValueTypeDefinition(
	valueTypeDefPB *sharedtypesv1.FunctionListValueTypeDefinition,
) (*function.ValueTypeDefinitionList, error) {
	if valueTypeDefPB == nil {
		return nil, nil
	}

	elementTypeDef, err := fromPBFunctionValueTypeDefinition(valueTypeDefPB.ElementType)
	if err != nil {
		return nil, err
	}

	return &function.ValueTypeDefinitionList{
		ElementType:          elementTypeDef,
		Label:                valueTypeDefPB.Label,
		Description:          valueTypeDefPB.Description,
		FormattedDescription: valueTypeDefPB.FormattedDescription,
	}, nil
}

func fromPBFunctionMapValueTypeDefinition(
	valueTypeDefPB *sharedtypesv1.FunctionMapValueTypeDefinition,
) (*function.ValueTypeDefinitionMap, error) {
	if valueTypeDefPB == nil {
		return nil, nil
	}

	valueTypeDef, err := fromPBFunctionValueTypeDefinition(valueTypeDefPB.ElementType)
	if err != nil {
		return nil, err
	}

	return &function.ValueTypeDefinitionMap{
		ElementType:          valueTypeDef,
		Label:                valueTypeDefPB.Label,
		Description:          valueTypeDefPB.Description,
		FormattedDescription: valueTypeDefPB.FormattedDescription,
	}, nil
}

func fromPBFunctionObjectValueTypeDefinition(
	valueTypeDefPB *sharedtypesv1.FunctionObjectValueTypeDefinition,
) (*function.ValueTypeDefinitionObject, error) {
	if valueTypeDefPB == nil {
		return nil, nil
	}

	attributeTypes, err := fromPBFunctionObjectAttributeTypes(
		valueTypeDefPB.AttributeTypes,
	)
	if err != nil {
		return nil, err
	}

	return &function.ValueTypeDefinitionObject{
		AttributeTypes:       attributeTypes,
		Label:                valueTypeDefPB.Label,
		Description:          valueTypeDefPB.Description,
		FormattedDescription: valueTypeDefPB.FormattedDescription,
	}, nil
}

func fromPBFunctionObjectAttributeTypes(
	attrTypes map[string]*sharedtypesv1.FunctionObjectAttributeType,
) (map[string]function.AttributeType, error) {
	if attrTypes == nil {
		return nil, nil
	}

	attributeTypes := make(map[string]function.AttributeType, len(attrTypes))
	for key, attrTypePB := range attrTypes {
		attrType, err := fromPBFunctionValueTypeDefinition(attrTypePB.Type)
		if err != nil {
			return nil, err
		}
		attributeTypes[key] = function.AttributeType{
			Type:           attrType,
			AllowNullValue: attrTypePB.AllowNullValue,
		}
	}

	return attributeTypes, nil
}

func fromPBFunctionTypeValueTypeDefinition(
	valueTypeDefPB *sharedtypesv1.FunctionTypeValueTypeDefinition,
) (*function.ValueTypeDefinitionFunction, error) {
	if valueTypeDefPB == nil {
		return nil, nil
	}

	functionDef, err := FromPBFunctionDefinition(valueTypeDefPB.FunctionType)
	if err != nil {
		return nil, err
	}

	return &function.ValueTypeDefinitionFunction{
		Definition:           derefFunctionDefinition(functionDef),
		Label:                valueTypeDefPB.Label,
		Description:          valueTypeDefPB.Description,
		FormattedDescription: valueTypeDefPB.FormattedDescription,
	}, nil
}

func fromPBAnyValueTypeDefinition(
	valueTypeDefPB *sharedtypesv1.FunctionAnyValueTypeDefinition,
) (*function.ValueTypeDefinitionAny, error) {
	if valueTypeDefPB == nil {
		return nil, nil
	}

	unionTypes, err := fromPBUnionValueTypeDefinitions(valueTypeDefPB.UnionTypes)
	if err != nil {
		return nil, err
	}

	return &function.ValueTypeDefinitionAny{
		Type:                 fromPBFunctionValueType(valueTypeDefPB.Type),
		UnionTypes:           unionTypes,
		Label:                valueTypeDefPB.Label,
		Description:          valueTypeDefPB.Description,
		FormattedDescription: valueTypeDefPB.FormattedDescription,
	}, nil
}

func fromPBUnionValueTypeDefinitions(
	unionTypes []*sharedtypesv1.FunctionValueTypeDefinition,
) ([]function.ValueTypeDefinition, error) {
	if unionTypes == nil {
		return nil, nil
	}

	valueTypeDefs := make([]function.ValueTypeDefinition, len(unionTypes))
	for i, unionTypePB := range unionTypes {
		unionTypeDef, err := fromPBFunctionValueTypeDefinition(unionTypePB)
		if err != nil {
			return nil, err
		}
		valueTypeDefs[i] = unionTypeDef
	}

	return valueTypeDefs, nil
}

func derefFunctionDefinition(
	funcDefPtr *function.Definition,
) function.Definition {
	if funcDefPtr == nil {
		return function.Definition{}
	}

	return *funcDefPtr
}

func fromPBFunctionValueType(
	valueTypePB sharedtypesv1.FunctionValueType,
) function.ValueType {
	switch valueTypePB {
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_STRING:
		return function.ValueTypeString
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_INT32:
		return function.ValueTypeInt32
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_INT64:
		return function.ValueTypeInt64
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_UINT32:
		return function.ValueTypeUint32
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_UINT64:
		return function.ValueTypeUint64
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_FLOAT32:
		return function.ValueTypeFloat32
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_FLOAT64:
		return function.ValueTypeFloat64
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_BOOL:
		return function.ValueTypeBool
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_LIST:
		return function.ValueTypeList
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_MAP:
		return function.ValueTypeMap
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_OBJECT:
		return function.ValueTypeObject
	case sharedtypesv1.FunctionValueType_FUNCTION_VALUE_TYPE_FUNCTION:
		return function.ValueTypeFunction
	}

	return function.ValueTypeAny
}

// FromPBDeployResourceRequest converts a DeployResourceRequest from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBDeployResourceRequest(
	req *sharedtypesv1.DeployResourceRequest,
) (*provider.ResourceDeployInput, error) {
	if req == nil {
		return nil, errors.New("deploy resource request cannot be nil")
	}

	changes, err := FromPBResourceChanges(req.Changes)
	if err != nil {
		return nil, err
	}

	providerCtx, err := FromPBProviderContext(req.Context)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceDeployInput{
		InstanceID:      req.InstanceId,
		InstanceName:    req.InstanceName,
		ResourceID:      req.ResourceId,
		Changes:         changes,
		ProviderContext: providerCtx,
	}, nil
}

// FromPBResourceSpecDefinition converts a ResourceSpecDefinition
// from a protobuf message to a core type compatible with the blueprint framework.
func FromPBResourceSpecDefinition(
	pbSpecDef *sharedtypesv1.ResourceSpecDefinition,
) (*provider.ResourceSpecDefinition, error) {
	if pbSpecDef == nil {
		return nil, nil
	}

	resourceDefinitionSchema, err := fromPBResourceDefinitionsSchema(pbSpecDef.Schema)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceSpecDefinition{
		Schema:              resourceDefinitionSchema,
		IDField:             pbSpecDef.IdField,
		DestroyBeforeCreate: pbSpecDef.DestroyBeforeCreate,
		TaggingSupport:      FromPBTaggingSupport(pbSpecDef.TaggingSupport),
	}, nil
}

func fromPBResourceDefinitionsSchema(
	pbSchema *sharedtypesv1.ResourceDefinitionsSchema,
) (*provider.ResourceDefinitionsSchema, error) {
	if pbSchema == nil {
		return nil, nil
	}

	attributes, err := fromPBResourceDefinitionsSchemaMap(pbSchema.Attributes)
	if err != nil {
		return nil, err
	}

	items, err := fromPBResourceDefinitionsSchema(pbSchema.Items)
	if err != nil {
		return nil, err
	}

	mapValues, err := fromPBResourceDefinitionsSchema(pbSchema.MapValues)
	if err != nil {
		return nil, err
	}

	oneOf, err := fromPBResourceDefinitionsSchemaSlice(pbSchema.OneOf)
	if err != nil {
		return nil, err
	}

	defaultValue, err := serialisation.FromMappingNodePB(
		pbSchema.DefaultValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	examples, err := FromPBMappingNodeSlice(pbSchema.Examples)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceDefinitionsSchema{
		Type:                 provider.ResourceDefinitionsSchemaType(pbSchema.Type),
		Label:                pbSchema.Label,
		Description:          pbSchema.Description,
		FormattedDescription: pbSchema.FormattedDescription,
		Attributes:           attributes,
		Items:                items,
		MapValues:            mapValues,
		OneOf:                oneOf,
		Required:             pbSchema.Required,
		Nullable:             pbSchema.Nullable,
		Default:              defaultValue,
		Examples:             examples,
		Computed:             pbSchema.Computed,
		MustRecreate:         pbSchema.MustRecreate,
		Sensitive:            pbSchema.Sensitive,
		IgnoreDrift:          pbSchema.IgnoreDrift,
		TrackDrift:           pbSchema.TrackDrift,
		SortArrayByField:     pbSchema.SortArrayByField,
	}, nil
}

func fromPBResourceDefinitionsSchemaMap(
	pbSchemaMap map[string]*sharedtypesv1.ResourceDefinitionsSchema,
) (map[string]*provider.ResourceDefinitionsSchema, error) {
	if pbSchemaMap == nil {
		return nil, nil
	}

	schemaMap := make(map[string]*provider.ResourceDefinitionsSchema, len(pbSchemaMap))
	for key, pbSchema := range pbSchemaMap {
		schema, err := fromPBResourceDefinitionsSchema(pbSchema)
		if err != nil {
			return nil, err
		}
		schemaMap[key] = schema
	}

	return schemaMap, nil
}

func fromPBResourceDefinitionsSchemaSlice(
	pbSchemaSlice []*sharedtypesv1.ResourceDefinitionsSchema,
) ([]*provider.ResourceDefinitionsSchema, error) {
	if pbSchemaSlice == nil {
		return nil, nil
	}

	schemaSlice := make([]*provider.ResourceDefinitionsSchema, len(pbSchemaSlice))
	for i, pbSchema := range pbSchemaSlice {
		schema, err := fromPBResourceDefinitionsSchema(pbSchema)
		if err != nil {
			return nil, err
		}
		schemaSlice[i] = schema
	}

	return schemaSlice, nil
}

// FromPBResourceChanges converts a Changes from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBResourceChanges(changes *sharedtypesv1.Changes) (*provider.Changes, error) {
	if changes == nil {
		return nil, nil
	}

	appliedResourceInfo, err := FromPBResourceInfo(changes.AppliedResourceInfo)
	if err != nil {
		return nil, err
	}

	modifiedFields, err := fromPBFieldChanges(changes.ModifiedFields)
	if err != nil {
		return nil, err
	}

	newFields, err := fromPBFieldChanges(changes.NewFields)
	if err != nil {
		return nil, err
	}

	newOutboundLinks, err := fromPBLinkChangesMap(changes.NewOutboundLinks)
	if err != nil {
		return nil, err
	}

	outboundLinkChanges, err := fromPBLinkChangesMap(changes.OutboundLinkChanges)
	if err != nil {
		return nil, err
	}

	return &provider.Changes{
		AppliedResourceInfo:       appliedResourceInfo,
		MustRecreate:              changes.MustRecreate,
		ModifiedFields:            modifiedFields,
		NewFields:                 newFields,
		RemovedFields:             changes.RemovedFields,
		UnchangedFields:           changes.UnchangedFields,
		ComputedFields:            changes.ComputedFields,
		FieldChangesKnownOnDeploy: changes.FieldChangesKnownOnDeploy,
		ConditionKnownOnDeploy:    changes.ConditionKnownOnDeploy,
		NewOutboundLinks:          newOutboundLinks,
		OutboundLinkChanges:       outboundLinkChanges,
		RemovedOutboundLinks:      changes.RemovedOutboundLinks,
	}, nil
}

// FromPBResourceInfo converts a ResourceInfo from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBResourceInfo(
	resourceInfo *sharedtypesv1.ResourceInfo,
) (provider.ResourceInfo, error) {
	if resourceInfo == nil {
		return provider.ResourceInfo{}, nil
	}

	resourceState, err := FromPBResourceState(resourceInfo.CurrentResourceState)
	if err != nil {
		return provider.ResourceInfo{}, err
	}

	resourceWithResolvedSubs, err := fromPBResolvedResource(resourceInfo.ResourceWithResolvedSubs)
	if err != nil {
		return provider.ResourceInfo{}, err
	}

	return provider.ResourceInfo{
		ResourceID:               resourceInfo.ResourceId,
		ResourceName:             resourceInfo.ResourceName,
		InstanceID:               resourceInfo.InstanceId,
		CurrentResourceState:     resourceState,
		ResourceWithResolvedSubs: resourceWithResolvedSubs,
	}, nil
}

func fromPBResolvedResource(
	resolvedResource *sharedtypesv1.ResolvedResource,
) (*provider.ResolvedResource, error) {
	if resolvedResource == nil {
		return nil, nil
	}

	description, err := serialisation.FromMappingNodePB(
		resolvedResource.Description,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	resolvedResourceMetadata, err := fromPBResolvedResourceMetadata(
		resolvedResource.Metadata,
	)
	if err != nil {
		return nil, err
	}

	spec, err := serialisation.FromMappingNodePB(
		resolvedResource.Spec,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	resolvedCondition, err := fromPBResolvedResourceCondition(
		resolvedResource.Condition,
	)
	if err != nil {
		return nil, err
	}

	linkSelector := serialisation.FromLinkSelectorPB(
		resolvedResource.LinkSelector,
	)

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: ResourceTypeToString(resolvedResource.Type),
		},
		Description:  description,
		Metadata:     resolvedResourceMetadata,
		Condition:    resolvedCondition,
		LinkSelector: linkSelector,
		Spec:         spec,
	}, nil
}

func fromPBResolvedResourceMetadata(
	pbResourceMetadata *sharedtypesv1.ResolvedResourceMetadata,
) (*provider.ResolvedResourceMetadata, error) {
	if pbResourceMetadata == nil {
		return nil, nil
	}

	displayName, err := serialisation.FromMappingNodePB(
		pbResourceMetadata.DisplayName,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	annotations, err := serialisation.FromMappingNodePB(
		pbResourceMetadata.Annotations,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	custom, err := serialisation.FromMappingNodePB(
		pbResourceMetadata.Custom,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedResourceMetadata{
		DisplayName: displayName,
		Annotations: annotations,
		Labels: &schema.StringMap{
			Values: pbResourceMetadata.Labels,
		},
		Custom: custom,
	}, nil
}

func fromPBResolvedResourceCondition(
	condition *sharedtypesv1.ResolvedResourceCondition,
) (*provider.ResolvedResourceCondition, error) {
	if condition == nil {
		return nil, nil
	}

	and, err := fromPBResolvedResourceConditions(condition.And)
	if err != nil {
		return nil, err
	}

	or, err := fromPBResolvedResourceConditions(condition.Or)
	if err != nil {
		return nil, err
	}

	not, err := fromPBResolvedResourceCondition(condition.Not)
	if err != nil {
		return nil, err
	}

	stringValue, err := serialisation.FromMappingNodePB(
		condition.StringValue,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &provider.ResolvedResourceCondition{
		And:         and,
		Or:          or,
		Not:         not,
		StringValue: stringValue,
	}, nil
}

func fromPBResolvedResourceConditions(
	conditionsPB []*sharedtypesv1.ResolvedResourceCondition,
) ([]*provider.ResolvedResourceCondition, error) {
	if conditionsPB == nil {
		return nil, nil
	}

	conditions := make([]*provider.ResolvedResourceCondition, len(conditionsPB))
	for i, conditionPB := range conditionsPB {
		condition, err := fromPBResolvedResourceCondition(conditionPB)
		if err != nil {
			return nil, err
		}
		conditions[i] = condition
	}

	return conditions, nil
}

func FromPBResourceState(
	pbResourceState *sharedtypesv1.ResourceState,
) (*state.ResourceState, error) {
	if pbResourceState == nil {
		return nil, nil
	}

	specData, err := serialisation.FromMappingNodePB(pbResourceState.SpecData, false /* optional */)
	if err != nil {
		return nil, err
	}

	resourceMetadataState, err := FromPBResourceMetadataState(pbResourceState.Metadata)
	if err != nil {
		return nil, err
	}

	lastDriftDetectedTimestamp := pbutils.IntPtrFromPBWrapper(pbResourceState.LastDriftDetectedTimestamp)

	durations, err := fromPBResourceCompletionDurations(pbResourceState.Durations)
	if err != nil {
		return nil, err
	}

	return &state.ResourceState{
		ResourceID:                 pbResourceState.Id,
		Name:                       pbResourceState.Name,
		Type:                       pbResourceState.Type,
		TemplateName:               pbResourceState.TemplateName,
		InstanceID:                 pbResourceState.InstanceId,
		Status:                     core.ResourceStatus(pbResourceState.Status),
		PreciseStatus:              core.PreciseResourceStatus(pbResourceState.PreciseStatus),
		LastStatusUpdateTimestamp:  int(pbResourceState.LastStatusUpdateTimestamp),
		LastDeployedTimestamp:      int(pbResourceState.LastDeployedTimestamp),
		LastDeployAttemptTimestamp: int(pbResourceState.LastDeployAttemptTimestamp),
		SpecData:                   specData,
		Description:                pbResourceState.Description,
		Metadata:                   resourceMetadataState,
		DependsOnResources:         pbResourceState.DependsOnResources,
		DependsOnChildren:          pbResourceState.DependsOnChildren,
		FailureReasons:             pbResourceState.FailureReasons,
		Drifted:                    pbResourceState.Drifted,
		LastDriftDetectedTimestamp: lastDriftDetectedTimestamp,
		Durations:                  durations,
		SystemMetadata:             FromPBSystemMetadataState(pbResourceState.SystemMetadata),
	}, nil
}

func fromPBResourceCompletionDurations(
	pbDurations *sharedtypesv1.ResourceCompletionDurations,
) (*state.ResourceCompletionDurations, error) {
	if pbDurations == nil {
		return nil, nil
	}

	configCompleteDuration := pbutils.DoublePtrFromPBWrapper(pbDurations.ConfigCompleteDuration)
	return &state.ResourceCompletionDurations{
		ConfigCompleteDuration: configCompleteDuration,
	}, nil
}

// FromPBResourceMetadataState converts a ResourceMetadataState from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBResourceMetadataState(
	pbResourceMetadataState *sharedtypesv1.ResourceMetadataState,
) (*state.ResourceMetadataState, error) {
	if pbResourceMetadataState == nil {
		return nil, nil
	}

	annotations, err := FromPBMappingNodeMap(pbResourceMetadataState.Annotations)
	if err != nil {
		return nil, err
	}

	custom, err := serialisation.FromMappingNodePB(
		pbResourceMetadataState.Custom,
		/* optional */ true,
	)
	if err != nil {
		return nil, err
	}

	return &state.ResourceMetadataState{
		DisplayName: pbResourceMetadataState.DisplayName,
		Annotations: annotations,
		Labels:      pbResourceMetadataState.Labels,
		Custom:      custom,
	}, nil
}

func fromPBFieldChanges(
	pbFieldChanges []*sharedtypesv1.FieldChange,
) ([]provider.FieldChange, error) {
	fieldChanges := make([]provider.FieldChange, len(pbFieldChanges))

	for i, pbFieldChange := range pbFieldChanges {
		prevValue, err := serialisation.FromMappingNodePB(pbFieldChange.PrevValue, true /* optional */)
		if err != nil {
			return nil, err
		}

		newValue, err := serialisation.FromMappingNodePB(pbFieldChange.NewValue, true /* optional */)
		if err != nil {
			return nil, err
		}

		fieldChanges[i] = provider.FieldChange{
			FieldPath:    pbFieldChange.FieldPath,
			PrevValue:    prevValue,
			NewValue:     newValue,
			MustRecreate: pbFieldChange.MustRecreate,
			Sensitive:    pbFieldChange.Sensitive,
		}
	}

	return fieldChanges, nil
}

func ptrsFromPBFieldChanges(
	pbFieldChanges []*sharedtypesv1.FieldChange,
) ([]*provider.FieldChange, error) {
	fieldChanges, err := fromPBFieldChanges(pbFieldChanges)
	if err != nil {
		return nil, err
	}

	fieldChangePtrs := make([]*provider.FieldChange, len(fieldChanges))
	for i, fieldChange := range fieldChanges {
		fieldChangePtrs[i] = &fieldChange
	}

	return fieldChangePtrs, nil
}

func fromPBLinkChangesMap(
	pbLinkChangesMap map[string]*sharedtypesv1.LinkChanges,
) (map[string]provider.LinkChanges, error) {
	if pbLinkChangesMap == nil {
		return nil, nil
	}

	linkChangesMap := make(map[string]provider.LinkChanges, len(pbLinkChangesMap))

	for key, pbLinkChanges := range pbLinkChangesMap {
		linkChanges, err := FromPBLinkChanges(pbLinkChanges)
		if err != nil {
			return nil, err
		}

		linkChangesMap[key] = linkChanges
	}

	return linkChangesMap, nil
}

// FromPBLinkChanges converts a LinkChanges from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBLinkChanges(
	pbLinkChanges *sharedtypesv1.LinkChanges,
) (provider.LinkChanges, error) {
	if pbLinkChanges == nil {
		return provider.LinkChanges{}, nil
	}

	modifiedFields, err := ptrsFromPBFieldChanges(pbLinkChanges.ModifiedFields)
	if err != nil {
		return provider.LinkChanges{}, err
	}

	newFields, err := ptrsFromPBFieldChanges(pbLinkChanges.NewFields)
	if err != nil {
		return provider.LinkChanges{}, err
	}

	return provider.LinkChanges{
		ModifiedFields:            modifiedFields,
		NewFields:                 newFields,
		RemovedFields:             pbLinkChanges.RemovedFields,
		UnchangedFields:           pbLinkChanges.UnchangedFields,
		FieldChangesKnownOnDeploy: pbLinkChanges.FieldChangesKnownOnDeploy,
	}, nil
}

// FromPBResourceHasStabilisedRequest converts a ResourceHasStabilisedRequest
// from a protobuf message to a core type compatible with the blueprint framework.
func FromPBResourceHasStabilisedRequest(
	req *sharedtypesv1.ResourceHasStabilisedRequest,
) (*provider.ResourceHasStabilisedInput, error) {
	if req == nil {
		return nil, errors.New("resource has stabilised request cannot be nil")
	}

	providerCtx, err := FromPBProviderContext(req.Context)
	if err != nil {
		return nil, err
	}

	resourceSpec, err := serialisation.FromMappingNodePB(req.ResourceSpec, false /* optional */)
	if err != nil {
		return nil, err
	}

	resourceMetadataState, err := FromPBResourceMetadataState(req.ResourceMetadata)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceHasStabilisedInput{
		InstanceID:       req.InstanceId,
		InstanceName:     req.InstanceName,
		ResourceID:       req.ResourceId,
		ResourceSpec:     resourceSpec,
		ResourceMetadata: resourceMetadataState,
		ProviderContext:  providerCtx,
	}, nil
}

// FromPBProviderContext converts a ProviderContext from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBProviderContext(pbProviderCtx *sharedtypesv1.ProviderContext) (provider.Context, error) {
	if pbProviderCtx == nil {
		return nil, nil
	}

	providerConfigVars, err := FromPBScalarMap(pbProviderCtx.ProviderConfigVariables)
	if err != nil {
		return nil, err
	}

	contextVars, err := FromPBScalarMap(pbProviderCtx.ContextVariables)
	if err != nil {
		return nil, err
	}

	taggingConfig := FromPBTaggingConfig(pbProviderCtx.TaggingConfig)

	return utils.ProviderContextFromVarMapsWithTagging(providerConfigVars, contextVars, taggingConfig), nil
}

// FromPBTaggingConfig converts a TaggingConfig from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBTaggingConfig(pbTaggingConfig *sharedtypesv1.TaggingConfig) *provider.TaggingConfig {
	if pbTaggingConfig == nil {
		return nil
	}

	return &provider.TaggingConfig{
		Prefix:                pbTaggingConfig.Prefix,
		DeployEngineVersion:   pbTaggingConfig.DeployEngineVersion,
		ProviderPluginID:      pbTaggingConfig.ProviderPluginId,
		ProviderPluginVersion: pbTaggingConfig.ProviderPluginVersion,
		Enabled:               pbTaggingConfig.Enabled,
	}
}

// FromPBTaggingSupport converts a TaggingSupport from a protobuf enum to a core type
// compatible with the blueprint framework.
func FromPBTaggingSupport(pbTaggingSupport sharedtypesv1.TaggingSupport) provider.TaggingSupport {
	switch pbTaggingSupport {
	case sharedtypesv1.TaggingSupport_TAGGING_SUPPORT_FULL:
		return provider.TaggingSupportFull
	case sharedtypesv1.TaggingSupport_TAGGING_SUPPORT_LABELS:
		return provider.TaggingSupportLabels
	default:
		return provider.TaggingSupportNone
	}
}

// FromPBSystemMetadataState converts a SystemMetadataState from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBSystemMetadataState(
	pbSystemMetadata *sharedtypesv1.SystemMetadataState,
) *state.SystemMetadataState {
	if pbSystemMetadata == nil {
		return nil
	}

	return &state.SystemMetadataState{
		Provenance: FromPBProvenanceState(pbSystemMetadata.Provenance),
	}
}

// FromPBProvenanceState converts a ProvenanceState from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBProvenanceState(pbProvenance *sharedtypesv1.ProvenanceState) *state.ProvenanceState {
	if pbProvenance == nil {
		return nil
	}

	return &state.ProvenanceState{
		ProvisionedBy:         pbProvenance.ProvisionedBy,
		DeployEngineVersion:   pbProvenance.DeployEngineVersion,
		ProviderPluginID:      pbProvenance.ProviderPluginId,
		ProviderPluginVersion: pbProvenance.ProviderPluginVersion,
		ProvisionedAt:         pbProvenance.ProvisionedAt,
	}
}

// ResourceTypeToString converts a ResourceType to a string.
func ResourceTypeToString(resourceType *sharedtypesv1.ResourceType) string {
	if resourceType == nil {
		return ""
	}

	return resourceType.Type
}

// FromPBDestroyResourceRequest converts a DestroyResourceRequest from a protobuf message to a core type
// compatible with the blueprint framework.
func FromPBDestroyResourceRequest(
	req *sharedtypesv1.DestroyResourceRequest,
) (*provider.ResourceDestroyInput, error) {
	if req == nil {
		return nil, errors.New("destroy resource request cannot be nil")
	}

	providerCtx, err := FromPBProviderContext(req.Context)
	if err != nil {
		return nil, err
	}

	resourceState, err := FromPBResourceState(req.ResourceState)
	if err != nil {
		return nil, err
	}

	return &provider.ResourceDestroyInput{
		InstanceID:      req.InstanceId,
		InstanceName:    req.InstanceName,
		ResourceID:      req.ResourceId,
		ResourceState:   resourceState,
		ProviderContext: providerCtx,
	}, nil
}

type functionCallContextInfo struct {
	stack    function.Stack
	params   core.BlueprintParams
	location *source.Meta
}

func fromPBFunctionCallContext(
	reqContext *sharedtypesv1.FunctionCallContext,
) (*functionCallContextInfo, error) {
	if reqContext == nil {
		return nil, errors.New("expected function call context to be non-nil")
	}

	stack, err := fromPBFunctionCallStack(reqContext.CallStack)
	if err != nil {
		return nil, err
	}

	params, err := fromPBBlueprintParams(reqContext.Params)
	if err != nil {
		return nil, err
	}

	location, err := fromPBSourceMeta(reqContext.CurrentLocation)
	if err != nil {
		return nil, err
	}

	return &functionCallContextInfo{
		stack:    stack,
		params:   params,
		location: location,
	}, nil
}

func fromPBFunctionCallStack(
	reqStack []*sharedtypesv1.FunctionCall,
) (function.Stack, error) {
	if reqStack == nil {
		return function.NewStack(), nil
	}

	stack := function.NewStack()
	for _, call := range reqStack {
		if call == nil {
			continue
		}
		location, err := fromPBSourceMeta(call.Location)
		if err != nil {
			return nil, err
		}
		stack.Push(&function.Call{
			FilePath:     call.FilePath,
			FunctionName: call.FunctionName,
			Location:     location,
		})
	}

	return stack, nil
}

func fromPBSourceMeta(
	pbLocation *sharedtypesv1.SourceMeta,
) (*source.Meta, error) {
	if pbLocation == nil {
		return nil, nil
	}

	startPosition, err := fromPBSourcePosition(pbLocation.StartPosition)
	if err != nil {
		return nil, err
	}

	endPosition, err := ptrFromPBSourcePosition(pbLocation.EndPosition)
	if err != nil {
		return nil, err
	}

	return &source.Meta{
		Position:    startPosition,
		EndPosition: endPosition,
	}, nil
}

func fromPBSourcePosition(
	pbPosition *sharedtypesv1.SourcePosition,
) (source.Position, error) {
	if pbPosition == nil {
		return source.Position{
			Line: -1,
		}, nil
	}

	return source.Position{
		Line:   int(pbPosition.Line),
		Column: int(pbPosition.Column),
	}, nil
}

func ptrFromPBSourcePosition(
	pbPosition *sharedtypesv1.SourcePosition,
) (*source.Position, error) {
	if pbPosition == nil {
		return nil, nil
	}

	position, err := fromPBSourcePosition(pbPosition)
	if err != nil {
		return nil, err
	}

	if position.Line == -1 {
		return nil, nil
	}

	return &position, nil
}

func fromPBBlueprintParams(
	reqParams *sharedtypesv1.BlueprintParams,
) (core.BlueprintParams, error) {
	if reqParams == nil {
		return nil, errors.New("expected blueprint params to be non-nil")
	}

	providerConfig, err := FromPBScalarMap(reqParams.ProviderConfigVariables)
	if err != nil {
		return nil, err
	}
	providerConfigExpanded := expandNamespacedConfig(providerConfig)

	transformerConfig, err := FromPBScalarMap(reqParams.TransformerConfigVariables)
	if err != nil {
		return nil, err
	}
	transformerConfigExpanded := expandNamespacedConfig(transformerConfig)

	contextVariables, err := FromPBScalarMap(reqParams.ContextVariables)
	if err != nil {
		return nil, err
	}

	blueprintVariables, err := FromPBScalarMap(reqParams.BlueprintVariables)
	if err != nil {
		return nil, err
	}

	return core.NewDefaultParams(
		providerConfigExpanded,
		transformerConfigExpanded,
		contextVariables,
		blueprintVariables,
	), nil
}

func fromPBFunctionCallArgs(
	reqArgs *sharedtypesv1.FunctionCallArgs,
) ([]any, error) {
	if reqArgs == nil {
		return []any{}, nil
	}

	deserialised, err := pbutils.ConvertPBAnyToInterface(reqArgs.Args)
	if err != nil {
		return nil, err
	}

	// Treat nil as an empty arguments slice.
	if utils.IsAnyNil(deserialised) {
		return []any{}, nil
	}

	args, ok := deserialised.([]any)
	if !ok {
		return nil, fmt.Errorf(
			"expected arguments to be a []any, got %T",
			deserialised,
		)
	}

	return args, nil
}

// FromPBMappingNodeMap converts a map of protobuf MappingNodes to a map of core MappingNodes
// compatible with the blueprint framework.
func FromPBMappingNodeMap(
	pbMap map[string]*schemapb.MappingNode,
) (map[string]*core.MappingNode, error) {
	if pbMap == nil {
		return nil, nil
	}

	coreMap := make(map[string]*core.MappingNode, len(pbMap))
	for key, pbNode := range pbMap {
		coreNode, err := serialisation.FromMappingNodePB(pbNode, true /* optional */)
		if err != nil {
			return nil, err
		}

		coreMap[key] = coreNode
	}

	return coreMap, nil
}

// FromPBMappingNodeSlice converts a slice of protobuf MappingNodes to a slice of core MappingNodes
// compatible with the blueprint framework.
func FromPBMappingNodeSlice(
	pbSlice []*schemapb.MappingNode,
) ([]*core.MappingNode, error) {
	if pbSlice == nil {
		return nil, nil
	}

	coreSlice := make([]*core.MappingNode, len(pbSlice))
	for index, pbNode := range pbSlice {
		coreNode, err := serialisation.FromMappingNodePB(pbNode, true /* optional */)
		if err != nil {
			return nil, err
		}

		coreSlice[index] = coreNode
	}

	return coreSlice, nil
}

// FromPBFunctionDefinitionResponse converts a FunctionDefinitionResponse from a protobuf message
// to a core type compatible with the blueprint framework.
func FromPBFunctionDefinitionResponse(
	response *sharedtypesv1.FunctionDefinition,
	action errorsv1.PluginAction,
) (*provider.FunctionGetDefinitionOutput, error) {
	if response == nil {
		return nil, nil
	}

	definition, err := FromPBFunctionDefinition(response)
	if err != nil {
		return nil, errorsv1.CreateGeneralError(
			err,
			action,
		)
	}

	return &provider.FunctionGetDefinitionOutput{
		Definition: definition,
	}, nil
}
