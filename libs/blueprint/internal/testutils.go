package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/mockclock"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

type FunctionRegistryMock struct {
	Functions map[string]provider.Function
	CallStack function.Stack
}

func (f *FunctionRegistryMock) ForCallContext(stack function.Stack) provider.FunctionRegistry {
	return &FunctionRegistryMock{
		Functions: f.Functions,
		CallStack: stack,
	}
}

func (f *FunctionRegistryMock) Call(
	ctx context.Context,
	functionName string,
	input *provider.FunctionCallInput,
) (*provider.FunctionCallOutput, error) {
	fnc, ok := f.Functions[functionName]
	if !ok {
		return nil, function.NewFuncCallError(
			fmt.Sprintf("function %s not found", functionName),
			function.FuncCallErrorCodeFunctionNotFound,
			input.CallContext.CallStackSnapshot(),
		)
	}
	f.CallStack.Push(&function.Call{
		FunctionName: functionName,
		Location:     nil,
	})
	output, err := fnc.Call(ctx, input)
	f.CallStack.Pop()
	return output, err
}

func (f *FunctionRegistryMock) GetDefinition(
	ctx context.Context,
	functionName string,
	input *provider.FunctionGetDefinitionInput,
) (*provider.FunctionGetDefinitionOutput, error) {
	fnc, ok := f.Functions[functionName]
	if !ok {
		return nil, fmt.Errorf("function %s not found", functionName)
	}
	defOutput, err := fnc.GetDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *FunctionRegistryMock) ListFunctions(
	ctx context.Context,
) ([]string, error) {
	functions := make([]string, 0, len(r.Functions))
	for function := range r.Functions {
		functions = append(functions, function)
	}
	return functions, nil
}

func (f *FunctionRegistryMock) HasFunction(ctx context.Context, functionName string) (bool, error) {
	_, ok := f.Functions[functionName]
	return ok, nil
}

type ResourceRegistryMock struct {
	Resources                 map[string]provider.Resource
	StateContainer            state.Container
	resourceLocks             map[string]*resourceLock
	resourceLockCheckInterval time.Duration
	resourceLockTimeout       time.Duration
	clock                     core.Clock
	resourceLocksMu           *sync.Mutex
}

type ResourceRegistryMockOption func(*ResourceRegistryMock)

func WithResourceRegistryLockCheckInterval(interval time.Duration) ResourceRegistryMockOption {
	return func(r *ResourceRegistryMock) {
		r.resourceLockCheckInterval = interval
	}
}

func WithResourceRegistryLockTimeout(timeout time.Duration) ResourceRegistryMockOption {
	return func(r *ResourceRegistryMock) {
		r.resourceLockTimeout = timeout
	}
}

func WithResourceRegistryClock(clock core.Clock) ResourceRegistryMockOption {
	return func(r *ResourceRegistryMock) {
		r.clock = clock
	}
}

func WithResourceRegistryStateContainer(stateContainer state.Container) ResourceRegistryMockOption {
	return func(r *ResourceRegistryMock) {
		r.StateContainer = stateContainer
	}
}

func NewResourceRegistryMock(
	resources map[string]provider.Resource,
	opts ...ResourceRegistryMockOption,
) *ResourceRegistryMock {
	registry := &ResourceRegistryMock{
		Resources:                 resources,
		StateContainer:            nil,
		resourceLocks:             make(map[string]*resourceLock),
		resourceLockCheckInterval: 10 * time.Millisecond,
		resourceLockTimeout:       200 * time.Millisecond,
		clock:                     &mockclock.StaticClock{},
		resourceLocksMu:           &sync.Mutex{},
	}

	for _, opt := range opts {
		opt(registry)
	}

	return registry
}

func (r *ResourceRegistryMock) HasResourceType(ctx context.Context, resourceType string) (bool, error) {
	_, ok := r.Resources[resourceType]
	return ok, nil
}

func (r *ResourceRegistryMock) GetSpecDefinition(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.GetSpecDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) GetTypeDescription(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.GetTypeDescription(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) ListResourceTypes(
	ctx context.Context,
) ([]string, error) {
	resourceTypes := make([]string, 0, len(r.Resources))
	for resourceType := range r.Resources {
		resourceTypes = append(resourceTypes, resourceType)
	}
	return resourceTypes, nil
}

func (r *ResourceRegistryMock) CustomValidate(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}
	defOutput, err := res.CustomValidate(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *ResourceRegistryMock) Deploy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDeployServiceInput,
) (*provider.ResourceDeployOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}

	return res.Deploy(ctx, input.DeployInput)
}

func (r *ResourceRegistryMock) Destroy(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceDestroyInput,
) error {
	res, ok := r.Resources[resourceType]
	if !ok {
		return fmt.Errorf("resource %s not found", resourceType)
	}

	return res.Destroy(ctx, input)
}

