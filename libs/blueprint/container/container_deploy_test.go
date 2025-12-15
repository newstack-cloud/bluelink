package container

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type ContainerDeployTestSuite struct {
	blueprint1Fixture blueprintDeployFixture
	blueprint2Fixture blueprintDeployFixture
	blueprint3Fixture blueprintDeployFixture
	blueprint4Fixture blueprintDeployFixture
	blueprint5Fixture blueprintDeployFixture
	stateContainer    state.Container
	fixture1Params    core.BlueprintParams
	fixture2Params    core.BlueprintParams
	fixture3Params    core.BlueprintParams
	fixture4Params    core.BlueprintParams
	fixture5Params    core.BlueprintParams
	suite.Suite
}

func (s *ContainerDeployTestSuite) SetupTest() {
	stateContainer := memstate.NewMemoryStateContainer()
	s.stateContainer = stateContainer
	fixtureInstances := []int{1, 3, 4, 5}
	err := populateCurrentState(fixtureInstances, stateContainer, "deploy")
	s.Require().NoError(err)

	skipRetryFailureForLinkNames := []string{
		// Transient failures are expected for "saveOrderFunction::ordersTable_0"
		// but not the other links between lambda functions and DynamoDB tables
		// in the same input blueprint.
		"saveOrderFunction::ordersTable_0",
		"saveOrderFunction::ordersTable_2",
	}
	providers := map[string]provider.Provider{
		"aws":     newTestAWSProvider(true /* alwaysStabilise */, skipRetryFailureForLinkNames, stateContainer),
		"example": newTestExampleProvider(),
		"core": providerhelpers.NewCoreProvider(
			stateContainer.Links(),
			core.BlueprintInstanceIDFromContext,
			os.Getwd,
			provider.NewFileSourceRegistry(),
			core.SystemClock{},
		),
	}
	specTransformers := map[string]transform.SpecTransformer{}
	// Speed up tests by reducing the polling interval for resource stability checks.
	resStabilityPollingConfig := &ResourceStabilityPollingConfig{
		PollingInterval: 10 * time.Millisecond,
		PollingTimeout:  1 * time.Second,
	}
	logger := core.NewNopLogger()
	s.Require().NoError(err)
	loader := NewDefaultLoader(
		providers,
		specTransformers,
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderResourceStabilityPollingConfig(resStabilityPollingConfig),
		WithLoaderLogger(logger),
	)

	s.fixture1Params = blueprint1DeployParams(
		/* includeInvoices */ false,
	)
	s.blueprint1Fixture, err = createBlueprintDeployFixture(
		"deploy",
		1,
		loader,
		s.fixture1Params,
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)

	s.fixture2Params = blueprint1DeployParams(
		/* includeInvoices */ true,
	)
	s.blueprint2Fixture, err = createBlueprintDeployFixture(
		"deploy",
		2,
		loader,
		s.fixture2Params,
		schema.JWCCSpecFormat,
	)
	s.Require().NoError(err)

	s.fixture3Params = blueprint3DeployParams()
	s.blueprint3Fixture, err = createBlueprintDeployFixture(
		"deploy",
		3,
		loader,
		s.fixture3Params,
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)

	s.fixture4Params = blueprint3DeployParams()
	s.blueprint4Fixture, err = createBlueprintDeployFixture(
		"deploy",
		4,
		loader,
		s.fixture4Params,
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)

	// Fixture 5 uses blueprint1's spec but with a state file that has
	// status=Updating (13) to test force deploy on stuck instances.
	s.fixture5Params = blueprint1DeployParams(
		/* includeInvoices */ false,
	)
	s.blueprint5Fixture, err = createBlueprintDeployFixture(
		"deploy",
		5,
		loader,
		s.fixture5Params,
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)
}

func (s *ContainerDeployTestSuite) Test_deploys_updates_to_existing_blueprint_instance() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		s.blueprint1Fixture.blueprintContainer,
		s.fixture1Params,
	)
	s.Require().NoError(changeStagingErr)

	err := s.blueprint1Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: "blueprint-instance-1",
			Changes:    changes,
			Rollback:   false,
		},
		channels,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	deploymentUpdateMessages := []DeploymentUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			resourceUpdateMessages = append(resourceUpdateMessages, msg)
		case msg := <-channels.ChildUpdateChan:
			childDeployUpdateMessages = append(childDeployUpdateMessages, msg)
		case msg := <-channels.LinkUpdateChan:
			linkDeployUpdateMessages = append(linkDeployUpdateMessages, msg)
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case msg := <-channels.DeploymentUpdateChan:
			deploymentUpdateMessages = append(deploymentUpdateMessages, msg)
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: resourceUpdateMessages,
		childDeployUpdateMessages:    childDeployUpdateMessages,
		linkDeployUpdateMessages:     linkDeployUpdateMessages,
		deploymentUpdateMessages:     deploymentUpdateMessages,
		finishedMessage:              finishedMessage,
	}
	assertDeployMessageOrder(actualMessages, s.blueprint1Fixture.expected, &s.Suite)

	instanceState, err := s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-1")
	s.Require().NoError(err)
	assertInstanceStateEquals(
		s.blueprint1Fixture.expectedInstanceState,
		&instanceState,
		&s.Suite,
	)
}

