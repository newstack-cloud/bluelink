package docgen

import (
	"context"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

func getProviderFunctionDocs(
	ctx context.Context,
	providerPlugin provider.Provider,
	params core.BlueprintParams,
	functionName string,
) (*PluginDocsFunction, error) {
	function, err := providerPlugin.Function(ctx, functionName)
	if err != nil {
		return nil, err
	}

	definitionOutput, err := function.GetDefinition(
		ctx,
		&provider.FunctionGetDefinitionInput{
			Params: params,
		},
	)
	if err != nil {
		return nil, err
	}

	parameterDocs := getFunctionParametersDocs(
		definitionOutput.Definition.Parameters,
	)

	returnDocs := getFunctionReturnDocs(
		definitionOutput.Definition.Return,
	)

	return &PluginDocsFunction{
		FunctionDefinition: FunctionDefinition{
			Parameters: parameterDocs,
			Return:     returnDocs,
		},
		Name:        functionName,
		Summary:     getProviderFunctionSummary(definitionOutput),
		Description: getProviderFunctionDescription(definitionOutput),
		Internal:    definitionOutput.Definition.Internal,
	}, nil
}

func getFunctionParametersDocs(
	parameters []function.Parameter,
) []*FunctionParameter {
	paramDocs := make([]*FunctionParameter, len(parameters))
	for i, param := range parameters {
		paramDocs[i] = toFunctionParameterDocs(param)
	}

	return paramDocs
}

func toFunctionParameterDocs(
	parameter function.Parameter,
) *FunctionParameter {
	switch concreteParam := parameter.(type) {
	case *function.ScalarParameter:
		return toScalarFunctionParameterDocs(concreteParam)
	case *function.ListParameter:
		return toListFunctionParameterDocs(concreteParam)
	case *function.MapParameter:
		return toMapFunctionParameterDocs(concreteParam)
	case *function.ObjectParameter:
		return toObjectFunctionParameterDocs(concreteParam)
	case *function.FunctionParameter:
		return toFunctionTypeFunctionParameterDocs(concreteParam)
	case *function.AnyParameter:
		return toAnyFunctionParameterDocs(concreteParam)
	case *function.VariadicParameter:
		return toVariadicFunctionParameterDocs(concreteParam)
	}

	return nil
}

func toScalarFunctionParameterDocs(
	concreteParam *function.ScalarParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)
	baseParam.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteParam.Type,
	)
	return baseParam
}

func toListFunctionParameterDocs(
	concreteParam *function.ListParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)
	baseParam.ElementValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteParam.ElementType,
	)

	return baseParam
}

func toMapFunctionParameterDocs(
	concreteParam *function.MapParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)
	baseParam.MapValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteParam.ElementType,
	)

	return baseParam
}

func toObjectFunctionParameterDocs(
	concreteParam *function.ObjectParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)
	baseParam.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteParam.ObjectValueType,
	)

	return baseParam
}

func toFunctionTypeFunctionParameterDocs(
	concreteParam *function.FunctionParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)
	baseParam.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteParam.FunctionType,
	)

	return baseParam
}

func toAnyFunctionParameterDocs(
	concreteParam *function.AnyParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)

	unionValueTypeDefDocs := make(
		[]*ValueTypeDefinition,
		len(concreteParam.UnionTypes),
	)
	for i, unionValueTypeDef := range concreteParam.UnionTypes {
		unionValueTypeDefDocs[i] = toValueTypeDefinitionDocs(
			unionValueTypeDef,
		)
	}

	baseParam.UnionValueTypeDefinitions = unionValueTypeDefDocs

	return baseParam
}

func toVariadicFunctionParameterDocs(
	concreteParam *function.VariadicParameter,
) *FunctionParameter {
	baseParam := createBaseParameter(concreteParam)
	baseParam.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteParam.Type,
	)
	baseParam.VariadicSingleType = concreteParam.SingleType
	baseParam.VariadicNamed = concreteParam.Named

	return baseParam
}

func createBaseParameter(
	param function.Parameter,
) *FunctionParameter {
	return &FunctionParameter{
		ParamType:      string(param.GetType()),
		Name:           param.GetName(),
		Label:          param.GetLabel(),
		Description:    getParameterTypeDescription(param),
		AllowNullValue: param.GetAllowNullValue(),
		Optional:       param.GetOptional(),
	}
}

func getParameterTypeDescription(
	param function.Parameter,
) string {
	if param == nil {
		return ""
	}

	if strings.TrimSpace(param.GetFormattedDescription()) != "" {
		return param.GetFormattedDescription()
	}

	return param.GetDescription()
}

