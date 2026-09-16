package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const mergedContributionsBlueprint = "__testdata/container/deploy/blueprint-merged-contributions.yml"

type MergedContributionsDeployTestSuite struct {
	suite.Suite
}

// Two links contributing to one resource cost one update of that resource, stating what
// both of them need it to say.
func (s *MergedContributionsDeployTestSuite) Test_updates_a_resource_with_the_contributions_of_every_link_writing_it() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedRole := &recordingRoleResource{}
	loader := newMergedContributionsLoader(stateContainer, deployedRole)

	messages, finished := s.deployMergedContributionsBlueprint(loader)
	s.Require().Equal(
		core.InstanceStatusDeployed,
		finished.Status,
		fmt.Sprintf("deploy failed: %v", finished.FailureReasons),
	)

	s.Require().ElementsMatch(
		[]string{"grant:saveOrderFunction", "grant:archiveOrderFunction"},
		s.policiesOfDeployedRole(deployedRole),
		"the role was deployed with only one link's contribution, so the other is not on it",
	)

	s.Require().Equal(
		map[string][]string{
			"spec.policies": {"archiveOrderFunction::ordersRole", "saveOrderFunction::ordersRole"},
		},
		contributorsReportedFor(messages, "ordersRole"),
		"the update does not say which links are the reason the role holds what it does",
	)
}

// The statuses an update carrying contributions reports must not be read as the resource's
// own deployment reaching an end.
func (s *MergedContributionsDeployTestSuite) Test_does_not_treat_an_update_carrying_contributions_as_the_resources_own_deployment() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedRole := &recordingRoleResource{}
	loader := newMergedContributionsLoader(stateContainer, deployedRole)

	messages, finished := s.deployMergedContributionsBlueprint(loader)
	s.Require().Equal(core.InstanceStatusDeployed, finished.Status)

	s.Require().Equal(
		1,
		countMessages(messages, "ordersRole", core.PreciseResourceStatusCreated),
		"the role was reported as finished more than once, or not at all",
	)
	s.Require().Equal(
		1,
		countMessages(messages, "ordersRole", core.PreciseResourceStatusLinkContributionsUpdated),
		"the two links contributing to the role did not cost exactly one update of it",
	)

	// Persisted status is the resource's own deployment and nothing else. An update
	// carrying contributions is not a deployment of the role the user asked for, and
	// recording it as one would leave state saying the role was last updated by a change
	// that never touched it.
	roleState, err := stateContainer.Resources().GetByName(
		context.Background(),
		finished.InstanceID,
		"ordersRole",
	)
	s.Require().NoError(err)
	s.Require().False(
		roleState.PreciseStatus.IsLinkContributionStatus(),
		"a status that is only ever reported on the event stream was recorded in state",
	)
	s.Require().Equal(
		core.PreciseResourceStatusCreated,
		roleState.PreciseStatus,
		"the update carrying contributions overwrote the status of the role's own deployment",
	)
	s.Require().Equal(core.ResourceStatusCreated, roleState.Status)
}

func (s *MergedContributionsDeployTestSuite) Test_fails_the_deployment_when_a_resource_cannot_be_given_what_its_links_contribute() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedRole := &recordingRoleResource{failContributionUpdate: true}
	loader := newMergedContributionsLoader(stateContainer, deployedRole)

	messages, finished := s.deployMergedContributionsBlueprint(loader)

	s.Require().NotEqual(
		core.InstanceStatusDeployed,
		finished.Status,
		"the deployment succeeded with the role missing what its links contribute to it",
	)
	s.Require().NotEmpty(finished.FailureReasons)
	s.Require().Equal(
		1,
		countMessages(messages, "ordersRole", core.PreciseResourceStatusLinkContributionsUpdateFailed),
	)

	// The role's own deployment succeeded, and the failure has to name what actually went
	// wrong rather than the resource that was deployed without incident.
	s.Require().Equal(
		1,
		countMessages(messages, "ordersRole", core.PreciseResourceStatusCreated),
	)
	s.Require().Contains(
		fmt.Sprintf("%v", finished.FailureReasons),
		"the role could not be given the statements its links need",
	)
}

