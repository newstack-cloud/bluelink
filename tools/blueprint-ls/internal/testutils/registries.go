// This file contains test helpers for language services
// including test registry implementations for resources,
// data sources, functions, and custom variable types.

package testutils

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type FunctionRegistryMock struct {
	Functions map[string]provider.Function
	CallStack function.Stack
}

func (f *FunctionRegistryMock) ForCallContext(stack function.Stack) provider.FunctionRegistry {
	return &FunctionRegistryMock{
		Functions: f.Functions,
		CallStack: stack,
	}
}

func (f *FunctionRegistryMock) Call(
	ctx context.Context,
	functionName string,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	fnc, ok := f.Functions[functionName]
	if !ok {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("function %s not found", functionName),
			function.FuncCallErrorCodeFunctionNotFound,
			input.CallContext.CallStackSnapshot(),
		)
	}
	f.CallStack.Push(&function.Call{
		FunctionName: functionName,
		Location:     nil,
	})
	output, err := fnc.Call(ctx, input)
	f.CallStack.Pop()
	return output, err
}

func (f *FunctionRegistryMock) GetDefinition(
	ctx context.Context,
	functionName string,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	fnc, ok := f.Functions[functionName]
	if !ok {
		return nil, fmt.Errorf("function %s not found", functionName)
	}
	defOutput, err := fnc.GetDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *FunctionRegistryMock) ListFunctions(
	ctx context.Context,
) ([]string, error) {
	functions := make([]string, 0, len(r.Functions))
	for function := range r.Functions {
		functions = append(functions, function)
	}
	return functions, nil
}

func (f *FunctionRegistryMock) HasFunction(ctx context.Context, functionName string) (bool, error) {
	_, ok := f.Functions[functionName]
	return ok, nil
}

type ResourceRegistryMock struct {
	Resources map[string]provider.Resource
}

func (r *ResourceRegistryMock) HasResourceType(ctx context.Context, resourceType string) (bool, error) {
	_, ok := r.Resources[resourceType]
	return ok, nil
}

func (r *ResourceRegistryMock) GetSpecDefinition(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.GetSpecDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) GetTypeDescription(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.GetTypeDescription(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) ListResourceTypes(
	ctx context.Context,
) ([]string, error) {
	resourceTypes := make([]string, 0, len(r.Resources))
	for resourceType := range r.Resources {
		resourceTypes = append(resourceTypes, resourceType)
	}
	return resourceTypes, nil
}

func (r *ResourceRegistryMock) CustomValidate(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.CustomValidate(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) Deploy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDeployServiceInput,
) (*provider.ResourceDeployOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.Deploy(ctx, input.DeployInput)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) Destroy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDestroyInput,
) error {
	res, ok := r.Resources[resourceType]
	if !ok {
		return fmt.Errorf("resource %s not found", resourceType)
	}
	return res.Destroy(ctx, input)
}

func (r *ResourceRegistryMock) GetStabilisedDependencies(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	return res.GetStabilisedDependencies(ctx, input)
}

func (r *ResourceRegistryMock) LookupResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (*state.ResourceState, error) {
	return nil, nil
}

func (r *ResourceRegistryMock) HasResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (bool, error) {
	return false, nil
}

func (r *ResourceRegistryMock) WithParams(
	params core.BlueprintParams,
) resourcehelpers.Registry {
	return r
}

func (r *ResourceRegistryMock) AcquireResourceLock(
	ctx context.Context,
	input *provider.AcquireResourceLockInput,
) error {
	return nil
}

func (r *ResourceRegistryMock) ReleaseResourceLock(
	ctx context.Context,
	instanceID string,
	resourceName string,
) {
}

func (r *ResourceRegistryMock) ReleaseResourceLocks(
	ctx context.Context,
	instanceID string,
) {
}

func (r *ResourceRegistryMock) ReleaseResourceLocksAcquiredBy(
	ctx context.Context,
	instanceID string,
	acquiredBy string,
) {
}

func (r *ResourceRegistryMock) ListTransformers(
	ctx context.Context,
) ([]string, error) {
	// Return empty list for tests - no transformers in mock
	return []string{}, nil
}

type DataSourceRegistryMock struct {
	DataSources map[string]provider.DataSource
}

func (r *DataSourceRegistryMock) HasDataSourceType(ctx context.Context, dataSourceType string) (bool, error) {
	_, ok := r.DataSources[dataSourceType]
	return ok, nil
}

func (r *DataSourceRegistryMock) Fetch(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.Fetch(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) GetSpecDefinition(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.GetSpecDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) GetTypeDescription(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetTypeDescriptionInput,
) (*provider.DataSourceGetTypeDescriptionOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.GetTypeDescription(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) GetFilterFields(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetFilterFieldsInput,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.GetFilterFields(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) ListDataSourceTypes(
	ctx context.Context,
) ([]string, error) {
	dataSourceTypes := make([]string, 0, len(r.DataSources))
	for dataSourceType := range r.DataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}
	return dataSourceTypes, nil
}

func (r *DataSourceRegistryMock) CustomValidate(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceValidateInput,
) (*provider.DataSourceValidateOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.CustomValidate(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

type CustomVarTypeRegistryMock struct {
	CustomVarTypes map[string]provider.CustomVariableType
}

func (r *CustomVarTypeRegistryMock) GetDescription(
	ctx context.Context,
	customVariableType string,
	input *provider.CustomVariableTypeGetDescriptionInput,
) (*provider.CustomVariableTypeGetDescriptionOutput, error) {
	res, ok := r.CustomVarTypes[customVariableType]
	if !ok {
		return nil, fmt.Errorf("custom variable type %s not found", customVariableType)
	}
	descOutput, err := res.GetDescription(ctx, input)
	if err != nil {
		return nil, err
	}
	return descOutput, nil
}

func (r *CustomVarTypeRegistryMock) HasCustomVariableType(ctx context.Context, customVariableType string) (bool, error) {
	_, ok := r.CustomVarTypes[customVariableType]
	return ok, nil
}

func (r *CustomVarTypeRegistryMock) GetOptions(
	ctx context.Context,
	customVariableType string,
	input *provider.CustomVariableTypeOptionsInput,
) (*provider.CustomVariableTypeOptionsOutput, error) {
	res, ok := r.CustomVarTypes[customVariableType]
	if !ok {
		return nil, fmt.Errorf("custom variable type %s not found", customVariableType)
	}
	optionsOutput, err := res.Options(ctx, input)
	if err != nil {
		return nil, err
	}
	return optionsOutput, nil
}

func (r *CustomVarTypeRegistryMock) ListCustomVariableTypes(
	ctx context.Context,
) ([]string, error) {
	customVarTypes := make([]string, 0, len(r.CustomVarTypes))
	for customVarType := range r.CustomVarTypes {
		customVarTypes = append(customVarTypes, customVarType)
	}
	return customVarTypes, nil
}