func getFunctionReturnDocs(
	returnType function.Return,
) *FunctionReturn {
	switch concreteReturnType := returnType.(type) {
	case *function.ScalarReturn:
		return toScalarFunctionReturnDocs(concreteReturnType)
	case *function.ListReturn:
		return toListFunctionReturnDocs(concreteReturnType)
	case *function.MapReturn:
		return toMapFunctionReturnDocs(concreteReturnType)
	case *function.ObjectReturn:
		return toObjectFunctionReturnDocs(concreteReturnType)
	case *function.FunctionReturn:
		return toFunctionTypeFunctionReturnDocs(concreteReturnType)
	case *function.AnyReturn:
		return toAnyFunctionReturnDocs(concreteReturnType)
	}

	return nil
}

func toScalarFunctionReturnDocs(
	concreteReturnType *function.ScalarReturn,
) *FunctionReturn {
	returnType := createBaseReturn(concreteReturnType)
	returnType.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteReturnType.Type,
	)
	return returnType
}

func toListFunctionReturnDocs(
	concreteReturnType *function.ListReturn,
) *FunctionReturn {
	returnType := createBaseReturn(concreteReturnType)
	returnType.ElementValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteReturnType.ElementType,
	)
	return returnType
}

func toMapFunctionReturnDocs(
	concreteReturnType *function.MapReturn,
) *FunctionReturn {
	returnType := createBaseReturn(concreteReturnType)
	returnType.MapValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteReturnType.ElementType,
	)
	return returnType
}

func toObjectFunctionReturnDocs(
	concreteReturnType *function.ObjectReturn,
) *FunctionReturn {
	returnType := createBaseReturn(concreteReturnType)
	returnType.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteReturnType.ObjectValueType,
	)
	return returnType
}

func toFunctionTypeFunctionReturnDocs(
	concreteReturnType *function.FunctionReturn,
) *FunctionReturn {
	returnType := createBaseReturn(concreteReturnType)
	returnType.ValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteReturnType.FunctionType,
	)
	return returnType
}

func toAnyFunctionReturnDocs(
	concreteReturnType *function.AnyReturn,
) *FunctionReturn {
	returnType := createBaseReturn(concreteReturnType)
	unionValueTypeDefDocs := make(
		[]*ValueTypeDefinition,
		len(concreteReturnType.UnionTypes),
	)
	for i, unionValueTypeDef := range concreteReturnType.UnionTypes {
		unionValueTypeDefDocs[i] = toValueTypeDefinitionDocs(
			unionValueTypeDef,
		)
	}
	returnType.UnionValueTypeDefinitions = unionValueTypeDefDocs

	return returnType
}

func createBaseReturn(
	returnType function.Return,
) *FunctionReturn {
	return &FunctionReturn{
		ReturnType:  string(returnType.GetType()),
		Description: getReturnTypeDescription(returnType),
	}
}

func getReturnTypeDescription(
	returnType function.Return,
) string {
	if returnType == nil {
		return ""
	}

	if strings.TrimSpace(returnType.GetFormattedDescription()) != "" {
		return returnType.GetFormattedDescription()
	}

	return returnType.GetDescription()
}

func toValueTypeDefinitionDocs(
	valueTypeDefinition function.ValueTypeDefinition,
) *ValueTypeDefinition {
	switch concreteValueType := valueTypeDefinition.(type) {
	case *function.ValueTypeDefinitionScalar:
		return toScalarValueTypeDefinitionDocs(
			concreteValueType,
		)
	case *function.ValueTypeDefinitionList:
		return toListValueTypeDefinitionDocs(
			concreteValueType,
		)
	case *function.ValueTypeDefinitionMap:
		return toMapValueTypeDefinitionDocs(
			concreteValueType,
		)
	case *function.ValueTypeDefinitionObject:
		return toObjectValueTypeDefinitionDocs(
			concreteValueType,
		)
	case *function.ValueTypeDefinitionFunction:
		return toFunctionTypeValueTypeDefinitionDocs(
			concreteValueType,
		)
	case *function.ValueTypeDefinitionAny:
		return toAnyValueTypeDefinitionDocs(
			concreteValueType,
		)
	}

	return nil
}

func toScalarValueTypeDefinitionDocs(
	concreteValueTypeDef *function.ValueTypeDefinitionScalar,
) *ValueTypeDefinition {
	valueTypeDef := createBaseValueTypeDefinition(
		concreteValueTypeDef,
	)
	valueTypeDef.StringChoices = concreteValueTypeDef.StringChoices
	return valueTypeDef
}

