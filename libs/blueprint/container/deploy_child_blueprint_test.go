package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

type ChildBlueprintDeployerTestSuite struct {
	suite.Suite
	fixtures map[int]*childBlueprintDeployerFixture
	logger   core.Logger
}

func (s *ChildBlueprintDeployerTestSuite) SetupTest() {
	fixtureInputs := s.fixtureInputs()
	s.fixtures = map[int]*childBlueprintDeployerFixture{}
	logger := core.NewNopLogger()
	s.logger = logger

	for _, fixtureInfo := range fixtureInputs {
		fixture, err := s.createFixture(
			fixtureInfo.number,
			fixtureInfo.childName,
			fmt.Sprintf("child-deploy-test--blueprint-instance-%d", fixtureInfo.number),
			fixtureInfo.failure,
		)
		s.Require().NoError(err)
		s.fixtures[fixtureInfo.number] = fixture
	}
}

func (s *ChildBlueprintDeployerTestSuite) Test_deploys_a_new_child_blueprint() {
	s.runDeployTest(
		s.fixtures[1],
		/* isNew */ true,
	)
}

func (s *ChildBlueprintDeployerTestSuite) runDeployTest(
	fixture *childBlueprintDeployerFixture,
	isNew bool,
) {
	ctx := context.Background()
	channels := CreateDeployChannels()
	stateContainer := internal.NewMemoryStateContainer()
	// Persist parent state so it can be accessed to get the instance name
	// during child deployment.
	err := stateContainer.Instances().Save(
		ctx,
		*fixture.parentInstanceStateSnapshot,
	)
	s.Require().NoError(err)

	deployState := NewDefaultDeploymentState()
	deployer := NewDefaultChildBlueprintDeployer(
		&dynamicIncludeSubstitutionResolver{
			resolvedIncludeFactory: s.createResolvedInclude,
		},
		newFSChildResolver(),
		s.createChildBlueprintLoader(fixture.deployEventSequence),
		stateContainer,
	)
	go deployer.Deploy(
		ctx,
		fixture.parentInstanceID,
		fixture.parentInstanceTreePath,
		fixture.includeTreePath,
		fixture.childNode,
		fixture.changes,
		&DeployContext{
			Rollback:              false,
			Destroying:            false,
			Channels:              channels,
			State:                 deployState,
			InstanceStateSnapshot: fixture.parentInstanceStateSnapshot,
			ParamOverrides:        deployLinkParams(),
			ResourceProviders:     map[string]provider.Provider{},
			ResourceTemplates:     map[string]string{},
			Logger:                s.logger,
		},
	)

	childDeployUpdateMessages := []ChildDeployUpdateMessage{}
	// Resource and link update messages should be passed through to the parent deployer
	// for deeper visibility into the deployment process to the end user without extra navigation
	// that requires further network requests.
	resourceDeployUpdateMessages := []ResourceDeployUpdateMessage{}
	linkDeployUpdateMessages := []LinkDeployUpdateMessage{}
	finishedMessage := (*ChildDeployUpdateMessage)(nil)
	for err == nil &&
		finishedMessage == nil {
		select {
		case msg := <-channels.ChildUpdateChan:
			childDeployUpdateMessages = append(childDeployUpdateMessages, msg)
			if isChildDeployFinishedMessage(msg, false, isNew) {
				finishedMessage = &msg
			}
		case msg := <-channels.ResourceUpdateChan:
			resourceDeployUpdateMessages = append(resourceDeployUpdateMessages, msg)
		case msg := <-channels.LinkUpdateChan:
			linkDeployUpdateMessages = append(linkDeployUpdateMessages, msg)
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	actualMessages := &actualMessages{
		resourceDeployUpdateMessages: resourceDeployUpdateMessages,
		childDeployUpdateMessages:    childDeployUpdateMessages,
		linkDeployUpdateMessages:     linkDeployUpdateMessages,
		deploymentUpdateMessages:     []DeploymentUpdateMessage{},
	}
	assertDeployMessageOrder(actualMessages, fixture.expectedMessages, &s.Suite)
	// The child deployer is not responsible for persisting the state of the child instance,
	// this is taken care of the blueprint container implementation that is used to deploy
	// the child blueprint under the hood.
	// For this reason, there is no need to assert the state of the child instance
	// after deployment here, the important thing for these tests is to ensure that each
	// of the expected messages are emitted in the correct order for the child blueprint
	// and each of its components.
}

func (s *ChildBlueprintDeployerTestSuite) fixtureInputs() []*childBlueprintDeployerFixtureInfo {
	return []*childBlueprintDeployerFixtureInfo{
		{
			number:    1,
			childName: "coreInfra",
			failure:   false,
		},
	}
}

func (s *ChildBlueprintDeployerTestSuite) createResolvedInclude(
	include *schema.Include,
) *subengine.ResolvedInclude {
	return &subengine.ResolvedInclude{
		// The only important thing for the purpose of these tests is that the path is correctly resolved
		// so that the child blueprint can be loaded.
		Path: core.MappingNodeFromString(*include.Path.Values[0].StringValue),
	}
}

func (s *ChildBlueprintDeployerTestSuite) createChildBlueprintLoader(
	deployEventSequence []*DeployEvent,
) ChildBlueprintLoaderFactory {
	return func(
		derivedFromTemplate []string,
		resourceTemplates map[string]string,
	) Loader {
		return &stubBlueprintContainerLoader{
			deployEventSequence: deployEventSequence,
		}
	}
}

func (s *ChildBlueprintDeployerTestSuite) createFixture(
	childBlueprintFixtureNo int,
	childName string,
	instanceID string,
	failure bool,
) (*childBlueprintDeployerFixture, error) {
	blueprintIncludeFilePath := fmt.Sprintf(
		"__testdata/container/child-blueprint-deployer/includes/%d--%s.json",
		childBlueprintFixtureNo,
		childName,
	)
	blueprintInclude, err := loadBlueprintIncludeFromFilePath(blueprintIncludeFilePath)
	if err != nil {
		return nil, err
	}

	childNode := &refgraph.ReferenceChainNode{
		ElementName:  core.ChildElementID(childName),
		Element:      blueprintInclude,
		References:   []*refgraph.ReferenceChainNode{},
		ReferencedBy: []*refgraph.ReferenceChainNode{},
		Paths:        []string{},
		Tags:         []string{},
	}

	expectedMessagesFilePath := fmt.Sprintf(
		"__testdata/container/child-blueprint-deployer/expected-messages/%d.json",
		childBlueprintFixtureNo,
	)
	expectedMessages, err := loadExpectedMessagesFromFile(expectedMessagesFilePath)
	if err != nil {
		return nil, err
	}

	deployEventSequenceFilePath := fmt.Sprintf(
		"__testdata/container/child-blueprint-deployer/deploy-event-sequences/%d.json",
		childBlueprintFixtureNo,
	)
	deployEventSequence, err := loadDeployEventSequenceFromFile(deployEventSequenceFilePath)
	if err != nil {
		return nil, err
	}

	parentInstanceStateFilePath := fmt.Sprintf(
		"__testdata/container/child-blueprint-deployer/current-parent-state/%d.json",
		childBlueprintFixtureNo,
	)
	parentInstanceStateSnapshot, err := internal.LoadInstanceState(
		parentInstanceStateFilePath,
	)
	if err != nil {
		return nil, err
	}

	changesFilePath := fmt.Sprintf(
		"__testdata/container/child-blueprint-deployer/changes/%d.json",
		childBlueprintFixtureNo,
	)
	changes, err := loadBlueprintChangesFromFile(changesFilePath)
	if err != nil {
		return nil, err
	}

	parentInstanceID := fmt.Sprintf("%s--parent", instanceID)
	includeTreePath := fmt.Sprintf("include.%s", childName)
	return &childBlueprintDeployerFixture{
		parentInstanceID:            parentInstanceID,
		parentInstanceTreePath:      parentInstanceID,
		includeTreePath:             includeTreePath,
		childNode:                   childNode,
		childInstanceID:             instanceID,
		parentInstanceStateSnapshot: parentInstanceStateSnapshot,
		changes:                     changes,
		deployEventSequence:         deployEventSequence,
		expectedMessages:            expectedMessages,
		failure:                     failure,
	}, nil
}

func loadBlueprintIncludeFromFilePath(
	schemaIncludeFilePath string,
) (*schema.Include, error) {
	schemaIncludeFileBytes, err := os.ReadFile(schemaIncludeFilePath)
	if err != nil {
		return nil, err
	}

	schemaInclude := &schema.Include{}
	err = json.Unmarshal(schemaIncludeFileBytes, schemaInclude)
	if err != nil {
		return nil, err
	}

	return schemaInclude, nil
}

func loadDeployEventSequenceFromFile(
	deployEventSequenceFilePath string,
) ([]*DeployEvent, error) {
	deployEventSequenceFileBytes, err := os.ReadFile(deployEventSequenceFilePath)
	if err != nil {
		return nil, err
	}

	deployEventSequence := []*DeployEvent{}
	err = json.Unmarshal(deployEventSequenceFileBytes, &deployEventSequence)
	if err != nil {
		return nil, err
	}

	return deployEventSequence, nil
}

type childBlueprintDeployerFixtureInfo struct {
	number    int
	childName string
	failure   bool
}

type childBlueprintDeployerFixture struct {
	parentInstanceID            string
	parentInstanceTreePath      string
	parentInstanceStateSnapshot *state.InstanceState
	includeTreePath             string
	childNode                   *refgraph.ReferenceChainNode
	childInstanceID             string
	changes                     *changes.BlueprintChanges
	deployEventSequence         []*DeployEvent
	expectedMessages            *expectedMessages
	failure                     bool
}

func isChildDeployFinishedMessage(
	msg ChildDeployUpdateMessage,
	rollback bool,
	isNew bool,
) bool {
	if rollback && isNew {
		// In the context of rolling back a child instance,
		// deploying a child instance is reversing the destroy operation.
		return (msg.Status == core.InstanceStatusDestroyRollbackFailed ||
			msg.Status == core.InstanceStatusDestroyRollbackComplete)
	}

	if rollback && !isNew {
		return (msg.Status == core.InstanceStatusUpdateRollbackFailed ||
			msg.Status == core.InstanceStatusUpdateRollbackComplete)
	}

	if !isNew {
		return (msg.Status == core.InstanceStatusUpdateFailed ||
			msg.Status == core.InstanceStatusUpdated)
	}

	return (msg.Status == core.InstanceStatusDeployFailed ||
		msg.Status == core.InstanceStatusDeployed)
}

func TestChildBlueprintDeployerTestSuite(t *testing.T) {
	suite.Run(t, new(ChildBlueprintDeployerTestSuite))
}
