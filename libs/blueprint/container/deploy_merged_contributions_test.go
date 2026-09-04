package container

import (
	"context"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/mockclock"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type MergedContributionsDeployerTestSuite struct {
	suite.Suite
}

// The update carries what every contributing link needs the resource to say, including
// the links this deployment did not run.
func (s *MergedContributionsDeployerTestSuite) Test_deploys_the_resource_with_every_links_contribution() {
	deployed := &capturingContributionResource{}
	deployCtx, stateContainer, messages := s.deployContext(deployed, map[string]*LinkDeployResult{
		"saveOrderFunction::ordersTable": s.produced("spec.policies", "dynamodb:PutItem"),
	}, []*state.LinkState{
		s.storedAppend("archiveFunction::appQueue", "spec.policies", "sqs:SendMessage"),
	})

	err := NewDefaultMergedContributionsDeployer(stateContainer, &mockclock.StaticClock{}).Deploy(
		context.Background(),
		"test-instance",
		"ordersRole",
		[]string{"saveOrderFunction::ordersTable"},
		deployCtx,
	)
	s.Require().NoError(err)

	s.Assert().ElementsMatch(
		[]string{"dynamodb:PutItem", "sqs:SendMessage"},
		s.itemsAt(deployed.spec, "$.policies"),
		"the link that did not run keeps its statement",
	)

	updates := drainResourceMessages(messages)
	s.Require().Len(updates, 2)
	s.Assert().Equal(
		core.PreciseResourceStatusUpdatingLinkContributions,
		updates[0].PreciseStatus,
	)
	s.Assert().Equal(
		core.PreciseResourceStatusLinkContributionsUpdated,
		updates[1].PreciseStatus,
	)
}

// The update has to be distinguishable from one the user's change asked for, and has to
// say which links are the reason the resource holds what it does.
func (s *MergedContributionsDeployerTestSuite) Test_reports_the_update_as_one_carrying_contributions() {
	deployed := &capturingContributionResource{}
	deployCtx, stateContainer, messages := s.deployContext(deployed, map[string]*LinkDeployResult{
		"saveOrderFunction::ordersTable": s.produced("spec.policies", "dynamodb:PutItem"),
	}, []*state.LinkState{
		s.storedAppend("archiveFunction::appQueue", "spec.policies", "sqs:SendMessage"),
	})

	err := NewDefaultMergedContributionsDeployer(stateContainer, &mockclock.StaticClock{}).Deploy(
		context.Background(),
		"test-instance",
		"ordersRole",
		[]string{"saveOrderFunction::ordersTable"},
		deployCtx,
	)
	s.Require().NoError(err)

	updates := drainResourceMessages(messages)
	s.Require().NotEmpty(updates)
	for _, update := range updates {
		s.Assert().True(
			update.FromLinkContributions,
			"an update carrying contributions must not read as a change to the blueprint",
		)
	}
	s.Assert().Equal(
		map[string][]string{
			"spec.policies": {
				"archiveFunction::appQueue",
				"saveOrderFunction::ordersTable",
			},
		},
		updates[0].LinkContributors,
	)
}

func (s *MergedContributionsDeployerTestSuite) Test_tells_the_provider_the_deployment_carries_contributions() {
	deployed := &capturingContributionResource{}
	deployCtx, stateContainer, _ := s.deployContext(deployed, map[string]*LinkDeployResult{
		"saveOrderFunction::ordersTable": s.produced("spec.policies", "dynamodb:PutItem"),
	}, nil)

	err := NewDefaultMergedContributionsDeployer(stateContainer, &mockclock.StaticClock{}).Deploy(
		context.Background(),
		"test-instance",
		"ordersRole",
		[]string{"saveOrderFunction::ordersTable"},
		deployCtx,
	)
	s.Require().NoError(err)

	s.Assert().True(
		deployed.fromLinkContributions,
		"the provider was given no way to tell this from a change the blueprint asked for",
	)
}

func (s *MergedContributionsDeployerTestSuite) Test_does_not_deploy_when_a_contribution_cannot_be_applied() {
	deployed := &capturingContributionResource{}
	unreadable := s.storedAppend("archiveFunction::appQueue", "spec.policies", "sqs:SendMessage")
	unreadable.Data = map[string]*core.MappingNode{}

	deployCtx, stateContainer, messages := s.deployContext(deployed, nil, []*state.LinkState{unreadable})

	err := NewDefaultMergedContributionsDeployer(stateContainer, &mockclock.StaticClock{}).Deploy(
		context.Background(),
		"test-instance",
		"ordersRole",
		nil,
		deployCtx,
	)
	s.Require().NoError(err)

	s.Assert().Nil(deployed.spec, "the resource was deployed without the contribution")

	updates := drainResourceMessages(messages)
	s.Require().Len(updates, 1)
	s.Assert().Equal(
		core.PreciseResourceStatusLinkContributionsUpdateFailed,
		updates[0].PreciseStatus,
	)
	s.Assert().Contains(updates[0].FailureReasons[0], "archiveFunction::appQueue")
}