// A resource whose links could not give it what they contribute must say so in state.
//
// The resource's own deployment succeeded, so state records it as created and nothing else
// in the persisted resource says the contribution update was ever attempted. Anyone
// inspecting the instance afterwards sees a healthy resource, and the next deployment
// stages against a state that claims the update it needs has already been applied. The
// failure is reported on the event stream and to the deployment as a whole, and neither of
// those is durable, once a deployment's events have aged out, state is the only account
// left.
//
// Recorded beside the resource's status rather than in it. The status stays the resource's
// own because a link contribution status is only ever reported on the stream, and the
// resource's own FailureReasons belong to its own deployment.
func (s *MergedContributionsDeployTestSuite) Test_records_why_a_resource_did_not_get_what_its_links_contribute() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedRole := &recordingRoleResource{failContributionUpdate: true}
	loader := newMergedContributionsLoader(stateContainer, deployedRole)

	_, finished := s.deployMergedContributionsBlueprint(loader)

	roleState, err := stateContainer.Resources().GetByName(
		context.Background(),
		finished.InstanceID,
		"ordersRole",
	)
	s.Require().NoError(err)
	s.Require().Len(
		roleState.LinkContributionFailures,
		1,
		"state records the role as deployed without saying its links could not write to it",
	)

	failure := roleState.LinkContributionFailures[0]
	s.Assert().Contains(
		fmt.Sprintf("%v", failure.Reasons),
		"the role could not be given the statements its links need",
	)
	s.Assert().ElementsMatch(
		[]string{"archiveOrderFunction::ordersRole", "saveOrderFunction::ordersRole"},
		failure.ContributingLinks,
		"the failure does not name the links the role is owed contributions from",
	)
	s.Assert().NotZero(failure.Timestamp)

	// The resource's own deployment said nothing, and must not be made to look as though
	// it did. A reader cannot tell a resource that failed to deploy from one that deployed
	// and was then left without its contributions if both are reported the same way.
	s.Assert().Empty(
		roleState.FailureReasons,
		"a contribution failure was written to the reasons for the resource's own deployment",
	)

	// The invariant the successful case relies on still holds. A status that is only ever
	// reported on the event stream must not be written to state.
	s.Assert().False(
		roleState.PreciseStatus.IsLinkContributionStatus(),
		"a stream-only status was recorded in state",
	)
	s.Assert().Equal(
		core.ResourceStatusCreated,
		roleState.Status,
		"the role's own deployment succeeded and its status must still say so",
	)
}

// A contribution that could not be composed reaches state as the link and field it belongs
// to, not only as a line of text.
func (s *MergedContributionsDeployTestSuite) Test_records_which_contributions_could_not_be_applied() {
	stateContainer := memstate.NewMemoryStateContainer()
	loader := newMergedContributionsLoaderWithLink(
		stateContainer,
		&recordingRoleResource{},
		&testLambdaIAMRoleLink{statesWholeField: true},
	)

	_, finished := s.deployMergedContributionsBlueprint(loader)
	s.Require().NotEqual(
		core.InstanceStatusDeployed,
		finished.Status,
		"the deployment succeeded with the role holding whichever link ran last",
	)

	roleState, err := stateContainer.Resources().GetByName(
		context.Background(),
		finished.InstanceID,
		"ordersRole",
	)
	s.Require().NoError(err)
	s.Require().Len(roleState.LinkContributionFailures, 1)

	failure := roleState.LinkContributionFailures[0]
	s.Require().Len(
		failure.UnappliedContributions,
		2,
		"state holds the failure without saying which contributions it was",
	)

	linkNames := []string{}
	for _, unapplied := range failure.UnappliedContributions {
		s.Assert().Equal("spec.policies", unapplied.FieldPath)
		s.Assert().NotEmpty(unapplied.Reason)
		linkNames = append(linkNames, unapplied.LinkName)
	}
	s.Assert().ElementsMatch(
		[]string{"archiveOrderFunction::ordersRole", "saveOrderFunction::ordersRole"},
		linkNames,
		"a disagreement that names only one of the links leaves the other unaccounted for",
	)
}

// A layer that applies successfully leaves no failure behind it.
//
// A failure that is recorded and never cleared reads as current for the life of the
// instance, so a retry that put the contributions where they belong would be
// indistinguishable from one that was never attempted.
func (s *MergedContributionsDeployTestSuite) Test_clears_a_recorded_contribution_failure_once_the_layer_applies() {
	inner := memstate.NewMemoryStateContainer()
	stateContainer := newContributionClearRecordingStateContainer(inner)

	loader := newMergedContributionsLoader(stateContainer, &recordingRoleResource{})
	_, finished := s.deployMergedContributionsBlueprint(loader)
	s.Require().Equal(core.InstanceStatusDeployed, finished.Status)

	roleState, err := inner.Resources().GetByName(
		context.Background(),
		finished.InstanceID,
		"ordersRole",
	)
	s.Require().NoError(err)

	s.Assert().Contains(
		stateContainer.resources.cleared(),
		clearedContributionLayer{resourceID: roleState.ResourceID, layerDepth: 0},
		"the layer applied without the failure held against it being cleared",
	)
}

