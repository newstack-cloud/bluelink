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
	"github.com/stretchr/testify/suite"
)

const (
	capabilityLayersBlueprint = "__testdata/container/deploy/blueprint-capability-layers.yml"
	networkAttached           = "test.vpc/network-attached"
)

type CapabilityLayersDeployTestSuite struct {
	suite.Suite
}

// A link requiring a capability its provider establishes by contributing has to run after
// that contribution reaches the resource, not just after the providing link settles.
func (s *CapabilityLayersDeployTestSuite) Test_runs_a_requiring_link_only_after_the_contributed_capability_lands() {
	deployedFunction := &recordingFunctionResource{}
	accessLink := &layerAccessLink{}
	loader := s.loader(memstate.NewMemoryStateContainer(), deployedFunction, accessLink)

	messages, finished := s.deploy(loader)
	s.Require().Equal(
		core.InstanceStatusDeployed,
		finished.Status,
		"deploy failed: %v",
		finished.FailureReasons,
	)

	s.Assert().Equal(
		"attached",
		accessLink.attachmentSeen(),
		"the access link ran against a function that had not been given its attachment",
	)

	s.Assert().Equal(
		[]int{0, 1},
		s.layerDepthsFor(messages, "saveOrderFunction"),
		"the function is written once per capability layer, in order",
	)
}

func (s *CapabilityLayersDeployTestSuite) Test_writes_a_resource_with_one_layer_of_contributors_once() {
	deployedFunction := &recordingFunctionResource{}
	loader := s.loader(memstate.NewMemoryStateContainer(), deployedFunction, &layerAccessLink{})

	messages, finished := s.deploy(loader)
	s.Require().Equal(core.InstanceStatusDeployed, finished.Status)

	s.Assert().Equal(
		[]int{1},
		s.layerDepthsFor(messages, "ordersRole"),
		"the role has no capability ordering over its contributors",
	)
}

// The layer carrying the capability states the resource's whole spec, so it has to carry
// what the links it does not yet include contributed last time. Dropping them would take
// fields off the resource between layers and put them back moments later.
func (s *CapabilityLayersDeployTestSuite) Test_an_early_layer_keeps_what_a_later_links_contribution_recorded() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedFunction := &recordingFunctionResource{}
	loader := s.loader(stateContainer, deployedFunction, &layerAccessLink{})

	_, finished := s.deploy(loader)
	s.Require().Equal(core.InstanceStatusDeployed, finished.Status)

	specs := deployedFunction.deployedSpecs()
	s.Require().GreaterOrEqual(len(specs), 2)

	// The final write carries both layers, whatever order they landed in.
	last := specs[len(specs)-1]
	s.Assert().Equal("attached", s.valueAt(last, "$.vpcConfig"))
	s.Assert().Equal("orders", s.valueAt(last, "$.environment.variables.TABLE_NAME"))
}

// A link waiting on a layer that failed is interrupted rather than left waiting or run
// without the capability it requires.
func (s *CapabilityLayersDeployTestSuite) Test_interrupts_a_link_waiting_on_a_layer_that_failed() {
	deployedFunction := &recordingFunctionResource{failContributionLayers: true}
	accessLink := &layerAccessLink{}
	loader := s.loader(memstate.NewMemoryStateContainer(), deployedFunction, accessLink)

	messages, finished := s.deploy(loader)

	s.Assert().NotEqual(
		core.InstanceStatusDeployed,
		finished.Status,
		"the function was never given what its links contribute, so the deployment did not succeed",
	)
	s.Assert().Empty(
		accessLink.attachmentSeen(),
		"the link ran against a function whose attachment could not be applied",
	)
	s.Assert().NotEmpty(
		s.failedLayersFor(messages, "saveOrderFunction"),
		"the layer that could not be applied was not reported",
	)
}

func (s *CapabilityLayersDeployTestSuite) failedLayersFor(
	messages []ResourceDeployUpdateMessage,
	resourceName string,
) []int {
	depths := []int{}
	for _, message := range messages {
		failed := message.PreciseStatus ==
			core.PreciseResourceStatusLinkContributionsUpdateFailed
		if message.ResourceName == resourceName && failed {
			depths = append(depths, message.ContributionLayerDepth)
		}
	}

	return depths
}

