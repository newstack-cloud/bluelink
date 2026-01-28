package testsuites

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/stretchr/testify/suite"
)

func assertPluginDocsEqual(
	expected *docgen.PluginDocs,
	actual *docgen.PluginDocs,
	testSuite *suite.Suite,
) {
	testSuite.Equal(expected.ID, actual.ID)
	testSuite.Equal(expected.DisplayName, actual.DisplayName)
	testSuite.Equal(expected.Description, actual.Description)
	testSuite.Equal(expected.Version, actual.Version)
	testSuite.Equal(expected.ProtocolVersions, actual.ProtocolVersions)
	testSuite.Equal(expected.Author, actual.Author)
	testSuite.Equal(expected.Repository, actual.Repository)
	assertPluginDocsConfigEqual(
		expected.Config,
		actual.Config,
		testSuite,
	)

	// Provider-specific assertions.
	assertPluginDocResourcesEqual(
		sortResources(expected.Resources),
		sortResources(actual.Resources),
		testSuite,
	)
	assertPluginDocLinksEqual(
		sortLinks(expected.Links),
		sortLinks(actual.Links),
		testSuite,
	)
	assertPluginDocDataSourcesEqual(
		sortDataSources(expected.DataSources),
		sortDataSources(actual.DataSources),
		testSuite,
	)
	assertPluginDocCustomVarTypesEqual(
		sortCustomVariableTypes(expected.CustomVarTypes),
		sortCustomVariableTypes(actual.CustomVarTypes),
		testSuite,
	)
	assertPluginDocFunctionsEqual(
		sortFunctions(expected.Functions),
		sortFunctions(actual.Functions),
		testSuite,
	)

	// Transformer-specific assertions.
	testSuite.Equal(expected.TransformName, actual.TransformName)
	assertPluginDocResourcesEqual(
		sortResources(expected.AbstractResources),
		sortResources(actual.AbstractResources),
		testSuite,
	)
}

func assertPluginDocsConfigEqual(
	expected *docgen.PluginDocsVersionConfig,
	actual *docgen.PluginDocsVersionConfig,
	testSuite *suite.Suite,
) {
	testSuite.Equal(expected.AllowAdditionalFields, actual.AllowAdditionalFields)
	testSuite.Equal(len(expected.Fields), len(actual.Fields))
	for key, expectedField := range expected.Fields {
		actualField, ok := actual.Fields[key]
		testSuite.True(ok)
		testSuite.Equal(expectedField, actualField)
	}
}

func assertMappingNodeSlicesEqual(
	expected []*core.MappingNode,
	actual []*core.MappingNode,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedNode := range expected {
		actualNode := actual[i]
		assertMappingNodeEqual(
			expectedNode,
			actualNode,
			testSuite,
		)
	}
}

func assertMappingNodeEqual(
	expected *core.MappingNode,
	actual *core.MappingNode,
	testSuite *suite.Suite,
) {
	if core.IsNilMappingNode(expected) && core.IsNilMappingNode(actual) {
		return
	}

	testSuite.NotNil(expected)
	testSuite.NotNil(actual)

	if expected.Fields != nil {
		assertMappingNodeFieldsEqual(
			expected.Fields,
			actual.Fields,
			testSuite,
		)
	}

	if expected.Scalar != nil {
		testSuite.Equal(expected.Scalar, actual.Scalar)
	}

	if expected.Items != nil {
		assertMappingNodeItemsEqual(
			expected.Items,
			actual.Items,
			testSuite,
		)
	}
}

func assertMappingNodeFieldsEqual(
	expected map[string]*core.MappingNode,
	actual map[string]*core.MappingNode,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for key, expectedField := range expected {
		actualField, ok := actual[key]
		testSuite.True(ok)
		assertMappingNodeEqual(
			expectedField,
			actualField,
			testSuite,
		)
	}
}

func assertMappingNodeItemsEqual(
	expected []*core.MappingNode,
	actual []*core.MappingNode,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedItem := range expected {
		actualItem := actual[i]
		assertMappingNodeEqual(
			expectedItem,
			actualItem,
			testSuite,
		)
	}
}

func assertScalarValueEqual(
	expected *core.ScalarValue,
	actual *core.ScalarValue,
	testSuite *suite.Suite,
) {
	if expected == nil && actual == nil {
		return
	}
	testSuite.NotNil(expected)
	testSuite.NotNil(actual)
	testSuite.Equal(expected, actual)
}
