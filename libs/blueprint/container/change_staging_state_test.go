package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ChangeStagingStateTestSuite struct {
	suite.Suite
}

// createTestChangeStagingState creates a ChangeStagingState for tests,
// with metadata changes initialized to avoid nil pointer dereferences.
func createTestChangeStagingState() ChangeStagingState {
	state := NewDefaultChangeStagingState()
	// Initialize metadata changes to avoid nil pointer dereference in ExtractBlueprintChanges
	state.UpdateMetadataChanges(&changes.MetadataChanges{}, nil)
	return state
}

func (s *ChangeStagingStateTestSuite) Test_ApplyResourceChanges_adds_new_resource_to_NewResources() {
	state := createTestChangeStagingState()
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "testResource",
		New:          true,
		Changes: provider.Changes{
			NewFields: []provider.FieldChange{
				{FieldPath: "spec.field1"},
			},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Contains(blueprintChanges.NewResources, "testResource")
	s.Empty(blueprintChanges.ResourceChanges)
}

func (s *ChangeStagingStateTestSuite) Test_ApplyResourceChanges_adds_removed_resource_to_RemovedResources() {
	state := createTestChangeStagingState()
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "testResource",
		Removed:      true,
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Contains(blueprintChanges.RemovedResources, "testResource")
	s.Empty(blueprintChanges.ResourceChanges)
}

func (s *ChangeStagingStateTestSuite) Test_ApplyResourceChanges_adds_resource_with_changes_to_ResourceChanges() {
	state := createTestChangeStagingState()
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "testResource",
		New:          false,
		Removed:      false,
		Changes: provider.Changes{
			ModifiedFields: []provider.FieldChange{
				{FieldPath: "spec.field1"},
			},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Contains(blueprintChanges.ResourceChanges, "testResource")
	s.Empty(blueprintChanges.NewResources)
}

func (s *ChangeStagingStateTestSuite) Test_ApplyResourceChanges_excludes_resource_with_no_field_changes() {
	state := createTestChangeStagingState()
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "testResource",
		New:          false,
		Removed:      false,
		Changes: provider.Changes{
			// Only unchanged fields, no modifications
			UnchangedFields: []string{"spec.field1", "spec.field2"},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Empty(blueprintChanges.ResourceChanges)
	s.Empty(blueprintChanges.NewResources)
	s.Empty(blueprintChanges.RemovedResources)
}

func (s *ChangeStagingStateTestSuite) Test_ApplyResourceChanges_includes_resource_with_new_fields() {
	state := createTestChangeStagingState()
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "testResource",
		New:          false,
		Removed:      false,
		Changes: provider.Changes{
			NewFields: []provider.FieldChange{
				{FieldPath: "spec.newField"},
			},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Contains(blueprintChanges.ResourceChanges, "testResource")
}

func (s *ChangeStagingStateTestSuite) Test_ApplyResourceChanges_includes_resource_with_removed_fields() {
	state := createTestChangeStagingState()
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "testResource",
		New:          false,
		Removed:      false,
		Changes: provider.Changes{
			RemovedFields: []string{"spec.removedField"},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Contains(blueprintChanges.ResourceChanges, "testResource")
}

func (s *ChangeStagingStateTestSuite) Test_ApplyLinkChanges_adds_new_link_to_NewOutboundLinks() {
	state := createTestChangeStagingState()

	// First add the resource that the link belongs to
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "resourceA",
		New:          true,
		Changes:      provider.Changes{},
	})

	// Then add the new link
	state.ApplyLinkChanges(LinkChangesMessage{
		ResourceAName: "resourceA",
		ResourceBName: "resourceB",
		New:           true,
		Changes: provider.LinkChanges{
			NewFields: []*provider.FieldChange{
				{FieldPath: "link.field1"},
			},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	resourceChanges := blueprintChanges.NewResources["resourceA"]
	s.Contains(resourceChanges.NewOutboundLinks, "resourceB")
}

func (s *ChangeStagingStateTestSuite) Test_ApplyLinkChanges_adds_removed_link_to_RemovedLinks() {
	state := createTestChangeStagingState()

	state.ApplyLinkChanges(LinkChangesMessage{
		ResourceAName: "resourceA",
		ResourceBName: "resourceB",
		Removed:       true,
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	s.Contains(blueprintChanges.RemovedLinks, "resourceA::resourceB")
}

func (s *ChangeStagingStateTestSuite) Test_ApplyLinkChanges_adds_link_with_changes_to_OutboundLinkChanges() {
	state := createTestChangeStagingState()

	// First add the resource that the link belongs to
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "resourceA",
		New:          false,
		Removed:      false,
		Changes: provider.Changes{
			ModifiedFields: []provider.FieldChange{
				{FieldPath: "spec.field1"},
			},
		},
	})

	// Then add the link with changes
	state.ApplyLinkChanges(LinkChangesMessage{
		ResourceAName: "resourceA",
		ResourceBName: "resourceB",
		New:           false,
		Removed:       false,
		Changes: provider.LinkChanges{
			ModifiedFields: []*provider.FieldChange{
				{FieldPath: "link.field1"},
			},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	resourceChanges := blueprintChanges.ResourceChanges["resourceA"]
	s.Contains(resourceChanges.OutboundLinkChanges, "resourceB")
}

func (s *ChangeStagingStateTestSuite) Test_ApplyLinkChanges_excludes_link_with_no_field_changes() {
	state := createTestChangeStagingState()

	// First add the resource that the link belongs to
	state.ApplyResourceChanges(ResourceChangesMessage{
		ResourceName: "resourceA",
		New:          false,
		Removed:      false,
		Changes: provider.Changes{
			ModifiedFields: []provider.FieldChange{
				{FieldPath: "spec.field1"},
			},
		},
	})

	// Then add a link with no field changes
	state.ApplyLinkChanges(LinkChangesMessage{
		ResourceAName: "resourceA",
		ResourceBName: "resourceB",
		New:           false,
		Removed:       false,
		Changes: provider.LinkChanges{
			// Only unchanged fields, no modifications
			UnchangedFields: []string{"link.field1", "link.field2"},
		},
	})

	blueprintChanges := state.ExtractBlueprintChanges()
	resourceChanges := blueprintChanges.ResourceChanges["resourceA"]
	s.Empty(resourceChanges.OutboundLinkChanges)
}

func TestChangeStagingStateTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeStagingStateTestSuite))
}
