package container

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/linkhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

const storedContributionBoundary = "boundary-from-link-data"

type MergedContributionsStoredTestSuite struct {
	suite.Suite
}

// A link that records what it contributes through its data rather than by producing
// contributions must still have those contributions composed.
//
// Contributions reach a resource by two routes. A link can produce them, or it can write
// its resources itself and record where each value lives through resource data mappings.
// The merged update supersedes a link's stored contributions with what it produced,
// on the basis that a link which has just run has reported everything it has to contribute.
// That holds only for a link that produces contributions. A link taking the other route
// produces none, so superseding replaces everything it contributes with nothing.
//
// The consequence is worse than a field going missing. The merged update states the
// resource's whole desired spec, so a contribution dropped from it is not absent, it is
// withdrawn as the update would instruct the provider to remove configuration the link had
// already written to the live resource.
func (s *MergedContributionsStoredTestSuite) Test_composes_contributions_a_link_recorded_rather_than_produced() {
	stateContainer := memstate.NewMemoryStateContainer()
	role := &boundaryRoleResource{}

	s.deployWithMappingOnlyLink(stateContainer, role)

	spec := role.specDeployedWithContributions()
	s.Require().NotNil(
		spec,
		"the role was never updated to carry what its links contribute",
	)

	boundary, err := core.GetPathValue("$.boundaryId", spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(
		boundary,
		"the contribution the link recorded through its data was dropped from the "+
			"merged update, which withdraws it from the live resource",
	)
	s.Assert().Equal(storedContributionBoundary, core.StringValue(boundary))
}

func (s *MergedContributionsStoredTestSuite) deployWithMappingOnlyLink(
	stateContainer state.Container,
	role *boundaryRoleResource,
) {
	awsProvider := &internal.ProviderMock{
		NamespaceValue: "aws",
		Resources: map[string]provider.Resource{
			lambdaFunctionResourceType: &roleLinkingLambdaResource{
				Resource: &internal.LambdaFunctionResource{
					CurrentDestroyAttempts:                   map[string]int{},
					CurrentDeployAttemps:                     map[string]int{},
					CurrentGetExternalStateAttemps:           map[string]int{},
					StabiliseResourceIDs:                     map[string]*internal.StubResourceStabilisationConfig{},
					AlwaysStabilise:                          true,
					CurrentStabiliseCalls:                    map[string]int{},
					FallbackToStateContainerForExternalState: true,
					StateContainer:                           stateContainer,
				},
			},
			iamRoleResourceType: role,
		},
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/iam/role": &mappingOnlyRoleLink{},
		},
		CustomVariableTypes: map[string]provider.CustomVariableType{},
		DataSources:         map[string]provider.DataSource{},
	}

	loader := NewDefaultLoader(
		map[string]provider.Provider{"aws": awsProvider},
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderResourceStabilityPollingConfig(&ResourceStabilityPollingConfig{
			PollingInterval: 10 * time.Millisecond,
			PollingTimeout:  1 * time.Second,
		}),
		WithLoaderLogger(core.NewNopLogger()),
	)

	params := newLinkProjectionParams()
	container, err := loader.Load(context.Background(), mergedContributionsBlueprint, params)
	s.Require().NoError(err)

	stagingChannels := createChangeStagingChannels()
	err = container.StageChanges(
		context.Background(),
		&StageChangesInput{},
		stagingChannels,
		params,
	)
	s.Require().NoError(err)

	stagedChanges, err := consumeStagedChangesForTest(stagingChannels)
	s.Require().NoError(err)

	deployChannels := CreateDeployChannels()
	err = container.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "MergedContributionsStoredInstance",
			Changes:      stagedChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	s.Require().NoError(err)

	collectDeployMessages(s.T(), deployChannels)
}

// A link that writes the resources it relates itself and records where each value it wrote
// lives, producing no contributions of its own. This is the shape of a link that predates
// the contributions mechanism, and of one whose writes go through a service SDK rather
// than through the resource's own spec.
type mappingOnlyRoleLink struct {
	testLambdaIAMRoleLink
}

func (l *mappingOnlyRoleLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	roleName := linkhelpers.GetResourceNameFromChanges(input.ResourceBChanges)

	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			ResourceDataMappings: map[string]string{
				fmt.Sprintf("%s::spec.boundaryId", roleName): "boundaryId",
			},
		},
	}, nil
}

func (l *mappingOnlyRoleLink) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	return &provider.LinkProduceResourceContributionsOutput{}, nil
}

func (l *mappingOnlyRoleLink) UpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	roleName := input.ResourceBInfo.ResourceName

	return &provider.LinkUpdateLinkedResourcesOutput{
		LinkData: core.MappingNodeFields(
			"boundaryId",
			core.MappingNodeFromString(storedContributionBoundary),
		),
		ResourceDataMappings: map[string]string{
			fmt.Sprintf("%s::spec.boundaryId", roleName): "boundaryId",
		},
	}, nil
}

func TestMergedContributionsStoredTestSuite(t *testing.T) {
	suite.Run(t, new(MergedContributionsStoredTestSuite))
}
