package changes

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// resourceInfoFixture5 creates a fixture for testing SortArrayByField functionality.
// The fixture contains tag arrays where the order differs between the new spec and current state,
// but the logical content (same key/value pairs) is identical.
// With SortArrayByField="key", no changes should be detected.
func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture5() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:               "test-resource-5",
		InstanceID:               "test-instance-5",
		ResourceName:             "resourceWithTags",
		CurrentResourceState:     s.resourceInfoFixture5CurrentState(),
		ResourceWithResolvedSubs: s.resourceInfoFixture5NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture5CurrentState() *state.ResourceState {
	// Tags stored in alphabetical order (as returned by GetExternalState after sorting)
	envKey := "env"
	envValue := "production"
	nameKey := "name"
	nameValue := "my-resource"
	teamKey := "team"
	teamValue := "platform"

	return &state.ResourceState{
		ResourceID:                 "test-resource-5",
		Name:                       "resourceWithTags",
		Type:                       "example/taggable",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"id": core.MappingNodeFromString("resource-id-5"),
				// Tags sorted by key: env, name, team
				"tags": {
					Items: []*core.MappingNode{
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &envKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &envValue}},
							},
						},
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &nameKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &nameValue}},
							},
						},
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &teamKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &teamValue}},
							},
						},
					},
				},
			},
		},
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture5NewResolvedResource() *provider.ResolvedResource {
	// Same tags but in different order (as user might define them in blueprint)
	// Order: team, name, env (reverse alphabetical)
	teamKey := "team"
	teamValue := "platform"
	nameKey := "name"
	nameValue := "my-resource"
	envKey := "env"
	envValue := "production"

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "example/taggable",
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				// Tags in user-defined order: team, name, env
				"tags": {
					Items: []*core.MappingNode{
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &teamKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &teamValue}},
							},
						},
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &nameKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &nameValue}},
							},
						},
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &envKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &envValue}},
							},
						},
					},
				},
			},
		},
	}
}

// resourceInfoFixture6 creates a fixture for testing SortArrayByField with actual changes.
// The tags have different orders AND one tag value has changed.
func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture6() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:               "test-resource-6",
		InstanceID:               "test-instance-6",
		ResourceName:             "resourceWithTagChanges",
		CurrentResourceState:     s.resourceInfoFixture6CurrentState(),
		ResourceWithResolvedSubs: s.resourceInfoFixture6NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture6CurrentState() *state.ResourceState {
	envKey := "env"
	envValue := "staging" // Will be changed to "production"
	nameKey := "name"
	nameValue := "my-resource"

	return &state.ResourceState{
		ResourceID:                 "test-resource-6",
		Name:                       "resourceWithTagChanges",
		Type:                       "example/taggable",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"id": core.MappingNodeFromString("resource-id-6"),
				// Tags sorted by key: env, name
				"tags": {
					Items: []*core.MappingNode{
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &envKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &envValue}},
							},
						},
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &nameKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &nameValue}},
							},
						},
					},
				},
			},
		},
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture6NewResolvedResource() *provider.ResolvedResource {
	// Tags in different order with one value changed
	nameKey := "name"
	nameValue := "my-resource"
	envKey := "env"
	envValue := "production" // Changed from "staging"

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "example/taggable",
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				// Tags in user-defined order: name, env (opposite of current state)
				"tags": {
					Items: []*core.MappingNode{
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &nameKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &nameValue}},
							},
						},
						{
							Fields: map[string]*core.MappingNode{
								"key":   {Scalar: &core.ScalarValue{StringValue: &envKey}},
								"value": {Scalar: &core.ScalarValue{StringValue: &envValue}},
							},
						},
					},
				},
			},
		},
	}
}