type clearedContributionLayer struct {
	resourceID string
	layerDepth int
}

type contributionClearRecordingStateContainer struct {
	state.Container
	resources *contributionClearRecordingResources
}

func newContributionClearRecordingStateContainer(
	inner state.Container,
) *contributionClearRecordingStateContainer {
	return &contributionClearRecordingStateContainer{
		Container: inner,
		resources: &contributionClearRecordingResources{
			ResourcesContainer: inner.Resources(),
		},
	}
}

func (c *contributionClearRecordingStateContainer) Resources() state.ResourcesContainer {
	return c.resources
}

type contributionClearRecordingResources struct {
	state.ResourcesContainer
	mu       sync.Mutex
	cleared_ []clearedContributionLayer
}

func (r *contributionClearRecordingResources) RemoveContributionFailure(
	ctx context.Context,
	resourceID string,
	layerDepth int,
) error {
	r.mu.Lock()
	r.cleared_ = append(r.cleared_, clearedContributionLayer{
		resourceID: resourceID,
		layerDepth: layerDepth,
	})
	r.mu.Unlock()

	return r.ResourcesContainer.RemoveContributionFailure(ctx, resourceID, layerDepth)
}

func (r *contributionClearRecordingResources) cleared() []clearedContributionLayer {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.cleared_)
}

// A failed contribution update has to name the links whose contributions it was carrying.
//
// The resource is the thing that did not get its contributions, so the outcome is recorded
// against it, but on its own that says a resource failed and nothing about why any link was
// writing to it. The successful update already reports its contributors, and the failure is
// where the attribution is worth more, since it is the only thing pointing at the link whose
// contribution has to change.
func (s *MergedContributionsDeployTestSuite) Test_names_the_contributing_links_when_the_update_fails() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedRole := &recordingRoleResource{failContributionUpdate: true}
	loader := newMergedContributionsLoader(stateContainer, deployedRole)

	messages, _ := s.deployMergedContributionsBlueprint(loader)

	var failure *ResourceDeployUpdateMessage
	for index, message := range messages {
		if message.ResourceName == "ordersRole" &&
			message.PreciseStatus == core.PreciseResourceStatusLinkContributionsUpdateFailed {
			failure = &messages[index]
			break
		}
	}

	s.Require().NotNil(failure, "the failed contribution update was never reported")
	s.Require().NotEmpty(
		failure.LinkContributors,
		"the failure names no link, so nothing points at the contribution that has to change",
	)
	s.Assert().Contains(
		failure.LinkContributors,
		"spec.policies",
		"the field the links contributed is not attributed to them",
	)
}

func (s *MergedContributionsDeployTestSuite) deployMergedContributionsBlueprint(
	loader Loader,
) ([]ResourceDeployUpdateMessage, *DeploymentFinishedMessage) {

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
			InstanceName: "MergedContributionsInstance",
			Changes:      stagedChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	s.Require().NoError(err)

	return s.collectUntilFinishForTest(deployChannels)
}

// Every resource update message the deployment produced, which is what a client rendering
// progress sees.
func (s *MergedContributionsDeployTestSuite) collectUntilFinishForTest(
	channels *DeployChannels,
) ([]ResourceDeployUpdateMessage, *DeploymentFinishedMessage) {
	return collectDeployMessages(s.T(), channels)
}

// Every resource update message a deployment produced, which is what a client rendering
// progress sees.
func collectDeployMessages(
	t *testing.T,
	channels *DeployChannels,
) ([]ResourceDeployUpdateMessage, *DeploymentFinishedMessage) {

	collected := []ResourceDeployUpdateMessage{}
	var err error
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for err == nil && finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			collected = append(collected, msg)
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(defaultDrainTimeout):
			err = errors.New(timeoutMessage)
		}
	}
	require.NoError(t, err)
	require.NotNil(t, finishedMessage)

	return collected, finishedMessage
}

func countMessages(
	messages []ResourceDeployUpdateMessage,
	resourceName string,
	preciseStatus core.PreciseResourceStatus,
) int {
	count := 0
	for _, message := range messages {
		if message.ResourceName == resourceName && message.PreciseStatus == preciseStatus {
			count += 1
		}
	}

	return count
}

func contributorsReportedFor(
	messages []ResourceDeployUpdateMessage,
	resourceName string,
) map[string][]string {
	for _, message := range messages {
		if message.ResourceName == resourceName && message.FromLinkContributions {
			return message.LinkContributors
		}
	}

	return nil
}

