package testsuites

import (
	"slices"

	"github.com/newstack-cloud/bluelink/tools/plugin-docgen/internal/docgen"
	"github.com/stretchr/testify/suite"
)

func assertPluginDocLinksEqual(
	expected []*docgen.PluginDocsLink,
	actual []*docgen.PluginDocsLink,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedLink := range expected {
		actualLink := actual[i]
		testSuite.Equal(expectedLink.Type, actualLink.Type)
		testSuite.Equal(expectedLink.Summary, actualLink.Summary)
		testSuite.Equal(expectedLink.Description, actualLink.Description)
		assertPluginDocLinkAnnotationDefinitionsEqual(
			expectedLink.AnnotationDefinitions,
			actualLink.AnnotationDefinitions,
			testSuite,
		)
	}
}

func assertPluginDocLinkAnnotationDefinitionsEqual(
	expected map[string]*docgen.PluginDocsLinkAnnotationDefinition,
	actual map[string]*docgen.PluginDocsLinkAnnotationDefinition,
	testSuite *suite.Suite,
) {
	testSuite.Equal(len(expected), len(actual))
	for i, expectedAnnotation := range expected {
		actualAnnotation, ok := actual[i]
		testSuite.True(ok)
		testSuite.Equal(expectedAnnotation, actualAnnotation)
	}
}

func sortLinks(
	links []*docgen.PluginDocsLink,
) []*docgen.PluginDocsLink {
	linksCopy := make([]*docgen.PluginDocsLink, len(links))
	copy(linksCopy, links)

	slices.SortFunc(linksCopy, func(
		a *docgen.PluginDocsLink,
		b *docgen.PluginDocsLink,
	) int {
		if a.Type > b.Type {
			return 1
		}

		if a.Type < b.Type {
			return -1
		}

		return 0
	})

	return linksCopy
}
