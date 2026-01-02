package changes

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// resourceInfoFixture7 creates a fixture for testing nullable fields with defaults.
// Scenario: User didn't provide delaySeconds or maximumMessageSize (nil in persisted state),
// but GetExternalState returns the default values (0 and 262144).
// With the nullable+default handling, no changes should be detected.
func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture7() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:               "test-resource-7",
		InstanceID:               "test-instance-7",
		ResourceName:             "queueWithNullableDefaults",
		CurrentResourceState:     s.resourceInfoFixture7CurrentState(),
		ResourceWithResolvedSubs: s.resourceInfoFixture7NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture7CurrentState() *state.ResourceState {
	queueName := "my-queue"

	return &state.ResourceState{
		ResourceID:                 "test-resource-7",
		Name:                       "queueWithNullableDefaults",
		Type:                       "example/nullable",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"queueName": {Scalar: &core.ScalarValue{StringValue: &queueName}},
				// delaySeconds and maximumMessageSize are nil (user didn't provide them)
				// because they are nullable and defaults were not populated
			},
		},
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture7NewResolvedResource() *provider.ResolvedResource {
	// Simulates GetExternalState returning the default values
	queueName := "my-queue"
	delaySeconds := 0       // Default value
	maxMessageSize := 262144 // Default value

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "example/nullable",
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"queueName":          {Scalar: &core.ScalarValue{StringValue: &queueName}},
				"delaySeconds":       {Scalar: &core.ScalarValue{IntValue: &delaySeconds}},
				"maximumMessageSize": {Scalar: &core.ScalarValue{IntValue: &maxMessageSize}},
			},
		},
	}
}

// resourceInfoFixture8 creates a fixture for testing nullable fields with non-default values.
// Scenario: User didn't provide delaySeconds (nil in persisted state),
// but GetExternalState returns a non-default value (30 instead of 0).
// This IS a real drift and should be detected.
func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture8() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:               "test-resource-8",
		InstanceID:               "test-instance-8",
		ResourceName:             "queueWithDriftedValue",
		CurrentResourceState:     s.resourceInfoFixture8CurrentState(),
		ResourceWithResolvedSubs: s.resourceInfoFixture8NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture8CurrentState() *state.ResourceState {
	queueName := "my-queue"

	return &state.ResourceState{
		ResourceID:                 "test-resource-8",
		Name:                       "queueWithDriftedValue",
		Type:                       "example/nullable",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"queueName": {Scalar: &core.ScalarValue{StringValue: &queueName}},
				// delaySeconds is nil (user didn't provide it)
			},
		},
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture8NewResolvedResource() *provider.ResolvedResource {
	// Simulates GetExternalState returning a NON-default value (real drift)
	queueName := "my-queue"
	delaySeconds := 30 // NOT the default (0), this is drift

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "example/nullable",
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"queueName":    {Scalar: &core.ScalarValue{StringValue: &queueName}},
				"delaySeconds": {Scalar: &core.ScalarValue{IntValue: &delaySeconds}},
			},
		},
	}
}

// resourceInfoFixture9 creates a fixture for testing nullable fields where user
// explicitly set a value that differs from external state.
// Scenario: User provided delaySeconds=10, but external state shows delaySeconds=30.
// This IS a real change and should be detected.
func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture9() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:               "test-resource-9",
		InstanceID:               "test-instance-9",
		ResourceName:             "queueWithExplicitValue",
		CurrentResourceState:     s.resourceInfoFixture9CurrentState(),
		ResourceWithResolvedSubs: s.resourceInfoFixture9NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture9CurrentState() *state.ResourceState {
	queueName := "my-queue"
	delaySeconds := 10 // User explicitly set this value

	return &state.ResourceState{
		ResourceID:                 "test-resource-9",
		Name:                       "queueWithExplicitValue",
		Type:                       "example/nullable",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"queueName":    {Scalar: &core.ScalarValue{StringValue: &queueName}},
				"delaySeconds": {Scalar: &core.ScalarValue{IntValue: &delaySeconds}},
			},
		},
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture9NewResolvedResource() *provider.ResolvedResource {
	// External state shows different value than what user configured
	queueName := "my-queue"
	delaySeconds := 30 // Different from persisted value (10)

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "example/nullable",
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"queueName":    {Scalar: &core.ScalarValue{StringValue: &queueName}},
				"delaySeconds": {Scalar: &core.ScalarValue{IntValue: &delaySeconds}},
			},
		},
	}
}
