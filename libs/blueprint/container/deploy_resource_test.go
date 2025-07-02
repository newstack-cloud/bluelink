package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ResourceDeployerTestSuite struct {
	suite.Suite
	deployer          ResourceDeployer
	fixtures          map[int]*resourceDeployerFixture
	resourceProviders map[string]provider.Provider
	stateContainer    state.Container
	logger            core.Logger
}

func (s *ResourceDeployerTestSuite) SetupTest() {
	resourceCache := core.NewCache[*provider.ResolvedResource]()
	s.stateContainer = memstate.NewMemoryStateContainer()
	s.deployer = NewDefaultResourceDeployer(
		&core.SystemClock{},
		&internal.StaticIDGenerator{
			ID: "test-resource-id",
		},
		provider.DefaultRetryPolicy,
		// Use a very short stabilisation
		// polling interval and timeout to speed up tests.
		&ResourceStabilityPollingConfig{
			PollingInterval: 1 * time.Millisecond,
			PollingTimeout:  20 * time.Millisecond,
		},
		&staticResourceSubstitutionResolver{
			resolvedResource: s.createResolvedResource(),
		},
		resourceCache,
		s.stateContainer,
	)
	fixtureInputs := s.fixtureInputs()
	s.fixtures = map[int]*resourceDeployerFixture{}

	logger := core.NewNopLogger()
	s.logger = logger

	for _, fixtureInfo := range fixtureInputs {
		fixture, err := s.createFixture(
			fixtureInfo.number,
			fixtureInfo.resourceName,
			fmt.Sprintf("resource-deploy-test--blueprint-instance-%d", fixtureInfo.number),
			s.createResourceDeployExpectedCachedOutput(fixtureInfo.failure, fixtureInfo.resourceName),
			fixtureInfo.failure,
		)
		s.Require().NoError(err)
		err = s.stateContainer.Instances().Save(
			context.Background(),
			*fixture.instanceStateSnapshot,
		)
		s.Require().NoError(err)
		s.fixtures[fixtureInfo.number] = fixture
	}

	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ false,
		/* skipRetryFailuresForLinkNames */ []string{},
		s.stateContainer,
	)

	s.resourceProviders = map[string]provider.Provider{
		"saveOrderFunction":    awsProvider,
		"processOrderFunction": awsProvider,
		"getOrderFunction":     awsProvider,
		"updateOrderFunction":  awsProvider,
		"listOrdersFunction":   awsProvider,
	}
}

func (s *ResourceDeployerTestSuite) Test_creates_a_new_resource() {
	s.runDeployTest(
		s.fixtures[1],
		/* rollingBack */ false,
	)
}

func (s *ResourceDeployerTestSuite) Test_updates_an_existing_resource() {
	s.runDeployTest(
		s.fixtures[2],
		/* rollingBack */ false,
	)
}

func (s *ResourceDeployerTestSuite) Test_rolls_back_resource_removal() {
	s.runDeployTest(
		s.fixtures[3],
		/* rollingBack */ true,
	)
}

func (s *ResourceDeployerTestSuite) Test_rolls_back_resource_update() {
	s.runDeployTest(
		s.fixtures[4],
		/* rollingBack */ true,
	)
}

func (s *ResourceDeployerTestSuite) Test_handles_resource_deploy_terminal_error() {
	s.runDeployTest(
		s.fixtures[5],
		/* rollingBack */ false,
	)
}

func (s *ResourceDeployerTestSuite) Test_handles_stabilise_timeout_error() {
	s.runDeployTest(
		s.fixtures[6],
		/* rollingBack */ false,
	)
}

func (s *ResourceDeployerTestSuite) runDeployTest(
	fixture *resourceDeployerFixture,
	rollingBack bool,
) {
	ctx := context.Background()
	channels := CreateDeployChannels()
	deployState := NewDefaultDeploymentState()
	go s.deployer.Deploy(
		ctx,
		fixture.instanceID,
		fixture.chainLinkNode,
		fixture.changes,
		&DeployContext{
			Rollback:              rollingBack,
			Destroying:            false,
			Channels:              channels,
			State:                 deployState,
			InstanceStateSnapshot: fixture.instanceStateSnapshot,
			ParamOverrides:        deployLinkParams(),
			ResourceProviders:     s.resourceProviders,
			ResourceTemplates:     map[string]string{},
			Logger:                s.logger,
		},
	)

	resourceDeployUpdateMessages := []ResourceDeployUpdateMessage{}
	finishedMessage := (*ResourceDeployUpdateMessage)(nil)
	var err error
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			resourceDeployUpdateMessages = append(resourceDeployUpdateMessages, msg)
			if isResourceDeployFinishedMessage(msg, rollingBack) {
				finishedMessage = &msg
			}
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: resourceDeployUpdateMessages,
		childDeployUpdateMessages:    []ChildDeployUpdateMessage{},
		linkDeployUpdateMessages:     []LinkDeployUpdateMessage{},
		deploymentUpdateMessages:     []DeploymentUpdateMessage{},
	}
	assertDeployMessageOrder(actualMessages, fixture.expectedMessages, &s.Suite)
	cachedOutput := deployState.GetResourceData(fixture.chainLinkNode.ResourceName)
	if !fixture.failure {
		s.Assert().NotNil(cachedOutput)
		s.Assert().Equal(fixture.expectedCachedOutput, cachedOutput.Spec)
	}

	resourceState, err := s.stateContainer.Resources().Get(
		ctx,
		actualMessages.resourceDeployUpdateMessages[0].ResourceID,
	)
	s.Assert().NoError(err)
	s.Assert().NotNil(resourceState)
}

