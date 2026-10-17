package changes

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// A change set is the surface a practitioner reads before approving a deployment, so a secret
// shown there is shown at the moment someone is looking. Schema declarations alone miss the
// field a secret was interpolated into, which is the case the resolver records.
type ResourceChangeSensitivityTestSuite struct {
	generator *defaultResourceChangeGenerator
	suite.Suite
}

func (s *ResourceChangeSensitivityTestSuite) SetupSuite() {
	s.generator = &defaultResourceChangeGenerator{}
}

func (s *ResourceChangeSensitivityTestSuite) Test_marks_a_field_a_secret_was_interpolated_into() {
	modified := s.modifiedFields([]string{"spec.tableName"})

	s.Assert().True(modified["spec.tableName"].Sensitive)
	s.Assert().False(
		modified["spec.region"].Sensitive,
		"a field with no recorded path was marked",
	)
}

func (s *ResourceChangeSensitivityTestSuite) Test_leaves_every_field_unmarked_when_nothing_was_recorded() {
	modified := s.modifiedFields(nil)

	s.Assert().False(modified["spec.tableName"].Sensitive)
	s.Assert().False(modified["spec.region"].Sensitive)
}

// A recorded path on a container covers the leaves below it, the same way a schema
// declaring a container sensitive does.
func (s *ResourceChangeSensitivityTestSuite) Test_a_recorded_container_path_covers_the_leaves_below_it() {
	modified := s.modifiedFields([]string{"spec.replicaRegions"})

	s.Assert().True(modified["spec.replicaRegions[0]"].Sensitive)
}

// Coverage is a path test, not a string prefix test.
func (s *ResourceChangeSensitivityTestSuite) Test_a_recorded_path_does_not_cover_a_name_that_extends_it() {
	modified := s.modifiedFields([]string{"spec.tableName"})

	s.Assert().False(modified["spec.tableNameSuffix"].Sensitive)
}

// An array the schema declares order-insensitive is sorted before comparison, so the index
// a path records is not the index the change set reports. The mark has to move with its
// item, left alone it lands on whichever item now sits at that index, hiding a value that
// is not secret and showing one that is.
func (s *ResourceChangeSensitivityTestSuite) Test_a_mark_moves_with_its_item_when_the_array_is_sorted() {
	modified := s.sortedArrayModifiedFields([]string{"spec.tags[1]"})

	// "alpha-secret" is at source index 1 and sorts to position 0.
	s.Assert().True(modified["spec.tags[0]"].Sensitive)
	s.Assert().False(
		modified["spec.tags[1]"].Sensitive,
		"the mark stayed on the index it recorded rather than following its item",
	)
}

func (s *ResourceChangeSensitivityTestSuite) Test_a_mark_reaches_a_path_nested_below_two_containers() {
	modified := s.sortedArrayModifiedFields([]string{"spec.config.env.PASSWORD"})

	s.Assert().True(modified["spec.config.env.PASSWORD"].Sensitive)
}

// The new-field branch builds its own FieldChange, so it needs marking of its own.
func (s *ResourceChangeSensitivityTestSuite) Test_marks_a_new_field_a_secret_reached() {
	resourceChanges := s.generateSortedArrayChanges([]string{"spec.config.env.REGION"})

	var region *provider.FieldChange
	for _, fieldChange := range resourceChanges.NewFields {
		if fieldChange.FieldPath == "spec.config.env.REGION" {
			region = &fieldChange
		}
	}
	s.Require().NotNil(region, "spec.config.env.REGION is not among the new fields")
	s.Assert().True(region.Sensitive)
}

func (s *ResourceChangeSensitivityTestSuite) Test_marks_metadata_a_secret_reached() {
	modified := s.metadataModifiedFields([]string{"metadata.custom.auditKey"})

	s.Assert().True(modified["metadata.custom.auditKey"].Sensitive)
	s.Assert().False(
		modified["metadata.custom.region"].Sensitive,
		"a sibling of the field holding the secret was marked",
	)
	s.Assert().False(modified["metadata.displayName"].Sensitive)
}

func (s *ResourceChangeSensitivityTestSuite) Test_marks_a_display_name_a_secret_reached() {
	modified := s.metadataModifiedFields([]string{"metadata.displayName"})

	s.Assert().True(modified["metadata.displayName"].Sensitive)
	s.Assert().False(modified["metadata.custom.auditKey"].Sensitive)
}

func (s *ResourceChangeSensitivityTestSuite) Test_a_secret_annotation_marks_that_entry_alone() {
	modified := s.metadataModifiedFields([]string{`metadata.annotations["example.v1"]`})

	s.Assert().True(modified[`metadata.annotations["example.v1"]`].Sensitive)
	s.Assert().False(
		modified[`metadata.annotations["example.v2"]`].Sensitive,
		"a sibling annotation was marked",
	)
}