func (s *CapabilityLayersDeployTestSuite) layerDepthsFor(
	messages []ResourceDeployUpdateMessage,
	resourceName string,
) []int {
	depths := []int{}
	for _, message := range messages {
		isApplied := message.PreciseStatus == core.PreciseResourceStatusLinkContributionsUpdated
		if message.ResourceName == resourceName && isApplied {
			depths = append(depths, message.ContributionLayerDepth)
		}
	}

	return depths
}

func (s *CapabilityLayersDeployTestSuite) valueAt(spec *core.MappingNode, path string) string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	return core.StringValue(value)
}

func (s *CapabilityLayersDeployTestSuite) deploy(
	loader Loader,
) ([]ResourceDeployUpdateMessage, *DeploymentFinishedMessage) {
	params := newLinkProjectionParams()
	container, err := loader.Load(context.Background(), capabilityLayersBlueprint, params)
	s.Require().NoError(err)

	stagingChannels := createChangeStagingChannels()
	s.Require().NoError(container.StageChanges(
		context.Background(),
		&StageChangesInput{},
		stagingChannels,
		params,
	))

	stagedChanges, err := consumeStagedChangesForTest(stagingChannels)
	s.Require().NoError(err)

	deployChannels := CreateDeployChannels()
	s.Require().NoError(container.Deploy(
		context.Background(),
		&DeployInput{InstanceName: "CapabilityLayersInstance", Changes: stagedChanges},
		deployChannels,
		params,
	))

	return collectDeployMessages(s.T(), deployChannels)
}

func (s *CapabilityLayersDeployTestSuite) loader(
	stateContainer state.Container,
	deployedFunction *recordingFunctionResource,
	accessLink *layerAccessLink,
) Loader {
	deployedFunction.Resource = &internal.LambdaFunctionResource{
		CurrentDestroyAttempts:                   map[string]int{},
		CurrentDeployAttemps:                     map[string]int{},
		CurrentGetExternalStateAttemps:           map[string]int{},
		StabiliseResourceIDs:                     map[string]*internal.StubResourceStabilisationConfig{},
		AlwaysStabilise:                          true,
		CurrentStabiliseCalls:                    map[string]int{},
		FallbackToStateContainerForExternalState: true,
		StateContainer:                           stateContainer,
	}
	accessLink.function = deployedFunction

	awsProvider := &internal.ProviderMock{
		NamespaceValue: "aws",
		Resources: map[string]provider.Resource{
			lambdaFunctionResourceType: deployedFunction,
			iamRoleResourceType:        &recordingRoleResource{},
			ec2VPCResourceType:         &layerVPCResource{},
		},
		Links: map[string]provider.Link{
			"aws/ec2/vpc::aws/lambda/function":  &layerPlacementLink{function: deployedFunction},
			"aws/lambda/function::aws/iam/role": accessLink,
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

func TestCapabilityLayersDeployTestSuite(t *testing.T) {
	suite.Run(t, new(CapabilityLayersDeployTestSuite))
}

// Every write the function was asked to make, so a test can see the layers land rather
// than only their outcome.
type recordingFunctionResource struct {
	provider.Resource
	// Fails the updates carrying contributions while leaving the function's own
	// deployment alone, which is what leaves a link waiting on a layer that never lands.
	failContributionLayers bool
	mu                     sync.Mutex
	specs                  []*core.MappingNode
}

func (r *recordingFunctionResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{iamRoleResourceType},
	}, nil
}

func (r *recordingFunctionResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	if r.failContributionLayers && input.FromLinkContributions {
		return nil, errors.New("the function could not be given what its links contribute")
	}

	spec := core.CopyMappingNode(
		input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec,
	)

	r.mu.Lock()
	r.specs = append(r.specs, spec)
	r.mu.Unlock()

	return r.Resource.Deploy(ctx, input)
}

func (r *recordingFunctionResource) deployedSpecs() []*core.MappingNode {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.specs)
}

// What the function looks like to a link reading it, which is whatever was last written.
func (r *recordingFunctionResource) liveAttachment() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := len(r.specs) - 1; index >= 0; index-- {
		attachment, _ := core.GetPathValue(
			"$.vpcConfig",
			r.specs[index],
			core.MappingNodeMaxTraverseDepth,
		)
		if attachment != nil {
			return core.StringValue(attachment)
		}
	}

	return "unattached"
}

type layerVPCResource struct {
	internal.IAMRoleResource
}

func (r *layerVPCResource) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: []string{lambdaFunctionResourceType},
	}, nil
}

func (r *layerVPCResource) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{Type: ec2VPCResourceType}, nil
}