func (s *MergedContributionsDeployTestSuite) policiesOfDeployedRole(role *recordingRoleResource) []string {

	spec := role.lastDeployedSpec()
	s.Require().NotNil(spec, "the role was never deployed with the contributions made to it")

	policies, err := core.GetPathValue("$.policies", spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(policies, "nothing was contributed to the role's policies")

	values := []string{}
	for _, policy := range policies.Items {
		values = append(values, core.StringValue(policy))
	}

	return values
}

func newMergedContributionsLoader(
	stateContainer state.Container,
	deployedRole *recordingRoleResource,
) Loader {
	return newMergedContributionsLoaderWithLink(
		stateContainer,
		deployedRole,
		&testLambdaIAMRoleLink{},
	)
}

func newMergedContributionsLoaderWithLink(
	stateContainer state.Container,
	deployedRole *recordingRoleResource,
	link provider.Link,
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
			iamRoleResourceType: deployedRole,
		},
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/iam/role": link,
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

// The shared lambda function, able to link to a role.
type roleLinkingLambdaResource struct {
	provider.Resource
}

func (r *roleLinkingLambdaResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{iamRoleResourceType},
	}, nil
}

// A role that records the spec it was asked to deploy, which is where the contributions
// made to it end up.
type recordingRoleResource struct {
	internal.IAMRoleResource
	mu sync.Mutex
	// Fails only the update that carries contributions, so the role's own deployment
	// succeeds and the deployment has nothing else to fail over.
	failContributionUpdate bool
	deployedAs             *core.MappingNode
}

func (r *recordingRoleResource) GetSpecDefinition(
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
					"policies": {
						Type:     provider.ResourceDefinitionsSchemaTypeArray,
						Items:    &provider.ResourceDefinitionsSchema{Type: provider.ResourceDefinitionsSchemaTypeString},
						Nullable: true,
					},
				},
			},
		},
	}, nil
}

func (r *recordingRoleResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	r.mu.Lock()
	failing := r.failContributionUpdate && input.FromLinkContributions
	r.deployedAs = core.CopyMappingNode(
		input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec,
	)
	r.mu.Unlock()

	if failing {
		return nil, errors.New("the role could not be given the statements its links need")
	}

	return r.IAMRoleResource.Deploy(ctx, input)
}

func (r *recordingRoleResource) lastDeployedSpec() *core.MappingNode {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.deployedAs
}

// A link that grants a function access through a shared role, contributing a policy
// statement to the role rather than writing it itself.
type testLambdaIAMRoleLink struct {
	// Makes each link state the whole value of the field rather than adding to it, so that
	// two links writing the same role disagree over what it should hold.
	statesWholeField bool
}

func (l *testLambdaIAMRoleLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	roleName := linkhelpers.GetResourceNameFromChanges(input.ResourceBChanges)

	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			// The declaration the join waits on. Without it the framework has no way to
			// know the role is contributed to before the links run.
			ResourceDataMappings: map[string]string{
				fmt.Sprintf("%s::spec.policies", roleName): "contributions[0]",
			},
		},
	}, nil
}

func (l *testLambdaIAMRoleLink) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	action := provider.ContributionActionAppend
	if l.statesWholeField {
		action = provider.ContributionActionSet
	}

	return &provider.LinkProduceResourceContributionsOutput{
		Contributions: []*provider.ResourceContribution{
			{
				ResourceName: input.ResourceBInfo.ResourceName,
				FieldPath:    "spec.policies",
				Value: core.MappingNodeFromString(
					fmt.Sprintf("grant:%s", input.ResourceAInfo.ResourceName),
				),
				Action: action,
			},
		},
	}, nil
}

func (l *testLambdaIAMRoleLink) UpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	return &provider.LinkUpdateLinkedResourcesOutput{}, nil
}

func (l *testLambdaIAMRoleLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testLambdaIAMRoleLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     provider.LinkPriorityResourceB,
		PriorityResourceType: iamRoleResourceType,
	}, nil
}

func (l *testLambdaIAMRoleLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: "aws/lambda/function::aws/iam/role",
	}, nil
}

func (l *testLambdaIAMRoleLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testLambdaIAMRoleLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{Kind: provider.LinkKindHard}, nil
}

func (l *testLambdaIAMRoleLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testLambdaIAMRoleLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *testLambdaIAMRoleLink) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{}, nil
}

func (l *testLambdaIAMRoleLink) GetCapabilities(
	ctx context.Context,
	input *provider.LinkGetCapabilitiesInput,
) (*provider.LinkGetCapabilitiesOutput, error) {
	return &provider.LinkGetCapabilitiesOutput{}, nil
}

func (l *testLambdaIAMRoleLink) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}

func TestMergedContributionsDeployTestSuite(t *testing.T) {
	suite.Run(t, new(MergedContributionsDeployTestSuite))
}
