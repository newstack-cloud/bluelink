package testutils

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

type MockBlueprintLoader struct {
	stubDiagnostics            []*core.Diagnostic
	clock                      commoncore.Clock
	instances                  state.InstancesContainer
	deployEventSequence        []container.DeployEvent
	changeStagingEventSequence []ChangeStagingEvent
	deployError                error
	changeStagingError         error
	// DestroyTracker tracks calls to the Destroy method for testing auto-rollback.
	DestroyTracker *DestroyTracker
	// DeployTracker tracks calls to the Deploy method for testing update rollback.
	DeployTracker *DeployTracker
	// DestroyEventSequence is the sequence of events to emit during destroy operations.
	destroyEventSequence []container.DeployEvent
	// RollbackDeployEventSequence is the sequence of events to emit during rollback deploy operations.
	rollbackDeployEventSequence []container.DeployEvent
}

// DestroyTracker tracks calls to the Destroy method for testing.
type DestroyTracker struct {
	mu sync.Mutex
	// DestroyCalls contains all the destroy inputs that were passed to the Destroy method.
	DestroyCalls []*container.DestroyInput
}

// NewDestroyTracker creates a new DestroyTracker.
func NewDestroyTracker() *DestroyTracker {
	return &DestroyTracker{
		DestroyCalls: []*container.DestroyInput{},
	}
}

// RecordCall records a destroy call.
func (t *DestroyTracker) RecordCall(input *container.DestroyInput) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.DestroyCalls = append(t.DestroyCalls, input)
}

// WasDestroyCalled returns true if Destroy was called at least once.
func (t *DestroyTracker) WasDestroyCalled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.DestroyCalls) > 0
}

// GetRollbackCalls returns only the destroy calls that were made as rollback operations.
func (t *DestroyTracker) GetRollbackCalls() []*container.DestroyInput {
	t.mu.Lock()
	defer t.mu.Unlock()
	var rollbackCalls []*container.DestroyInput
	for _, call := range t.DestroyCalls {
		if call.Rollback {
			rollbackCalls = append(rollbackCalls, call)
		}
	}
	return rollbackCalls
}

// DeployTracker tracks calls to the Deploy method for testing.
type DeployTracker struct {
	mu sync.Mutex
	// DeployCalls contains all the deploy inputs that were passed to the Deploy method.
	DeployCalls []*container.DeployInput
}

// NewDeployTracker creates a new DeployTracker.
func NewDeployTracker() *DeployTracker {
	return &DeployTracker{
		DeployCalls: []*container.DeployInput{},
	}
}

// RecordCall records a deploy call.
func (t *DeployTracker) RecordCall(input *container.DeployInput) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.DeployCalls = append(t.DeployCalls, input)
}

// WasDeployCalled returns true if Deploy was called at least once.
func (t *DeployTracker) WasDeployCalled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.DeployCalls) > 0
}

// GetRollbackDeployCalls returns only the deploy calls that were made as rollback operations.
func (t *DeployTracker) GetRollbackDeployCalls() []*container.DeployInput {
	t.mu.Lock()
	defer t.mu.Unlock()
	var rollbackCalls []*container.DeployInput
	for _, call := range t.DeployCalls {
		if call.Rollback {
			rollbackCalls = append(rollbackCalls, call)
		}
	}
	return rollbackCalls
}

type MockBlueprintLoaderOption func(*MockBlueprintLoader)

func WithMockBlueprintLoaderDeployError(err error) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.deployError = err
	}
}

func WithMockBlueprintLoaderChangeStagingError(err error) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.changeStagingError = err
	}
}

// WithDestroyTracker configures a DestroyTracker to track destroy calls.
func WithDestroyTracker(tracker *DestroyTracker) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.DestroyTracker = tracker
	}
}

// WithDestroyEventSequence configures the event sequence for destroy operations.
func WithDestroyEventSequence(events []container.DeployEvent) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.destroyEventSequence = events
	}
}

// WithDeployTracker configures a DeployTracker to track deploy calls.
func WithDeployTracker(tracker *DeployTracker) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.DeployTracker = tracker
	}
}

// WithRollbackDeployEventSequence configures the event sequence for rollback deploy operations.
func WithRollbackDeployEventSequence(events []container.DeployEvent) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.rollbackDeployEventSequence = events
	}
}

