package changes

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

// Blueprint-wide metadata is staged through the same mapping node walk as a resource's
// metadata, but its recorded paths arrive on the resolve result rather than on a resolved
// resource, since there is no resource to carry them.
type MetadataChangeSensitivityTestSuite struct {
	suite.Suite
}

func (s *MetadataChangeSensitivityTestSuite) Test_marks_a_metadata_field_a_secret_reached() {
	metadataChanges := &MetadataChanges{}
	CollectMetadataChanges(
		metadataChanges,
		metadataResolveResult([]string{"metadata.auditKey"}),
		map[string]*core.MappingNode{
			"build":    core.MappingNodeFromString("webpack"),
			"auditKey": core.MappingNodeFromString("previous-audit"),
		},
	)

	modified := fieldChangesByPath(
		&s.Suite,
		metadataChanges.ModifiedFields,
		"metadata.auditKey",
		"metadata.build",
	)
	s.Assert().True(modified["metadata.auditKey"].Sensitive)
	s.Assert().False(modified["metadata.build"].Sensitive)
}

func (s *MetadataChangeSensitivityTestSuite) Test_leaves_every_field_unmarked_when_nothing_was_recorded() {
	metadataChanges := &MetadataChanges{}
	CollectMetadataChanges(
		metadataChanges,
		metadataResolveResult(nil),
		map[string]*core.MappingNode{
			"build":    core.MappingNodeFromString("webpack"),
			"auditKey": core.MappingNodeFromString("previous-audit"),
		},
	)

	modified := fieldChangesByPath(
		&s.Suite,
		metadataChanges.ModifiedFields,
		"metadata.auditKey",
	)
	s.Assert().False(modified["metadata.auditKey"].Sensitive)
}

// A record is shared with everything resolved on the way, so the result carries paths for
// values and data sources too. None of them can match a metadata path, and this pins that
// they do not mark one.
func (s *MetadataChangeSensitivityTestSuite) Test_is_not_marked_by_a_path_from_another_element() {
	metadataChanges := &MetadataChanges{}
	CollectMetadataChanges(
		metadataChanges,
		metadataResolveResult([]string{"values.connection.value.password"}),
		map[string]*core.MappingNode{
			"auditKey": core.MappingNodeFromString("previous-audit"),
		},
	)

	modified := fieldChangesByPath(&s.Suite, metadataChanges.ModifiedFields, "metadata.auditKey")
	s.Assert().False(modified["metadata.auditKey"].Sensitive)
}

func metadataResolveResult(sensitivePaths []string) *subengine.ResolveInMappingNodeResult {
	return &subengine.ResolveInMappingNodeResult{
		ResolvedMappingNode: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"build":    core.MappingNodeFromString("esbuild"),
				"auditKey": core.MappingNodeFromString("audit-super-secret"),
			},
		},
		SensitivePaths: sensitivePaths,
	}
}

// Keyed by path, with every asserted path required to be present: a missing entry reads as
// "not sensitive" and would pass regardless of what the code does.
func fieldChangesByPath(
	s *suite.Suite,
	fieldChanges []provider.FieldChange,
	paths ...string,
) map[string]provider.FieldChange {
	byPath := map[string]provider.FieldChange{}
	for _, fieldChange := range fieldChanges {
		byPath[fieldChange.FieldPath] = fieldChange
	}

	for _, path := range paths {
		_, found := byPath[path]
		s.Require().True(found, "%s is not among the field changes", path)
	}

	return byPath
}

func TestMetadataChangeSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(MetadataChangeSensitivityTestSuite))
}
