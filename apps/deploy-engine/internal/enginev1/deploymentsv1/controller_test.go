package deploymentsv1

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

var (
	// Saturday, 3 May 2025 14:27:22 UTC
	testTime = time.Unix(1746282442, 0).UTC()
)

type ControllerTestSuite struct {
	suite.Suite
	ctrl                       *Controller
	ctrlFailingIDGenerators    *Controller
	ctrlStreamErrors           *Controller
	eventStore                 manage.Events
	changesetStore             manage.Changesets
	reconciliationResultsStore *testutils.MockReconciliationResultsStore
	instances                  state.InstancesContainer
	client                     *http.Client
}

func (s *ControllerTestSuite) SetupTest() {
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}
	blueprintLoader := testutils.NewMockBlueprintLoader(
		[]*core.Diagnostic{},
		clock,
		stateContainer.Instances(),
		// Leave instance ID empty for the deploy event sequence
		// so it can be populated based on the instance ID provided
		// in the request or generated in the deployment process.
		deployEventSequence( /* instanceID */ ""),
		changeStagingEventSequence(),
	)
	s.eventStore = testutils.NewMockEventStore(
		map[string]*manage.Event{},
	)
	s.changesetStore = testutils.NewMockChangesetStore(
		map[string]*manage.Changeset{},
	)
	s.reconciliationResultsStore = testutils.NewMockReconciliationResultsStore(
		map[string]*manage.ReconciliationResult{},
	).(*testutils.MockReconciliationResultsStore)
	s.instances = stateContainer.Instances()
	dependencies := &typesv1.Dependencies{
		EventStore: s.eventStore,
		ValidationStore: testutils.NewMockBlueprintValidationStore(
			map[string]*manage.BlueprintValidation{},
		),
		ChangesetStore:             s.changesetStore,
		ReconciliationResultsStore: s.reconciliationResultsStore,
		Instances:                  s.instances,
		Exports:                    stateContainer.Exports(),
		IDGenerator:                core.NewUUIDGenerator(),
		EventIDGenerator:           utils.NewUUIDv7Generator(),
		ValidationLoader:           blueprintLoader,
		DeploymentLoader:           blueprintLoader,
		BlueprintResolver:          &testutils.MockBlueprintResolver{},
		ParamsProvider: params.NewDefaultProvider(
			map[string]*core.ScalarValue{},
		),
		PluginConfigPreparer: testutils.NewMockPluginConfigPreparer(
			pluginConfigPreparerFixtures,
		),
		Clock:  clock,
		Logger: core.NewNopLogger(),
	}
	s.ctrl = NewController(
		/* changesetRetentionPeriod */ 10*time.Second,
		/* reconciliationResultsRetentionPeriod */ 10*time.Second,
		/* deploymentTimeout */ 10*time.Second,
		/* drainTimeout */ 100*time.Millisecond,
		dependencies,
	)

	depsWithFailingIDGenerators := testutils.CopyDependencies(dependencies)
	failingIDGenerator := &testutils.FailingIDGenerator{}
	depsWithFailingIDGenerators.IDGenerator = failingIDGenerator
	depsWithFailingIDGenerators.EventIDGenerator = failingIDGenerator
	s.ctrlFailingIDGenerators = NewController(
		/* changesetRetentionPeriod */ 10*time.Second,
		/* reconciliationResultsRetentionPeriod */ 10*time.Second,
		/* deploymentTimeout */ 10*time.Second,
		/* drainTimeout */ 100*time.Millisecond,
		depsWithFailingIDGenerators,
	)

	depsWithStreamErrors := testutils.CopyDependencies(dependencies)
	streamErrorsBlueprintLoader := testutils.NewMockBlueprintLoader(
		[]*core.Diagnostic{},
		clock,
		stateContainer.Instances(),
		deployEventSequence( /* instanceID */ ""),
		changeStagingEventSequence(),
		testutils.WithMockBlueprintLoaderChangeStagingError(
			errors.New("change staging error"),
		),
		testutils.WithMockBlueprintLoaderDeployError(
			errors.New("deploy error"),
		),
	)
	depsWithStreamErrors.DeploymentLoader = streamErrorsBlueprintLoader
	s.ctrlStreamErrors = NewController(
		/* changesetRetentionPeriod */ 10*time.Second,
		/* reconciliationResultsRetentionPeriod */ 10*time.Second,
		/* deploymentTimeout */ 10*time.Second,
		/* drainTimeout */ 100*time.Millisecond,
		depsWithStreamErrors,
	)

	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}

	helpersv1.SetupRequestBodyValidator()
}

