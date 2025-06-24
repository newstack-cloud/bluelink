package testsuites

import (
	"slices"

	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/stretchr/testify/suite"
)

func assertPluginDocFunctionsEqual(
	expected []*docgen.PluginDocsFunction,
	actual []*docgen.PluginDocsFunction,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedFunction := range expected {
		actualFunction := actual[i]
		testSuite.Equal(expectedFunction.Name, actualFunction.Name)
		testSuite.Equal(expectedFunction.Summary, actualFunction.Summary)
		testSuite.Equal(expectedFunction.Description, actualFunction.Description)
		assertFunctionDefinitionsEqual(
			&expectedFunction.FunctionDefinition,
			&actualFunction.FunctionDefinition,
			testSuite,
		)
	}
}

func assertFunctionDefinitionsEqual(
	expected *docgen.FunctionDefinition,
	actual *docgen.FunctionDefinition,
	testSuite *suite.Suite,
) {
	if expected == nil && actual == nil {
		return
	}

	testSuite.Equal(len(expected.Parameters), len(actual.Parameters))
	for i, expectedParameter := range expected.Parameters {
		actualParameter := actual.Parameters[i]
		assertFunctionParametersEqual(
			expectedParameter,
			actualParameter,
			testSuite,
		)
	}

	assertFunctionReturnTypesEqual(
		expected.Return,
		actual.Return,
		testSuite,
	)
}

func assertFunctionParametersEqual(
	expected *docgen.FunctionParameter,
	actual *docgen.FunctionParameter,
	testSuite *suite.Suite,
) {
	testSuite.Equal(expected.ParamType, actual.ParamType)
	testSuite.Equal(expected.Name, actual.Name)
	testSuite.Equal(expected.Label, actual.Label)
	testSuite.Equal(expected.Description, actual.Description)
	testSuite.Equal(expected.AllowNullValue, actual.AllowNullValue)
	testSuite.Equal(expected.Optional, actual.Optional)

	assertFunctionValueTypeDefinitionsEqual(
		expected.ValueTypeDefinition,
		actual.ValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionsEqual(
		expected.ElementValueTypeDefinition,
		actual.ElementValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionsEqual(
		expected.MapValueTypeDefinition,
		actual.MapValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionSlicesEqual(
		expected.UnionValueTypeDefinitions,
		actual.UnionValueTypeDefinitions,
		testSuite,
	)

	testSuite.Equal(expected.VariadicNamed, actual.VariadicNamed)
	testSuite.Equal(expected.VariadicSingleType, actual.VariadicSingleType)
}

func assertFunctionReturnTypesEqual(
	expected *docgen.FunctionReturn,
	actual *docgen.FunctionReturn,
	testSuite *suite.Suite,
) {
	testSuite.Equal(expected.ReturnType, actual.ReturnType)
	testSuite.Equal(expected.Description, actual.Description)

	assertFunctionValueTypeDefinitionsEqual(
		expected.ValueTypeDefinition,
		actual.ValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionsEqual(
		expected.ElementValueTypeDefinition,
		actual.ElementValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionsEqual(
		expected.MapValueTypeDefinition,
		actual.MapValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionSlicesEqual(
		expected.UnionValueTypeDefinitions,
		actual.UnionValueTypeDefinitions,
		testSuite,
	)
}

func assertFunctionValueTypeDefinitionSlicesEqual(
	expected []*docgen.ValueTypeDefinition,
	actual []*docgen.ValueTypeDefinition,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedValueTypeDefinition := range expected {
		actualValueTypeDefinition := actual[i]
		assertFunctionValueTypeDefinitionsEqual(
			expectedValueTypeDefinition,
			actualValueTypeDefinition,
			testSuite,
		)
	}
}

func assertFunctionValueTypeDefinitionsEqual(
	expected *docgen.ValueTypeDefinition,
	actual *docgen.ValueTypeDefinition,
	testSuite *suite.Suite,
) {
	if expected == nil && actual == nil {
		return
	}

	testSuite.NotNil(expected)
	testSuite.NotNil(actual)

	testSuite.Equal(expected.Type, actual.Type)
	testSuite.Equal(expected.Label, actual.Label)
	testSuite.Equal(expected.Description, actual.Description)
	testSuite.Equal(expected.StringChoices, actual.StringChoices)

	assertFunctionValueTypeDefinitionsEqual(
		expected.ElementValueTypeDefinition,
		actual.ElementValueTypeDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionsEqual(
		expected.MapValueTypeDefinition,
		actual.MapValueTypeDefinition,
		testSuite,
	)

	assertAttributeValueTypeDefinitionsEqual(
		expected.AttributeValueTypeDefinitions,
		actual.AttributeValueTypeDefinitions,
		testSuite,
	)

	assertFunctionDefinitionsEqual(
		expected.FunctionDefinition,
		actual.FunctionDefinition,
		testSuite,
	)

	assertFunctionValueTypeDefinitionSlicesEqual(
		expected.UnionValueTypeDefinitions,
		actual.UnionValueTypeDefinitions,
		testSuite,
	)
}

func assertAttributeValueTypeDefinitionsEqual(
	expected map[string]*docgen.AttributeType,
	actual map[string]*docgen.AttributeType,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for key, expectedAttributeType := range expected {
		actualAttributeType, ok := actual[key]
		testSuite.True(ok)
		testSuite.Equal(expectedAttributeType.Nullable, actualAttributeType.Nullable)
		assertFunctionValueTypeDefinitionsEqual(
			&actualAttributeType.ValueTypeDefinition,
			&actualAttributeType.ValueTypeDefinition,
			testSuite,
		)
	}
}

func sortFunctions(
	functions []*docgen.PluginDocsFunction,
) []*docgen.PluginDocsFunction {
	functionsCopy := make([]*docgen.PluginDocsFunction, len(functions))
	copy(functionsCopy, functions)

	slices.SortFunc(functionsCopy, func(
		a *docgen.PluginDocsFunction,
		b *docgen.PluginDocsFunction,
	) int {
		if a.Name > b.Name {
			return 1
		}

		if a.Name < b.Name {
			return -1
		}

		return 0
	})

	return functionsCopy
}
