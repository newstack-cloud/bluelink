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
	// ReconciliationTracker tracks calls to CheckReconciliation and ApplyReconciliation.
	ReconciliationTracker *ReconciliationTracker
	// checkReconciliationResult is the result to return from CheckReconciliation.
	checkReconciliationResult *container.ReconciliationCheckResult
	// checkReconciliationError is the error to return from CheckReconciliation.
	checkReconciliationError error
	// applyReconciliationResult is the result to return from ApplyReconciliation.
	applyReconciliationResult *container.ApplyReconciliationResult
	// applyReconciliationError is the error to return from ApplyReconciliation.
	applyReconciliationError error
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

// ReconciliationTracker tracks calls to CheckReconciliation and ApplyReconciliation for testing.
type ReconciliationTracker struct {
	mu sync.Mutex
	// CheckCalls contains all the check reconciliation inputs that were passed to CheckReconciliation.
	CheckCalls []*container.CheckReconciliationInput
	// ApplyCalls contains all the apply reconciliation inputs that were passed to ApplyReconciliation.
	ApplyCalls []*container.ApplyReconciliationInput
}

// NewReconciliationTracker creates a new ReconciliationTracker.
func NewReconciliationTracker() *ReconciliationTracker {
	return &ReconciliationTracker{
		CheckCalls: []*container.CheckReconciliationInput{},
		ApplyCalls: []*container.ApplyReconciliationInput{},
	}
}

// RecordCheckCall records a check reconciliation call.
func (t *ReconciliationTracker) RecordCheckCall(input *container.CheckReconciliationInput) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.CheckCalls = append(t.CheckCalls, input)
}

// RecordApplyCall records an apply reconciliation call.
func (t *ReconciliationTracker) RecordApplyCall(input *container.ApplyReconciliationInput) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ApplyCalls = append(t.ApplyCalls, input)
}

// WasCheckCalled returns true if CheckReconciliation was called at least once.
func (t *ReconciliationTracker) WasCheckCalled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.CheckCalls) > 0
}

// WasApplyCalled returns true if ApplyReconciliation was called at least once.
func (t *ReconciliationTracker) WasApplyCalled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.ApplyCalls) > 0
}

// GetCheckCalls returns a copy of all check reconciliation calls.
func (t *ReconciliationTracker) GetCheckCalls() []*container.CheckReconciliationInput {
	t.mu.Lock()
	defer t.mu.Unlock()
	calls := make([]*container.CheckReconciliationInput, len(t.CheckCalls))
	copy(calls, t.CheckCalls)
	return calls
}

// GetApplyCalls returns a copy of all apply reconciliation calls.
func (t *ReconciliationTracker) GetApplyCalls() []*container.ApplyReconciliationInput {
	t.mu.Lock()
	defer t.mu.Unlock()
	calls := make([]*container.ApplyReconciliationInput, len(t.ApplyCalls))
	copy(calls, t.ApplyCalls)
	return calls
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

// WithReconciliationTracker configures a ReconciliationTracker to track reconciliation calls.
func WithReconciliationTracker(tracker *ReconciliationTracker) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.ReconciliationTracker = tracker
	}
}

// WithCheckReconciliationResult configures the result to return from CheckReconciliation.
func WithCheckReconciliationResult(result *container.ReconciliationCheckResult) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.checkReconciliationResult = result
	}
}

// WithCheckReconciliationError configures an error to return from CheckReconciliation.
func WithCheckReconciliationError(err error) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.checkReconciliationError = err
	}
}

// WithApplyReconciliationResult configures the result to return from ApplyReconciliation.
func WithApplyReconciliationResult(result *container.ApplyReconciliationResult) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.applyReconciliationResult = result
	}
}

// WithApplyReconciliationError configures an error to return from ApplyReconciliation.
func WithApplyReconciliationError(err error) MockBlueprintLoaderOption {
	return func(loader *MockBlueprintLoader) {
		loader.applyReconciliationError = err
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
		reconciliationTracker:       m.ReconciliationTracker,
		checkReconciliationResult:   m.checkReconciliationResult,
		checkReconciliationError:    m.checkReconciliationError,
		applyReconciliationResult:   m.applyReconciliationResult,
		applyReconciliationError:    m.applyReconciliationError,
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
		reconciliationTracker:       m.ReconciliationTracker,
		checkReconciliationResult:   m.checkReconciliationResult,
		checkReconciliationError:    m.checkReconciliationError,
		applyReconciliationResult:   m.applyReconciliationResult,
		applyReconciliationError:    m.applyReconciliationError,
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
		reconciliationTracker:       m.ReconciliationTracker,
		checkReconciliationResult:   m.checkReconciliationResult,
		checkReconciliationError:    m.checkReconciliationError,
		applyReconciliationResult:   m.applyReconciliationResult,
		applyReconciliationError:    m.applyReconciliationError,
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
	reconciliationTracker       *ReconciliationTracker
	checkReconciliationResult   *container.ReconciliationCheckResult
	checkReconciliationError    error
	applyReconciliationResult   *container.ApplyReconciliationResult
	applyReconciliationError    error
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

func (m *MockBlueprintContainer) CheckReconciliation(
	ctx context.Context,
	input *container.CheckReconciliationInput,
	paramOverrides core.BlueprintParams,
) (*container.ReconciliationCheckResult, error) {
	// Track the check call if a tracker is configured
	if m.reconciliationTracker != nil {
		m.reconciliationTracker.RecordCheckCall(input)
	}

	if m.checkReconciliationError != nil {
		return nil, m.checkReconciliationError
	}

	if m.checkReconciliationResult != nil {
		return m.checkReconciliationResult, nil
	}

	// Return an empty result if no result was configured
	return &container.ReconciliationCheckResult{
		InstanceID:     input.InstanceID,
		Resources:      []container.ResourceReconcileResult{},
		Links:          []container.LinkReconcileResult{},
		HasInterrupted: false,
		HasDrift:       false,
	}, nil
}

func (m *MockBlueprintContainer) ApplyReconciliation(
	ctx context.Context,
	input *container.ApplyReconciliationInput,
	paramOverrides core.BlueprintParams,
) (*container.ApplyReconciliationResult, error) {
	// Track the apply call if a tracker is configured
	if m.reconciliationTracker != nil {
		m.reconciliationTracker.RecordApplyCall(input)
	}

	if m.applyReconciliationError != nil {
		return nil, m.applyReconciliationError
	}

	if m.applyReconciliationResult != nil {
		return m.applyReconciliationResult, nil
	}

	// Return a default result if no result was configured
	return &container.ApplyReconciliationResult{
		InstanceID:       input.InstanceID,
		ResourcesUpdated: len(input.ResourceActions),
		LinksUpdated:     len(input.LinkActions),
		Errors:           []container.ReconciliationError{},
	}, nil
}
