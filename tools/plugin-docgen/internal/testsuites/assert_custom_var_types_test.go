package testsuites

import (
	"slices"

	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/stretchr/testify/suite"
)

func assertPluginDocCustomVarTypesEqual(
	expected []*docgen.PluginDocsCustomVarType,
	actual []*docgen.PluginDocsCustomVarType,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedCustomVarType := range expected {
		actualCustomVarType := actual[i]
		testSuite.Equal(expectedCustomVarType.Type, actualCustomVarType.Type)
		testSuite.Equal(expectedCustomVarType.Label, actualCustomVarType.Label)
		testSuite.Equal(expectedCustomVarType.Summary, actualCustomVarType.Summary)
		testSuite.Equal(expectedCustomVarType.Description, actualCustomVarType.Description)
		assertPluginDocCustomVarTypeOptionsEqual(
			expectedCustomVarType.Options,
			actualCustomVarType.Options,
			testSuite,
		)
		testSuite.Equal(expectedCustomVarType.Examples, actualCustomVarType.Examples)
	}
}

func assertPluginDocCustomVarTypeOptionsEqual(
	expected map[string]*docgen.PluginDocsCustomVarTypeOption,
	actual map[string]*docgen.PluginDocsCustomVarTypeOption,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedOption := range expected {
		actualOption, ok := actual[i]
		testSuite.True(ok)
		testSuite.Equal(expectedOption, actualOption)
	}
}

func sortCustomVariableTypes(
	customVarTypes []*docgen.PluginDocsCustomVarType,
) []*docgen.PluginDocsCustomVarType {
	customVarTypesCopy := make([]*docgen.PluginDocsCustomVarType, len(customVarTypes))
	copy(customVarTypesCopy, customVarTypes)

	slices.SortFunc(customVarTypesCopy, func(
		a *docgen.PluginDocsCustomVarType,
		b *docgen.PluginDocsCustomVarType,
	) int {
		if a.Type > b.Type {
			return 1
		}

		if a.Type < b.Type {
			return -1
		}

		return 0
	})

	return customVarTypesCopy
}
