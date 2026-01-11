package validationv1

import (
	"net/http"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/typesv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/params"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/stretchr/testify/suite"
)

var (
	// Saturday, 3 May 2025 14:27:22 UTC
	testTime = time.Unix(1746282442, 0).UTC()
)

type ControllerTestSuite struct {
	suite.Suite
	ctrl                    *Controller
	ctrlFailingIDGenerators *Controller
	eventStore              manage.Events
	validationStore         manage.Validation
	client                  *http.Client
}

func (s *ControllerTestSuite) SetupTest() {
	stateContainer := testutils.NewMemoryStateContainer()
	clock := &testutils.MockClock{
		StaticTime: testTime,
	}
	blueprintLoader := testutils.NewMockBlueprintLoader(
		stubDiagnostics,
		clock,
		stateContainer.Instances(),
		/* deployEventSequence */ []container.DeployEvent{},
		/* changeStagingEventSequence */ []testutils.ChangeStagingEvent{},
	)
	s.eventStore = testutils.NewMockEventStore(
		map[string]*manage.Event{},
	)
	s.validationStore = testutils.NewMockBlueprintValidationStore(
		map[string]*manage.BlueprintValidation{},
	)
	dependencies := &typesv1.Dependencies{
		EventStore:      s.eventStore,
		ValidationStore: s.validationStore,
		ChangesetStore: testutils.NewMockChangesetStore(
			map[string]*manage.Changeset{},
		),
		CleanupOperationsStore: testutils.NewMockCleanupOperationsStore(
			map[string]*manage.CleanupOperation{},
		),
		Instances:         stateContainer.Instances(),
		Exports:           stateContainer.Exports(),
		IDGenerator:       core.NewUUIDGenerator(),
		EventIDGenerator:  utils.NewUUIDv7Generator(),
		ValidationLoader:  blueprintLoader,
		DeploymentLoader:  blueprintLoader,
		BlueprintResolver: &testutils.MockBlueprintResolver{},
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
		10*time.Second,
		dependencies,
	)
	depsWithFailingIDGenerators := testutils.CopyDependencies(dependencies)
	failingIDGenerator := &testutils.FailingIDGenerator{}
	depsWithFailingIDGenerators.IDGenerator = failingIDGenerator
	depsWithFailingIDGenerators.EventIDGenerator = failingIDGenerator
	s.ctrlFailingIDGenerators = NewController(
		10*time.Second,
		depsWithFailingIDGenerators,
	)
	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}

	helpersv1.SetupRequestBodyValidator()
}

var (
	stubDiagnostics = []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "Validation failed due to invalid version",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   1,
						Column: 20,
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
		{
			Level:   core.DiagnosticLevelWarning,
			Message: "Validation warning",
			Range: &core.DiagnosticRange{
				Start: &source.Meta{
					Position: source.Position{
						Line:   3,
						Column: 10,
					},
				},
				End: &source.Meta{
					Position: source.Position{
						Line:   4,
						Column: 15,
					},
				},
			},
		},
	}
)

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
		"warnings value": {
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "Warning message",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   3,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   3,
							Column: 5,
						},
					},
				},
			},
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "Another warning",
				Range: &core.DiagnosticRange{
					Start: &source.Meta{
						Position: source.Position{
							Line:   4,
							Column: 1,
						},
					},
					End: &source.Meta{
						Position: source.Position{
							Line:   4,
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
