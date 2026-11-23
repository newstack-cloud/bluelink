package container

import (
	"context"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	bperrors "github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Without RequireNewInstance, a name with no ID is resolved to the instance holding it and
// deployed onto it, so a caller that means to create would update another instance instead.
func (s *ContainerDeployTestSuite) Test_a_new_instance_is_not_deployed_under_a_name_in_use() {
	changes, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		s.blueprint2Fixture.blueprintContainer,
		s.fixture2Params,
	)
	s.Require().NoError(err)

	err = s.blueprint2Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName:       "BlueprintInstance1",
			RequireNewInstance: true,
			Changes:            changes,
		},
		CreateDeployChannels(),
		s.fixture2Params,
	)
	s.Assert().ErrorIs(err, state.ErrInstanceAlreadyExists)

	existing, err := s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-1")
	s.Require().NoError(err)
	s.Assert().Equal(core.InstanceStatusDeployed, existing.Status)
}

// Both deploys get past any lookup of the name before either has created its instance, so
// only the atomic create can tell them apart.
func (s *ContainerDeployTestSuite) Test_concurrent_new_instances_with_the_same_name_only_one_is_created() {
	changes, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		s.blueprint2Fixture.blueprintContainer,
		s.fixture2Params,
	)
	s.Require().NoError(err)

	startBarrier := make(chan struct{})
	var deploysDone sync.WaitGroup
	deployErrs := make([]error, 2)
	channels := []*DeployChannels{CreateDeployChannels(), CreateDeployChannels()}
	for i := range deployErrs {
		deploysDone.Add(1)
		go func() {
			defer deploysDone.Done()
			<-startBarrier
			deployErrs[i] = s.blueprint2Fixture.blueprintContainer.Deploy(
				context.Background(),
				&DeployInput{
					InstanceName:       "ConcurrentNewInstance",
					RequireNewInstance: true,
					Changes:            changes,
				},
				channels[i],
				s.fixture2Params,
			)
		}()
	}
	close(startBarrier)
	deploysDone.Wait()

	winner := 0
	if deployErrs[0] != nil {
		winner = 1
	}
	s.Require().NoError(deployErrs[winner])
	s.Assert().ErrorIs(deployErrs[1-winner], state.ErrInstanceAlreadyExists)

	finished, err := collectDeployFinish(channels[winner])
	s.Require().NoError(err)
	instanceID, err := s.stateContainer.Instances().LookupIDByName(
		context.Background(),
		"ConcurrentNewInstance",
	)
	s.Require().NoError(err)
	s.Assert().Equal(finished.InstanceID, instanceID)
}

// A deploy given an ID would otherwise deploy to that ID's instance, which is the takeover
// RequireNewInstance is there to prevent.
func (s *ContainerDeployTestSuite) Test_a_new_instance_cannot_be_required_for_a_given_instance_id() {
	changes, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		s.blueprint2Fixture.blueprintContainer,
		s.fixture2Params,
	)
	s.Require().NoError(err)

	err = s.blueprint2Fixture.blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID:         "blueprint-instance-1",
			InstanceName:       "BlueprintInstance1",
			RequireNewInstance: true,
			Changes:            changes,
		},
		CreateDeployChannels(),
		s.fixture2Params,
	)
	runErr, isRunErr := err.(*bperrors.RunError)
	s.Require().True(isRunErr, "expected a run error, got %v", err)
	s.Assert().Equal(ErrorReasonCodeInstanceIDWithRequireNewInstance, runErr.ReasonCode)

	existing, err := s.stateContainer.Instances().Get(context.Background(), "blueprint-instance-1")
	s.Require().NoError(err)
	s.Assert().Equal(core.InstanceStatusDeployed, existing.Status)
}

// A deploy by name alone whose lookup missed the name, because another deploy created the
// instance after the lookup, must not deploy under its generated ID with no instance record.
func (s *ContainerDeployTestSuite) Test_a_deploy_by_name_that_loses_the_create_race_is_rejected() {
	stateContainer := s.stateContainerWithNameHolder()
	blueprintContainer, changes, params := s.loadNewInstanceDeploy(
		&lookupMissesStateContainer{Container: stateContainer},
	)

	err := blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{InstanceName: nameHolder.InstanceName, Changes: changes},
		CreateDeployChannels(),
		params,
	)
	s.Assert().ErrorIs(err, state.ErrInstanceAlreadyExists)
}

// An ID with no record names a new instance, so a name conflict on creating it can only be
// another instance's, and the deploy must not go ahead as if the ID's record existed.
func (s *ContainerDeployTestSuite) Test_a_deploy_to_an_unknown_id_under_a_name_in_use_is_rejected() {
	stateContainer := s.stateContainerWithNameHolder()
	blueprintContainer, changes, params := s.loadNewInstanceDeploy(stateContainer)

	err := blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID:   "instance-without-a-record",
			InstanceName: nameHolder.InstanceName,
			Changes:      changes,
		},
		CreateDeployChannels(),
		params,
	)
	s.Assert().ErrorIs(err, state.ErrInstanceAlreadyExists)

	_, err = stateContainer.Instances().Get(context.Background(), "instance-without-a-record")
	s.Assert().True(state.IsInstanceNotFound(err))
}

var nameHolder = state.InstanceState{
	InstanceID:   "instance-holding-the-name",
	InstanceName: "NameInUse",
	Status:       core.InstanceStatusDeployed,
}

func (s *ContainerDeployTestSuite) stateContainerWithNameHolder() state.Container {
	stateContainer := memstate.NewMemoryStateContainer()
	s.Require().NoError(stateContainer.Instances().Save(context.Background(), nameHolder))
	return stateContainer
}

func (s *ContainerDeployTestSuite) loadNewInstanceDeploy(
	stateContainer state.Container,
) (BlueprintContainer, *changes.BlueprintChanges, core.BlueprintParams) {
	loader := NewDefaultLoader(
		map[string]provider.Provider{
			"aws": &internal.ProviderMock{
				NamespaceValue: "aws",
				Resources: map[string]provider.Resource{
					"aws/lambda2/function": &internal.Lambda2FunctionResource{},
				},
			},
		},
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint11.yml",
		params,
	)
	s.Require().NoError(err)

	staged, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		blueprintContainer,
		params,
	)
	s.Require().NoError(err)
	return blueprintContainer, staged, params
}

// Answers every name lookup as not found, as a lookup that ran before another deploy
// created the instance would have.
type lookupMissesStateContainer struct {
	state.Container
}

func (c *lookupMissesStateContainer) Instances() state.InstancesContainer {
	return &lookupMissesInstances{InstancesContainer: c.Container.Instances()}
}

type lookupMissesInstances struct {
	state.InstancesContainer
}

func (i *lookupMissesInstances) LookupIDByName(_ context.Context, instanceName string) (string, error) {
	return "", state.InstanceNotFoundError(instanceName)
}
