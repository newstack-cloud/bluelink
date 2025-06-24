package testsuites

import (
	"slices"

	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/stretchr/testify/suite"
)

func assertPluginDocDataSourcesEqual(
	expected []*docgen.PluginDocsDataSource,
	actual []*docgen.PluginDocsDataSource,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedDataSource := range expected {
		actualDataSource := actual[i]
		testSuite.Equal(expectedDataSource.Type, actualDataSource.Type)
		testSuite.Equal(expectedDataSource.Label, actualDataSource.Label)
		testSuite.Equal(expectedDataSource.Summary, actualDataSource.Summary)
		testSuite.Equal(expectedDataSource.Description, actualDataSource.Description)
		assertPluginDocsDataSourceSpecsEqual(
			expectedDataSource.Specification,
			actualDataSource.Specification,
			testSuite,
		)
		testSuite.Equal(expectedDataSource.Examples, actualDataSource.Examples)
	}
}

func assertPluginDocsDataSourceSpecsEqual(
	expected *docgen.PluginDocsDataSourceSpec,
	actual *docgen.PluginDocsDataSourceSpec,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected.Fields), len(actual.Fields))
	for key, expectedField := range expected.Fields {
		actualField, ok := actual.Fields[key]
		testSuite.True(ok)
		testSuite.Equal(expectedField, actualField)
	}
}

func sortDataSources(
	dataSources []*docgen.PluginDocsDataSource,
) []*docgen.PluginDocsDataSource {
	dataSourcesCopy := make([]*docgen.PluginDocsDataSource, len(dataSources))
	copy(dataSourcesCopy, dataSources)

	slices.SortFunc(dataSourcesCopy, func(
		a *docgen.PluginDocsDataSource,
		b *docgen.PluginDocsDataSource,
	) int {
		if a.Type > b.Type {
			return 1
		}

		if a.Type < b.Type {
			return -1
		}

		return 0
	})

	return dataSourcesCopy
}