func NewMockBlueprintLoader(
	stubDiagnostics []*core.Diagnostic,
	clock commoncore.Clock,
	instances state.InstancesContainer,
	deployEventSequence []container.DeployEvent,
	changeStagingEventSequence []ChangeStagingEvent,
	opts ...MockBlueprintLoaderOption,
) container.Loader {
	loader := &MockBlueprintLoader{
		stubDiagnostics:            stubDiagnostics,
		clock:                      clock,
		instances:                  instances,
		deployEventSequence:        deployEventSequence,
		changeStagingEventSequence: changeStagingEventSequence,
	}

	for _, opt := range opts {
		opt(loader)
	}

	return loader
}

func (m *MockBlueprintLoader) Load(
	ctx context.Context,
	blueprintSpecFile string,
	params core.BlueprintParams,
) (container.BlueprintContainer, error) {
	return &MockBlueprintContainer{
		stubDiagnostics:             m.stubDiagnostics,
		clock:                       m.clock,
		instances:                   m.instances,
		deployEventSequence:         m.deployEventSequence,
		changeStagingEventSequence:  m.changeStagingEventSequence,
		deployError:                 m.deployError,
		changeStagingError:          m.changeStagingError,
		destroyTracker:              m.DestroyTracker,
		deployTracker:               m.DeployTracker,
		destroyEventSequence:        m.destroyEventSequence,
		rollbackDeployEventSequence: m.rollbackDeployEventSequence,
	}, nil
}

func (m *MockBlueprintLoader) Validate(
	ctx context.Context,
	blueprintSpecFile string,
	params core.BlueprintParams,
) (*container.ValidationResult, error) {
	return &container.ValidationResult{
		Diagnostics: m.stubDiagnostics,
	}, nil
}

func (m *MockBlueprintLoader) LoadString(
	ctx context.Context,
	blueprintSpec string,
	inputFormat schema.SpecFormat,
	params core.BlueprintParams,
) (container.BlueprintContainer, error) {
	return &MockBlueprintContainer{
		stubDiagnostics:             m.stubDiagnostics,
		clock:                       m.clock,
		instances:                   m.instances,
		deployEventSequence:         m.deployEventSequence,
		changeStagingEventSequence:  m.changeStagingEventSequence,
		deployError:                 m.deployError,
		changeStagingError:          m.changeStagingError,
		destroyTracker:              m.DestroyTracker,
		deployTracker:               m.DeployTracker,
		destroyEventSequence:        m.destroyEventSequence,
		rollbackDeployEventSequence: m.rollbackDeployEventSequence,
	}, nil
}

func (m *MockBlueprintLoader) ValidateString(
	ctx context.Context,
	blueprintSpec string,
	inputFormat schema.SpecFormat,
	params core.BlueprintParams,
) (*container.ValidationResult, error) {
	return &container.ValidationResult{
		Diagnostics: m.stubDiagnostics,
	}, nil
}

func (m *MockBlueprintLoader) LoadFromSchema(
	ctx context.Context,
	blueprintSchema *schema.Blueprint,
	params core.BlueprintParams,
) (container.BlueprintContainer, error) {
	return &MockBlueprintContainer{
		stubDiagnostics:             m.stubDiagnostics,
		clock:                       m.clock,
		instances:                   m.instances,
		deployEventSequence:         m.deployEventSequence,
		changeStagingEventSequence:  m.changeStagingEventSequence,
		deployError:                 m.deployError,
		changeStagingError:          m.changeStagingError,
		destroyTracker:              m.DestroyTracker,
		deployTracker:               m.DeployTracker,
		destroyEventSequence:        m.destroyEventSequence,
		rollbackDeployEventSequence: m.rollbackDeployEventSequence,
	}, nil
}

func (m *MockBlueprintLoader) ValidateFromSchema(
	ctx context.Context,
	blueprintSchema *schema.Blueprint,
	params core.BlueprintParams,
) (*container.ValidationResult, error) {
	return &container.ValidationResult{
		Diagnostics: m.stubDiagnostics,
	}, nil
}

type MockBlueprintContainer struct {
	stubDiagnostics             []*core.Diagnostic
	clock                       commoncore.Clock
	instances                   state.InstancesContainer
	deployEventSequence         []container.DeployEvent
	changeStagingEventSequence  []ChangeStagingEvent
	changeStagingError          error
	deployError                 error
	destroyTracker              *DestroyTracker
	deployTracker               *DeployTracker
	destroyEventSequence        []container.DeployEvent
	rollbackDeployEventSequence []container.DeployEvent
}