// A resource that links contribute to is either created by this deployment, which has saved it
// by the time its links settle, or already in state. One that is in neither
// is a resource a link declared a contribution to that does not exist,
// and staying silent about it leaves the deployment waiting for an update that will never be reported.
func (s *MergedContributionsDeployerTestSuite) Test_reports_a_resource_that_is_not_deployed() {
	deployed := &capturingContributionResource{}
	deployCtx, stateContainer, messages := s.deployContext(deployed, nil, nil)

	err := NewDefaultMergedContributionsDeployer(stateContainer, &mockclock.StaticClock{}).Deploy(
		context.Background(),
		"test-instance",
		"neverDeployedRole",
		nil,
		deployCtx,
	)
	s.Require().NoError(err)

	s.Assert().Nil(deployed.spec)

	updates := drainResourceMessages(messages)
	s.Require().Len(updates, 1)
	s.Assert().Equal(
		core.PreciseResourceStatusLinkContributionsUpdateFailed,
		updates[0].PreciseStatus,
	)
	s.Assert().Contains(updates[0].FailureReasons[0], "neverDeployedRole")
}

func (s *MergedContributionsDeployerTestSuite) deployContext(
	deployed provider.Resource,
	deployResults map[string]*LinkDeployResult,
	storedLinks []*state.LinkState,
) (*DeployContext, state.Container, chan ResourceDeployUpdateMessage) {
	deployState := NewDefaultDeploymentState()
	for linkName, result := range deployResults {
		deployState.SetLinkDeployResult(linkName, result)
	}

	links := map[string]*state.LinkState{}
	for _, linkState := range storedLinks {
		links[linkState.Name] = linkState
	}

	// Buffered so the deployer's status updates do not block on a reader, which is what
	// the deployment event loop is in a real deployment.
	channels := &DeployChannels{
		ResourceUpdateChan: make(chan ResourceDeployUpdateMessage, 10),
	}

	instanceState := state.InstanceState{
		InstanceID:   "test-instance",
		InstanceName: "TestInstance",
		ResourceIDs:  map[string]string{"ordersRole": "orders-role-id"},
		Resources: map[string]*state.ResourceState{
			"orders-role-id": {
				ResourceID: "orders-role-id",
				Name:       "ordersRole",
				Type:       "aws/iam/role",
				InstanceID: "test-instance",
				SpecData:   core.MappingNodeFields(),
			},
		},
		Links: links,
	}

	// The resource is read from live state rather than the snapshot, since a resource
	// links contribute to is commonly created by the same deployment that runs them.
	stateContainer := memstate.NewMemoryStateContainer()
	s.Require().NoError(
		stateContainer.Instances().Save(context.Background(), instanceState),
	)

	return &DeployContext{
		State:                 deployState,
		Channels:              channels,
		InstanceStateSnapshot: &instanceState,
		InputChanges:          &changes.BlueprintChanges{},
		ResourceProviders: map[string]provider.Provider{
			"ordersRole": &contributionTestProvider{resource: deployed},
		},
		ParamOverrides: core.NewDefaultParams(
			map[string]map[string]*core.ScalarValue{},
			map[string]map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{},
		),
	}, stateContainer, channels.ResourceUpdateChan
}

func (s *MergedContributionsDeployerTestSuite) produced(
	fieldPath string,
	value string,
) *LinkDeployResult {
	return &LinkDeployResult{
		Contributions: []*provider.ResourceContribution{
			{
				ResourceName: "ordersRole",
				FieldPath:    fieldPath,
				Value:        core.MappingNodeFromString(value),
				Action:       provider.ContributionActionAppend,
			},
		},
	}
}

func (s *MergedContributionsDeployerTestSuite) storedAppend(
	linkName string,
	fieldPath string,
	value string,
) *state.LinkState {
	contributed := ContributionsToLinkData([]*provider.ResourceContribution{
		{
			ResourceName: "ordersRole",
			FieldPath:    fieldPath,
			Value:        core.MappingNodeFromString(value),
			Action:       provider.ContributionActionAppend,
		},
	})

	return &state.LinkState{
		Name:                 linkName,
		Data:                 contributed.Data,
		ResourceDataMappings: contributed.ResourceDataMappings,
		ContributionRecords:  contributed.ContributionRecords,
	}
}

func (s *MergedContributionsDeployerTestSuite) itemsAt(
	spec *core.MappingNode,
	path string,
) []string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	items := []string{}
	for _, item := range value.Items {
		items = append(items, core.StringValue(item))
	}

	return items
}

func drainResourceMessages(
	messages chan ResourceDeployUpdateMessage,
) []ResourceDeployUpdateMessage {
	drained := []ResourceDeployUpdateMessage{}
	for {
		select {
		case message := <-messages:
			drained = append(drained, message)
		default:
			return drained
		}
	}
}

// Records the spec it was asked to deploy, which is what the update carries, and why it
// was asked to deploy it.
type capturingContributionResource struct {
	provider.Resource
	spec                  *core.MappingNode
	fromLinkContributions bool
}

func (r *capturingContributionResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	r.spec = core.CopyMappingNode(
		input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec,
	)
	r.fromLinkContributions = input.FromLinkContributions

	return &provider.ResourceDeployOutput{}, nil
}

type contributionTestProvider struct {
	provider.Provider
	resource provider.Resource
}

func (p *contributionTestProvider) Resource(
	ctx context.Context,
	resourceType string,
) (provider.Resource, error) {
	return p.resource, nil
}

func TestMergedContributionsDeployerTestSuite(t *testing.T) {
	suite.Run(t, new(MergedContributionsDeployerTestSuite))
}