func deployEventSequence(instanceID string) []container.DeployEvent {
	return []container.DeployEvent{
		{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				InstanceID:      instanceID,
				Status:          core.InstanceStatusPreparing,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			ResourceUpdateEvent: &container.ResourceDeployUpdateMessage{
				InstanceID:      instanceID,
				ResourceID:      "resource-1",
				ResourceName:    "Resource1",
				Status:          core.ResourceStatusCreating,
				PreciseStatus:   core.PreciseResourceStatusCreating,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			LinkUpdateEvent: &container.LinkDeployUpdateMessage{
				InstanceID:      instanceID,
				LinkID:          "link-1",
				LinkName:        "Resource1::Resource2",
				Status:          core.LinkStatusCreating,
				PreciseStatus:   core.PreciseLinkStatusUpdatingResourceA,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			ChildUpdateEvent: &container.ChildDeployUpdateMessage{
				ParentInstanceID: instanceID,
				ChildInstanceID:  "child-instance-1",
				ChildName:        "coreInfra",
				Status:           core.InstanceStatusDeploying,
				UpdateTimestamp:  testTime.Unix(),
			},
		},
		{
			DeploymentUpdateEvent: &container.DeploymentUpdateMessage{
				InstanceID:      instanceID,
				Status:          core.InstanceStatusDeploying,
				UpdateTimestamp: testTime.Unix(),
			},
		},
		{
			FinishEvent: &container.DeploymentFinishedMessage{
				InstanceID:      instanceID,
				Status:          core.InstanceStatusDeployed,
				UpdateTimestamp: testTime.Unix(),
				EndOfStream:     true,
			},
		},
	}
}

func changeStagingEventSequence() []testutils.ChangeStagingEvent {
	return []testutils.ChangeStagingEvent{
		{
			ResourceChangesEvent: &container.ResourceChangesMessage{
				ResourceName: "Resource1",
				Removed:      true,
			},
		},
		{
			ChildChangesEvent: &container.ChildChangesMessage{
				ChildBlueprintName: "coreInfra",
				New:                true,
				Changes: changes.BlueprintChanges{
					NewResources: map[string]provider.Changes{
						"childResource1": {
							ComputedFields: []string{"spec.id"},
						},
					},
				},
			},
		},
		{
			LinkChangesEvent: &container.LinkChangesMessage{
				ResourceAName: "Resource1",
				ResourceBName: "Resource2",
				Changes: provider.LinkChanges{
					FieldChangesKnownOnDeploy: []string{"Resource1.policy"},
				},
			},
		},
		{
			FinalBlueprintChanges: &changes.BlueprintChanges{
				ResourceChanges: map[string]provider.Changes{
					"Resource1": {
						NewOutboundLinks: map[string]provider.LinkChanges{
							"Resource2": {
								FieldChangesKnownOnDeploy: []string{"Resource1.policy"},
							},
						},
					},
				},
				NewChildren: map[string]changes.NewBlueprintDefinition{
					"coreInfra": {
						NewResources: map[string]provider.Changes{
							"childResource1": {
								ComputedFields: []string{"spec.id"},
							},
						},
					},
				},
			},
		},
	}
}

var (
	pluginConfigPreparerFixtures = map[string][]*core.Diagnostic{
		"invalid value": {
			{
				Level:   core.DiagnosticLevelError,
				Message: "Invalid value provided",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   1,
							Column: 5,
						},
					},
				},
			},
			{
				Level:   core.DiagnosticLevelError,
				Message: "Another error",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   2,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   2,
							Column: 5,
						},
					},
				},
			},
		},
	}
)

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}
