package container

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// Stages a destroy-shaped changeset for the given
// instance by invoking StageChanges with Destroy=true on the public
// BlueprintContainer API. Mirrors the (*ContainerDeployTestSuite).stageChanges
// pattern used elsewhere in the suite.
func (s *ContainerDeployTestSuite) stageDestroyChanges(
	ctx context.Context,
	instanceID string,
	container BlueprintContainer,
	params core.BlueprintParams,
) (*changes.BlueprintChanges, error) {
	channels := createChangeStagingChannels()
	err := container.StageChanges(
		ctx,
		&StageChangesInput{
			InstanceID: instanceID,
			Destroy:    true,
		},
		channels,
		params,
	)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case <-channels.ResourceChangesChan:
		case changeSet := <-channels.CompleteChan:
			return &changeSet, nil
		case err := <-channels.ErrChan:
			return nil, err
		case <-time.After(60 * time.Second):
			return nil, errors.New(timeoutMessage)
		}
	}
}

// Test_concurrent_deploys_on_same_instance_only_one_wins exercises the atomic
// CAS guard in ClaimForDeployment. Two BlueprintContainer instances loaded
// independently from the same blueprint source (matching an HA deploy engine
// topology with multiple replicas sharing a single state backend) issue
// Deploy requests against the same idle instance (status = Deployed). Both
// goroutines pass the cheap isInstanceInProgress pre-check; only the CAS can
// resolve the race.
//
// Expected public-API outcome: exactly one Deploy reaches a successful
// terminal status; the other's DeploymentFinishedMessage reports a failed
// status carrying the in-progress rejection reason from
// instanceInProgressFailedMessage.
func (s *ContainerDeployTestSuite) Test_concurrent_deploys_on_same_instance_only_one_wins() {
	containerA, err := s.loadBlueprintContainer(1, schema.YAMLSpecFormat, s.fixture1Params)
	s.Require().NoError(err)
	containerB, err := s.loadBlueprintContainer(1, schema.YAMLSpecFormat, s.fixture1Params)
	s.Require().NoError(err)

	changesA, err := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		containerA,
		s.fixture1Params,
	)
	s.Require().NoError(err)
	changesB, err := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		containerB,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	channelsA := CreateDeployChannels()
	channelsB := CreateDeployChannels()

	// A sync barrier gets both goroutines as close to the same start as
	// possible so they race through the Get/pre-check/claim sequence.
	startBarrier := make(chan struct{})
	var kickoffWG sync.WaitGroup
	kickoffWG.Add(2)

	deploy := func(container BlueprintContainer, changes *changes.BlueprintChanges, channels *DeployChannels) {
		kickoffWG.Done()
		<-startBarrier
		_ = container.Deploy(
			context.Background(),
			&DeployInput{
				InstanceID: "blueprint-instance-1",
				Changes:    changes,
				Rollback:   false,
			},
			channels,
			s.fixture1Params,
		)
	}

	go deploy(containerA, changesA, channelsA)
	go deploy(containerB, changesB, channelsB)

	kickoffWG.Wait()
	close(startBarrier)

	finishA, errA := collectDeployFinish(channelsA)
	finishB, errB := collectDeployFinish(channelsB)
	s.Require().NoError(errA)
	s.Require().NoError(errB)
	s.Require().NotNil(finishA)
	s.Require().NotNil(finishB)

	// Against a freshly-deployed instance the only reason either deploy
	// would not reach its natural success terminal is the in-progress guard
	// kicking the loser out. So exactly one of the two should succeed.
	aSucceeded := finishA.Status == core.InstanceStatusUpdated ||
		finishA.Status == core.InstanceStatusDeployed
	bSucceeded := finishB.Status == core.InstanceStatusUpdated ||
		finishB.Status == core.InstanceStatusDeployed

	s.Assert().True(
		aSucceeded != bSucceeded,
		"expected exactly one concurrent deploy to succeed, with the other "+
			"rejected by the in-progress guard; a status=%s reasons=%v b status=%s reasons=%v",
		finishA.Status,
		finishA.FailureReasons,
		finishB.Status,
		finishB.FailureReasons,
	)
}

