package container

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type ContainerDestroyTestSuite struct {
	blueprint1Fixture blueprintDeployFixture
	blueprint2Fixture blueprintDeployFixture
	blueprint3Fixture blueprintDeployFixture
	blueprint4Fixture blueprintDeployFixture
	stateContainer    state.Container
	suite.Suite
}

func (s *ContainerDestroyTestSuite) SetupTest() {
	stateContainer := memstate.NewMemoryStateContainer()
	s.stateContainer = stateContainer
	fixtureInstances := []int{1, 2, 3, 4}
	err := populateCurrentState(fixtureInstances, stateContainer, "destroy")
	s.Require().NoError(err)

	providers := map[string]provider.Provider{
		"aws": newTestAWSProvider(
			/* alwaysStabilise */ false,
			/* skipRetryFailuresForLinkNames */ []string{},
			stateContainer,
		),
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
	logger := core.NewNopLogger()
	loader := NewDefaultLoader(
		providers,
		specTransformers,
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(logger),
	)

	s.blueprint1Fixture, err = createBlueprintDeployFixture(
		"destroy",
		1,
		loader,
		baseBlueprintParams(),
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)

	s.blueprint2Fixture, err = createBlueprintDeployFixture(
		"destroy",
		2,
		loader,
		baseBlueprintParams(),
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)

	s.blueprint3Fixture, err = createBlueprintDeployFixture(
		"destroy",
		3,
		loader,
		baseBlueprintParams(),
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)

	s.blueprint4Fixture, err = createBlueprintDeployFixture(
		"destroy",
		4,
		loader,
		baseBlueprintParams(),
		schema.YAMLSpecFormat,
	)
	s.Require().NoError(err)
}

func (s *ContainerDestroyTestSuite) Test_destroys_blueprint_instance_with_child_blueprint() {
	channels := CreateDeployChannels()
	s.blueprint1Fixture.blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceID: "blueprint-instance-1",
			Changes:    blueprint1RemovalChanges(),
			Rollback:   false,
		},
		channels,
		blueprintDestroyParams(),
	)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	deploymentUpdateMessages := []DeploymentUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	var err error
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

	_, err = s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-1")
	s.Assert().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *ContainerDestroyTestSuite) Test_destroys_blueprint_instance_with_child_blueprint_by_name() {
	channels := CreateDeployChannels()
	s.blueprint1Fixture.blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceName: "BlueprintInstance1",
			Changes:      blueprint1RemovalChanges(),
			Rollback:     false,
		},
		channels,
		blueprintDestroyParams(),
	)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	deploymentUpdateMessages := []DeploymentUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	var err error
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

	_, err = s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-1")
	s.Assert().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *ContainerDestroyTestSuite) Test_destroys_blueprint_instance_as_deployment_rollback() {
	channels := CreateDeployChannels()
	s.blueprint2Fixture.blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceID: "blueprint-instance-2",
			Changes:    blueprint2RemovalChanges(),
			Rollback:   true,
		},
		channels,
		blueprintDestroyParams(),
	)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	deploymentUpdateMessages := []DeploymentUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	var err error
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

	_, err = s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-2")
	s.Assert().Error(err)
	stateErr, isStateErr := err.(*state.Error)
	s.Assert().True(isStateErr)
	s.Assert().Equal(state.ErrInstanceNotFound, stateErr.Code)
}

func (s *ContainerDestroyTestSuite) Test_fails_to_destroys_blueprint_instance_due_to_terminal_resource_impl_error() {
	channels := CreateDeployChannels()
	s.blueprint3Fixture.blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceID: "blueprint-instance-3",
			Changes:    blueprint3RemovalChanges(),
			Rollback:   false,
		},
		channels,
		blueprintDestroyParams(),
	)

	resourceUpdateMessages := []ResourceDeployUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	var err error
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.ResourceUpdateChan:
			resourceUpdateMessages = append(resourceUpdateMessages, msg)
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: resourceUpdateMessages,
		childDeployUpdateMessages:    []ChildDeployUpdateMessage{},
		linkDeployUpdateMessages:     []LinkDeployUpdateMessage{},
		deploymentUpdateMessages:     []DeploymentUpdateMessage{},
		finishedMessage:              finishedMessage,
	}
	assertDeployMessageOrder(actualMessages, s.blueprint3Fixture.expected, &s.Suite)

	instance, err := s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-3")
	s.Assert().NoError(err)
	s.Assert().Equal("blueprint-instance-3", instance.InstanceID)
}

