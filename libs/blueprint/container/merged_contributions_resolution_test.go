package container

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

const (
	mergedContributionsComputedRefBlueprint = "__testdata/container/deploy/" +
		"blueprint-merged-contributions-computed-ref.yml"
	// What permissionsBoundary's computed id resolves to once it has been deployed, which
	// is the value the reference to it in ordersRole must carry.
	deployedBoundaryID = "arn:aws:iam:us-east-1:123456789012:role:permissionsBoundary"
)

type MergedContributionsResolutionTestSuite struct {
	suite.Suite
}

// A resource updated to carry link contributions must be deployed with the values its
// references resolved, not with the values they had while changes were staged.
//
// A reference to a field computed by another resource cannot be resolved before that
// resource is deployed, so the staged spec holds nothing for it. The resource's own
// deployment resolves it, which is why creating it works. The update that carries link
// contributions composes onto a spec of its own, and taking the staged one there deploys
// the resource a second time with every reference emptied. What reaches the provider
// is a spec asking for the referenced fields to be unset, and a provider that diffs
// against the live resource asks the upstream API to null them.
//
// The fields a link contributes are unaffected, so the failure is invisible in what the
// links did and shows up only as the resource's own references being blanked.
func (s *MergedContributionsResolutionTestSuite) Test_deploys_contributions_onto_resolved_references() {
	stateContainer := memstate.NewMemoryStateContainer()
	role := &boundaryRoleResource{}

	finished := s.deployComputedRefBlueprint(stateContainer, role)
	s.Require().Equal(
		core.InstanceStatusDeployed,
		finished.Status,
		"the deployment did not succeed",
	)

	contributionSpec := role.specDeployedWithContributions()
	s.Require().NotNil(
		contributionSpec,
		"the role was never updated with the contributions its links made",
	)

	boundaryID, err := core.GetPathValue(
		"$.boundaryId",
		contributionSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NoError(err)
	s.Require().NotNil(
		boundaryID,
		"the update carrying contributions dropped the role's reference to the boundary",
	)
	s.Assert().Equal(
		deployedBoundaryID,
		core.StringValue(boundaryID),
		"the update carrying contributions deployed a reference resolved before the "+
			"resource it points at existed",
	)
}

// The resource's own deployment already resolved the reference, so this pins that the
// update is the step that loses it rather than the reference never resolving at all.
func (s *MergedContributionsResolutionTestSuite) Test_resolves_the_reference_for_the_resources_own_deployment() {
	stateContainer := memstate.NewMemoryStateContainer()
	role := &boundaryRoleResource{}

	s.deployComputedRefBlueprint(stateContainer, role)

	ownSpec := role.specDeployedOnOwnDeployment()
	s.Require().NotNil(ownSpec, "the role was never deployed in its own right")

	boundaryID, err := core.GetPathValue(
		"$.boundaryId",
		ownSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NoError(err)
	s.Require().NotNil(boundaryID)
	s.Assert().Equal(deployedBoundaryID, core.StringValue(boundaryID))
}

func (s *MergedContributionsResolutionTestSuite) deployComputedRefBlueprint(
	stateContainer state.Container,
	role *boundaryRoleResource,
) *DeploymentFinishedMessage {
	loader := newComputedRefContributionsLoader(stateContainer, role)
	params := newLinkProjectionParams()
	container, err := loader.Load(
		context.Background(),
		mergedContributionsComputedRefBlueprint,
		params,
	)
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
			InstanceName: "MergedContributionsComputedRefInstance",
			Changes:      stagedChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	s.Require().NoError(err)

	_, finished := collectDeployMessages(s.T(), deployChannels)
	return finished
}

func newComputedRefContributionsLoader(
	stateContainer state.Container,
	role *boundaryRoleResource,
) Loader {
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
			"aws/lambda/function::aws/iam/role": &testLambdaIAMRoleLink{},
		},
		CustomVariableTypes: map[string]provider.CustomVariableType{},
		DataSources:         map[string]provider.DataSource{},
	}

	return NewDefaultLoader(
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
}

// A role carrying a reference to a field another resource computes, recording the spec it
// was deployed with on each of the two occasions it is deployed, its own deployment, and
// the update that carries what its links contribute.
type boundaryRoleResource struct {
	internal.IAMRoleResource
	mu                sync.Mutex
	ownDeploymentSpec *core.MappingNode
	contributionsSpec *core.MappingNode
}

func (r *boundaryRoleResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"boundaryId": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Nullable: true,
					},
					"policies": {
						Type: provider.ResourceDefinitionsSchemaTypeArray,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
						Nullable: true,
					},
				},
			},
		},
	}, nil
}

func (r *boundaryRoleResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	spec := core.CopyMappingNode(
		input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec,
	)

	r.mu.Lock()
	if input.FromLinkContributions {
		r.contributionsSpec = spec
	} else {
		r.ownDeploymentSpec = spec
	}
	r.mu.Unlock()

	return r.IAMRoleResource.Deploy(ctx, input)
}

func (r *boundaryRoleResource) specDeployedWithContributions() *core.MappingNode {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.contributionsSpec
}

func (r *boundaryRoleResource) specDeployedOnOwnDeployment() *core.MappingNode {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ownDeploymentSpec
}

func TestMergedContributionsResolutionTestSuite(t *testing.T) {
	suite.Run(t, new(MergedContributionsResolutionTestSuite))
}
