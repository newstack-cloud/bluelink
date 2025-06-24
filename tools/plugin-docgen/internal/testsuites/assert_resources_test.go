package testsuites

import (
	"slices"

	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/stretchr/testify/suite"
)

func assertPluginDocResourcesEqual(
	expected []*docgen.PluginDocsResource,
	actual []*docgen.PluginDocsResource,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedResource := range expected {
		actualResource := actual[i]
		testSuite.Equal(expectedResource.Type, actualResource.Type)
		testSuite.Equal(expectedResource.Label, actualResource.Label)
		testSuite.Equal(expectedResource.Description, actualResource.Description)
		testSuite.Equal(expectedResource.Summary, actualResource.Summary)
		assertPluginDocResourceSpecsEqual(
			expectedResource.Specification,
			actualResource.Specification,
			testSuite,
		)
		testSuite.Equal(expectedResource.Examples, actualResource.Examples)
		testSuite.Equal(expectedResource.CanLinkTo, actualResource.CanLinkTo)
	}
}

func assertPluginDocResourceSpecsEqual(
	expected *docgen.PluginDocResourceSpec,
	actual *docgen.PluginDocResourceSpec,
	testSuite *suite.Suite,
) {
	testSuite.Equal(expected.IDField, actual.IDField)
	assertPluginDocResourceSpecSchemasEqual(
		expected.Schema,
		actual.Schema,
		testSuite,
	)
}

func assertPluginDocResourceSpecSchemasEqual(
	expected *docgen.PluginDocResourceSpecSchema,
	actual *docgen.PluginDocResourceSpecSchema,
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
	testSuite.Equal(expected.Nullable, actual.Nullable)
	testSuite.Equal(expected.Computed, actual.Computed)
	testSuite.Equal(expected.MustRecreate, actual.MustRecreate)
	assertMappingNodeEqual(
		expected.Default,
		actual.Default,
		testSuite,
	)
	assertMappingNodeSlicesEqual(
		expected.Examples,
		actual.Examples,
		testSuite,
	)
	assertResourceSpecSchemaAttributesEqual(
		expected.Attributes,
		actual.Attributes,
		testSuite,
	)
	testSuite.Equal(expected.Required, actual.Required)
	assertPluginDocResourceSpecSchemasEqual(
		expected.Items,
		actual.Items,
		testSuite,
	)
	assertPluginDocResourceSpecSchemasEqual(
		expected.MapValues,
		actual.MapValues,
		testSuite,
	)
	assertPluginDocResourceSpecUnionSchemasEqual(
		expected.OneOf,
		actual.OneOf,
		testSuite,
	)
}

func assertPluginDocResourceSpecUnionSchemasEqual(
	expected []*docgen.PluginDocResourceSpecSchema,
	actual []*docgen.PluginDocResourceSpecSchema,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedSchema := range expected {
		actualSchema := actual[i]
		assertPluginDocResourceSpecSchemasEqual(
			expectedSchema,
			actualSchema,
			testSuite,
		)
	}
}

func assertResourceSpecSchemaAttributesEqual(
	expected map[string]*docgen.PluginDocResourceSpecSchema,
	actual map[string]*docgen.PluginDocResourceSpecSchema,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for key, expectedAttribute := range expected {
		actualAttribute, ok := actual[key]
		testSuite.True(ok)
		assertPluginDocResourceSpecSchemasEqual(
			expectedAttribute,
			actualAttribute,
			testSuite,
		)
	}
}

func sortResources(
	resources []*docgen.PluginDocsResource,
) []*docgen.PluginDocsResource {
	resourcesCopy := make([]*docgen.PluginDocsResource, len(resources))
	copy(resourcesCopy, resources)

	slices.SortFunc(resourcesCopy, func(
		a *docgen.PluginDocsResource,
		b *docgen.PluginDocsResource,
	) int {
		if a.Type > b.Type {
			return 1
		}

		if a.Type < b.Type {
			return -1
		}

		return 0
	})

	return resourcesCopy
}
