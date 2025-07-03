package container

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type LinkDeployerTestSuite struct {
	suite.Suite
	deployer         *defaultLinkDeployer
	fixtures         map[int]*linkDeployerFixture
	stateContainer   state.Container
	resourceRegistry resourcehelpers.Registry
	logger           core.Logger
}

func (s *LinkDeployerTestSuite) SetupTest() {
	s.stateContainer = memstate.NewMemoryStateContainer()
	s.deployer = &defaultLinkDeployer{
		clock:          &core.SystemClock{},
		stateContainer: s.stateContainer,
	}
	s.resourceRegistry = &internal.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/iam/role": &internal.IAMRoleResource{},
		},
	}
	fixtureInputs := s.fixtureInputs()
	s.fixtures = map[int]*linkDeployerFixture{}

	logger := core.NewNopLogger()
	s.logger = logger

	for _, fixtureInfo := range fixtureInputs {
		fixture, err := s.createFixture(
			fixtureInfo.number,
			fixtureInfo.linkName,
			fmt.Sprintf("link-deploy-test--blueprint-instance-%d", fixtureInfo.number),
			// For destroying links, the output for the deployer will not be used.
			// In all cases, the test link implementation will produce
			// the same output for destroy, create and update deployments.
			s.createFixtureDeployExpectedOutput(fixtureInfo.failure),
		)
		s.Require().NoError(err)
		s.stateContainer.Instances().Save(
			context.Background(),
			*fixture.instanceStateSnapshot,
		)
		s.Require().NoError(err)
		s.fixtures[fixtureInfo.number] = fixture
	}
}

func (s *LinkDeployerTestSuite) Test_creates_a_new_link() {
	s.runDeployTest(
		s.fixtures[1],
		provider.LinkUpdateTypeCreate,
		/* destroying */ false,
		/* rollingBack */ false,
	)
}

func (s *LinkDeployerTestSuite) Test_updates_an_existing_link() {
	s.runDeployTest(
		s.fixtures[2],
		provider.LinkUpdateTypeUpdate,
		/* destroying */ false,
		/* rollingBack */ false,
	)
}

func (s *LinkDeployerTestSuite) Test_destroys_a_link() {
	s.runDeployTest(
		s.fixtures[3],
		provider.LinkUpdateTypeDestroy,
		/* destroying */ true,
		/* rollingBack */ false,
	)
}

func (s *LinkDeployerTestSuite) Test_link_creation_rollback() {
	s.runDeployTest(
		s.fixtures[4],
		provider.LinkUpdateTypeDestroy,
		/* destroying */ true,
		/* rollingBack */ true,
	)
}

func (s *LinkDeployerTestSuite) Test_link_update_rollback() {
	s.runDeployTest(
		s.fixtures[5],
		provider.LinkUpdateTypeUpdate,
		/* destroying */ false,
		/* rollingBack */ true,
	)
}

func (s *LinkDeployerTestSuite) Test_link_destroy_rollback() {
	s.runDeployTest(
		s.fixtures[6],
		provider.LinkUpdateTypeCreate,
		/* destroying */ false,
		/* rollingBack */ true,
	)
}

func (s *LinkDeployerTestSuite) Test_handles_resource_b_update_terminal_failure() {
	s.runDeployTest(
		s.fixtures[7],
		provider.LinkUpdateTypeCreate,
		/* destroying */ false,
		/* rollingBack */ false,
	)
}

func (s *LinkDeployerTestSuite) Test_handles_resource_a_update_terminal_failure() {
	s.runDeployTest(
		s.fixtures[8],
		provider.LinkUpdateTypeCreate,
		/* destroying */ false,
		/* rollingBack */ false,
	)
}

func (s *LinkDeployerTestSuite) Test_handles_intermediary_resources_update_terminal_failure() {
	s.runDeployTest(
		s.fixtures[9],
		provider.LinkUpdateTypeCreate,
		/* destroying */ false,
		/* rollingBack */ false,
	)
}