func (s *ContainerDestroyTestSuite) Test_fails_to_destroys_blueprint_instance_due_to_terminal_link_impl_error() {
	channels := CreateDeployChannels()
	s.blueprint4Fixture.blueprintContainer.Destroy(
		context.Background(),
		&DestroyInput{
			InstanceID: "blueprint-instance-4",
			Changes:    blueprint4RemovalChanges(),
			Rollback:   false,
		},
		channels,
		blueprintDestroyParams(),
	)

	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	finishedMessage := (*DeploymentFinishedMessage)(nil)
	var err error
	for err == nil &&
		finishedMessage == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case msg := <-channels.LinkUpdateChan:
			linkDeployUpdateMessages = append(linkDeployUpdateMessages, msg)
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: []ResourceDeployUpdateMessage{},
		childDeployUpdateMessages:    []ChildDeployUpdateMessage{},
		linkDeployUpdateMessages:     linkDeployUpdateMessages,
		deploymentUpdateMessages:     []DeploymentUpdateMessage{},
		finishedMessage:              finishedMessage,
	}
	assertDeployMessageOrder(actualMessages, s.blueprint4Fixture.expected, &s.Suite)

	instance, err := s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-4")
	s.Assert().NoError(err)
	s.Assert().Equal("blueprint-instance-4", instance.InstanceID)
}

func blueprint1RemovalChanges() *changes.BlueprintChanges {
	return &changes.BlueprintChanges{
		RemovedResources: []string{
			"ordersTable_0",
			"ordersTable_1",
			"saveOrderFunction",
			"invoicesTable",
		},
		RemovedChildren: []string{
			"coreInfra",
		},
		RemovedLinks: []string{
			"saveOrderFunction::ordersTable_0",
			"saveOrderFunction::ordersTable_1",
		},
		RemovedExports: []string{
			"environment",
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"coreInfra": {
				RemovedResources: []string{
					"complexResource",
				},
				RemovedChildren: []string{},
				RemovedLinks:    []string{},
				RemovedExports:  []string{},
			},
		},
	}
}

func blueprint2RemovalChanges() *changes.BlueprintChanges {
	return &changes.BlueprintChanges{
		RemovedResources: []string{
			"ordersTable_0",
			"ordersTable_1",
			"preprocessOrderFunction",
			"invoicesTable",
		},
		RemovedChildren: []string{
			"coreInfra",
		},
		RemovedLinks: []string{
			"preprocessOrderFunction::ordersTable_0",
			"preprocessOrderFunction::ordersTable_1",
		},
		RemovedExports: []string{
			"environment",
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"coreInfra": {
				RemovedResources: []string{
					"complexResource",
				},
				RemovedChildren: []string{},
				RemovedLinks:    []string{},
				RemovedExports:  []string{},
			},
		},
	}
}

func blueprint3RemovalChanges() *changes.BlueprintChanges {
	return &changes.BlueprintChanges{
		RemovedResources: []string{
			"ordersTable_0",
			"ordersTable_1",
			"failingOrderFunction",
			"invoicesTable",
		},
		RemovedChildren: []string{
			"coreInfra",
		},
		RemovedLinks: []string{
			"failingOrderFunction::ordersTable_0",
			"failingOrderFunction::ordersTable_1",
		},
		RemovedExports: []string{
			"environment",
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"coreInfra": {
				RemovedResources: []string{
					"complexResource",
				},
				RemovedChildren: []string{},
				RemovedLinks:    []string{},
				RemovedExports:  []string{},
			},
		},
	}
}

func blueprint4RemovalChanges() *changes.BlueprintChanges {
	return &changes.BlueprintChanges{
		RemovedResources: []string{
			"ordersTableFailingLink_0",
			"ordersTable_1",
			"preprocessOrderFunction",
			"invoicesTable",
		},
		RemovedChildren: []string{
			"coreInfra",
		},
		RemovedLinks: []string{
			"preprocessOrderFunction::ordersTableFailingLink_0",
			"preprocessOrderFunction::ordersTable_1",
		},
		RemovedExports: []string{
			"environment",
		},
		ChildChanges: map[string]changes.BlueprintChanges{
			"coreInfra": {
				RemovedResources: []string{
					"complexResource",
				},
				RemovedChildren: []string{},
				RemovedLinks:    []string{},
				RemovedExports:  []string{},
			},
		},
	}
}

func blueprintDestroyParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

func TestContainerDestroyTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerDestroyTestSuite))
}