func (r *ResourceRegistryMock) GetStabilisedDependencies(
	ctx context.Context,
	resourceType string,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	res, ok := r.Resources[resourceType]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", resourceType)
	}

	return res.GetStabilisedDependencies(ctx, input)
}

func (r *ResourceRegistryMock) LookupResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (*state.ResourceState, error) {
	resourceImpl, hasResourceImpl := r.Resources[input.ResourceType]
	if !hasResourceImpl {
		return nil, fmt.Errorf(
			"resource type %s not found in the registry",
			input.ResourceType,
		)
	}

	definition, err := resourceImpl.GetSpecDefinition(
		ctx,
		&provider.ResourceGetSpecDefinitionInput{
			ProviderContext: input.ProviderContext,
		},
	)
	if err != nil {
		return nil, err
	}

	if definition == nil || definition.SpecDefinition == nil {
		return nil, fmt.Errorf(
			"resource spec definition for resource type %s is empty",
			input.ResourceType,
		)
	}

	idField := definition.SpecDefinition.IDField
	instance, err := r.StateContainer.Instances().Get(ctx, input.InstanceID)
	if err != nil {
		return nil, err
	}

	return extractResourceByExternalID(
		idField,
		input.ExternalID,
		input.ResourceType,
		&instance,
	), nil
}

func extractResourceByExternalID(
	idField string,
	externalID string,
	resourceType string,
	instance *state.InstanceState,
) *state.ResourceState {
	if instance == nil {
		return nil
	}

	for _, resource := range instance.Resources {
		fieldPath := substitutions.RenderFieldPath("$.%s", idField)
		idFieldValue, _ := core.GetPathValue(
			fieldPath,
			resource.SpecData,
			core.MappingNodeMaxTraverseDepth,
		)
		if idFieldValue != nil &&
			core.StringValue(idFieldValue) == externalID &&
			resource.Type == resourceType {
			return resource
		}
	}

	return nil
}

func (r *ResourceRegistryMock) HasResourceInState(
	ctx context.Context,
	input *provider.ResourceLookupInput,
) (bool, error) {
	resourceState, err := r.LookupResourceInState(ctx, input)
	if err != nil {
		return false, err
	}

	if resourceState == nil {
		return false, nil
	}

	return true, nil
}

type resourceLock struct {
	// The ID of the blueprint instance that the lock is acquired in.
	instanceID string
	// The name of the resource that the lock is acquired on.
	resourceName string
	// The time when the lock was acquired.
	// This is used to determine if the lock has timed out.
	lockTime time.Time
	// The identifier of the caller that acquired the lock.
	acquiredBy string
}

func (r *ResourceRegistryMock) AcquireResourceLock(
	ctx context.Context,
	input *provider.AcquireResourceLockInput,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			r.resourceLocksMu.Lock()
			lockKey := createResourceLockKey(input.InstanceID, input.ResourceName)
			if r.checkLock(lockKey) {
				r.resourceLocks[lockKey] = &resourceLock{
					instanceID:   input.InstanceID,
					resourceName: input.ResourceName,
					lockTime:     r.clock.Now(),
					acquiredBy:   input.AcquiredBy,
				}
				r.resourceLocksMu.Unlock()
				return nil
			}
			r.resourceLocksMu.Unlock()
			time.Sleep(r.resourceLockCheckInterval)
		}
	}
}

// The resource locks mutex must be held when calling this method.
func (r *ResourceRegistryMock) checkLock(lockKey string) bool {
	if lock, exists := r.resourceLocks[lockKey]; exists {
		// If the lock exists, check if it has timed out.
		if r.clock.Now().Sub(lock.lockTime) < r.resourceLockTimeout {
			// Lock is still held, cannot acquire.
			return false
		}
		// Lock has timed out, remove it.
		delete(r.resourceLocks, lockKey)
	}

	return true
}

func (r *ResourceRegistryMock) ReleaseResourceLock(
	ctx context.Context,
	instanceID string,
	resourceName string,
) {
	r.resourceLocksMu.Lock()
	defer r.resourceLocksMu.Unlock()

	lockKey := createResourceLockKey(instanceID, resourceName)
	delete(r.resourceLocks, lockKey)
}

func (r *ResourceRegistryMock) ReleaseResourceLocks(ctx context.Context, instanceID string) {
	r.resourceLocksMu.Lock()
	defer r.resourceLocksMu.Unlock()

	for lockKey := range r.resourceLocks {
		if lockKeyHasInstanceID(lockKey, instanceID) {
			delete(r.resourceLocks, lockKey)
		}
	}
}