func (s *LinkDeployerTestSuite) runDeployTest(
	fixture *linkDeployerFixture,
	updateType provider.LinkUpdateType,
	destroying bool,
	rollingBack bool,
) {
	ctx := context.Background()
	channels := CreateDeployChannels()
	deployState := NewDefaultDeploymentState()
	go func() {
		s.deployer.Deploy(
			ctx,
			fixture.linkElement,
			fixture.instanceID,
			fixture.instanceName,
			updateType,
			&testLambdaDynamoDBTableLink{
				resourceAUpdateAttempts: map[string]int{},
				failResourceANames: []string{
					"saveOrderFunctionFail",
				},
				failResourceBNames: []string{
					"ordersTableFail",
				},
				failIntermediariesUpdateLinkNames: []string{
					"saveOrderFunctionFail2::ordersTableFail2",
				},
				skipRetryFailuresForInstance: []string{
					// To reduce time it takes to complete tests,
					// skip retry failures for all but one test case.
					"link-deploy-test--blueprint-instance-2",
					"link-deploy-test--blueprint-instance-3",
					"link-deploy-test--blueprint-instance-4",
					"link-deploy-test--blueprint-instance-5",
					"link-deploy-test--blueprint-instance-6",
					"link-deploy-test--blueprint-instance-7",
					"link-deploy-test--blueprint-instance-8",
					"link-deploy-test--blueprint-instance-9",
				},
			},
			&DeployContext{
				Rollback:              rollingBack,
				Destroying:            destroying,
				Channels:              channels,
				State:                 deployState,
				InstanceStateSnapshot: fixture.instanceStateSnapshot,
				ParamOverrides:        deployLinkParams(),
				ResourceTemplates:     map[string]string{},
				ResourceRegistry:      s.resourceRegistry,
				Logger:                s.logger,
			},
			provider.DefaultRetryPolicy,
		)
	}()

	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	finishedMessage := (*LinkDeployUpdateMessage)(nil)
	var err error
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.LinkUpdateChan:
			linkDeployUpdateMessages = append(linkDeployUpdateMessages, msg)
			if isLinkDeployFinishedMessage(msg, rollingBack) {
				finishedMessage = &msg
			}
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	result := deployState.GetLinkDeployResult(fixture.linkElement.LinkName)
	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: []ResourceDeployUpdateMessage{},
		childDeployUpdateMessages:    []ChildDeployUpdateMessage{},
		linkDeployUpdateMessages:     linkDeployUpdateMessages,
		deploymentUpdateMessages:     []DeploymentUpdateMessage{},
	}
	assertDeployMessageOrder(actualMessages, fixture.expectedMessages, &s.Suite)
	s.Assert().Equal(fixture.expectedOutput, result)

	linkState, err := s.stateContainer.Links().Get(
		ctx,
		actualMessages.linkDeployUpdateMessages[0].LinkID,
	)
	s.Assert().NoError(err)
	s.Assert().NotNil(linkState)
}

func (s *LinkDeployerTestSuite) fixtureInputs() []*linkDeployerFixtureInfo {
	return []*linkDeployerFixtureInfo{
		{
			number:   1,
			linkName: "saveOrderFunction::ordersTable",
			failure:  false,
		},
		{
			number:   2,
			linkName: "saveOrderFunction::ordersTable",
			failure:  false,
		},
		{
			number:   3,
			linkName: "saveOrderFunction::ordersTable",
			failure:  false,
		},
		{
			number:   4,
			linkName: "saveOrderFunction::ordersTable",
			failure:  false,
		},
		{
			number:   5,
			linkName: "saveOrderFunction::ordersTable",
			failure:  false,
		},
		{
			number:   6,
			linkName: "saveOrderFunction::ordersTable",
			failure:  false,
		},
		{
			number:   7,
			linkName: "saveOrderFunction::ordersTableFail",
			failure:  true,
		},
		{
			number:   8,
			linkName: "saveOrderFunctionFail::ordersTable",
			failure:  true,
		},
		{
			number:   9,
			linkName: "saveOrderFunctionFail2::ordersTableFail2",
			failure:  true,
		},
	}
}

