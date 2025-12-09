package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

const (
	blueprint1InstanceID            = "blueprint-instance-1"
	blueprint1InstanceName          = "BlueprintInstance1"
	blueprint2InstanceID            = "blueprint-instance-2"
	blueprint3InstanceID            = "blueprint-instance-3"
	blueprint3ChildInstanceID       = "blueprint-instance-3-child-core-infra"
	blueprint4InstanceID            = "blueprint-instance-4"
	blueprint5InstanceID            = "blueprint-instance-5"
	blueprint6InstanceID            = "blueprint-instance-6"
	blueprint7InstanceID            = "blueprint-instance-7"
	blueprint8NoMetadataInstanceID  = "blueprint-instance-8-no-metadata"
)

const timeoutMessage = "timed out waiting for changes to be staged"

type ContainerChangeStagingTestSuite struct {
	blueprint1Container            BlueprintContainer
	blueprint2Container            BlueprintContainer
	blueprint3Container            BlueprintContainer
	blueprint4Container            BlueprintContainer
	blueprint5Container            BlueprintContainer
	blueprint6Container            BlueprintContainer
	blueprint7Container            BlueprintContainer
	blueprint8NoMetadataContainer  BlueprintContainer
	suite.Suite
}

func (s *ContainerChangeStagingTestSuite) SetupSuite() {
	stateContainer := memstate.NewMemoryStateContainer()
	err := s.populateCurrentState(stateContainer)
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

	blueprint1Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint1.yml",
		baseBlueprintParams(),
	)
	s.Require().NoError(err)
	s.blueprint1Container = blueprint1Container

	blueprint2Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint2.yml",
		createBlueprint2Params(),
	)
	s.Require().NoError(err)
	s.blueprint2Container = blueprint2Container

	blueprint3Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint3.yml",
		baseBlueprintParams(),
	)
	s.Require().NoError(err)
	s.blueprint3Container = blueprint3Container

	blueprint4Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint4.yml",
		createBlueprint4Params(),
	)
	s.Require().NoError(err)
	s.blueprint4Container = blueprint4Container

	blueprint5Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint5.yml",
		baseBlueprintParams(),
	)
	s.Require().NoError(err)
	s.blueprint5Container = blueprint5Container

	blueprint6Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint6.yml",
		baseBlueprintParams(),
	)
	s.Require().NoError(err)
	s.blueprint6Container = blueprint6Container

	blueprint7Container, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint7.yml",
		baseBlueprintParams(),
	)
	s.Require().NoError(err)
	s.blueprint7Container = blueprint7Container

	blueprint8NoMetadataContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint8-no-metadata.yml",
		baseBlueprintParams(),
	)
	s.Require().NoError(err)
	s.blueprint8NoMetadataContainer = blueprint8NoMetadataContainer
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_to_existing_blueprint_instance() {
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint1Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint1InstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	resourceChangeMessages := []ResourceChangesMessage{}
	childChangeMessages := []ChildChangesMessage{}
	linkChangeMessages := []LinkChangesMessage{}
	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil &&
		(fullChangeSet == nil ||
			len(resourceChangeMessages) < 6 ||
			len(childChangeMessages) < 1 ||
			len(linkChangeMessages) < 3) {
		select {
		case msg := <-channels.ResourceChangesChan:
			resourceChangeMessages = append(resourceChangeMessages, msg)
		case msg := <-channels.ChildChangesChan:
			childChangeMessages = append(childChangeMessages, msg)
		case msg := <-channels.LinkChangesChan:
			linkChangeMessages = append(linkChangeMessages, msg)
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	err = testhelpers.Snapshot(normaliseBlueprintChanges(fullChangeSet))
	s.Require().NoError(err)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_to_existing_blueprint_instance_by_name() {
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint1Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceName: blueprint1InstanceName,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	resourceChangeMessages := []ResourceChangesMessage{}
	childChangeMessages := []ChildChangesMessage{}
	linkChangeMessages := []LinkChangesMessage{}
	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil &&
		(fullChangeSet == nil ||
			len(resourceChangeMessages) < 6 ||
			len(childChangeMessages) < 1 ||
			len(linkChangeMessages) < 3) {
		select {
		case msg := <-channels.ResourceChangesChan:
			resourceChangeMessages = append(resourceChangeMessages, msg)
		case msg := <-channels.ChildChangesChan:
			childChangeMessages = append(childChangeMessages, msg)
		case msg := <-channels.LinkChangesChan:
			linkChangeMessages = append(linkChangeMessages, msg)
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	err = testhelpers.Snapshot(normaliseBlueprintChanges(fullChangeSet))
	s.Require().NoError(err)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_for_a_new_blueprint_instance() {
	channels := createChangeStagingChannels()
	params := createBlueprint2Params()

	err := s.blueprint2Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint2InstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	resourceChangeMessages := []ResourceChangesMessage{}
	childChangeMessages := []ChildChangesMessage{}
	linkChangeMessages := []LinkChangesMessage{}
	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil &&
		(fullChangeSet == nil ||
			len(resourceChangeMessages) < 6 ||
			len(childChangeMessages) < 1 ||
			len(linkChangeMessages) < 5) {
		select {
		case msg := <-channels.ResourceChangesChan:
			resourceChangeMessages = append(resourceChangeMessages, msg)
		case msg := <-channels.ChildChangesChan:
			childChangeMessages = append(childChangeMessages, msg)
		case msg := <-channels.LinkChangesChan:
			linkChangeMessages = append(linkChangeMessages, msg)
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	err = testhelpers.Snapshot(normaliseBlueprintChanges(fullChangeSet))
	s.Require().NoError(err)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_for_destroying_a_blueprint_instance() {
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint5Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint5InstanceID,
			Destroy:    true,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil && fullChangeSet == nil {
		select {
		// For destroy operations, we only expect to see the complete message
		// as resources can be efficiently collected synchronously in one go based on the
		// current persisted state of the instance.
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	err = testhelpers.Snapshot(normaliseBlueprintChanges(fullChangeSet))
	s.Require().NoError(err)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_fails_for_cyclic_dependency_between_blueprint_instances() {
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint3Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint3InstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	for err == nil {
		select {
		case <-channels.ResourceChangesChan:
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case <-channels.CompleteChan:
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

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_fails_when_max_blueprint_depth_is_exceeded() {
	channels := createChangeStagingChannels()
	params := createBlueprint4Params()

	err := s.blueprint4Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint4InstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	for err == nil {
		select {
		case <-channels.ResourceChangesChan:
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case <-channels.CompleteChan:
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Assert().Error(err)
	runErr, isRunErr := err.(*bperrors.RunError)
	s.Assert().True(isRunErr)
	s.Assert().Equal(
		ErrorReasonCodeMaxBlueprintDepthExceeded,
		runErr.ReasonCode,
	)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_when_removed_resource_has_dependents_still_in_blueprint() {
	// The expected behaviour is that the elements that previously depended on the removed resource
	// must be recreated based on the assumption that if the resource is removed and the new version
	// of the blueprint has been successfully loaded, then the dependents
	// that remain in the blueprint must no longer be depend on the removed resource.
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint6Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint6InstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	resourceChangeMessages := []ResourceChangesMessage{}
	childChangeMessages := []ChildChangesMessage{}
	linkChangeMessages := []LinkChangesMessage{}
	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil &&
		(fullChangeSet == nil ||
			len(resourceChangeMessages) < 7 ||
			len(childChangeMessages) < 1 ||
			len(linkChangeMessages) < 3) {
		select {
		case msg := <-channels.ResourceChangesChan:
			resourceChangeMessages = append(resourceChangeMessages, msg)
		case msg := <-channels.ChildChangesChan:
			childChangeMessages = append(childChangeMessages, msg)
		case msg := <-channels.LinkChangesChan:
			linkChangeMessages = append(linkChangeMessages, msg)
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	err = testhelpers.Snapshot(normaliseBlueprintChanges(fullChangeSet))
	s.Require().NoError(err)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_when_removed_child_has_dependents_still_in_blueprint() {
	// The expected behaviour is that the elements that previously depended on the removed child
	// must be recreated based on the assumption that if the child blueprint is removed and the
	// new version of the host blueprint has been successfully loaded, then the dependents
	// that remain in the blueprint must no longer be depend on the removed resource.
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint7Container.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint7InstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	resourceChangeMessages := []ResourceChangesMessage{}
	childChangeMessages := []ChildChangesMessage{}
	linkChangeMessages := []LinkChangesMessage{}
	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil &&
		(fullChangeSet == nil ||
			len(resourceChangeMessages) < 6 ||
			len(childChangeMessages) < 2 ||
			len(linkChangeMessages) < 3) {
		select {
		case msg := <-channels.ResourceChangesChan:
			resourceChangeMessages = append(resourceChangeMessages, msg)
		case msg := <-channels.ChildChangesChan:
			childChangeMessages = append(childChangeMessages, msg)
		case msg := <-channels.LinkChangesChan:
			linkChangeMessages = append(linkChangeMessages, msg)
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	err = testhelpers.Snapshot(normaliseBlueprintChanges(fullChangeSet))
	s.Require().NoError(err)
}

func (s *ContainerChangeStagingTestSuite) Test_stage_changes_for_blueprint_without_metadata() {
	// This test verifies that staging changes works correctly when the blueprint
	// has no metadata section. Previously this would cause a nil pointer dereference
	// in ExtractBlueprintChanges when accessing MetadataChanges.
	channels := createChangeStagingChannels()
	params := baseBlueprintParams()

	err := s.blueprint8NoMetadataContainer.StageChanges(
		context.Background(),
		&StageChangesInput{
			InstanceID: blueprint8NoMetadataInstanceID,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	resourceChangeMessages := []ResourceChangesMessage{}
	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil &&
		(fullChangeSet == nil ||
			len(resourceChangeMessages) < 1) {
		select {
		case msg := <-channels.ResourceChangesChan:
			resourceChangeMessages = append(resourceChangeMessages, msg)
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		case <-time.After(60 * time.Second):
			err = errors.New(timeoutMessage)
		}
	}
	s.Require().NoError(err)

	// Verify that MetadataChanges is properly initialized (empty, not nil)
	s.Assert().NotNil(fullChangeSet)
	s.Assert().Empty(fullChangeSet.MetadataChanges.NewFields)
	s.Assert().Empty(fullChangeSet.MetadataChanges.ModifiedFields)
	s.Assert().Empty(fullChangeSet.MetadataChanges.RemovedFields)
	s.Assert().Empty(fullChangeSet.MetadataChanges.UnchangedFields)
}

func (s *ContainerChangeStagingTestSuite) populateCurrentState(stateContainer state.Container) error {

	err := s.populateBlueprintCurrentState(stateContainer, blueprint1InstanceID, 1)
	if err != nil {
		return err
	}

	err = s.populateBlueprint3CyclicCurrentState(stateContainer)
	if err != nil {
		return err
	}

	err = s.populateBlueprintCurrentState(stateContainer, blueprint5InstanceID, 5)
	if err != nil {
		return err
	}

	err = s.populateBlueprintCurrentState(stateContainer, blueprint6InstanceID, 6)
	if err != nil {
		return err
	}

	return s.populateBlueprint7CurrentStateWithRemovedChild(stateContainer)
}

func (s *ContainerChangeStagingTestSuite) populateBlueprintCurrentState(
	stateContainer state.Container,
	instanceID string,
	blueprintNo int,
) error {
	blueprintCurrentState, err := internal.LoadInstanceState(
		fmt.Sprintf(
			"__testdata/container/change-staging/current-state/blueprint%d.json",
			blueprintNo,
		),
	)
	if err != nil {
		return err
	}
	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprintCurrentState,
	)
	if err != nil {
		return err
	}

	blueprintChildCurrentState, err := internal.LoadInstanceState(
		fmt.Sprintf(
			"__testdata/container/change-staging/current-state/blueprint%d-child-core-infra.json",
			blueprintNo,
		),
	)
	if err != nil {
		return err
	}

	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprintChildCurrentState,
	)
	if err != nil {
		return err
	}

	return stateContainer.Children().Attach(
		context.Background(),
		instanceID,
		blueprintChildCurrentState.InstanceID,
		"coreInfra",
	)
}

func (s *ContainerChangeStagingTestSuite) populateBlueprint3CyclicCurrentState(
	stateContainer state.Container,
) error {
	blueprint3CurrentState, err := internal.LoadInstanceState(
		"__testdata/container/change-staging/current-state/blueprint3.json",
	)
	if err != nil {
		return err
	}
	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprint3CurrentState,
	)
	if err != nil {
		return err
	}

	blueprint3ChildCurrentState, err := internal.LoadInstanceState(
		"__testdata/container/change-staging/current-state/blueprint3-child-core-infra.json",
	)
	if err != nil {
		return err
	}

	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprint3ChildCurrentState,
	)
	if err != nil {
		return err
	}

	// Creates cycle between blueprint1 and blueprint3
	err = stateContainer.Children().Attach(
		context.Background(),
		blueprint3InstanceID,
		blueprint3ChildCurrentState.InstanceID,
		"coreInfra",
	)
	if err != nil {
		return err
	}

	return stateContainer.Children().Attach(
		context.Background(),
		blueprint3ChildInstanceID,
		blueprint3CurrentState.InstanceID,
		"ordersApi",
	)
}

func (s *ContainerChangeStagingTestSuite) populateBlueprint7CurrentStateWithRemovedChild(
	stateContainer state.Container,
) error {
	err := s.populateBlueprintCurrentState(stateContainer, blueprint7InstanceID, 7)
	if err != nil {
		return err
	}

	blueprint7ChildToBeRemoved, err := internal.LoadInstanceState(
		"__testdata/container/change-staging/current-state/blueprint7-child-networking.json",
	)
	if err != nil {
		return err
	}

	err = stateContainer.Instances().Save(
		context.Background(),
		*blueprint7ChildToBeRemoved,
	)
	if err != nil {
		return err
	}

	return stateContainer.Children().Attach(
		context.Background(),
		blueprint7InstanceID,
		blueprint7ChildToBeRemoved.InstanceID,
		"networking",
	)
}

func baseBlueprintParams() core.BlueprintParams {
	environment := "production-env"
	enableOrderTableTrigger := true
	region := "us-west-2"
	deployOrdersTableToRegions := "[\"us-west-2\",\"us-east-1\"]"
	relatedInfo := "[{\"id\":\"test-info-1\"},{\"id\":\"test-info-2\"}]"
	includeInvoices := false
	orderTablesConfig := "[{\"name\":\"orders-1\"},{\"name\":\"orders-2\"}]"
	blueprintVars := map[string]*core.ScalarValue{
		"environment": {
			StringValue: &environment,
		},
		"region": {
			StringValue: &region,
		},
		"deployOrdersTableToRegions": {
			StringValue: &deployOrdersTableToRegions,
		},
		"enableOrderTableTrigger": {
			BoolValue: &enableOrderTableTrigger,
		},
		"relatedInfo": {
			StringValue: &relatedInfo,
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

func createBlueprint2Params() core.BlueprintParams {
	baseParams := baseBlueprintParams()
	includeInvoices := true
	return baseParams.WithBlueprintVariables(
		map[string]*core.ScalarValue{
			"includeInvoices": {
				BoolValue: &includeInvoices,
			},
		},
		true,
	)
}

func createBlueprint4Params() core.BlueprintParams {
	return createBlueprint2Params()
}

func TestContainerChangesStagingTestSuite(t *testing.T) {
	suite.Run(t, new(ContainerChangeStagingTestSuite))
}
