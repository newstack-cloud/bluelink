package changes

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture2() *provider.ResourceInfo {

	return &provider.ResourceInfo{
		ResourceID:               "",
		InstanceID:               "test-instance-1",
		ResourceName:             "complexResource",
		CurrentResourceState:     (*state.ResourceState)(nil),
		ResourceWithResolvedSubs: s.resourceInfoFixture2NewResolvedResource(),
	}
}

func (s *ResourceChangeGeneratorTestSuite) resourceInfoFixture2NewResolvedResource() *provider.ResolvedResource {
	newDisplayName := "Test Complex Resource Updated"
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
					// To be resolved on deployment
					"test.annotation.v1": (*core.MappingNode)(nil),
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
					// To be resolved on deployment
					"url": (*core.MappingNode)(nil),
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
