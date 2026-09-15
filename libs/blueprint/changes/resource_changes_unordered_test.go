package changes

import (
	"context"
	"testing"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type UnorderedArrayChangesTestSuite struct {
	suite.Suite
	generator ResourceChangeGenerator
}

func (s *UnorderedArrayChangesTestSuite) SetupTest() {
	s.generator = NewDefaultResourceChangeGenerator()
}

func (s *UnorderedArrayChangesTestSuite) Test_reports_no_change_when_an_unordered_array_is_reordered() {
	changes, err := s.generator.GenerateChanges(
		context.Background(),
		s.resourceInfo(
			stringArray("subnet-a", "subnet-b", "subnet-c"),
			stringArray("subnet-c", "subnet-a", "subnet-b"),
		),
		&unorderedFieldsResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	s.Assert().Empty(
		changes.ModifiedFields,
		"the same values in a different order were reported as a change",
	)
	s.Assert().Empty(changes.NewFields)
	s.Assert().Empty(changes.RemovedFields)
}

// Ignoring order must not become ignoring the field.
func (s *UnorderedArrayChangesTestSuite) Test_reports_a_change_when_an_unordered_arrays_values_differ() {
	changes, err := s.generator.GenerateChanges(
		context.Background(),
		s.resourceInfo(
			stringArray("subnet-a", "subnet-b"),
			stringArray("subnet-a", "subnet-z"),
		),
		&unorderedFieldsResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	s.Assert().NotEmpty(
		changes.ModifiedFields,
		"a changed value was not reported because the field's order is ignored",
	)
}

func (s *UnorderedArrayChangesTestSuite) resourceInfo(
	newItems *bpcore.MappingNode,
	currentItems *bpcore.MappingNode,
) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:   "unordered-resource",
		InstanceID:   "unordered-instance",
		ResourceName: "resourceWithUnorderedArray",
		CurrentResourceState: &state.ResourceState{
			ResourceID:    "unordered-resource",
			Name:          "resourceWithUnorderedArray",
			Type:          "example/unordered",
			Status:        bpcore.ResourceStatusCreated,
			PreciseStatus: bpcore.PreciseResourceStatusCreated,
			SpecData: bpcore.MappingNodeFields(
				"subnetIds", currentItems,
			),
		},
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{Value: "example/unordered"},
			Spec: bpcore.MappingNodeFields(
				"subnetIds", newItems,
			),
		},
	}
}

type unorderedFieldsResource struct {
	internal.ExampleTaggableResource
}

func (r *unorderedFieldsResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"subnetIds": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
						IgnoreItemOrder: true,
					},
				},
			},
		},
	}, nil
}

func stringArray(values ...string) *bpcore.MappingNode {
	items := make([]*bpcore.MappingNode, 0, len(values))
	for _, value := range values {
		items = append(items, bpcore.MappingNodeFromString(value))
	}

	return &bpcore.MappingNode{Items: items}
}

func TestUnorderedArrayChangesTestSuite(t *testing.T) {
	suite.Run(t, new(UnorderedArrayChangesTestSuite))
}
