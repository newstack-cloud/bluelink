package changes

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture3() *provider.ResourceInfo {

	return &provider.ResourceInfo{
		ResourceID:               "test-resource-1",
		InstanceID:               "test-instance-1",
		ResourceName:             "complexResource",
		CurrentResourceState:     s.resourceInfoFixture3CurrentState(),
		ResourceWithResolvedSubs: s.resourceInfoFixture3NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture3CurrentState() *state.ResourceState {
	itemID := "test-item-1"
	currentEndpoint1 := "http://example.com/1"
	currentPrimaryPort := 8080
	currentIpv4Enabled := true
	currentSpecMetadataValue1 := "value1"
	currentSpecMetadataValue2 := "value2"
	currentMetadataCustomURL := "http://example.com"
	currentMetadataProtocol1 := "https"
	currentMetadataProtocol2 := "wss"
	otherItemValue := "other-item-value"
	vendorTag1 := "vendor-tag-1"
	vendorTag2 := "vendor-tag-2"
	vendorTag3 := "vendor-tag-3"
	localTag1 := "local-tag-1"
	localTag2 := "local-tag-2"
	firstAnnotationValue := "first-annotation-value"
	secondAnnotationValue := "second-annotation-value"
	originalAnnotationValue := "original-annotation-value"

	return &state.ResourceState{
		ResourceID:                 "test-resource-1",
		Name:                       "complexResource",
		Type:                       "example/complex",
		Status:                     core.ResourceStatusCreated,
		PreciseStatus:              core.PreciseResourceStatusCreated,
		LastDeployedTimestamp:      1732969676,
		LastDeployAttemptTimestamp: 1732969676,
		SpecData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"id": {
					Scalar: &core.ScalarValue{
						StringValue: &itemID,
					},
				},
				"itemConfig": {
					Fields: map[string]*core.MappingNode{
						"endpoints": {
							Items: []*core.MappingNode{
								{
									Scalar: &core.ScalarValue{
										StringValue: &currentEndpoint1,
									},
								},
							},
						},
						"primaryPort": {
							Scalar: &core.ScalarValue{
								IntValue: &currentPrimaryPort,
							},
						},
						"ipv4": {
							Scalar: &core.ScalarValue{
								BoolValue: &currentIpv4Enabled,
							},
						},
						"metadata": {
							Fields: map[string]*core.MappingNode{
								"value1": {
									Scalar: &core.ScalarValue{
										StringValue: &currentSpecMetadataValue1,
									},
								},
								"value2": {
									Scalar: &core.ScalarValue{
										StringValue: &currentSpecMetadataValue2,
									},
								},
							},
						},
					},
				},
				"otherItemConfig": {
					Scalar: &core.ScalarValue{
						StringValue: &otherItemValue,
					},
				},
				"vendorTags": {
					Items: []*core.MappingNode{
						{
							Scalar: &core.ScalarValue{
								StringValue: &vendorTag1,
							},
						},
						{
							Scalar: &core.ScalarValue{
								StringValue: &vendorTag2,
							},
						},
						{
							Scalar: &core.ScalarValue{
								StringValue: &vendorTag3,
							},
						},
					},
				},
			},
		},
		Metadata: &state.ResourceMetadataState{
			DisplayName: "Test Complex Resource",
			Annotations: map[string]*core.MappingNode{
				"test.annotation.v1": {
					Scalar: &core.ScalarValue{
						StringValue: &firstAnnotationValue,
					},
				},
				"test.annotation.v2": {
					Scalar: &core.ScalarValue{
						StringValue: &secondAnnotationValue,
					},
				},
				"test.annotation.original-v3": {
					Scalar: &core.ScalarValue{
						StringValue: &originalAnnotationValue,
					},
				},
			},
			Labels: map[string]string{
				"app":   "test-app-v1",
				"squad": "portal-squad",
			},
			Custom: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"url": {
						Scalar: &core.ScalarValue{
							StringValue: &currentMetadataCustomURL,
						},
					},
					"protocol": {
						Items: []*core.MappingNode{
							{
								Scalar: &core.ScalarValue{
									StringValue: &currentMetadataProtocol1,
								},
							},
							{
								Scalar: &core.ScalarValue{
									StringValue: &currentMetadataProtocol2,
								},
							},
						},
					},
					"localTags": {
						Items: []*core.MappingNode{
							{
								Scalar: &core.ScalarValue{
									StringValue: &localTag1,
								},
							},
							{
								Scalar: &core.ScalarValue{
									StringValue: &localTag2,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture3NewResolvedResource() *provider.ResolvedResource {
	newDisplayName := "Test Complex Resource Updated"
	firstAnnotationValue := "first-annotation-value"
	secondAnnotationValue := "second-annotation-value"
	thirdAnnotationValue := "third-annotation-value"
	newEndpoint1 := "http://example.com/new/1"
	newEndpoint2 := "http://example.com/new/2"
	newEndpoint3 := "http://example.com/new/3"
	newPrimaryPort := 8081
	newIpv4Enabled := false
	newSpecMetadataValue1 := "new-value1"
	newScore := 1.309
	newMetadataProtocol := "https"
	otherItemValue := "other-item-value"
	vendorTag := "vendor-tag-1"
	localTag := "local-tag-1"

	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "example/complex",
		},
		Metadata: &provider.ResolvedResourceMetadata{
			DisplayName: &core.MappingNode{
				Scalar: &core.ScalarValue{
					StringValue: &newDisplayName,
				},
			},
			Annotations: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"test.annotation.v1": {
						Scalar: &core.ScalarValue{
							StringValue: &firstAnnotationValue,
						},
					},
					"test.annotation.v2": {
						Scalar: &core.ScalarValue{
							StringValue: &secondAnnotationValue,
						},
					},
					"test.annotation.v3": {
						Scalar: &core.ScalarValue{
							StringValue: &thirdAnnotationValue,
						},
					},
				},
			},
			Labels: &schema.StringMap{
				Values: map[string]string{
					"app": "test-app-v2",
					"env": "production",
				},
			},
			Custom: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"protocol": {
						Scalar: &core.ScalarValue{
							StringValue: &newMetadataProtocol,
						},
					},
					"localTags": {
						Items: []*core.MappingNode{
							{
								Scalar: &core.ScalarValue{
									StringValue: &localTag,
								},
							},
						},
					},
					// the resource change stager is expected to stop
					// traversing nested structures at the max traversal depth
					// (core.MappingNodeMaxTraverseDepth)
					// No entries should be added to the changes for this field.
					"deeplyNested": buildDeeplyNestedMappingNode(250, "nested"),
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"itemConfig": {
					Fields: map[string]*core.MappingNode{
						"endpoints": {
							Items: []*core.MappingNode{
								{
									Scalar: &core.ScalarValue{
										StringValue: &newEndpoint1,
									},
								},
								{
									Scalar: &core.ScalarValue{
										StringValue: &newEndpoint2,
									},
								},
								{
									Scalar: &core.ScalarValue{
										StringValue: &newEndpoint3,
									},
								},
								// To be resolved on deployment
								(*core.MappingNode)(nil),
							},
						},
						"primaryPort": {
							Scalar: &core.ScalarValue{
								IntValue: &newPrimaryPort,
							},
						},
						"ipv4": {
							Scalar: &core.ScalarValue{
								BoolValue: &newIpv4Enabled,
							},
						},
						"score": {
							Scalar: &core.ScalarValue{
								FloatValue: &newScore,
							},
						},
						// 25 levels deep exceeds core.MappingNodeMaxTraverseDepth
						// so the resource change stager should not traverse the full structure
						// for the "deepConfig" field.
						// No entries should be added to the changes for this field.
						"deepConfig": buildDeeplyNestedMappingNode(25, "item"),
						"metadata": {
							Fields: map[string]*core.MappingNode{
								"value1": {
									Scalar: &core.ScalarValue{
										StringValue: &newSpecMetadataValue1,
									},
								},
								// "value2" key/value pair has been removed.
							},
						},
					},
				},
				"otherItemConfig": {
					Scalar: &core.ScalarValue{
						StringValue: &otherItemValue,
					},
				},
				"vendorTags": {
					Items: []*core.MappingNode{
						{
							Scalar: &core.ScalarValue{
								StringValue: &vendorTag,
							},
						},
					},
				},
			},
		},
	}
}

func buildDeeplyNestedMappingNode(depth int, fieldPrefix string) *core.MappingNode {
	if depth == 0 {
		return nil
	}

	fieldName := fmt.Sprintf("%s%d", fieldPrefix, depth)
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			fieldName: buildDeeplyNestedMappingNode(depth-1, fieldPrefix),
		},
	}
}