func (s *ContainerDeployTestSuite) Test_deploys_updates_to_existing_blueprint_instance_by_name() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		s.blueprint1Fixture.blueprintContainer,
		s.fixture1Params,
	)
	s.Require().NoError(changeStagingErr)

	err := s.blueprint1Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			// user-defined name provided instead of ID,
			// deploy should resolve the ID to select the correct
			// instance to update.
			InstanceName: "BlueprintInstance1",
			Changes:      changes,
			Rollback:     false,
		},
		channels,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	deploymentUpdateMessages := []DeploymentUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			resourceUpdateMessages = append(resourceUpdateMessages, msg)
		case msg := <-channels.ChildUpdateChan:
			childDeployUpdateMessages = append(childDeployUpdateMessages, msg)
		case msg := <-channels.LinkUpdateChan:
			linkDeployUpdateMessages = append(linkDeployUpdateMessages, msg)
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case msg := <-channels.DeploymentUpdateChan:
			deploymentUpdateMessages = append(deploymentUpdateMessages, msg)
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: resourceUpdateMessages,
		childDeployUpdateMessages:    childDeployUpdateMessages,
		linkDeployUpdateMessages:     linkDeployUpdateMessages,
		deploymentUpdateMessages:     deploymentUpdateMessages,
		finishedMessage:              finishedMessage,
	}
	assertDeployMessageOrder(actualMessages, s.blueprint1Fixture.expected, &s.Suite)

	instanceState, err := s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-1")
	s.Require().NoError(err)
	assertInstanceStateEquals(
		s.blueprint1Fixture.expectedInstanceState,
		&instanceState,
		&s.Suite,
	)
}

func (s *ContainerDeployTestSuite) Test_deploys_new_blueprint_instance() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		s.blueprint2Fixture.blueprintContainer,
		s.fixture2Params,
	)
	s.Require().NoError(changeStagingErr)

	err := s.blueprint2Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			// An ID must not be provided for a new blueprint instance,
			// the container will generate it.
			//
			// An instance name, however, must be provided for a new
			// deployment.
			InstanceName: "BlueprintInstance2",
			Changes:      changes,
			Rollback:     false,
		},
		channels,
		s.fixture2Params,
	)
	s.Require().NoError(err)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	deploymentUpdateMessages := []DeploymentUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			resourceUpdateMessages = append(resourceUpdateMessages, msg)
		case msg := <-channels.ChildUpdateChan:
			childDeployUpdateMessages = append(childDeployUpdateMessages, msg)
		case msg := <-channels.LinkUpdateChan:
			linkDeployUpdateMessages = append(linkDeployUpdateMessages, msg)
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case msg := <-channels.DeploymentUpdateChan:
			deploymentUpdateMessages = append(deploymentUpdateMessages, msg)
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: resourceUpdateMessages,
		childDeployUpdateMessages:    childDeployUpdateMessages,
		linkDeployUpdateMessages:     linkDeployUpdateMessages,
		deploymentUpdateMessages:     deploymentUpdateMessages,
		finishedMessage:              finishedMessage,
	}
	assertDeployMessageOrder(actualMessages, s.blueprint2Fixture.expected, &s.Suite)

	instanceState, err := s.stateContainer.Instances().Get(
		context.Background(),
		actualMessages.finishedMessage.InstanceID,
	)
	s.Require().NoError(err)
	assertInstanceStateEquals(
		s.blueprint2Fixture.expectedInstanceState,
		&instanceState,
		&s.Suite,
	)
}

func (s *ContainerDeployTestSuite) Test_fails_to_deploy_new_blueprint_instance_when_name_is_missing() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		s.blueprint2Fixture.blueprintContainer,
		s.fixture2Params,
	)
	s.Require().NoError(changeStagingErr)

	err := s.blueprint2Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			// An ID must not be provided for a new blueprint instance,
			// the container will generate it.
			//
			// An instance name, however, must be provided for a new
			// deployment but is missing here.
			Changes:  changes,
			Rollback: false,
		},
		channels,
		s.fixture2Params,
	)
	s.Require().Error(err)
	runErr, isRunErr := err.(*bperrors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(
		ErrorReasonCodeMissingNameForNewInstance,
		runErr.ReasonCode,
	)
}

