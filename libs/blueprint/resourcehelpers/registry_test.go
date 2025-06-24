package resourcehelpers

import (
	"context"
	"slices"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	. "gopkg.in/check.v1"
)

type RegistryTestSuite struct {
	resourceRegistry Registry
	testResource     *testExampleResource
}

var _ = Suite(&RegistryTestSuite{})

func (s *RegistryTestSuite) SetUpTest(c *C) {
	testRes := newTestExampleResource()

	providers := map[string]provider.Provider{
		"test": &testProvider{
			resources: map[string]provider.Resource{
				"test/exampleResource": testRes,
			},
			namespace: "test",
		},
	}

	testAbstractRes := newTestExampleAbstractResource()

	transformers := map[string]transform.SpecTransformer{
		"celerity-test": &testSpecTransformer{
			abstractResources: map[string]transform.AbstractResource{
				"test/exampleAbstractResource": testAbstractRes,
			},
		},
	}

	s.testResource = testRes.(*testExampleResource)
	s.resourceRegistry = NewRegistry(
		providers,
		transformers,
		time.Millisecond,
		/* params */ nil,
	)
}

func (s *RegistryTestSuite) Test_get_spec_definition(c *C) {
	output, err := s.resourceRegistry.GetSpecDefinition(
		context.TODO(),
		"test/exampleResource",
		&provider.ResourceGetSpecDefinitionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.SpecDefinition, DeepEquals, s.testResource.definition)

	// Second time should be cached and produce the same result.
	output, err = s.resourceRegistry.GetSpecDefinition(
		context.TODO(),
		"test/exampleResource",
		&provider.ResourceGetSpecDefinitionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.SpecDefinition, DeepEquals, s.testResource.definition)
}

func (s *RegistryTestSuite) Test_has_resource_type(c *C) {
	hasResourceType, err := s.resourceRegistry.HasResourceType(context.TODO(), "test/exampleResource")
	c.Assert(err, IsNil)
	c.Assert(hasResourceType, Equals, true)

	hasResourceType, err = s.resourceRegistry.HasResourceType(context.TODO(), "test/otherResource")
	c.Assert(err, IsNil)
	c.Assert(hasResourceType, Equals, false)
}

func (s *RegistryTestSuite) Test_get_type_description(c *C) {
	output, err := s.resourceRegistry.GetTypeDescription(
		context.TODO(),
		"test/exampleResource",
		&provider.ResourceGetTypeDescriptionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.MarkdownDescription, Equals, s.testResource.markdownDescription)
	c.Assert(output.PlainTextDescription, Equals, s.testResource.plainTextDescription)
}

func (s *RegistryTestSuite) Test_list_resource_types(c *C) {
	resourceTypes, err := s.resourceRegistry.ListResourceTypes(
		context.TODO(),
	)
	c.Assert(err, IsNil)

	containsTestExampleResource := slices.Contains(
		resourceTypes,
		"test/exampleResource",
	)
	c.Assert(containsTestExampleResource, Equals, true)

	containsTestExampleAbstractResource := slices.Contains(
		resourceTypes,
		"test/exampleAbstractResource",
	)
	c.Assert(containsTestExampleAbstractResource, Equals, true)

	// Second time should be cached and produce the same result.
	resourceTypesCached, err := s.resourceRegistry.ListResourceTypes(
		context.TODO(),
	)
	c.Assert(err, IsNil)

	containsCachedTestExampleResource := slices.Contains(
		resourceTypesCached,
		"test/exampleResource",
	)
	c.Assert(containsCachedTestExampleResource, Equals, true)

	containsCachedTestExampleAbstractResource := slices.Contains(
		resourceTypesCached,
		"test/exampleAbstractResource",
	)
	c.Assert(containsCachedTestExampleAbstractResource, Equals, true)
}

func (s *RegistryTestSuite) Test_deploy_resource(c *C) {
	deployInput := &provider.ResourceDeployInput{
		InstanceID: "test-blueprint-id",
		ResourceID: "test-resource-id",
		Changes: &provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				ResourceID:   "test-resource-id",
				ResourceName: "testResource",
				InstanceID:   "test-blueprint-id",
				ResourceWithResolvedSubs: &provider.ResolvedResource{
					Metadata: &provider.ResolvedResourceMetadata{
						DisplayName: core.MappingNodeFromString("Test Example Resource"),
						Annotations: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"annotation.v1": core.MappingNodeFromString("annotationValue"),
							},
						},
						Labels: &schema.StringMap{
							Values: map[string]string{"key": "value"},
						},
					},
				},
			},
			ComputedFields: []string{"spec.id"},
		},
	}

	output, err := s.resourceRegistry.Deploy(
		context.TODO(),
		"test/exampleResource",
		&provider.ResourceDeployServiceInput{
			DeployInput:     deployInput,
			WaitUntilStable: true,
		},
	)
	c.Assert(err, IsNil)
	c.Assert(output, DeepEquals, &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString("test-example-resource-item-id-1"),
		},
	})
}

func (s *RegistryTestSuite) Test_produces_error_for_missing_provider(c *C) {
	_, err := s.resourceRegistry.HasResourceType(context.TODO(), "otherProvider/otherResource")
	c.Assert(err, NotNil)
	runErr, isRunErr := err.(*errors.RunError)
	c.Assert(isRunErr, Equals, true)
	c.Assert(runErr.ReasonCode, Equals, provider.ErrorReasonCodeItemTypeProviderNotFound)
	c.Assert(runErr.Error(), Equals, "run error: run failed as the provider with namespace \"otherProvider\" "+
		"was not found for resource type \"otherProvider/otherResource\"")
}
