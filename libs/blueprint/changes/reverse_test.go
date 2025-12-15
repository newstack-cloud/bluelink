package changes

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ReverseChangesetTestSuite struct {
	suite.Suite
}

func (s *ReverseChangesetTestSuite) Test_returns_nil_when_original_is_nil() {
	previousState := &state.InstanceState{}
	result, err := ReverseChangeset( /* original */ nil, previousState)
	s.NoError(err)
	s.Nil(result)
}

func (s *ReverseChangesetTestSuite) Test_returns_nil_when_previous_state_is_nil() {
	original := &BlueprintChanges{}
	result, err := ReverseChangeset(original /* previousState */, nil)
	s.NoError(err)
	s.Nil(result)
}

func (s *ReverseChangesetTestSuite) Test_returns_nil_when_both_are_nil() {
	result, err := ReverseChangeset( /* original */ nil /* previousState */, nil)
	s.NoError(err)
	s.Nil(result)
}

func (s *ReverseChangesetTestSuite) Test_reverses_field_change() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				ModifiedFields: []provider.FieldChange{
					{
						FieldPath: "spec.config.value",
						PrevValue: core.MappingNodeFromString("old-value"),
						NewValue:  core.MappingNodeFromString("new-value"),
					},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.Require().Len(resourceChanges.ModifiedFields, 1)
	s.Equal("spec.config.value", resourceChanges.ModifiedFields[0].FieldPath)
	s.Equal("new-value", core.StringValue(resourceChanges.ModifiedFields[0].PrevValue))
	s.Equal("old-value", core.StringValue(resourceChanges.ModifiedFields[0].NewValue))
}

func (s *ReverseChangesetTestSuite) Test_reverses_field_change_preserves_flags() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				ModifiedFields: []provider.FieldChange{
					{
						FieldPath:    "spec.password",
						PrevValue:    core.MappingNodeFromString("old-secret"),
						NewValue:     core.MappingNodeFromString("new-secret"),
						MustRecreate: true,
						Sensitive:    true,
					},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.Require().Len(resourceChanges.ModifiedFields, 1)
	s.True(resourceChanges.ModifiedFields[0].MustRecreate)
	s.True(resourceChanges.ModifiedFields[0].Sensitive)
}

func (s *ReverseChangesetTestSuite) Test_reverses_modified_resource_fields() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				ModifiedFields: []provider.FieldChange{
					{
						FieldPath: "spec.replicas",
						PrevValue: core.MappingNodeFromInt(3),
						NewValue:  core.MappingNodeFromInt(5),
					},
					{
						FieldPath: "spec.name",
						PrevValue: core.MappingNodeFromString("old-name"),
						NewValue:  core.MappingNodeFromString("new-name"),
					},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Require().Contains(reversed.ResourceChanges, "myResource")

	resourceChanges := reversed.ResourceChanges["myResource"]
	s.Require().Len(resourceChanges.ModifiedFields, 2)

	// First field
	s.Equal("spec.replicas", resourceChanges.ModifiedFields[0].FieldPath)
	s.Equal(5, core.IntValue(resourceChanges.ModifiedFields[0].PrevValue))
	s.Equal(3, core.IntValue(resourceChanges.ModifiedFields[0].NewValue))

	// Second field
	s.Equal("spec.name", resourceChanges.ModifiedFields[1].FieldPath)
	s.Equal("new-name", core.StringValue(resourceChanges.ModifiedFields[1].PrevValue))
	s.Equal("old-name", core.StringValue(resourceChanges.ModifiedFields[1].NewValue))
}