func (s *ResourceChangeSensitivityTestSuite) Test_a_recorded_annotation_map_still_covers_its_entries() {
	modified := s.metadataModifiedFields([]string{"metadata.annotations"})

	s.Assert().True(modified[`metadata.annotations["example.v1"]`].Sensitive)
}

func (s *ResourceChangeSensitivityTestSuite) metadataChanges(
	sensitiveMetadataPaths []string,
) *provider.Changes {
	resourceChanges, err := s.generator.GenerateChanges(
		context.Background(),
		metadataResourceInfo(sensitiveMetadataPaths),
		&internal.DynamoDBTableResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	return resourceChanges
}

func (s *ResourceChangeSensitivityTestSuite) Test_marks_new_metadata_a_secret_reached() {
	newFields := fieldChangesByPath(&s.Suite,
		s.metadataChanges([]string{"metadata.custom.addedKey"}).NewFields,
		"metadata.custom.addedKey",
	)

	s.Assert().True(newFields["metadata.custom.addedKey"].Sensitive)
}

func (s *ResourceChangeSensitivityTestSuite) Test_marks_a_retyped_metadata_field_a_secret_reached() {
	modified := fieldChangesByPath(&s.Suite,
		s.metadataChanges([]string{"metadata.custom.retypedKey"}).ModifiedFields,
		"metadata.custom.retypedKey",
	)

	s.Assert().True(modified["metadata.custom.retypedKey"].Sensitive)
}

// A shallow change carries its whole subtree, so a secret recorded inside it is inside what
// the change prints. The reported path is the container, which no recorded descendant path
// matches on the at-or-above test the descending walks use.
func (s *ResourceChangeSensitivityTestSuite) Test_marks_a_shallow_change_holding_a_secret_below_it() {
	modified := fieldChangesByPath(&s.Suite,
		s.metadataChanges([]string{"metadata.custom.retypedKey.nested"}).ModifiedFields,
		"metadata.custom.retypedKey",
	)

	s.Assert().True(modified["metadata.custom.retypedKey"].Sensitive)
	s.Assert().False(
		modified["metadata.custom.auditKey"].Sensitive,
		"an unrelated field was marked",
	)
}

func (s *ResourceChangeSensitivityTestSuite) metadataModifiedFields(
	sensitiveMetadataPaths []string,
) map[string]provider.FieldChange {
	return fieldChangesByPath(&s.Suite,
		s.metadataChanges(sensitiveMetadataPaths).ModifiedFields,
		"metadata.displayName",
		"metadata.custom.auditKey",
		"metadata.custom.region",
		`metadata.annotations["example.v1"]`,
		`metadata.annotations["example.v2"]`,
	)
}

func metadataResourceInfo(sensitiveMetadataPaths []string) *provider.ResourceInfo {
	resourceInfo := sensitivityResourceInfo(nil)
	resourceInfo.ResourceWithResolvedSubs.SensitiveMetadataPaths = sensitiveMetadataPaths
	resourceInfo.ResourceWithResolvedSubs.Metadata = &provider.ResolvedResourceMetadata{
		DisplayName: core.MappingNodeFromString("Orders Table"),
		Annotations: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"example.v1": core.MappingNodeFromString("new-annotation"),
				"example.v2": core.MappingNodeFromString("new-sibling"),
			},
		},
		Custom: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"auditKey": core.MappingNodeFromString("audit-super-secret"),
				"region":   core.MappingNodeFromString("eu-west-2"),
				// Absent from the current state, so this reaches the new-field branch.
				"addedKey": core.MappingNodeFromString("added-super-secret"),
				// A scalar in state and an object now, so the walk falls through to the
				// shallow comparison it makes when the two types disagree.
				"retypedKey": {
					Fields: map[string]*core.MappingNode{
						"nested": core.MappingNodeFromString("retyped-super-secret"),
					},
				},
			},
		},
	}
	resourceInfo.CurrentResourceState.Metadata = &state.ResourceMetadataState{
		DisplayName: "Previous Table",
		Annotations: map[string]*core.MappingNode{
			"example.v1": core.MappingNodeFromString("previous-annotation"),
			"example.v2": core.MappingNodeFromString("previous-sibling"),
		},
		Custom: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"auditKey":   core.MappingNodeFromString("previous-audit"),
				"region":     core.MappingNodeFromString("eu-west-1"),
				"retypedKey": core.MappingNodeFromString("previously-a-scalar"),
			},
		},
	}

	return resourceInfo
}