// Test_concurrent_new_instance_deploys_with_same_user_supplied_id_only_one_wins
// exercises the atomic-create guard added by InitialiseAndClaim. Two
// concurrent deploys target the same user-supplied InstanceID that does not
// yet exist in state. Both pass the Get → zero-state path, both reach
// saveNewInstance; only InitialiseAndClaim's backend-native create-if-absent
// CAS can resolve the race.
//
// Expected public-API outcome: exactly one deploy reaches a successful
// terminal status; the other is rejected by either ErrInstanceAlreadyExists
// (its InitialiseAndClaim lost) then ErrVersionConflict at the subsequent
// ClaimForDeployment — surfaced as the in-progress rejection reason.
func (s *ContainerDeployTestSuite) Test_concurrent_new_instance_deploys_with_same_user_supplied_id_only_one_wins() {
	const sharedNewID = "user-supplied-new-id"
	containerA, err := s.loadBlueprintContainer(2, schema.JWCCSpecFormat, s.fixture2Params)
	s.Require().NoError(err)
	containerB, err := s.loadBlueprintContainer(2, schema.JWCCSpecFormat, s.fixture2Params)
	s.Require().NoError(err)

	// Stage once against an empty instanceID so the produced change set
	// describes a new-instance deploy. Both concurrent deploys replay the
	// same change set against the same user-supplied instance ID.
	//
	// Change staging is run against both containers so each one operates
	// with its own unique change set.
	sharedChanges, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		containerA,
		s.fixture2Params,
	)
	s.Require().NoError(err)
	_, err = s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		containerB,
		s.fixture2Params,
	)
	s.Require().NoError(err)

	channelsA := CreateDeployChannels()
	channelsB := CreateDeployChannels()

	startBarrier := make(chan struct{})
	var kickoffWG sync.WaitGroup
	kickoffWG.Add(2)

	deploy := func(container BlueprintContainer, channels *DeployChannels) {
		kickoffWG.Done()
		<-startBarrier
		_ = container.Deploy(
			context.Background(),
			&DeployInput{
				InstanceID:   sharedNewID,
				InstanceName: "BlueprintInstance2",
				Changes:      sharedChanges,
				Rollback:     false,
			},
			channels,
			s.fixture2Params,
		)
	}

	go deploy(containerA, channelsA)
	go deploy(containerB, channelsB)

	kickoffWG.Wait()
	close(startBarrier)

	finishA, errA := collectDeployFinish(channelsA)
	finishB, errB := collectDeployFinish(channelsB)
	s.Require().NoError(errA)
	s.Require().NoError(errB)
	s.Require().NotNil(finishA)
	s.Require().NotNil(finishB)

	aSucceeded := finishA.Status == core.InstanceStatusUpdated ||
		finishA.Status == core.InstanceStatusDeployed
	bSucceeded := finishB.Status == core.InstanceStatusUpdated ||
		finishB.Status == core.InstanceStatusDeployed

	s.Assert().True(
		aSucceeded != bSucceeded,
		"expected exactly one concurrent new-instance deploy to succeed, "+
			"with the other rejected by the atomic-create guard; "+
			"a status=%s reasons=%v b status=%s reasons=%v",
		finishA.Status,
		finishA.FailureReasons,
		finishB.Status,
		finishB.FailureReasons,
	)

	// The shared state should reflect exactly one instance record with the
	// user-supplied ID, at a version bumped beyond the initial claim.
	instanceState, err := s.stateContainer.Instances().Get(context.Background(), sharedNewID)
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(instanceState.Version, int64(1))
}

// loadBlueprintContainer loads a fresh BlueprintContainer from the same
// blueprint fixture file the suite uses so tests can exercise concurrent
// operations across independent container instances sharing the same state
// backend. Mirrors an HA deploy engine topology (multiple replicas + one
// shared state store).
func (s *ContainerDeployTestSuite) loadBlueprintContainer(
	fixtureNo int,
	format schema.SpecFormat,
	params core.BlueprintParams,
) (BlueprintContainer, error) {
	extension := "yml"
	if format == schema.JWCCSpecFormat {
		extension = "jsonc"
	}
	return s.loader.Load(
		context.Background(),
		fmt.Sprintf("__testdata/container/deploy/blueprint%d.%s", fixtureNo, extension),
		params,
	)
}

