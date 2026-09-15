package subengine

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ResolveAnnotationsForDeploymentTestSuite struct {
	SubstitutionResourceResolverTestSuite
}

// An annotation that could not be resolved while changes were staged has to be resolved
// again when the resource is deployed.
//
// An annotation referencing another element's value cannot be resolved before that element
// exists, so staging leaves it holding nothing. Deployment resolves what staging could not,
// and an annotation left out of that pass would be wiped. Nothing reports
// it, the annotation is present, so it reads as resolved, and the value it should carry is
// simply absent from the deployed resource.
//
// Annotations are what a provider reads to decide how to treat a resource, so one that
// silently arrives empty is a resource configured differently from what the blueprint
// asked for, with nothing said about it.
func (s *ResolveAnnotationsForDeploymentTestSuite) Test_resolves_an_annotation_left_unresolved_by_change_staging() {
	blueprint := s.specFixtureSchemas[resolveInResourcePartialAnnotationsFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		params,
	)

	// Staged before the child blueprint it references has been deployed, which is what
	// leaves the annotation with nothing in it.
	stagedResult, err := subResolver.ResolveInResource(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor: ResolveForChangeStaging,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(stagedResult.ResolvedResource.Metadata)

	stagedRegion := stagedResult.ResolvedResource.Metadata.Annotations.Fields["aws.dynamodb.region"]
	s.Require().Empty(
		core.StringValue(stagedRegion),
		"the annotation resolved while staging, so this no longer covers the case it was written for",
	)

	s.Require().NoError(s.saveChildBlueprintRegion())

	ctx := context.WithValue(
		context.Background(),
		core.BlueprintInstanceIDKey,
		testInstanceID,
	)
	deployedResult, err := subResolver.ResolveInResource(
		ctx,
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor:        ResolveForDeployment,
			PartiallyResolved: stagedResult.ResolvedResource,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(deployedResult.ResolvedResource.Metadata)

	region := deployedResult.ResolvedResource.Metadata.Annotations.Fields["aws.dynamodb.region"]
	s.Require().NotNil(region, "the annotation was dropped entirely")
	s.Assert().Equal(
		"eu-west-1",
		core.StringValue(region),
		"the annotation staging could not resolve was carried over as it was rather "+
			"than resolved against the state the deployment can see",
	)
}

// The literal annotation beside it is resolved while staging and must survive the second
// pass untouched, so the fix cannot be to re-resolve everything blindly.
func (s *ResolveAnnotationsForDeploymentTestSuite) Test_keeps_an_annotation_change_staging_resolved() {
	blueprint := s.specFixtureSchemas[resolveInResourcePartialAnnotationsFixtureName]
	spec := internal.NewBlueprintSpecMock(blueprint)
	params := resolveInResourceTestParams()
	subResolver := NewDefaultSubstitutionResolver(
		&Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		spec,
		params,
	)

	stagedResult, err := subResolver.ResolveInResource(
		context.TODO(),
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{ResolveFor: ResolveForChangeStaging},
	)
	s.Require().NoError(err)

	s.Require().NoError(s.saveChildBlueprintRegion())

	ctx := context.WithValue(
		context.Background(),
		core.BlueprintInstanceIDKey,
		testInstanceID,
	)
	deployedResult, err := subResolver.ResolveInResource(
		ctx,
		"ordersTable",
		blueprint.Resources.Values["ordersTable"],
		&ResolveResourceTargetInfo{
			ResolveFor:        ResolveForDeployment,
			PartiallyResolved: stagedResult.ResolvedResource,
		},
	)
	s.Require().NoError(err)

	trigger := deployedResult.ResolvedResource.Metadata.Annotations.Fields["aws.dynamodb.trigger"]
	s.Require().NotNil(trigger, "an annotation resolved while staging was dropped")
	s.Assert().True(core.BoolValue(trigger))
}

func (s *ResolveAnnotationsForDeploymentTestSuite) saveChildBlueprintRegion() error {
	err := s.stateContainer.Instances().Save(context.Background(), state.InstanceState{
		InstanceID: testInstanceID,
	})
	if err != nil {
		return err
	}

	childBlueprintRegion := "eu-west-1"
	err = s.stateContainer.Instances().Save(context.Background(), state.InstanceState{
		InstanceID: testChildInstanceID,
		Exports: map[string]*state.ExportState{
			"region": {
				Value: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &childBlueprintRegion},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return s.stateContainer.Children().Attach(
		context.Background(),
		testInstanceID,
		testChildInstanceID,
		"coreInfra",
	)
}

func TestResolveAnnotationsForDeploymentTestSuite(t *testing.T) {
	suite.Run(t, new(ResolveAnnotationsForDeploymentTestSuite))
}