func (s *ReverseChangesetTestSuite) Test_new_resources_become_removed() {
	original := &BlueprintChanges{
		NewResources: map[string]provider.Changes{
			"newResource1": {},
			"newResource2": {},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.NewResources, 0)
	s.ElementsMatch(
		[]string{"newResource1", "newResource2"},
		reversed.RemovedResources,
	)
}

func (s *ReverseChangesetTestSuite) Test_removed_resources_become_new() {
	original := &BlueprintChanges{
		RemovedResources: []string{"removedResource1", "removedResource2"},
	}
	previousState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"removedResource1": "res-id-1",
			"removedResource2": "res-id-2",
		},
		Resources: map[string]*state.ResourceState{
			"res-id-1": {
				ResourceID: "res-id-1",
				Name:       "removedResource1",
				Type:       "test/resource",
			},
			"res-id-2": {
				ResourceID: "res-id-2",
				Name:       "removedResource2",
				Type:       "test/resource",
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.RemovedResources, 0)
	s.Contains(reversed.NewResources, "removedResource1")
	s.Contains(reversed.NewResources, "removedResource2")
}

func (s *ReverseChangesetTestSuite) Test_new_fields_become_removed() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				NewFields: []provider.FieldChange{
					{FieldPath: "spec.newField1"},
					{FieldPath: "spec.newField2"},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.ElementsMatch(
		[]string{"spec.newField1", "spec.newField2"},
		resourceChanges.RemovedFields,
	)
}

func (s *ReverseChangesetTestSuite) Test_removed_fields_become_new() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				RemovedFields: []string{"spec.oldField1", "spec.oldField2"},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.Require().Len(resourceChanges.NewFields, 2)
	s.Equal("spec.oldField1", resourceChanges.NewFields[0].FieldPath)
	s.Equal("spec.oldField2", resourceChanges.NewFields[1].FieldPath)
}