// collectDeployFinish drains a DeployChannels set until a
// DeploymentFinishedMessage arrives, forwarding through the other
// per-element channels. Returns the finish message and any error surfaced
// on ErrChan or a timeout sentinel.
func collectDeployFinish(channels *DeployChannels) (*DeploymentFinishedMessage, error) {
	for {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case <-channels.DeploymentUpdateChan:
		case msg := <-channels.FinishChan:
			return &msg, nil
		case err := <-channels.ErrChan:
			return nil, err
		case <-time.After(60 * time.Second):
			return nil, errors.New(timeoutMessage)
		}
	}
}

// Test_concurrent_deploy_and_destroy_on_same_instance_only_one_wins exercises
// the atomic CAS guard across action types. A Deploy and a Destroy are issued
// against the same idle instance at the same time; only the CAS resolves the
// race (both pass the status-based pre-check since the starting status is
// Deployed, not an in-progress state).
//
// Expected public-API outcome: exactly one of the two operations reaches a
// successful terminal status; the other's FinishChan message carries the
// in-progress rejection reason.
func (s *ContainerDeployTestSuite) Test_concurrent_deploy_and_destroy_on_same_instance_only_one_wins() {
	deployContainer, err := s.loadBlueprintContainer(1, schema.YAMLSpecFormat, s.fixture1Params)
	s.Require().NoError(err)
	destroyContainer, err := s.loadBlueprintContainer(1, schema.YAMLSpecFormat, s.fixture1Params)
	s.Require().NoError(err)

	deployChanges, err := s.stageChanges(
		context.Background(),
		"blueprint-instance-1",
		deployContainer,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	destroyChanges, err := s.stageDestroyChanges(
		context.Background(),
		"blueprint-instance-1",
		destroyContainer,
		s.fixture1Params,
	)
	s.Require().NoError(err)

	deployChannels := CreateDeployChannels()
	destroyChannels := CreateDeployChannels()

	startBarrier := make(chan struct{})
	var kickoffWG sync.WaitGroup
	kickoffWG.Add(2)

	go func() {
		kickoffWG.Done()
		<-startBarrier
		_ = deployContainer.Deploy(
			context.Background(),
			&DeployInput{
				InstanceID: "blueprint-instance-1",
				Changes:    deployChanges,
				Rollback:   false,
			},
			deployChannels,
			s.fixture1Params,
		)
	}()

	go func() {
		kickoffWG.Done()
		<-startBarrier
		destroyContainer.Destroy(
			context.Background(),
			&DestroyInput{
				InstanceID: "blueprint-instance-1",
				Changes:    destroyChanges,
				Rollback:   false,
			},
			destroyChannels,
			s.fixture1Params,
		)
	}()

	kickoffWG.Wait()
	close(startBarrier)

	deployFinish, deployErr := collectDeployFinish(deployChannels)
	destroyFinish, destroyErr := collectDeployFinish(destroyChannels)
	s.Require().NoError(deployErr)
	s.Require().NoError(destroyErr)
	s.Require().NotNil(deployFinish)
	s.Require().NotNil(destroyFinish)

	// Against a freshly-deployed instance the only reason either operation
	// would not reach its natural success terminal is the in-progress guard
	// kicking the loser out. So exactly one of the two should be a success
	// and the loser's FailureReasons should name the attempted action in its
	// in-progress rejection wording.
	deploySucceeded := deployFinish.Status == core.InstanceStatusUpdated ||
		deployFinish.Status == core.InstanceStatusDeployed
	destroySucceeded := destroyFinish.Status == core.InstanceStatusDestroyed

	s.Require().True(
		deploySucceeded != destroySucceeded,
		"expected exactly one of the concurrent deploy/destroy operations to succeed, "+
			"with the other rejected by the in-progress guard; "+
			"deploy status=%s destroy status=%s",
		deployFinish.Status,
		destroyFinish.Status,
	)

	if deploySucceeded {
		s.Assert().Equal(
			[]string{instanceInProgressFailedMessage(
				"blueprint-instance-1",
				destroyClaimAction,
				false,
			)},
			destroyFinish.FailureReasons,
		)
	} else {
		s.Assert().Equal(
			[]string{instanceInProgressFailedMessage(
				"blueprint-instance-1",
				deployClaimAction,
				false,
			)},
			deployFinish.FailureReasons,
		)
	}
}