func (s *ResourceDeployerTestSuite) createResourceDeployExpectedCachedOutput(
	failure bool,
	resourceName string,
) *core.MappingNode {
	if failure {
		return nil
	}

	id := fmt.Sprintf(
		"arn:aws:lambda:us-east-1:123456789012:function:%s",
		resourceName,
	)
	handler := fmt.Sprintf(
		"src/%s.handler",
		strings.TrimSuffix(resourceName, "Function"),
	)
	return &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"id":      core.MappingNodeFromString(id),
			"handler": core.MappingNodeFromString(handler),
		},
	}
}

func (s *ResourceDeployerTestSuite) createResolvedResource() *provider.ResolvedResource {
	description := "Function that saves an order to the database."
	return &provider.ResolvedResource{
		Type: &schema.ResourceTypeWrapper{
			Value: "aws/lambda/function",
		},
		Description: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &description,
			},
		},
		LinkSelector: &schema.LinkSelector{
			ByLabel: &schema.StringMap{
				Values: map[string]string{
					"app": "orders",
				},
			},
		},
		Spec: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"handler": core.MappingNodeFromString("src/saveOrder.handler"),
			},
		},
	}
}

func (s *ResourceDeployerTestSuite) fixtureInputs() []*resourceDeployerFixtureInfo {
	return []*resourceDeployerFixtureInfo{
		{
			number:       1,
			resourceName: "saveOrderFunction",
			failure:      false,
		},
		{
			number:       2,
			resourceName: "processOrderFunction",
			failure:      false,
		},
		{
			number:       3,
			resourceName: "getOrderFunction",
			failure:      false,
		},
		{
			number:       4,
			resourceName: "updateOrderFunction",
			failure:      false,
		},
		{
			number:       5,
			resourceName: "updateOrderFunction",
			failure:      true,
		},
		{
			number:       6,
			resourceName: "listOrdersFunction",
			failure:      true,
		},
	}
}

func (s *ResourceDeployerTestSuite) createFixture(
	resourceFixtureNo int,
	resourceName string,
	instanceID string,
	expectedCachedOutput *core.MappingNode,
	failure bool,
) (*resourceDeployerFixture, error) {
	resourceSchemaFilePath := fmt.Sprintf(
		"__testdata/container/resource-deployer/resources/%d--%s.json",
		resourceFixtureNo,
		resourceName,
	)
	schemaResource, err := loadSchemaResourceFromFile(resourceSchemaFilePath)
	if err != nil {
		return nil, err
	}

	chainLinkNode := &links.ChainLinkNode{
		ResourceName: resourceName,
		Resource:     schemaResource,
	}

	expectedMessagesFilePath := fmt.Sprintf(
		"__testdata/container/resource-deployer/expected-messages/%d.json",
		resourceFixtureNo,
	)
	expectedMessages, err := loadExpectedMessagesFromFile(expectedMessagesFilePath)
	if err != nil {
		return nil, err
	}

	instanceStateFilePath := fmt.Sprintf(
		"__testdata/container/resource-deployer/current-state/%d.json",
		resourceFixtureNo,
	)
	instanceStateSnapshot, err := internal.LoadInstanceState(
		instanceStateFilePath,
	)
	if err != nil {
		return nil, err
	}

	changesFilePath := fmt.Sprintf(
		"__testdata/container/resource-deployer/changes/%d.json",
		resourceFixtureNo,
	)
	changes, err := loadBlueprintChangesFromFile(changesFilePath)
	if err != nil {
		return nil, err
	}

	return &resourceDeployerFixture{
		chainLinkNode:         chainLinkNode,
		instanceID:            instanceID,
		instanceStateSnapshot: instanceStateSnapshot,
		changes:               changes,
		expectedMessages:      expectedMessages,
		expectedCachedOutput:  expectedCachedOutput,
		failure:               failure,
	}, nil
}

func loadSchemaResourceFromFile(
	schemaResourceFilePath string,
) (*schema.Resource, error) {
	schemaResourceFileBytes, err := os.ReadFile(schemaResourceFilePath)
	if err != nil {
		return nil, err
	}

	schemaResource := &schema.Resource{}
	err = json.Unmarshal(schemaResourceFileBytes, schemaResource)
	if err != nil {
		return nil, err
	}

	return schemaResource, nil
}

func isResourceDeployFinishedMessage(
	msg ResourceDeployUpdateMessage,
	rollback bool,
) bool {
	if rollback {
		return (msg.Status == core.ResourceStatusRollbackFailed ||
			msg.Status == core.ResourceStatusRollbackComplete) &&
			!msg.CanRetry
	}

	return (msg.Status == core.ResourceStatusCreateFailed ||
		msg.Status == core.ResourceStatusCreated ||
		msg.Status == core.ResourceStatusUpdateFailed ||
		msg.Status == core.ResourceStatusUpdated ||
		msg.Status == core.ResourceStatusDestroyFailed ||
		msg.Status == core.ResourceStatusDestroyed) &&
		!msg.CanRetry
}

type resourceDeployerFixtureInfo struct {
	number       int
	resourceName string
	failure      bool
}

type resourceDeployerFixture struct {
	chainLinkNode         *links.ChainLinkNode
	instanceID            string
	instanceStateSnapshot *state.InstanceState
	changes               *changes.BlueprintChanges
	expectedMessages      *expectedMessages
	expectedCachedOutput  *core.MappingNode
	failure               bool
}

func TestResourceDeployerTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceDeployerTestSuite))
}