func (s *ReverseChangesetTestSuite) Test_new_exports_become_removed() {
	original := &BlueprintChanges{
		NewExports: map[string]provider.FieldChange{
			"export1": {FieldPath: "resources.myResource.spec.output"},
			"export2": {FieldPath: "resources.myResource.spec.id"},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.NewExports, 0)
	s.ElementsMatch(
		[]string{"export1", "export2"},
		reversed.RemovedExports,
	)
}

func (s *ReverseChangesetTestSuite) Test_removed_exports_become_new() {
	original := &BlueprintChanges{
		RemovedExports: []string{"oldExport1", "oldExport2"},
	}
	previousState := &state.InstanceState{
		Exports: map[string]*state.ExportState{
			"oldExport1": {
				Field: "resources.myResource.spec.field1",
				Value: core.MappingNodeFromString("value1"),
				Type:  schema.ExportTypeString,
			},
			"oldExport2": {
				Field: "resources.myResource.spec.field2",
				Value: core.MappingNodeFromInt(42),
				Type:  schema.ExportTypeInteger,
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.RemovedExports, 0)
	s.Contains(reversed.NewExports, "oldExport1")
	s.Contains(reversed.NewExports, "oldExport2")
	s.Equal("resources.myResource.spec.field1", reversed.NewExports["oldExport1"].FieldPath)
	s.Equal("value1", core.StringValue(reversed.NewExports["oldExport1"].NewValue))
	s.Equal("resources.myResource.spec.field2", reversed.NewExports["oldExport2"].FieldPath)
	s.Equal(42, core.IntValue(reversed.NewExports["oldExport2"].NewValue))
}

func (s *ReverseChangesetTestSuite) Test_reverses_export_changes() {
	original := &BlueprintChanges{
		ExportChanges: map[string]provider.FieldChange{
			"myExport": {
				FieldPath: "resources.myResource.spec.output",
				PrevValue: core.MappingNodeFromString("old-output"),
				NewValue:  core.MappingNodeFromString("new-output"),
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Contains(reversed.ExportChanges, "myExport")
	s.Equal("new-output", core.StringValue(reversed.ExportChanges["myExport"].PrevValue))
	s.Equal("old-output", core.StringValue(reversed.ExportChanges["myExport"].NewValue))
}

func (s *ReverseChangesetTestSuite) Test_new_children_become_removed() {
	original := &BlueprintChanges{
		NewChildren: map[string]NewBlueprintDefinition{
			"childBlueprint1": {},
			"childBlueprint2": {},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.NewChildren, 0)
	s.ElementsMatch(
		[]string{"childBlueprint1", "childBlueprint2"},
		reversed.RemovedChildren,
	)
}

func (s *ReverseChangesetTestSuite) Test_removed_children_become_new() {
	original := &BlueprintChanges{
		RemovedChildren: []string{"oldChild1"},
	}
	previousState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"oldChild1": {
				InstanceID: "child-instance-1",
				ResourceIDs: map[string]string{
					"childResource": "child-res-id",
				},
				Resources: map[string]*state.ResourceState{
					"child-res-id": {
						ResourceID: "child-res-id",
						Name:       "childResource",
						Type:       "test/childResource",
					},
				},
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.RemovedChildren, 0)
	s.Contains(reversed.NewChildren, "oldChild1")
	s.Contains(reversed.NewChildren["oldChild1"].NewResources, "childResource")
}

func (s *ReverseChangesetTestSuite) Test_reverses_child_changes_recursively() {
	childChanges := BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"nestedResource": {
				ModifiedFields: []provider.FieldChange{
					{
						FieldPath: "spec.value",
						PrevValue: core.MappingNodeFromString("nested-old"),
						NewValue:  core.MappingNodeFromString("nested-new"),
					},
				},
			},
		},
	}

	original := &BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"childBlueprint": childChanges,
		},
	}
	previousState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID: "child-instance",
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Contains(reversed.ChildChanges, "childBlueprint")

	childReversed := reversed.ChildChanges["childBlueprint"]
	s.Contains(childReversed.ResourceChanges, "nestedResource")

	nestedChanges := childReversed.ResourceChanges["nestedResource"]
	s.Require().Len(nestedChanges.ModifiedFields, 1)
	s.Equal("nested-new", core.StringValue(nestedChanges.ModifiedFields[0].PrevValue))
	s.Equal("nested-old", core.StringValue(nestedChanges.ModifiedFields[0].NewValue))
}

func (s *ReverseChangesetTestSuite) Test_reverses_metadata_changes() {
	original := &BlueprintChanges{
		MetadataChanges: MetadataChanges{
			ModifiedFields: []provider.FieldChange{
				{
					FieldPath: "metadata.labels.app",
					PrevValue: core.MappingNodeFromString("old-app"),
					NewValue:  core.MappingNodeFromString("new-app"),
				},
			},
			NewFields: []provider.FieldChange{
				{FieldPath: "metadata.annotations.new"},
			},
			RemovedFields: []string{"metadata.annotations.old"},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)

	// Modified fields should be reversed
	s.Require().Len(reversed.MetadataChanges.ModifiedFields, 1)
	s.Equal("new-app", core.StringValue(reversed.MetadataChanges.ModifiedFields[0].PrevValue))
	s.Equal("old-app", core.StringValue(reversed.MetadataChanges.ModifiedFields[0].NewValue))

	// New fields become removed
	s.ElementsMatch([]string{"metadata.annotations.new"}, reversed.MetadataChanges.RemovedFields)

	// Removed fields become new
	s.Require().Len(reversed.MetadataChanges.NewFields, 1)
	s.Equal("metadata.annotations.old", reversed.MetadataChanges.NewFields[0].FieldPath)
}

func (s *ReverseChangesetTestSuite) Test_new_outbound_links_become_removed() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				NewOutboundLinks: map[string]provider.LinkChanges{
					"linkedResource1": {},
					"linkedResource2": {},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.ElementsMatch(
		[]string{"linkedResource1", "linkedResource2"},
		resourceChanges.RemovedOutboundLinks,
	)
}

func (s *ReverseChangesetTestSuite) Test_removed_outbound_links_become_new() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				RemovedOutboundLinks: []string{"oldLink1", "oldLink2"},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.Contains(resourceChanges.NewOutboundLinks, "oldLink1")
	s.Contains(resourceChanges.NewOutboundLinks, "oldLink2")
}

func (s *ReverseChangesetTestSuite) Test_reverses_outbound_link_changes() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				OutboundLinkChanges: map[string]provider.LinkChanges{
					"linkedResource": {
						ModifiedFields: []*provider.FieldChange{
							{
								FieldPath: "linkData.priority",
								PrevValue: core.MappingNodeFromInt(1),
								NewValue:  core.MappingNodeFromInt(2),
							},
						},
					},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	s.Contains(resourceChanges.OutboundLinkChanges, "linkedResource")

	linkChanges := resourceChanges.OutboundLinkChanges["linkedResource"]
	s.Require().Len(linkChanges.ModifiedFields, 1)
	s.Equal(2, core.IntValue(linkChanges.ModifiedFields[0].PrevValue))
	s.Equal(1, core.IntValue(linkChanges.ModifiedFields[0].NewValue))
}

func (s *ReverseChangesetTestSuite) Test_reverses_outbound_link_new_and_removed_fields() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"myResource": {
				OutboundLinkChanges: map[string]provider.LinkChanges{
					"linkedResource": {
						NewFields: []*provider.FieldChange{
							{FieldPath: "linkData.newField"},
						},
						RemovedFields: []string{"linkData.oldField"},
					},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	resourceChanges := reversed.ResourceChanges["myResource"]
	linkChanges := resourceChanges.OutboundLinkChanges["linkedResource"]

	// New fields become removed
	s.ElementsMatch([]string{"linkData.newField"}, linkChanges.RemovedFields)

	// Removed fields become new
	s.Require().Len(linkChanges.NewFields, 1)
	s.Equal("linkData.oldField", linkChanges.NewFields[0].FieldPath)
}

func (s *ReverseChangesetTestSuite) Test_collects_links_to_remove() {
	original := &BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"resource1": {
				NewOutboundLinks: map[string]provider.LinkChanges{
					"link1": {},
				},
			},
		},
		NewResources: map[string]provider.Changes{
			"newResource": {
				NewOutboundLinks: map[string]provider.LinkChanges{
					"link2": {},
					"link3": {},
				},
			},
		},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.ElementsMatch([]string{"link1", "link2", "link3"}, reversed.RemovedLinks)
}

func (s *ReverseChangesetTestSuite) Test_skips_removed_resource_without_id() {
	original := &BlueprintChanges{
		RemovedResources: []string{"unknownResource"},
	}
	previousState := &state.InstanceState{
		ResourceIDs: map[string]string{},
		Resources:   map[string]*state.ResourceState{},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.NewResources, 0)
}

func (s *ReverseChangesetTestSuite) Test_skips_removed_child_without_state() {
	original := &BlueprintChanges{
		RemovedChildren: []string{"unknownChild"},
	}
	previousState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Len(reversed.NewChildren, 0)
}

func (s *ReverseChangesetTestSuite) Test_preserves_unchanged_exports() {
	original := &BlueprintChanges{
		UnchangedExports: []string{"unchangedExport1", "unchangedExport2"},
	}
	previousState := &state.InstanceState{}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.ElementsMatch(
		[]string{"unchangedExport1", "unchangedExport2"},
		reversed.UnchangedExports,
	)
}

func (s *ReverseChangesetTestSuite) Test_builds_new_child_from_state_with_nested_children() {
	original := &BlueprintChanges{
		RemovedChildren: []string{"childBlueprint"},
	}
	previousState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childBlueprint": {
				InstanceID: "child-instance",
				ResourceIDs: map[string]string{
					"childResource": "child-res-id",
				},
				Resources: map[string]*state.ResourceState{
					"child-res-id": {
						ResourceID: "child-res-id",
						Name:       "childResource",
						Type:       "test/childResource",
					},
				},
				ChildBlueprints: map[string]*state.InstanceState{
					"nestedChild": {
						InstanceID: "nested-instance",
						ResourceIDs: map[string]string{
							"nestedResource": "nested-res-id",
						},
						Resources: map[string]*state.ResourceState{
							"nested-res-id": {
								ResourceID: "nested-res-id",
								Name:       "nestedResource",
								Type:       "test/nestedResource",
							},
						},
					},
				},
				Exports: map[string]*state.ExportState{
					"childExport": {
						Field: "resources.childResource.spec.output",
						Value: core.MappingNodeFromString("exported-value"),
					},
				},
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Require().NoError(err)
	s.Require().NotNil(reversed)
	s.Require().Contains(reversed.NewChildren, "childBlueprint")

	newChild := reversed.NewChildren["childBlueprint"]
	s.Contains(newChild.NewResources, "childResource")
	s.Contains(newChild.NewChildren, "nestedChild")
	s.Contains(newChild.NewChildren["nestedChild"].NewResources, "nestedResource")
	s.Contains(newChild.NewExports, "childExport")
}

func (s *ReverseChangesetTestSuite) Test_returns_error_when_max_depth_exceeded_in_child_changes() {
	// Build a deeply nested structure that exceeds MaxReverseChangesetDepth (5)
	// We need 6 levels of nesting to exceed the limit (depth starts at 0)
	deepestChanges := BlueprintChanges{
		ResourceChanges: map[string]provider.Changes{
			"deepResource": {
				ModifiedFields: []provider.FieldChange{
					{FieldPath: "spec.value"},
				},
			},
		},
	}

	// Build nested child changes from inside out
	level5Changes := BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"level6": deepestChanges,
		},
	}
	level4Changes := BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"level5": level5Changes,
		},
	}
	level3Changes := BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"level4": level4Changes,
		},
	}
	level2Changes := BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"level3": level3Changes,
		},
	}

	original := &BlueprintChanges{
		ChildChanges: map[string]BlueprintChanges{
			"level2": level2Changes,
		},
	}

	// Build corresponding nested state
	previousState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"level2": {
				InstanceID: "level2-instance",
				ChildBlueprints: map[string]*state.InstanceState{
					"level3": {
						InstanceID: "level3-instance",
						ChildBlueprints: map[string]*state.InstanceState{
							"level4": {
								InstanceID: "level4-instance",
								ChildBlueprints: map[string]*state.InstanceState{
									"level5": {
										InstanceID: "level5-instance",
										ChildBlueprints: map[string]*state.InstanceState{
											"level6": {
												InstanceID: "level6-instance",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Nil(reversed)
	s.Require().Error(err)
	runErr, isRunErr := err.(*bperrors.RunError)
	s.Require().True(isRunErr)
	s.Equal(ErrorReasonCodeMaxReverseDepthExceeded, runErr.ReasonCode)
}

func (s *ReverseChangesetTestSuite) Test_returns_error_when_max_depth_exceeded_in_removed_children() {
	// Build a deeply nested state structure that exceeds MaxReverseChangesetDepth (5)
	// when rebuilding removed children from state
	original := &BlueprintChanges{
		RemovedChildren: []string{"level1"},
	}

	// Build deeply nested state (6 levels deep to exceed the limit of 5)
	previousState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"level1": {
				InstanceID: "level1-instance",
				ChildBlueprints: map[string]*state.InstanceState{
					"level2": {
						InstanceID: "level2-instance",
						ChildBlueprints: map[string]*state.InstanceState{
							"level3": {
								InstanceID: "level3-instance",
								ChildBlueprints: map[string]*state.InstanceState{
									"level4": {
										InstanceID: "level4-instance",
										ChildBlueprints: map[string]*state.InstanceState{
											"level5": {
												InstanceID: "level5-instance",
												ChildBlueprints: map[string]*state.InstanceState{
													"level6": {
														InstanceID: "level6-instance",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	reversed, err := ReverseChangeset(original, previousState)

	s.Nil(reversed)
	s.Require().Error(err)
	runErr, isRunErr := err.(*bperrors.RunError)
	s.Require().True(isRunErr)
	s.Equal(ErrorReasonCodeMaxReverseDepthExceeded, runErr.ReasonCode)
}

func TestReverseChangesetTestSuite(t *testing.T) {
	suite.Run(t, new(ReverseChangesetTestSuite))
}