func (m *MockBlueprintContainer) StageChanges(
	ctx context.Context,
	input *container.StageChangesInput,
	channels *container.ChangeStagingChannels,
	paramOverrides core.BlueprintParams,
) error {
	go func() {
		if m.changeStagingError != nil {
			channels.ErrChan <- m.changeStagingError
			return
		}

		for _, event := range m.changeStagingEventSequence {
			if event.ResourceChangesEvent != nil {
				channels.ResourceChangesChan <- *event.ResourceChangesEvent
			}
			if event.ChildChangesEvent != nil {
				channels.ChildChangesChan <- *event.ChildChangesEvent
			}
			if event.LinkChangesEvent != nil {
				channels.LinkChangesChan <- *event.LinkChangesEvent
			}
			if event.FinalBlueprintChanges != nil {
				channels.CompleteChan <- *event.FinalBlueprintChanges
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return nil
}

func (m *MockBlueprintContainer) Deploy(
	ctx context.Context,
	input *container.DeployInput,
	channels *container.DeployChannels,
	paramOverrides core.BlueprintParams,
) error {
	// Track the deploy call if a tracker is configured
	if m.deployTracker != nil {
		m.deployTracker.RecordCall(input)
	}

	instanceID := input.InstanceID
	if instanceID == "" {
		instanceID = uuid.New().String()
	}

	// Use rollback event sequence if this is a rollback deploy and one is configured
	eventSequence := m.deployEventSequence
	if input.Rollback && len(m.rollbackDeployEventSequence) > 0 {
		eventSequence = m.rollbackDeployEventSequence
	}

	go func() {
		currentTimestamp := m.clock.Now().Unix()
		err := m.instances.Save(
			ctx,
			state.InstanceState{
				InstanceID:                instanceID,
				InstanceName:              input.InstanceName,
				Status:                    core.InstanceStatusPreparing,
				LastStatusUpdateTimestamp: int(currentTimestamp),
			},
		)
		if err != nil {
			channels.ErrChan <- err
			return
		}

		for i, event := range eventSequence {
			if event.ResourceUpdateEvent != nil {
				event.ResourceUpdateEvent.InstanceID = instanceID
				channels.ResourceUpdateChan <- *event.ResourceUpdateEvent
			}
			if event.ChildUpdateEvent != nil {
				event.ChildUpdateEvent.ParentInstanceID = instanceID
				channels.ChildUpdateChan <- *event.ChildUpdateEvent
			}
			if event.LinkUpdateEvent != nil {
				event.LinkUpdateEvent.InstanceID = instanceID
				channels.LinkUpdateChan <- *event.LinkUpdateEvent
			}
			if event.DeploymentUpdateEvent != nil {
				event.DeploymentUpdateEvent.InstanceID = instanceID
				channels.DeploymentUpdateChan <- *event.DeploymentUpdateEvent
				// The first deployment update event needs to be sent to the caller
				// in order for the deploy engine to obtain an instance ID.
				// If an error for the stream is configured, it should be sent
				// after this event.
				if i == 0 && m.deployError != nil {
					channels.ErrChan <- m.deployError
					return
				}
			}
			if event.FinishEvent != nil {
				event.FinishEvent.InstanceID = instanceID
				channels.FinishChan <- *event.FinishEvent
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return nil
}

func (m *MockBlueprintContainer) Destroy(
	ctx context.Context,
	input *container.DestroyInput,
	channels *container.DeployChannels,
	paramOverrides core.BlueprintParams,
) {
	// Track the destroy call if a tracker is configured
	if m.destroyTracker != nil {
		m.destroyTracker.RecordCall(input)
	}

	// Emit destroy events if configured
	if len(m.destroyEventSequence) > 0 {
		go func() {
			for _, event := range m.destroyEventSequence {
				if event.ResourceUpdateEvent != nil {
					event.ResourceUpdateEvent.InstanceID = input.InstanceID
					channels.ResourceUpdateChan <- *event.ResourceUpdateEvent
				}
				if event.DeploymentUpdateEvent != nil {
					event.DeploymentUpdateEvent.InstanceID = input.InstanceID
					channels.DeploymentUpdateChan <- *event.DeploymentUpdateEvent
				}
				if event.FinishEvent != nil {
					event.FinishEvent.InstanceID = input.InstanceID
					channels.FinishChan <- *event.FinishEvent
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
}

func (m *MockBlueprintContainer) SpecLinkInfo() links.SpecLinkInfo {
	return nil
}

func (m *MockBlueprintContainer) BlueprintSpec() speccore.BlueprintSpec {
	return nil
}

func (m *MockBlueprintContainer) RefChainCollector() refgraph.RefChainCollector {
	return nil
}

func (m *MockBlueprintContainer) ResourceTemplates() map[string]string {
	return map[string]string{}
}

func (m *MockBlueprintContainer) Diagnostics() []*core.Diagnostic {
	return m.stubDiagnostics
}
