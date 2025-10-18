package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type testProvider struct {
	functions      map[string]Function
	resources      map[string]Resource
	links          map[string]Link
	dataSources    map[string]DataSource
	customVarTypes map[string]CustomVariableType
	namespace      string
}

func (p *testProvider) Namespace(ctx context.Context) (string, error) {
	return p.namespace, nil
}

func (p *testProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return nil, nil
}

func (p *testProvider) Resource(ctx context.Context, resourceType string) (Resource, error) {
	resource, ok := p.resources[resourceType]
	if !ok {
		return nil, errors.New("resource not found")
	}
	return resource, nil
}

func (p *testProvider) DataSource(ctx context.Context, dataSourceType string) (DataSource, error) {
	dataSource, ok := p.dataSources[dataSourceType]
	if !ok {
		return nil, errors.New("data source not found")
	}
	return dataSource, nil
}

func (p *testProvider) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (Link, error) {
	link, ok := p.links[core.LinkType(resourceTypeA, resourceTypeB)]
	if !ok {
		return nil, errors.New("link not found")
	}
	return link, nil
}

func (p *testProvider) CustomVariableType(ctx context.Context, customVariableType string) (CustomVariableType, error) {
	customVarType, ok := p.customVarTypes[customVariableType]
	if !ok {
		return nil, errors.New("custom variable type not found")
	}
	return customVarType, nil
}

func (p *testProvider) ListFunctions(ctx context.Context) ([]string, error) {
	functionNames := []string{}
	for name := range p.functions {
		functionNames = append(functionNames, name)
	}
	return functionNames, nil
}

func (p *testProvider) ListResourceTypes(ctx context.Context) ([]string, error) {
	resourceTypes := []string{}
	for resourceType := range p.resources {
		resourceTypes = append(resourceTypes, resourceType)
	}
	return resourceTypes, nil
}

func (p *testProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	linkTypes := []string{}
	for linkType := range p.links {
		linkTypes = append(linkTypes, linkType)
	}
	return linkTypes, nil
}

func (p *testProvider) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	dataSourceTypes := []string{}
	for dataSourceType := range p.dataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}
	return dataSourceTypes, nil
}

func (p *testProvider) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	customVarTypes := []string{}
	for customVarType := range p.customVarTypes {
		customVarTypes = append(customVarTypes, customVarType)
	}
	return customVarTypes, nil
}

func (p *testProvider) Function(ctx context.Context, functionName string) (Function, error) {
	function, ok := p.functions[functionName]
	if !ok {
		return nil, errors.New("function not found")
	}
	return function, nil
}

func (p *testProvider) RetryPolicy(ctx context.Context) (*RetryPolicy, error) {
	return &RetryPolicy{
		MaxRetries: 3,
		// The first retry delay is 1 millisecond
		FirstRetryDelay: 0.001,
		// The maximum delay between retries is 10 milliseconds.
		MaxDelay:      0.01,
		BackoffFactor: 0.5,
		// Make the retry behaviour more deterministic for tests by disabling jitter.
		Jitter: false,
	}, nil
}

type testSubstrFunction struct {
	definition *function.Definition
}

func newTestSubstrFunction() Function {
	return &testSubstrFunction{
		definition: &function.Definition{
			Description: "Extracts a substring from the given string.",
			FormattedDescription: "Extracts a substring from the given string.\n\n" +
				"**Examples:**\n\n" +
				"```\n${substr(values.cacheClusterConfig.host, 0, 3)}\n```",
			Parameters: []function.Parameter{
				&function.ScalarParameter{
					Type: &function.ValueTypeDefinitionScalar{
						Label: "string",
						Type:  function.ValueTypeString,
					},
					Description: "A valid string Scalar, reference or function call yielding a return value " +
						"representing the string to extract the substring from.",
				},
				&function.ScalarParameter{
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Description: "The index of the first character to include in the substring.",
				},
				&function.ScalarParameter{
					Type: &function.ValueTypeDefinitionScalar{
						Label: "integer",
						Type:  function.ValueTypeInt64,
					},
					Optional: true,
					Description: "The index of the last character to include in the substring. " +
						"If not provided, the substring will include all characters from the start index to the end of the string.",
				},
			},
			Return: &function.ScalarReturn{
				Type: &function.ValueTypeDefinitionScalar{
					Label: "string",
					Type:  function.ValueTypeString,
				},
				Description: "The substring extracted from the provided string.",
			},
		},
	}
}

