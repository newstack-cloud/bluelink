package stageui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ExportCountingTestSuite struct {
	suite.Suite
}

func TestExportCountingTestSuite(t *testing.T) {
	suite.Run(t, new(ExportCountingTestSuite))
}

func (s *ExportCountingTestSuite) Test_countExportChanges_nil_returns_zeros() {
	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(nil)

	s.Equal(0, newCount)
	s.Equal(0, modifiedCount)
	s.Equal(0, removedCount)
	s.Equal(0, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_counts_new_exports() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {NewValue: stringNode("value1")},
			"export2": {NewValue: stringNode("value2")},
		},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(2, newCount)
	s.Equal(0, modifiedCount)
	s.Equal(0, removedCount)
	s.Equal(0, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_counts_modified_exports() {
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"export1": {
				PrevValue: stringNode("old1"),
				NewValue:  stringNode("new1"),
			},
			"export2": {
				PrevValue: stringNode("old2"),
				NewValue:  stringNode("new2"),
			},
		},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(0, newCount)
	s.Equal(2, modifiedCount)
	s.Equal(0, removedCount)
	s.Equal(0, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_counts_removed_exports() {
	bc := &changes.BlueprintChanges{
		RemovedExports: []string{"export1", "export2", "export3"},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(0, newCount)
	s.Equal(0, modifiedCount)
	s.Equal(3, removedCount)
	s.Equal(0, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_counts_unchanged_exports() {
	bc := &changes.BlueprintChanges{
		UnchangedExports: []string{"export1", "export2"},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(0, newCount)
	s.Equal(0, modifiedCount)
	s.Equal(0, removedCount)
	s.Equal(2, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_export_with_nil_prevValue_counted_as_new() {
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"export1": {NewValue: stringNode("value1")}, // nil prevValue = new
		},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(1, newCount)
	s.Equal(0, modifiedCount)
	s.Equal(0, removedCount)
	s.Equal(0, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_resolve_on_deploy_counted_as_unchanged_when_no_other_changes() {
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"computedExport": {
				PrevValue: stringNode("old-computed"),
				NewValue:  nil, // resolve-on-deploy placeholder
			},
		},
		ResolveOnDeploy: []string{"exports.computedExport"},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(0, newCount)
	s.Equal(0, modifiedCount)
	s.Equal(0, removedCount)
	s.Equal(1, unchangedCount, "resolve-on-deploy should be counted as unchanged when no other changes")
}

func (s *ExportCountingTestSuite) Test_countExportChanges_resolve_on_deploy_counted_as_modified_when_has_other_changes() {
	bc := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource": {},
		},
		ExportChanges: map[string]provider.FieldChange{
			"computedExport": {
				PrevValue: stringNode("old-computed"),
				NewValue:  nil, // resolve-on-deploy placeholder
			},
		},
		ResolveOnDeploy: []string{"exports.computedExport"},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(0, newCount)
	s.Equal(1, modifiedCount, "resolve-on-deploy should be counted as modified when there are other changes")
	s.Equal(0, removedCount)
	s.Equal(0, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_countExportChanges_mixed_types() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"newExport": {NewValue: stringNode("new")},
		},
		ExportChanges: map[string]provider.FieldChange{
			"modifiedExport": {
				PrevValue: stringNode("old"),
				NewValue:  stringNode("updated"),
			},
		},
		RemovedExports:   []string{"removedExport"},
		UnchangedExports: []string{"unchangedExport"},
	}

	newCount, modifiedCount, removedCount, unchangedCount := countExportChanges(bc)

	s.Equal(1, newCount)
	s.Equal(1, modifiedCount)
	s.Equal(1, removedCount)
	s.Equal(1, unchangedCount)
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_new_resources() {
	bc := &changes.BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource": {},
		},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_removed_resources() {
	bc := &changes.BlueprintChanges{
		RemovedResources: []string{"removedResource"},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_resource_changes() {
	bc := &changes.BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"changedResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.replicas", PrevValue: intNode(2), NewValue: intNode(4)},
				},
			},
		},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_new_children() {
	bc := &changes.BlueprintChanges{
		NewChildren: map[string]changes.NewBlueprintDefinition{
			"newChild": {},
		},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_removed_children() {
	bc := &changes.BlueprintChanges{
		RemovedChildren: []string{"removedChild"},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_false_for_only_resolve_on_deploy_exports() {
	bc := &changes.BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"computedExport": {
				PrevValue: stringNode("old"),
				NewValue:  nil,
			},
		},
		ResolveOnDeploy: []string{"exports.computedExport"},
	}

	s.False(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_new_exports() {
	bc := &changes.BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"newExport": {NewValue: stringNode("value")},
		},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_returns_true_for_removed_exports() {
	bc := &changes.BlueprintChanges{
		RemovedExports: []string{"removedExport"},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_hasActualChangesInChangeset_recursively_checks_children() {
	bc := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"childBlueprint": {
				NewResources: map[string]provider.Changes{
					"childNewResource": {},
				},
			},
		},
	}

	s.True(hasActualChangesInChangeset(bc))
}

func (s *ExportCountingTestSuite) Test_isResolveOnDeployPlaceholder_returns_true_when_matches() {
	change := &provider.FieldChange{
		PrevValue: stringNode("old"),
		NewValue:  nil,
	}
	resolveOnDeploy := []string{"exports.computedField"}

	result := isResolveOnDeployPlaceholder("computedField", change, resolveOnDeploy)

	s.True(result)
}

func (s *ExportCountingTestSuite) Test_isResolveOnDeployPlaceholder_returns_false_when_newValue_not_nil() {
	change := &provider.FieldChange{
		PrevValue: stringNode("old"),
		NewValue:  stringNode("new"),
	}
	resolveOnDeploy := []string{"exports.computedField"}

	result := isResolveOnDeployPlaceholder("computedField", change, resolveOnDeploy)

	s.False(result)
}

func (s *ExportCountingTestSuite) Test_isResolveOnDeployPlaceholder_returns_false_when_not_in_list() {
	change := &provider.FieldChange{
		PrevValue: stringNode("old"),
		NewValue:  nil,
	}
	resolveOnDeploy := []string{"exports.otherField"}

	result := isResolveOnDeployPlaceholder("computedField", change, resolveOnDeploy)

	s.False(result)
}

// Helper functions for creating MappingNodes
func stringNode(s string) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{StringValue: &s},
	}
}

func intNode(i int) *core.MappingNode {
	return &core.MappingNode{
		Scalar: &core.ScalarValue{IntValue: &i},
	}
}