func (s *ContainerDeployTestSuite) Test_fails_to_deploy_blueprint_with_cycle() {
	channels := CreateDeployChannels()
	changes := fixture3Changes()

	// Ensure the parent blueprint is attached as a child of the core infra
	// to create a cycle in state to ensure the cycle check is tested.
	attachErr := s.stateContainer.Children().Attach(
		context.Background(),
		"blueprint-instance-3-child-core-infra",
		"blueprint-instance-3",
		"appInfra",
	)
	s.Require().NoError(attachErr)

	err := s.blueprint3Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: "blueprint-instance-3",
			Changes:    changes,
			Rollback:   false,
		},
		channels,
		s.fixture3Params,
	)
	s.Require().NoError(err)

	var finishMsg *DeploymentFinishedMessage
	for err == nil && finishMsg == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishMsg = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Assert().Error(err)
	runErr, isRunErr := err.(*bperrors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(
		ErrorReasonCodeBlueprintCycleDetected,
		runErr.ReasonCode,
	)
}

func (s *ContainerDeployTestSuite) Test_fails_to_deploy_blueprint_instance_already_being_deployed() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		"blueprint-instance-4",
		s.blueprint4Fixture.blueprintContainer,
		s.fixture4Params,
	)
	s.Require().NoError(changeStagingErr)

	err := s.blueprint3Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: "blueprint-instance-4",
			Changes:    changes,
			Rollback:   false,
		},
		channels,
		s.fixture4Params,
	)
	s.Require().NoError(err)

	var finishMsg *DeploymentFinishedMessage
	for err == nil && finishMsg == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishMsg = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Assert().NoError(err)
	s.Assert().NotNil(finishMsg)
	s.Assert().Equal(core.InstanceStatusUpdateFailed, finishMsg.Status)
	s.Assert().Equal([]string{
		instanceInProgressDeployFailedMessage("blueprint-instance-4", false),
	}, finishMsg.FailureReasons)
}

func (s *ContainerDeployTestSuite) Test_force_deploys_blueprint_instance_stuck_in_updating() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		"blueprint-instance-5",
		s.blueprint5Fixture.blueprintContainer,
		s.fixture5Params,
	)
	s.Require().NoError(changeStagingErr)

	// Blueprint instance 5 is in "Updating" status (from fixture setup) simulating
	// a stuck instance that crashed during an update operation.
	// With Force=true, the deployment should proceed instead of failing.
	err := s.blueprint5Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: "blueprint-instance-5",
			Changes:    changes,
			Rollback:   false,
			Force:      true, // Force bypasses the in-progress check
		},
		channels,
		s.fixture5Params,
	)
	s.Require().NoError(err)

	var finishMsg *DeploymentFinishedMessage
	for err == nil && finishMsg == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishMsg = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Assert().NoError(err)
	s.Assert().NotNil(finishMsg)
	// With force=true, the deployment should succeed (status Updated)
	// since the instance already has resources and this is an update.
	s.Assert().Equal(core.InstanceStatusUpdated, finishMsg.Status)
}

func (s *ContainerDeployTestSuite) Test_context_cancellation_drains_in_progress_items() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		s.blueprint1Fixture.blueprintContainer,
		s.fixture1Params,
	)
	s.Require().NoError(changeStagingErr)

	// Create a context that is already cancelled to simulate mid-deployment cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.blueprint1Fixture.blueprintContainer.Deploy(
		ctx,
		&DeployInput{
			InstanceID: "blueprint-instance-1",
			Changes:    changes,
			Rollback:   false,
		},
		channels,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	// Collect remaining messages - we should get a proper error message
	// due to the draining behavior, not just hang forever
	var finishMsg *DeploymentFinishedMessage
	var channelErr error
	for channelErr == nil && finishMsg == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishMsg = &msg
		case <-channels.DeploymentUpdateChan:
		case channelErr = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			channelErr = errors.New(timeoutMessage)
		}
	}

	// We should get a context cancelled error propagated
	s.Assert().Error(channelErr)
	s.Assert().ErrorIs(channelErr, context.Canceled)
}