func (f *testSubstrFunction) GetDefinition(
	ctx context.Context,
	input *FunctionGetDefinitionInput,
) (*FunctionGetDefinitionOutput, error) {
	return &FunctionGetDefinitionOutput{
		Definition: f.definition,
	}, nil
}

func (f *testSubstrFunction) Call(
	ctx context.Context,
	input *FunctionCallInput,
) (*FunctionCallOutput, error) {
	var inputStr string
	var start int64
	if err := input.Arguments.GetMultipleVars(ctx, &inputStr, &start); err != nil {
		return nil, err
	}

	var end int64
	if err := input.Arguments.GetVar(ctx, 2, &end); err != nil {
		end = int64(len(inputStr))
	}

	if start > end {
		return nil, function.NewFuncCallError(
			"start index cannot be greater than end index",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if start < 0 || end < 0 {
		return nil, function.NewFuncCallError(
			"start and end indices cannot be negative",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if start > int64(len(inputStr)-1) {
		return nil, function.NewFuncCallError(
			"start index cannot be greater than the last element index in the string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	if end > int64(len(inputStr)) {
		return nil, function.NewFuncCallError(
			"end index cannot be greater than the length of the string",
			function.FuncCallErrorCodeInvalidInput,
			input.CallContext.CallStackSnapshot(),
		)
	}

	return &FunctionCallOutput{
		ResponseData: inputStr[start:end],
	}, nil
}

type functionCallArgsMock struct {
	args    []any
	callCtx FunctionCallContext
}

func (f *functionCallArgsMock) Get(ctx context.Context, position int) (any, error) {
	return f.args[position], nil
}

func (f *functionCallArgsMock) GetVar(ctx context.Context, position int, target any) error {
	val := reflect.ValueOf(target)
	if position >= len(f.args) {
		return function.NewFuncCallError(
			fmt.Sprintf("argument at index %d not found", position),
			function.FuncCallErrorCodeFunctionCall,
			f.callCtx.CallStackSnapshot(),
		)
	}

	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return function.NewFuncCallError(
			"target to read argument into is not a pointer",
			function.FuncCallErrorCodeInvalidArgumentType,
			f.callCtx.CallStackSnapshot(),
		)
	}

	argVal := reflect.ValueOf(f.args[position])
	// Allow interface{} as a target type so that the caller can carry out type assertions
	// when an argument can be of multiple types.
	if targetVal.Elem().Kind() != reflect.Interface && targetVal.Elem().Kind() != argVal.Kind() {
		return function.NewFuncCallError(
			fmt.Sprintf(
				"argument at index %d is of type %s, but target is of type %s",
				position,
				argVal.Kind(),
				targetVal.Elem().Kind(),
			),
			function.FuncCallErrorCodeInvalidArgumentType,
			f.callCtx.CallStackSnapshot(),
		)
	}

	val.Elem().Set(reflect.ValueOf(f.args[position]))
	return nil
}

func (f *functionCallArgsMock) GetMultipleVars(ctx context.Context, targets ...any) error {
	for i := 0; i < len(f.args); i += 1 {
		if i < len(targets) {
			targetVal := reflect.ValueOf(targets[i])
			if targetVal.Kind() != reflect.Ptr {
				return function.NewFuncCallError(
					fmt.Sprintf("target at index %d to read argument into is not a pointer", i),
					function.FuncCallErrorCodeInvalidArgumentType,
					f.callCtx.CallStackSnapshot(),
				)
			}

			argVal := reflect.ValueOf(f.args[i])
			// Allow interface{} as a target type so that the caller can carry out type assertions
			// when an argument can be of multiple types.
			if targetVal.Elem().Kind() != reflect.Interface && targetVal.Elem().Kind() != argVal.Kind() {
				return function.NewFuncCallError(
					fmt.Sprintf(
						"argument at index %d is of type %s, but target is of type %s",
						i,
						argVal.Kind(),
						targetVal.Elem().Kind(),
					),
					function.FuncCallErrorCodeInvalidArgumentType,
					f.callCtx.CallStackSnapshot(),
				)
			}
			targetVal.Elem().Set(argVal)
		}
	}

	if len(targets) > len(f.args) {
		expectedText := fmt.Sprintf("%d arguments expected", len(targets))
		if len(targets) == 1 {
			expectedText = "1 argument expected"
		}
		argsText := fmt.Sprintf(", but %d arguments were passed into function", len(f.args))
		if len(f.args) == 1 {
			argsText = ", but 1 argument was passed into function"
		}

		return function.NewFuncCallError(
			fmt.Sprintf(
				"%s%s",
				expectedText,
				argsText,
			),
			function.FuncCallErrorCodeFunctionCall,
			f.callCtx.CallStackSnapshot(),
		)
	}
	return nil
}

func (f *functionCallArgsMock) Export(ctx context.Context) ([]any, error) {
	return f.args, nil
}

type functionCallContextMock struct {
	params             *core.ParamsImpl
	registry           FunctionRegistry
	callStack          function.Stack
	fileSourceRegistry FileSourceRegistry
	currentLocation    *source.Meta
}

func (f *functionCallContextMock) Registry() FunctionRegistry {
	return f.registry
}

func (f *functionCallContextMock) Params() core.BlueprintParams {
	return f.params
}

func (f *functionCallContextMock) FileSourceRegistry() FileSourceRegistry {
	if f.fileSourceRegistry == nil {
		return NewFileSourceRegistry()
	}
	return f.fileSourceRegistry
}

func (f *functionCallContextMock) NewCallArgs(args ...any) FunctionCallArguments {
	return &functionCallArgsMock{args: args, callCtx: f}
}

func (f *functionCallContextMock) CallStackSnapshot() []*function.Call {
	// Take a copy of the current call stack.
	return f.callStack.Snapshot()
}

func (f *functionCallContextMock) CurrentLocation() *source.Meta {
	return f.currentLocation
}

type testExampleDataSource struct {
	definition               *DataSourceSpecDefinition
	filterFields             map[string]*DataSourceFilterSchema
	markdownDescription      string
	plainTextDescription     string
	emulateTransientFailures bool
	// Tracks the number of fetch attempts for all calls for the data source type.
	// This is used to emulate transient failures when fetching data from data sources,
	// the data source registry will retry fetching data until the
	// fetch attempt count exceeds the max deploy attempts.
	currentFetchAttempts int
	mu                   sync.Mutex
}

func newTestExampleDataSource(emulateTransientFailures bool) DataSource {
	return &testExampleDataSource{
		definition: &DataSourceSpecDefinition{
			Fields: map[string]*DataSourceSpecSchema{
				"name": {
					Type: DataSourceSpecTypeString,
				},
			},
		},
		filterFields: map[string]*DataSourceFilterSchema{
			"metadata.id": {
				Type:                 DataSourceFilterSearchValueTypeString,
				Description:          "The unique identifier of the resource.",
				FormattedDescription: "The **unique** identifier of the resource.",
				SupportedOperators: []schema.DataSourceFilterOperator{
					schema.DataSourceFilterOperatorEquals,
				},
			},
		},
		markdownDescription:      "## test/exampleDataSource\n\nThis is a test data source.",
		plainTextDescription:     "test/exampleDataSource\n\nThis is a test data source.",
		emulateTransientFailures: emulateTransientFailures,
		currentFetchAttempts:     0,
	}
}

func (d *testExampleDataSource) GetSpecDefinition(
	ctx context.Context,
	input *DataSourceGetSpecDefinitionInput,
) (*DataSourceGetSpecDefinitionOutput, error) {
	return &DataSourceGetSpecDefinitionOutput{
		SpecDefinition: d.definition,
	}, nil
}

func (d *testExampleDataSource) Fetch(
	ctx context.Context,
	input *DataSourceFetchInput,
) (*DataSourceFetchOutput, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.emulateTransientFailures {
		d.currentFetchAttempts += 1

		// Provider retry policy allows for a maximum of 3 attempts before failing.
		if d.currentFetchAttempts < 3 {
			return nil, &RetryableError{
				ChildError: errors.New("fetch failed due to transient error"),
			}
		}
	}

	testName := "test"
	return &DataSourceFetchOutput{
		Data: map[string]*core.MappingNode{
			"name": {
				Scalar: &core.ScalarValue{
					StringValue: &testName,
				},
			},
		},
	}, nil
}

func (d *testExampleDataSource) GetType(
	ctx context.Context,
	input *DataSourceGetTypeInput,
) (*DataSourceGetTypeOutput, error) {
	return &DataSourceGetTypeOutput{
		Type: "test/exampleDataSource",
	}, nil
}

func (d *testExampleDataSource) GetTypeDescription(
	ctx context.Context,
	input *DataSourceGetTypeDescriptionInput,
) (*DataSourceGetTypeDescriptionOutput, error) {
	return &DataSourceGetTypeDescriptionOutput{
		MarkdownDescription:  d.markdownDescription,
		PlainTextDescription: d.plainTextDescription,
	}, nil
}

func (d *testExampleDataSource) GetExamples(
	ctx context.Context,
	input *DataSourceGetExamplesInput,
) (*DataSourceGetExamplesOutput, error) {
	return &DataSourceGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}

func (d *testExampleDataSource) GetFilterFields(
	ctx context.Context,
	input *DataSourceGetFilterFieldsInput,
) (*DataSourceGetFilterFieldsOutput, error) {
	return &DataSourceGetFilterFieldsOutput{
		FilterFields: d.filterFields,
	}, nil
}

func (d *testExampleDataSource) CustomValidate(
	ctx context.Context,
	input *DataSourceValidateInput,
) (*DataSourceValidateOutput, error) {
	return &DataSourceValidateOutput{
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelError,
				Message: "This is a test diagnostic.",
			},
		},
	}, nil
}

type testEC2InstanceTypeCustomVariableType struct {
	markdownDescription  string
	plainTextDescription string
}

func newTestEC2InstanceCustomVarType() CustomVariableType {
	return &testEC2InstanceTypeCustomVariableType{
		markdownDescription:  "## aws/ec2/instanceType\n\nThis is a test custom variable type.",
		plainTextDescription: "aws/ec2/instanceType\n\nThis is a test custom variable type.",
	}
}

func (t *testEC2InstanceTypeCustomVariableType) Options(
	ctx context.Context,
	input *CustomVariableTypeOptionsInput,
) (*CustomVariableTypeOptionsOutput, error) {
	t2nano := "t2.nano"
	t2micro := "t2.micro"
	t2small := "t2.small"
	t2medium := "t2.medium"
	t2large := "t2.large"
	t2xlarge := "t2.xlarge"
	t22xlarge := "t2.2xlarge"
	return &CustomVariableTypeOptionsOutput{
		Options: map[string]*CustomVariableTypeOption{
			t2nano: {
				Value: &core.ScalarValue{
					StringValue: &t2nano,
				},
			},
			t2micro: {
				Value: &core.ScalarValue{
					StringValue: &t2micro,
				},
			},
			t2small: {
				Value: &core.ScalarValue{
					StringValue: &t2small,
				},
			},
			t2medium: {
				Value: &core.ScalarValue{
					StringValue: &t2medium,
				},
			},
			t2large: {
				Value: &core.ScalarValue{
					StringValue: &t2large,
				},
			},
			t2xlarge: {
				Value: &core.ScalarValue{
					StringValue: &t2xlarge,
				},
			},
			t22xlarge: {
				Value: &core.ScalarValue{
					StringValue: &t22xlarge,
				},
			},
		},
	}, nil
}

func (t *testEC2InstanceTypeCustomVariableType) GetType(
	ctx context.Context,
	input *CustomVariableTypeGetTypeInput,
) (*CustomVariableTypeGetTypeOutput, error) {
	return &CustomVariableTypeGetTypeOutput{
		Type: "aws/ec2/instanceType",
	}, nil
}

func (t *testEC2InstanceTypeCustomVariableType) GetDescription(
	ctx context.Context,
	input *CustomVariableTypeGetDescriptionInput,
) (*CustomVariableTypeGetDescriptionOutput, error) {
	return &CustomVariableTypeGetDescriptionOutput{
		MarkdownDescription:  t.markdownDescription,
		PlainTextDescription: t.plainTextDescription,
	}, nil
}

func (t *testEC2InstanceTypeCustomVariableType) GetExamples(
	ctx context.Context,
	input *CustomVariableTypeGetExamplesInput,
) (*CustomVariableTypeGetExamplesOutput, error) {
	return &CustomVariableTypeGetExamplesOutput{
		PlainTextExamples: []string{},
		MarkdownExamples:  []string{},
	}, nil
}