func (r *layerVPCResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"cidr": {Type: provider.ResourceDefinitionsSchemaTypeString},
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
				},
			},
		},
	}, nil
}

// Places the function in the VPC by contributing its attachment rather than writing it,
// and tells other links the attachment is established.
type layerPlacementLink struct {
	layerLinkBase
	function *recordingFunctionResource
}

func (l *layerPlacementLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{Type: "aws/ec2/vpc::aws/lambda/function"}, nil
}

func (l *layerPlacementLink) GetCapabilities(
	ctx context.Context,
	input *provider.LinkGetCapabilitiesInput,
) (*provider.LinkGetCapabilitiesOutput, error) {
	return &provider.LinkGetCapabilitiesOutput{
		Provides: []provider.LinkCapability{
			{Name: networkAttached, Resource: provider.LinkPriorityResourceB},
		},
	}, nil
}

func (l *layerPlacementLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	functionName := linkhelpers.GetResourceNameFromChanges(input.ResourceBChanges)

	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			ResourceDataMappings: map[string]string{
				fmt.Sprintf("%s::spec.vpcConfig", functionName): "contributions[0]",
			},
		},
	}, nil
}

func (l *layerPlacementLink) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	return &provider.LinkProduceResourceContributionsOutput{
		Contributions: []*provider.ResourceContribution{
			{
				ResourceName: input.ResourceBInfo.ResourceName,
				FieldPath:    "spec.vpcConfig",
				Value:        core.MappingNodeFromString("attached"),
				Action:       provider.ContributionActionSet,
			},
		},
	}, nil
}

// Grants the function access through the role, reading the function's live attachment to
// decide what to do, which is why it requires the capability the placement link provides.
type layerAccessLink struct {
	layerLinkBase
	mu       sync.Mutex
	function *recordingFunctionResource
	seen     string
}

func (l *layerAccessLink) attachmentSeen() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.seen
}

func (l *layerAccessLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{Type: "aws/lambda/function::aws/iam/role"}, nil
}

func (l *layerAccessLink) GetCapabilities(
	ctx context.Context,
	input *provider.LinkGetCapabilitiesInput,
) (*provider.LinkGetCapabilitiesOutput, error) {
	return &provider.LinkGetCapabilitiesOutput{
		Requires: []provider.LinkCapability{
			{Name: networkAttached, Resource: provider.LinkPriorityResourceA},
		},
	}, nil
}

func (l *layerAccessLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	functionName := linkhelpers.GetResourceNameFromChanges(input.ResourceAChanges)
	roleName := linkhelpers.GetResourceNameFromChanges(input.ResourceBChanges)

	return &provider.LinkStageChangesOutput{
		Changes: &provider.LinkChanges{
			ResourceDataMappings: map[string]string{
				fmt.Sprintf("%s::spec.environment.variables.TABLE_NAME", functionName): "contributions[0]",
				fmt.Sprintf("%s::spec.policies", roleName):                             "contributions[1]",
			},
		},
	}, nil
}

func (l *layerAccessLink) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	// The reason this link requires the capability: what it does depends on the function
	// as the provider left it, not as the blueprint declares it.
	l.mu.Lock()
	l.seen = l.function.liveAttachment()
	l.mu.Unlock()

	return &provider.LinkProduceResourceContributionsOutput{
		Contributions: []*provider.ResourceContribution{
			{
				ResourceName: input.ResourceAInfo.ResourceName,
				FieldPath:    "spec.environment.variables.TABLE_NAME",
				Value:        core.MappingNodeFromString("orders"),
				Action:       provider.ContributionActionSet,
			},
			{
				ResourceName: input.ResourceBInfo.ResourceName,
				FieldPath:    "spec.policies",
				Value:        core.MappingNodeFromString("dynamodb:PutItem"),
				Action:       provider.ContributionActionAppend,
			},
		},
	}, nil
}

// The parts of the link interface neither test link has an opinion about.
type layerLinkBase struct{}

func (l *layerLinkBase) UpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	return &provider.LinkUpdateLinkedResourcesOutput{}, nil
}

func (l *layerLinkBase) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *layerLinkBase) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource: provider.LinkPriorityResourceNone,
	}, nil
}

func (l *layerLinkBase) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *layerLinkBase) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{Kind: provider.LinkKindSoft}, nil
}

func (l *layerLinkBase) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *layerLinkBase) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *layerLinkBase) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{}, nil
}

func (l *layerLinkBase) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}