func (s *ContainerDeployTestSuite) Test_context_timeout_during_deployment_returns_error() {
	channels := CreateDeployChannels()
	// Stage changes before deploying to get a change set derived from the test fixture
	// state without having to manually create a change set fixture.
	changes, changeStagingErr := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		s.blueprint1Fixture.blueprintContainer,
		s.fixture1Params,
	)
	s.Require().NoError(changeStagingErr)

	// Create a context that is already timed out
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	err := s.blueprint1Fixture.blueprintContainer.Deploy(
		ctx,
		&DeployInput{
			InstanceID: "blueprint-instance-1",
			Changes:    changes,
			Rollback:   false,
		},
		channels,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	// Collect messages - we should get an error due to the timeout
	// The draining behavior ensures we don't hang forever
	var finishMsg *DeploymentFinishedMessage
	var channelErr error
	for channelErr == nil && finishMsg == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishMsg = &msg
		case <-channels.DeploymentUpdateChan:
		case channelErr = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			channelErr = errors.New(timeoutMessage)
		}
	}

	// We should get a context deadline exceeded error propagated
	s.Assert().Error(channelErr)
	s.Assert().ErrorIs(channelErr, context.DeadlineExceeded)
}

func (s *ContainerDeployTestSuite) stageChanges(
	ctx context.Context,
	instanceID string,
	container BlueprintContainer,
	params core.BlueprintParams,
) (*changes.BlueprintChanges, error) {
	changeStagingChannels := createChangeStagingChannels()
	err := container.StageChanges(
		ctx,
		&StageChangesInput{
			InstanceID: instanceID,
		},
		changeStagingChannels,
		params,
	)
	if err != nil {
		return nil, err
	}

	changes := &changes.BlueprintChanges{}
	for {
		select {
		case <-changeStagingChannels.ChildChangesChan:
		case <-changeStagingChannels.LinkChangesChan:
		case <-changeStagingChannels.ResourceChangesChan:
		case changeSet := <-changeStagingChannels.CompleteChan:
			changes = &changeSet
			return changes, nil
		case err := <-changeStagingChannels.ErrChan:
			return nil, err
		case <-time.After(60 * time.Second):
			return nil, errors.New(timeoutMessage)
		}
	}
}

func blueprint1DeployParams(includeInvoices bool) core.BlueprintParams {
	environment := "production-env"
	enableOrderTableTrigger := true
	region := "us-west-2"
	deployOrdersTableToRegions := "[\"us-west-2\",\"us-east-1\",\"eu-west-1\"]"
	orderTablesConfig := `
		[
			{
				"name": "orders-us-west-2"
			},
			{
				"name": "orders-us-east-1"
			},
			{
				"name": "orders-eu-west-1"
			}
		]
	`
	blueprintVars := map[string]*core.ScalarValue{
		"environment": {
			StringValue: &environment,
		},
		"enableOrderTableTrigger": {
			BoolValue: &enableOrderTableTrigger,
		},
		"region": {
			StringValue: &region,
		},
		"deployOrdersTableToRegions": {
			StringValue: &deployOrdersTableToRegions,
		},
		"includeInvoices": {
			BoolValue: &includeInvoices,
		},
		"orderTablesConfig": {
			StringValue: &orderTablesConfig,
		},
	}

	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		blueprintVars,
	)
}

func blueprint3DeployParams() core.BlueprintParams {
	region := "us-west-2"
	environment := "production-env"

	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			"region": {
				StringValue: &region,
			},
			"environment": {
				StringValue: &environment,
			},
		},
	)
}

func fixture3Changes() *changes.BlueprintChanges {
	changes := &changes.BlueprintChanges{
		ChildChanges: map[string]changes.BlueprintChanges{
			"coreInfra": {
				ResourceChanges: map[string]provider.Changes{
					"complexResource": {
						AppliedResourceInfo: provider.ResourceInfo{
							ResourceID:   "complex-resource-id",
							ResourceName: "complexResource",
							InstanceID:   "blueprint-instance-3-child-core-infra",
							ResourceWithResolvedSubs: &provider.ResolvedResource{
								Type: &schema.ResourceTypeWrapper{
									Value: "example/complex",
								},
								Spec: &core.MappingNode{
									Fields: map[string]*core.MappingNode{
										"itemConfig": {
											Fields: map[string]*core.MappingNode{
												"endpoints": {
													Items: []*core.MappingNode{
														core.MappingNodeFromString("https://example.com/1"),
														core.MappingNodeFromString("https://example.com/2"),
													},
												},
											},
										},
									},
								},
							},
						},
						ModifiedFields: []provider.FieldChange{
							{
								FieldPath: "spec.itemConfig.endpoints[0]",
								PrevValue: core.MappingNodeFromString("https://old.example.com/1"),
								NewValue:  core.MappingNodeFromString("https://example.com/1"),
							},
						},
					},
				},
				ChildChanges: map[string]changes.BlueprintChanges{},
			},
		},
	}
	// Ensure there is a change set for the cyclic reference to ensure the max depth check
	// is tested, as an empty change set for the cyclic reference would not trigger the check.
	changes.ChildChanges["coreInfra"].ChildChanges["appInfra"] = *changes

	return changes
}

func TestContainerDeployTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerDeployTestSuite))
}