func (s *LinkDeployerTestSuite) createFixture(
	linkFixtureNo int,
	linkName string,
	instanceID string,
	expectedOutput *LinkDeployResult,
) (*linkDeployerFixture, error) {

	linkElement := &LinkIDInfo{
		LinkID:   fmt.Sprintf("test-link-%d", linkFixtureNo),
		LinkName: linkName,
	}

	expectedMessagesFilePath := fmt.Sprintf(
		"__testdata/container/link-deployer/expected-messages/%d.json",
		linkFixtureNo,
	)
	expectedMessages, err := loadExpectedMessagesFromFile(expectedMessagesFilePath)
	if err != nil {
		return nil, err
	}

	instanceStateFilePath := fmt.Sprintf(
		"__testdata/container/link-deployer/current-state/%d.json",
		linkFixtureNo,
	)
	instanceStateSnapshot, err := internal.LoadInstanceState(
		instanceStateFilePath,
	)
	if err != nil {
		return nil, err
	}

	return &linkDeployerFixture{
		linkElement:           linkElement,
		instanceID:            instanceID,
		instanceName:          instanceID,
		instanceStateSnapshot: instanceStateSnapshot,
		expectedMessages:      expectedMessages,
		expectedOutput:        expectedOutput,
	}, nil
}

func (s *LinkDeployerTestSuite) createFixtureDeployExpectedOutput(
	failure bool,
) *LinkDeployResult {
	if failure {
		return (*LinkDeployResult)(nil)
	}

	return &LinkDeployResult{
		IntermediaryResourceStates: []*state.LinkIntermediaryResourceState{
			{
				ResourceID: "testRoleID",
				ResourceSpecData: &core.MappingNode{
					Fields: map[string]*core.MappingNode{
						"id": core.MappingNodeFromString("arn:aws:iam:us-east-1:123456789012:role:testRole"),
					},
				},
			},
		},
		LinkData: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"saveOrderFunction": {
					Fields: map[string]*core.MappingNode{
						"environmentVariables": {
							Fields: map[string]*core.MappingNode{
								"TABLE_NAME_ordersTable":   core.MappingNodeFromString("production-orders"),
								"TABLE_REGION_ordersTable": core.MappingNodeFromString("eu-west-2"),
							},
						},
					},
				},
				"ordersTable":              core.MappingNodeFromString("testResourceBValue"),
				"testIntermediaryResource": core.MappingNodeFromString("testIntermediaryResourceValue"),
			},
		},
		ResourceDataMappings: map[string]string{
			"saveOrderFunction::spec.environment.variables.TABLE_NAME_ordersTable":   "saveOrderFunction.environmentVariables.TABLE_NAME_ordersTable",
			"saveOrderFunction::spec.environment.variables.TABLE_REGION_ordersTable": "saveOrderFunction.environmentVariables.TABLE_REGION_ordersTable",
		},
	}
}

func deployLinkParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

func isLinkDeployFinishedMessage(
	msg LinkDeployUpdateMessage,
	rollback bool,
) bool {
	if rollback {
		return (msg.Status == core.LinkStatusCreateRollbackFailed ||
			msg.Status == core.LinkStatusCreateRollbackComplete ||
			msg.Status == core.LinkStatusUpdateRollbackFailed ||
			msg.Status == core.LinkStatusUpdateRollbackComplete ||
			msg.Status == core.LinkStatusDestroyRollbackFailed ||
			msg.Status == core.LinkStatusDestroyRollbackComplete) &&
			!msg.CanRetryCurrentStage
	}

	return (msg.Status == core.LinkStatusCreateFailed ||
		msg.Status == core.LinkStatusCreated ||
		msg.Status == core.LinkStatusUpdateFailed ||
		msg.Status == core.LinkStatusUpdated ||
		msg.Status == core.LinkStatusDestroyFailed ||
		msg.Status == core.LinkStatusDestroyed) &&
		!msg.CanRetryCurrentStage
}

type linkDeployerFixtureInfo struct {
	number   int
	linkName string
	failure  bool
}

type linkDeployerFixture struct {
	linkElement           *LinkIDInfo
	instanceID            string
	instanceName          string
	instanceStateSnapshot *state.InstanceState
	expectedMessages      *expectedMessages
	expectedOutput        *LinkDeployResult
}

func TestLinkDeployerTestSuite(t *testing.T) {
	suite.Run(t, new(LinkDeployerTestSuite))
}