func (s *ResourceChangeSensitivityTestSuite) generateSortedArrayChanges(
	sensitiveSpecPaths []string,
) *provider.Changes {
	resourceChanges, err := s.generator.GenerateChanges(
		context.Background(),
		sortedArrayResourceInfo(sensitiveSpecPaths),
		&sortedTagsResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	return resourceChanges
}

func (s *ResourceChangeSensitivityTestSuite) sortedArrayModifiedFields(
	sensitiveSpecPaths []string,
) map[string]provider.FieldChange {
	resourceChanges := s.generateSortedArrayChanges(sensitiveSpecPaths)

	byPath := map[string]provider.FieldChange{}
	for _, fieldChange := range resourceChanges.ModifiedFields {
		byPath[fieldChange.FieldPath] = fieldChange
	}

	for _, path := range []string{"spec.tags[0]", "spec.tags[1]", "spec.config.env.PASSWORD"} {
		_, found := byPath[path]
		s.Require().True(found, "%s is not among the modified fields", path)
	}

	return byPath
}

// Declares its tags array order-insensitive, so change staging sorts it before comparing.
// Defined here rather than on a shared fixture resource, which several other suites snapshot.
type sortedTagsResource struct {
	*internal.DynamoDBTableResource
}

func (r *sortedTagsResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"tags": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
						IgnoreItemOrder: true,
					},
					// Two levels below the spec root, so propagation through the object
					// and map walks is load-bearing rather than inherited from the root.
					"config": {
						Type: provider.ResourceDefinitionsSchemaTypeObject,
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"env": {
								Type: provider.ResourceDefinitionsSchemaTypeMap,
								MapValues: &provider.ResourceDefinitionsSchema{
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func sortedArrayResourceInfo(sensitiveSpecPaths []string) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:   "test-resource-2",
		InstanceID:   "test-instance-1",
		ResourceName: "taggedResource",
		CurrentResourceState: &state.ResourceState{
			ResourceID: "test-resource-2",
			Name:       "taggedResource",
			Type:       "example/sortedTags",
			Status:     core.ResourceStatusCreated,
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"tags": core.MappingNodeFromStringSlice(
						[]string{"previous-one", "previous-two"},
					),
					"config": {
						Fields: map[string]*core.MappingNode{
							"env": {
								Fields: map[string]*core.MappingNode{
									"PASSWORD": core.MappingNodeFromString("previous"),
								},
							},
						},
					},
				},
			},
		},
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{Value: "example/sortedTags"},
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					// Sorting reverses these, so the secret moves from index 1 to index 0.
					"tags": core.MappingNodeFromStringSlice(
						[]string{"zeta-plain", "alpha-secret"},
					),
					"config": {
						Fields: map[string]*core.MappingNode{
							"env": {
								Fields: map[string]*core.MappingNode{
									"PASSWORD": core.MappingNodeFromString("new-secret"),
									// Absent from the current state, so this reaches the
									// new-field branch rather than the modified one.
									"REGION": core.MappingNodeFromString("eu-west-2"),
								},
							},
						},
					},
				},
			},
			SensitiveSpecPaths: sensitiveSpecPaths,
		},
	}
}

func (s *ResourceChangeSensitivityTestSuite) modifiedFields(
	sensitiveSpecPaths []string,
) map[string]provider.FieldChange {
	resourceChanges, err := s.generator.GenerateChanges(
		context.Background(),
		sensitivityResourceInfo(sensitiveSpecPaths),
		&internal.DynamoDBTableResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	byPath := map[string]provider.FieldChange{}
	for _, fieldChange := range resourceChanges.ModifiedFields {
		byPath[fieldChange.FieldPath] = fieldChange
	}

	// Every field the assertions reach has to have actually changed, or a missing entry
	// would read as "not sensitive" and pass regardless of what the code does.
	for _, path := range []string{
		"spec.tableName",
		"spec.tableNameSuffix",
		"spec.region",
		"spec.replicaRegions[0]",
	} {
		_, found := byPath[path]
		s.Require().True(found, "%s is not among the modified fields", path)
	}

	return byPath
}

func sensitivityResourceInfo(sensitiveSpecPaths []string) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:   "test-resource-1",
		InstanceID:   "test-instance-1",
		ResourceName: "ordersTable",
		CurrentResourceState: &state.ResourceState{
			ResourceID: "test-resource-1",
			Name:       "ordersTable",
			Type:       "aws/dynamodb/table",
			Status:     core.ResourceStatusCreated,
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"tableName":       core.MappingNodeFromString("previous-table"),
					"tableNameSuffix": core.MappingNodeFromString("previous-suffix"),
					"region":          core.MappingNodeFromString("eu-west-1"),
					"replicaRegions": core.MappingNodeFromStringSlice(
						[]string{"us-east-1"},
					),
				},
			},
		},
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{Value: "aws/dynamodb/table"},
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"tableName":       core.MappingNodeFromString("orders-super-secret"),
					"tableNameSuffix": core.MappingNodeFromString("new-suffix"),
					"region":          core.MappingNodeFromString("eu-west-2"),
					"replicaRegions": core.MappingNodeFromStringSlice(
						[]string{"eu-west-3"},
					),
				},
			},
			SensitiveSpecPaths: sensitiveSpecPaths,
		},
	}
}

func TestResourceChangeSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceChangeSensitivityTestSuite))
}