func (r *ResourceRegistryMock) ReleaseResourceLocksAcquiredBy(
	ctx context.Context,
	instanceID string,
	acquiredBy string,
) {
	r.resourceLocksMu.Lock()
	defer r.resourceLocksMu.Unlock()

	for lockKey, lock := range r.resourceLocks {
		if lockKeyHasInstanceID(lockKey, instanceID) && lock.acquiredBy == acquiredBy {
			delete(r.resourceLocks, lockKey)
		}
	}
}

func createResourceLockKey(instanceID, resourceName string) string {
	return fmt.Sprintf("%s:%s", instanceID, resourceName)
}

func lockKeyHasInstanceID(lockKey, instanceID string) bool {
	instanceIDPrefix := fmt.Sprintf("%s:", instanceID)
	return strings.HasPrefix(lockKey, instanceIDPrefix)
}

func (r *ResourceRegistryMock) WithParams(
	params core.BlueprintParams,
) resourcehelpers.Registry {
	return &ResourceRegistryMock{
		Resources:                 r.Resources,
		StateContainer:            r.StateContainer,
		resourceLocks:             r.resourceLocks,
		resourceLockCheckInterval: r.resourceLockCheckInterval,
		resourceLockTimeout:       r.resourceLockTimeout,
		clock:                     r.clock,
		resourceLocksMu:           r.resourceLocksMu,
	}
}

type DataSourceRegistryMock struct {
	DataSources map[string]provider.DataSource
}

func (r *DataSourceRegistryMock) HasDataSourceType(ctx context.Context, dataSourceType string) (bool, error) {
	_, ok := r.DataSources[dataSourceType]
	return ok, nil
}

func (r *DataSourceRegistryMock) GetSpecDefinition(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetSpecDefinitionInput,
) (*provider.DataSourceGetSpecDefinitionOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.GetSpecDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) GetTypeDescription(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetTypeDescriptionInput,
) (*provider.DataSourceGetTypeDescriptionOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.GetTypeDescription(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) GetFilterFields(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceGetFilterFieldsInput,
) (*provider.DataSourceGetFilterFieldsOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.GetFilterFields(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) ListDataSourceTypes(
	ctx context.Context,
) ([]string, error) {
	dataSourceTypes := make([]string, 0, len(r.DataSources))
	for dataSourceType := range r.DataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}
	return dataSourceTypes, nil
}

func (r *DataSourceRegistryMock) CustomValidate(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceValidateInput,
) (*provider.DataSourceValidateOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.CustomValidate(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

func (r *DataSourceRegistryMock) Fetch(
	ctx context.Context,
	dataSourceType string,
	input *provider.DataSourceFetchInput,
) (*provider.DataSourceFetchOutput, error) {
	res, ok := r.DataSources[dataSourceType]
	if !ok {
		return nil, fmt.Errorf("data source %s not found", dataSourceType)
	}
	defOutput, err := res.Fetch(ctx, input)
	if err != nil {
		return nil, err
	}
	return defOutput, nil
}

// UnpackLoadError recursively unpacks a LoadError that can contain child errors.
// This will recursively unpack the first child error until it reaches the last child error.
func UnpackLoadError(err error) (*errors.LoadError, bool) {
	loadErr, ok := err.(*errors.LoadError)
	if ok && len(loadErr.ChildErrors) > 0 {
		return UnpackLoadError(loadErr.ChildErrors[0])
	}
	return loadErr, ok
}

// UnpackError recursively unpacks a LoadError that can contain child errors.
// This is to be used when the terminating error is not a LoadError.
func UnpackError(err error) (error, bool) {
	loadErr, ok := err.(*errors.LoadError)
	if ok && len(loadErr.ChildErrors) > 0 {
		return UnpackError(loadErr.ChildErrors[0])
	}
	return err, ok
}

func OrderStringSlice(fields []string) []string {
	orderedFields := make([]string, len(fields))
	copy(orderedFields, fields)
	slices.Sort(orderedFields)
	return orderedFields
}

func LoadInstanceState(
	stateSnapshotFile string,
) (*state.InstanceState, error) {
	currentStateBytes, err := os.ReadFile(stateSnapshotFile)
	if err != nil {
		return nil, err
	}

	currentState := &state.InstanceState{}
	err = json.Unmarshal(currentStateBytes, currentState)
	if err != nil {
		return nil, err
	}

	return currentState, nil
}

func LoadStringFromFile(
	filePath string,
) (string, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(fileBytes), nil
}

type StaticIDGenerator struct {
	ID string
}

func (m *StaticIDGenerator) GenerateID() (string, error) {
	return m.ID, nil
}

// StubResourceStabilisationConfig provides configuration for the test
// resource implementations to simulate eventual resource stabilisation.
type StubResourceStabilisationConfig struct {
	// The number of attempts to wait for a resource to stabilise
	// before giving up.
	// Set this to -1 for a resource that should never stabilise.
	StabilisesAfterAttempts int
}