func toListValueTypeDefinitionDocs(
	concreteValueTypeDef *function.ValueTypeDefinitionList,
) *ValueTypeDefinition {
	valueTypeDef := createBaseValueTypeDefinition(
		concreteValueTypeDef,
	)
	valueTypeDef.ElementValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteValueTypeDef.ElementType,
	)
	return valueTypeDef
}

func toMapValueTypeDefinitionDocs(
	concreteValueTypeDef *function.ValueTypeDefinitionMap,
) *ValueTypeDefinition {
	valueTypeDef := createBaseValueTypeDefinition(
		concreteValueTypeDef,
	)
	valueTypeDef.MapValueTypeDefinition = toValueTypeDefinitionDocs(
		concreteValueTypeDef.ElementType,
	)
	return valueTypeDef
}

func toObjectValueTypeDefinitionDocs(
	concreteValueTypeDef *function.ValueTypeDefinitionObject,
) *ValueTypeDefinition {
	valueTypeDef := createBaseValueTypeDefinition(
		concreteValueTypeDef,
	)
	valueTypeDef.AttributeValueTypeDefinitions = map[string]*AttributeType{}
	for key, attrValueTypeDef := range concreteValueTypeDef.AttributeTypes {
		valueTypeDef.AttributeValueTypeDefinitions[key] = toObjectAttributeTypeDocs(
			attrValueTypeDef,
		)
	}
	return valueTypeDef
}

func toObjectAttributeTypeDocs(
	attributeTypeDef function.AttributeType,
) *AttributeType {
	return &AttributeType{
		ValueTypeDefinition: *toValueTypeDefinitionDocs(
			attributeTypeDef.Type,
		),
		Nullable: attributeTypeDef.AllowNullValue,
	}
}

func toFunctionTypeValueTypeDefinitionDocs(
	concreteValueTypeDef *function.ValueTypeDefinitionFunction,
) *ValueTypeDefinition {
	valueTypeDef := createBaseValueTypeDefinition(
		concreteValueTypeDef,
	)
	valueTypeDef.FunctionDefinition = &FunctionDefinition{
		Parameters: getFunctionParametersDocs(
			concreteValueTypeDef.Definition.Parameters,
		),
		Return: getFunctionReturnDocs(
			concreteValueTypeDef.Definition.Return,
		),
	}

	return valueTypeDef
}

func toAnyValueTypeDefinitionDocs(
	concreteValueTypeDef *function.ValueTypeDefinitionAny,
) *ValueTypeDefinition {
	valueTypeDef := createBaseValueTypeDefinition(
		concreteValueTypeDef,
	)
	unionValueTypeDefDocs := make(
		[]*ValueTypeDefinition,
		len(concreteValueTypeDef.UnionTypes),
	)
	for i, unionValueTypeDef := range concreteValueTypeDef.UnionTypes {
		unionValueTypeDefDocs[i] = toValueTypeDefinitionDocs(
			unionValueTypeDef,
		)
	}
	valueTypeDef.UnionValueTypeDefinitions = unionValueTypeDefDocs

	return valueTypeDef
}

func createBaseValueTypeDefinition(
	valueTypeDef function.ValueTypeDefinition,
) *ValueTypeDefinition {
	return &ValueTypeDefinition{
		Type:        string(valueTypeDef.GetType()),
		Label:       valueTypeDef.GetLabel(),
		Description: getValueTypeDefinitionDescription(valueTypeDef),
	}
}

func getValueTypeDefinitionDescription(
	valueTypeDef function.ValueTypeDefinition,
) string {
	if valueTypeDef == nil {
		return ""
	}

	if strings.TrimSpace(valueTypeDef.GetFormattedDescription()) != "" {
		return valueTypeDef.GetFormattedDescription()
	}

	return valueTypeDef.GetDescription()
}

func getProviderFunctionSummary(
	output *provider.FunctionGetDefinitionOutput,
) string {
	if output.Definition == nil {
		return ""
	}

	if strings.TrimSpace(output.Definition.FormattedSummary) != "" {
		return output.Definition.FormattedSummary
	}

	if strings.TrimSpace(output.Definition.Summary) != "" {
		return output.Definition.Summary
	}

	return truncateDescription(getProviderFunctionDescription(output), 120)
}

func getProviderFunctionDescription(
	output *provider.FunctionGetDefinitionOutput,
) string {
	if output.Definition == nil {
		return ""
	}

	if strings.TrimSpace(output.Definition.FormattedDescription) != "" {
		return output.Definition.FormattedDescription
	}

	return output.Definition.Description
}
