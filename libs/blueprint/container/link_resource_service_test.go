package container

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// Locks a shared resource during the intermediary phase without naming itself as the
// acquirer, which is what a link implementation does: it has no reason to know how the
// deployer releases locks, and over the plugin protocol the field is filled in for it.
type sharedLockLink struct {
	provider.Link
	sharedResourceName string
}

func (l *sharedLockLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	err := input.ResourceService.AcquireResourceLock(
		ctx,
		&provider.AcquireResourceLockInput{
			InstanceID:   input.ResourceAInfo.InstanceID,
			ResourceName: l.sharedResourceName,
		},
	)
	if err != nil {
		return nil, err
	}

	return l.Link.UpdateIntermediaryResources(ctx, input)
}

func sharedLockAWSProvider(
	stateContainer state.Container,
	sharedResourceName string,
) provider.Provider {
	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{},
		stateContainer,
	)
	mock := awsProvider.(*internal.ProviderMock)

	locking := map[string]provider.Link{}
	for linkType, linkImpl := range mock.Links {
		locking[linkType] = &sharedLockLink{
			Link:               linkImpl,
			sharedResourceName: sharedResourceName,
		}
	}
	mock.Links = locking

	return mock
}

// Every link in this blueprint locks the same resource, so the deployment only finishes
// if each lock is released when the link that took it is done with it.
//
// The deployer releases a link's locks by acquirer, keyed on the link ID, and a link
// implementation does not supply that. Left unattributed, the first lock was never
// matched by the release and every later link waited on a holder that had already
// finished, which is a deployment that hangs rather than fails. A lock is never expired
// on age, so nothing else would have recovered it.
func (s *ContainerDeployTestSuite) Test_a_lock_taken_by_a_link_is_released_when_its_phase_ends() {
	stateContainer := memstate.NewMemoryStateContainer()

	loader := NewDefaultLoader(
		map[string]provider.Provider{
			"aws":     sharedLockAWSProvider(stateContainer, "ordersTable_0"),
			"example": newTestExampleProvider(),
			"core": providerhelpers.NewCoreProvider(
				stateContainer.Links(),
				core.BlueprintInstanceIDFromContext,
				os.Getwd,
				provider.NewFileSourceRegistry(),
				core.SystemClock{},
			),
		},
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)

	params := blueprint1DeployParams( /* includeInvoices */ true)
	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint1.yml",
		params,
	)
	s.Require().NoError(err)

	deployChanges, err := s.stageChanges(
		context.Background(),
		/* instanceID */ "",
		blueprintContainer,
		params,
	)
	s.Require().NoError(err)

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "SharedLinkLockInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage := (*DeploymentFinishedMessage)(nil)
	for err == nil && finishedMessage == nil {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case msg := <-channels.FinishChan:
			finishedMessage = &msg
		case <-channels.DeploymentUpdateChan:
		case err = <-channels.ErrChan:
		case <-time.After(defaultDrainTimeout):
			err = errors.New("timed out waiting for the deployment to finish")
		}
	}

	s.Require().NoError(
		err,
		"a link's lock outliving its phase leaves every later link waiting on a holder "+
			"that has already finished",
	)
	s.Require().NotNil(finishedMessage)
}
