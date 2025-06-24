package changes

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture4() *provider.ResourceInfo {

	return &provider.ResourceInfo{
		ResourceID:           "test-resource-4",
		InstanceID:           "test-instance-1",
		ResourceName:         "complexResource",
		CurrentResourceState: s.resourceInfoFixture4CurrentState(),
		// Reuse the example complex resource as the new spec for the resource.
		ResourceWithResolvedSubs: s.resourceInfoFixture1NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture4CurrentState() *state.ResourceState {
	return &state.ResourceState{
		ResourceID: "test-resource-1",
		Name:       "complexResource",
		// Resource type is being updated from "example/old-complex" to "example/complex"
		Type:                       "example/old-complex",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData:                   &core.MappingNode{},
		Metadata:                   &state.ResourceMetadataState{},
	}
}
