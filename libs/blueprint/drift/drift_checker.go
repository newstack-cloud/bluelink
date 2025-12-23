package drift

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// Checker is an interface for behaviour
// that can be used to check if resources within
// a blueprint have drifted from the current state
// persisted with the blueprint framework.
// This is useful to detect situations where resources
// in an upstream provider (e.g. an AWS account) have been modified
// manually or by other means, and the blueprint state
// is no longer in sync with the actual state of the
// resources.
// A checker is only responsible for checking and persisting
// drift, the course of action to resolve the drift is
// left to the user.
type Checker interface {
	// CheckDrift checks the drift of all resources in the blueprint
	// with the given instance ID.
	// This will always check the drift with the upstream provider,
	// the state container can be used to retrieve the last known
	// drift state that was previously checked.
	// In most cases, this method will persist the results of the
	// drift check with the configured state container.
	// This returns a map of resource IDs to their drift state ONLY
	// if the resource has drifted from the last known state.
	CheckDrift(
		ctx context.Context,
		instanceID string,
		params core.BlueprintParams,
	) (map[string]*state.ResourceDriftState, error)
	// CheckResourceDrift checks the drift of a single resource
	// with the given instance ID and resource ID.
	// This will always check the drift with the upstream provider,
	// the state container can be used to retrieve the last known
	// drift state that was previously checked.
	// In most cases, this method will persist the results of the
	// drift check with the configured state container.
	// This will return nil if the resource has not drifted from
	// the last known state.
	CheckResourceDrift(
		ctx context.Context,
		instanceID string,
		instanceName string,
		resourceID string,
		params core.BlueprintParams,
	) (*state.ResourceDriftState, error)
	// CheckInterruptedResources detects interrupted resources and determines
	// their actual state from the cloud, but does NOT update persisted state.
	// Returns results for user review before applying reconciliation.
	// This is typically called during change staging to allow users to
	// review and approve state updates for resources that were interrupted
	// during a previous deployment.
	CheckInterruptedResources(
		ctx context.Context,
		instanceID string,
		params core.BlueprintParams,
	) ([]ReconcileResult, error)
	// ApplyReconciliation updates persisted state based on reconcile results.
	// This should be called after the user approves the reconciliation.
	ApplyReconciliation(
		ctx context.Context,
		results []ReconcileResult,
	) error
	// CheckLinkDrift checks drift for a single link.
	// Uses ResourceDataMappings to compare link.Data against resource external state.
	// Also checks intermediary resources via GetIntermediaryExternalState.
	// This will persist the results of the drift check with the configured state container.
	// Returns nil if the link has not drifted.
	CheckLinkDrift(
		ctx context.Context,
		instanceID string,
		linkID string,
		params core.BlueprintParams,
	) (*state.LinkDriftState, error)
	// CheckAllLinkDrift checks drift for all links in an instance.
	// Uses ResourceDataMappings to compare link.Data against resource external state.
	// Also checks intermediary resources via GetIntermediaryExternalState.
	// This will persist the results of the drift check with the configured state container.
	// Returns a map of link IDs to their drift state ONLY if the link has drifted.
	CheckAllLinkDrift(
		ctx context.Context,
		instanceID string,
		params core.BlueprintParams,
	) (map[string]*state.LinkDriftState, error)

	// CheckDriftWithState is like CheckDrift but accepts pre-fetched instance state.
	// Use this to avoid redundant state fetches when the caller already has the state.
	CheckDriftWithState(
		ctx context.Context,
		instanceState *state.InstanceState,
		params core.BlueprintParams,
	) (map[string]*state.ResourceDriftState, error)
	// CheckInterruptedResourcesWithState is like CheckInterruptedResources but accepts
	// pre-fetched instance state.
	CheckInterruptedResourcesWithState(
		ctx context.Context,
		instanceState *state.InstanceState,
		params core.BlueprintParams,
	) ([]ReconcileResult, error)
	// CheckAllLinkDriftWithState is like CheckAllLinkDrift but accepts pre-fetched
	// instance state.
	CheckAllLinkDriftWithState(
		ctx context.Context,
		instanceState *state.InstanceState,
		params core.BlueprintParams,
	) (map[string]*state.LinkDriftState, error)
}

type defaultChecker struct {
	stateContainer  state.Container
	providers       map[string]provider.Provider
	changeGenerator changes.ResourceChangeGenerator
	clock           core.Clock
	logger          core.Logger
}

// NewDefaultChecker creates a new instance
// of the default drift checker implementation.
func NewDefaultChecker(
	stateContainer state.Container,
	providers map[string]provider.Provider,
	changeGenerator changes.ResourceChangeGenerator,
	clock core.Clock,
	logger core.Logger,
) Checker {
	return &defaultChecker{
		stateContainer,
		providers,
		changeGenerator,
		clock,
		logger,
	}
}

func (c *defaultChecker) CheckDrift(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
) (map[string]*state.ResourceDriftState, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)
	instances := c.stateContainer.Instances()

	instanceLogger.Info(
		fmt.Sprintf("Fetching instance state for instance %s", instanceID),
	)
	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		instanceLogger.Debug(
			fmt.Sprintf("Failed to fetch instance state for instance %s", instanceID),
		)
		return nil, err
	}

	return c.checkDriftWithState(ctx, &instanceState, params, instanceLogger)
}

func (c *defaultChecker) CheckDriftWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) (map[string]*state.ResourceDriftState, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceState.InstanceID),
	)
	return c.checkDriftWithState(ctx, instanceState, params, instanceLogger)
}

func (c *defaultChecker) checkDriftWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	instanceLogger core.Logger,
) (map[string]*state.ResourceDriftState, error) {
	driftResults := map[string]*state.ResourceDriftState{}
	for _, resource := range instanceState.Resources {
		resourceLogger := instanceLogger.WithFields(
			core.StringLogField("resourceId", resource.ResourceID),
		)
		resourceLogger.Debug(
			fmt.Sprintf("Checking drift for resource %s", resource.ResourceID),
		)
		resourceDrift, err := c.checkResourceDrift(
			ctx,
			resource,
			instanceState.InstanceName,
			params,
			resourceLogger,
		)
		if err != nil {
			instanceLogger.Debug(
				fmt.Sprintf("Failed to check drift for resource %s", resource.ResourceID),
				core.StringLogField("resourceId", resource.ResourceID),
				core.ErrorLogField("error", err),
			)
			return nil, err
		}

		if resourceDrift != nil {
			driftResults[resource.ResourceID] = resourceDrift
		}
	}

	return driftResults, nil
}

func (c *defaultChecker) CheckResourceDrift(
	ctx context.Context,
	instanceID string,
	instanceName string,
	resourceID string,
	params core.BlueprintParams,
) (*state.ResourceDriftState, error) {
	resourceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("resourceId", resourceID),
	)
	resources := c.stateContainer.Resources()
	links := c.stateContainer.Links()

	resourceLogger.Info(
		fmt.Sprintf("Fetching state for resource %s", resourceID),
	)
	resourceState, err := resources.Get(ctx, resourceID)
	if err != nil {
		resourceLogger.Debug(
			fmt.Sprintf("Failed to fetch state for resource %s", resourceID),
		)
		return nil, err
	}

	linksWithResourceDataMappings, err := links.ListWithResourceDataMappings(
		ctx,
		instanceID,
		resourceState.Name,
	)
	if err != nil {
		resourceLogger.Debug(
			fmt.Sprintf("Failed to fetch resource data mappings for resource %s", resourceID),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	finalResourceState, err := applyLinksToResourceState(
		&resourceState,
		linksWithResourceDataMappings,
	)
	if err != nil {
		resourceLogger.Debug(
			fmt.Sprintf("Failed to apply links to resource state for resource %s", resourceID),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	return c.checkResourceDrift(ctx, &finalResourceState, instanceName, params, resourceLogger)
}

func (c *defaultChecker) checkResourceDrift(
	ctx context.Context,
	resource *state.ResourceState,
	instanceName string,
	params core.BlueprintParams,
	resourceLogger core.Logger,
) (*state.ResourceDriftState, error) {
	resourceLogger.Debug(
		"Loading resource plugin implementation for resource type",
		core.StringLogField("resourceType", resource.Type),
	)
	providerNamespace := provider.ExtractProviderFromItemType(resource.Type)
	resourceImpl, resourceProvider, err := c.getResourceImplementation(ctx, providerNamespace, resource.Type)
	if err != nil {
		resourceLogger.Debug(
			"Failed to load resource plugin implementation for resource type",
			core.StringLogField("resourceType", resource.Type),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	resourceLogger.Debug(
		"Loading retry policy for resource provider",
	)
	policy, err := c.getRetryPolicy(
		ctx,
		resourceProvider,
		provider.DefaultRetryPolicy,
	)
	if err != nil {
		resourceLogger.Debug(
			"Failed to load retry policy for resource provider",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	resourceLogger.Info(
		"Retrieving external state for the resource from the provider",
	)
	retryCtx := provider.CreateRetryContext(policy)
	providerCtx := provider.NewProviderContextFromParams(
		providerNamespace,
		params,
	)
	externalStateOutput, err := c.getResourceExternalState(
		ctx,
		resourceImpl,
		&provider.ResourceGetExternalStateInput{
			InstanceID:              resource.InstanceID,
			InstanceName:            instanceName,
			ResourceID:              resource.ResourceID,
			CurrentResourceSpec:     resource.SpecData,
			CurrentResourceMetadata: resource.Metadata,
			ProviderContext:         providerCtx,
		},
		retryCtx,
		resourceLogger,
	)
	if err != nil {
		return nil, err
	}

	if externalStateOutput == nil {
		resourceLogger.Debug(
			"External state for the resource is nil, moving on",
		)
		return nil, nil
	}

	driftedResourceInfo := createDriftedResourceInfo(
		resource,
		externalStateOutput,
	)
	resourceChanges, err := c.changeGenerator.GenerateChanges(
		ctx,
		driftedResourceInfo,
		resourceImpl,
		/* resolveOnDeploy */ []string{},
		params,
	)
	if err != nil {
		return nil, err
	}

	specDefinitionOutput, err := resourceImpl.GetSpecDefinition(ctx, &provider.ResourceGetSpecDefinitionInput{
		ProviderContext: providerCtx,
	})
	if err != nil {
		resourceLogger.Debug(
			"Failed to get spec definition for resource required for "+
				"determining changes that should be considered for drift",
			core.StringLogField("resourceType", resource.Type),
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	// Filter the resource changes to only include changes to fields that
	// are not marked to be ignored for drift checking in the spec definition.
	finalResourceChanges := filterChanges(
		resourceChanges,
		specDefinitionOutput.SpecDefinition,
		resourceLogger,
	)

	if !hasChanges(finalResourceChanges) {
		resourceLogger.Debug(
			"No changes detected indicating that the resource has not drifted" +
				", updating resource state as not drifted",
		)
		_, err = c.stateContainer.Resources().RemoveDrift(
			ctx,
			resource.ResourceID,
		)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	resourceLogger.Debug(
		"Changes have been detected indicating that the resource has drifted" +
			", updating resource state to reflect this",
	)

	currentTime := int(c.clock.Now().Unix())
	driftState := state.ResourceDriftState{
		ResourceID:   resource.ResourceID,
		ResourceName: resource.Name,
		SpecData:     resource.SpecData,
		Difference:   toResourceDriftChanges(finalResourceChanges),
		Timestamp:    &currentTime,
	}

	err = c.stateContainer.Resources().SaveDrift(
		ctx,
		driftState,
	)
	if err != nil {
		return nil, err
	}

	return &driftState, nil
}

func (c *defaultChecker) getResourceExternalState(
	ctx context.Context,
	resource provider.Resource,
	input *provider.ResourceGetExternalStateInput,
	retryCtx *provider.RetryContext,
	resourceLogger core.Logger,
) (*provider.ResourceGetExternalStateOutput, error) {
	getExternalStateStartTime := c.clock.Now()
	externalStateOutput, err := resource.GetExternalState(ctx, input)
	if err != nil {
		if provider.IsRetryableError(err) {
			resourceLogger.Debug(
				"retryable error occurred during external resource state retrieval",
				core.IntegerLogField("attempt", int64(retryCtx.Attempt)),
				core.ErrorLogField("error", err),
			)
			return c.handleGetResourceExternalStateRetry(
				ctx,
				resource,
				input,
				provider.RetryContextWithStartTime(
					retryCtx,
					getExternalStateStartTime,
				),
				resourceLogger,
			)
		}

		return nil, err
	}

	return externalStateOutput, nil
}

func (c *defaultChecker) handleGetResourceExternalStateRetry(
	ctx context.Context,
	resource provider.Resource,
	input *provider.ResourceGetExternalStateInput,
	retryCtx *provider.RetryContext,
	resourceLogger core.Logger,
) (*provider.ResourceGetExternalStateOutput, error) {
	currentAttemptDuration := c.clock.Since(
		retryCtx.AttemptStartTime,
	)
	nextRetryCtx := provider.RetryContextWithNextAttempt(retryCtx, currentAttemptDuration)

	if !nextRetryCtx.ExceededMaxRetries {
		waitTimeMs := provider.CalculateRetryWaitTimeMS(nextRetryCtx.Policy, nextRetryCtx.Attempt)
		time.Sleep(time.Duration(waitTimeMs) * time.Millisecond)
		return c.getResourceExternalState(
			ctx,
			resource,
			input,
			nextRetryCtx,
			resourceLogger,
		)
	}

	resourceLogger.Debug(
		"resource external state retrieval failed after reaching the maximum number of retries",
		core.IntegerLogField("attempt", int64(nextRetryCtx.Attempt)),
		core.IntegerLogField("maxRetries", int64(nextRetryCtx.Policy.MaxRetries)),
	)

	return nil, nil
}

func (c *defaultChecker) getResourceImplementation(
	ctx context.Context,
	providerNamespace string,
	resourceType string,
) (provider.Resource, provider.Provider, error) {
	provider, ok := c.providers[providerNamespace]
	if !ok {
		return nil, nil, fmt.Errorf("provider %s not found", providerNamespace)
	}

	resourceImpl, err := provider.Resource(ctx, resourceType)
	if err != nil {
		return nil, nil, err
	}

	return resourceImpl, provider, nil
}

func (c *defaultChecker) getRetryPolicy(
	ctx context.Context,
	resourceProvider provider.Provider,
	defaultRetryPolicy *provider.RetryPolicy,
) (*provider.RetryPolicy, error) {
	retryPolicy, err := resourceProvider.RetryPolicy(ctx)
	if err != nil {
		return nil, err
	}

	if retryPolicy == nil {
		return defaultRetryPolicy, nil
	}

	return retryPolicy, nil
}

func createDriftedResourceInfo(
	resource *state.ResourceState,
	externalStateOutput *provider.ResourceGetExternalStateOutput,
) *provider.ResourceInfo {
	resourceFromExternalState := externalStateOutput.ResourceSpecState
	return &provider.ResourceInfo{
		ResourceID:           resource.ResourceID,
		ResourceName:         resource.Name,
		CurrentResourceState: resource,
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{
				Value: resource.Type,
			},
			Description: core.MappingNodeFromString(resource.Description),
			Metadata:    createResolvedResourceMetadata(resource),
			Spec:        resourceFromExternalState,
		},
	}
}

func createResolvedResourceMetadata(
	resource *state.ResourceState,
) *provider.ResolvedResourceMetadata {
	if resource.Metadata == nil {
		return nil
	}

	return &provider.ResolvedResourceMetadata{
		DisplayName: core.MappingNodeFromString(
			resource.Metadata.DisplayName,
		),
		Annotations: &core.MappingNode{
			Fields: resource.Metadata.Annotations,
		},
		Labels: &schema.StringMap{
			Values: resource.Metadata.Labels,
		},
		Custom: resource.Metadata.Custom,
	}
}

func filterChanges(
	changes *provider.Changes,
	specDefinition *provider.ResourceSpecDefinition,
	resourceLogger core.Logger,
) *provider.Changes {
	if specDefinition == nil || specDefinition.Schema == nil {
		resourceLogger.Debug(
			"Spec definition is nil, all changes will be considered as drift",
		)
		return changes
	}

	filteredChanges := &provider.Changes{
		AppliedResourceInfo:       changes.AppliedResourceInfo,
		ModifiedFields:            withoutSpecFieldChanges(changes.ModifiedFields),
		NewFields:                 withoutSpecFieldChanges(changes.NewFields),
		RemovedFields:             withoutSpecFields(changes.RemovedFields),
		MustRecreate:              changes.MustRecreate,
		ComputedFields:            changes.ComputedFields,
		UnchangedFields:           changes.UnchangedFields,
		FieldChangesKnownOnDeploy: changes.FieldChangesKnownOnDeploy,
		ConditionKnownOnDeploy:    changes.ConditionKnownOnDeploy,
		NewOutboundLinks:          changes.NewOutboundLinks,
		OutboundLinkChanges:       changes.OutboundLinkChanges,
		RemovedOutboundLinks:      changes.RemovedOutboundLinks,
	}

	// Walk the schema, for each non-computed field, check if there are any changes
	// in the modified, new or removed fields, if so, and IgnoreDrift is true, then
	// omit the change from the final change set
	filterSchemaChanges(specDefinition.Schema, changes, filteredChanges, "$", 0, resourceLogger)

	return filteredChanges
}

func withoutSpecFieldChanges(
	sourceFieldChanges []provider.FieldChange,
) []provider.FieldChange {
	filteredChanges := []provider.FieldChange{}
	for _, fieldChange := range sourceFieldChanges {
		if !isFieldInSpec(fieldChange.FieldPath) {
			filteredChanges = append(filteredChanges, fieldChange)
		}
	}
	return filteredChanges
}

func withoutSpecFields(
	sourceRemovedFields []string,
) []string {
	filteredRemovedFields := []string{}
	for _, removedField := range sourceRemovedFields {
		if !isFieldInSpec(removedField) {
			filteredRemovedFields = append(filteredRemovedFields, removedField)
		}
	}
	return filteredRemovedFields
}

func filterSchemaChanges(
	schema *provider.ResourceDefinitionsSchema,
	source *provider.Changes,
	destination *provider.Changes,
	schemaPath string,
	depth int,
	resourceLogger core.Logger,
) {

	if schema == nil ||
		depth >= core.MappingNodeMaxTraverseDepth ||
		// Computed value that should not be tracked for drift.
		schema.Computed && !schema.TrackDrift {
		return
	}

	filteredModifiedFields := filterFieldChanges(
		schema,
		source.ModifiedFields,
		schemaPath,
		resourceLogger,
	)
	destination.ModifiedFields = append(destination.ModifiedFields, filteredModifiedFields...)

	filteredNewFields := filterFieldChanges(
		schema,
		source.NewFields,
		schemaPath,
		resourceLogger,
	)
	destination.NewFields = append(destination.NewFields, filteredNewFields...)

	filteredRemovedFields := filterRemovedFields(
		schema,
		source.RemovedFields,
		schemaPath,
		resourceLogger,
	)
	destination.RemovedFields = append(destination.RemovedFields, filteredRemovedFields...)

	if schema.Type == provider.ResourceDefinitionsSchemaTypeObject {
		for attrName, attrSchema := range schema.Attributes {
			attrPath := substitutions.RenderFieldPath(schemaPath, attrName)
			filterSchemaChanges(
				attrSchema,
				source,
				destination,
				attrPath,
				depth+1,
				resourceLogger,
			)
		}
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeArray {
		// "[*]" in the path matches any index in the array.
		arrayPath := fmt.Sprintf("%s[*]", schemaPath)
		filterSchemaChanges(
			schema.Items,
			source,
			destination,
			arrayPath,
			depth+1,
			resourceLogger,
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeMap {
		// ".*" in the path matches any key in the map.
		mapPath := fmt.Sprintf("%s.*", schemaPath)
		filterSchemaChanges(
			schema.Items,
			source,
			destination,
			mapPath,
			depth+1,
			resourceLogger,
		)
	}

	if schema.Type == provider.ResourceDefinitionsSchemaTypeUnion && !schema.IgnoreDrift {
		for _, unionSchema := range schema.OneOf {
			filterSchemaChanges(
				unionSchema,
				source,
				destination,
				// Use the same schema path for union schemas, as they are not nested
				// within the parent schema, but rather represent an alternative
				// schema for the same field.
				schemaPath,
				depth+1,
				resourceLogger,
			)
		}
	}
}

func filterFieldChanges(
	schema *provider.ResourceDefinitionsSchema,
	sourceFieldChanges []provider.FieldChange,
	schemaPath string,
	logger core.Logger,
) []provider.FieldChange {
	filteredChanges := []provider.FieldChange{}

	for _, fieldChange := range sourceFieldChanges {
		if isFieldInSpec(fieldChange.FieldPath) {
			fieldSearchPath := core.ReplaceSpecWithRoot(fieldChange.FieldPath)
			pathsEqual, _ := core.PathMatchesPattern(
				fieldSearchPath,
				schemaPath,
			)
			if pathsEqual && !schema.IgnoreDrift {
				filteredChanges = append(
					filteredChanges,
					provider.FieldChange{
						FieldPath:    fieldChange.FieldPath,
						PrevValue:    fieldChange.PrevValue,
						NewValue:     fieldChange.NewValue,
						MustRecreate: fieldChange.MustRecreate,
						Sensitive:    schema.Sensitive,
					},
				)
			} else if pathsEqual && schema.IgnoreDrift {
				logger.Debug(
					fmt.Sprintf(
						"Ignoring drift for new or modified field %s as it is marked to be ignored",
						fieldChange.FieldPath,
					),
				)
			}
		}
	}

	return filteredChanges
}

func filterRemovedFields(
	schema *provider.ResourceDefinitionsSchema,
	sourceRemovedFields []string,
	schemaPath string,
	logger core.Logger,
) []string {
	filteredRemovedFields := []string{}

	for _, removedField := range sourceRemovedFields {
		if isFieldInSpec(removedField) {
			fieldSearchPath := core.ReplaceSpecWithRoot(removedField)
			pathsEqual, _ := core.PathMatchesPattern(
				fieldSearchPath,
				schemaPath,
			)
			if pathsEqual && !schema.IgnoreDrift {
				filteredRemovedFields = append(filteredRemovedFields, removedField)
			} else if pathsEqual && schema.IgnoreDrift {
				logger.Debug(
					fmt.Sprintf(
						"Ignoring drift for removed field %s as it is marked to be ignored",
						removedField,
					),
				)
			}
		}
	}

	return filteredRemovedFields
}

func isFieldInSpec(fieldPath string) bool {
	return strings.HasPrefix(fieldPath, "spec.") ||
		strings.HasPrefix(fieldPath, "spec[")
}

func hasChanges(
	changes *provider.Changes,
) bool {
	return len(changes.ModifiedFields) > 0 ||
		len(changes.NewFields) > 0 ||
		len(changes.RemovedFields) > 0
}

func toResourceDriftChanges(changes *provider.Changes) *state.ResourceDriftChanges {
	return &state.ResourceDriftChanges{
		ModifiedFields:  toResourceDriftFieldChanges(changes.ModifiedFields),
		NewFields:       toResourceDriftFieldChanges(changes.NewFields),
		RemovedFields:   changes.RemovedFields,
		UnchangedFields: changes.UnchangedFields,
	}
}

func toResourceDriftFieldChanges(
	fieldChanges []provider.FieldChange,
) []*state.ResourceDriftFieldChange {
	return commoncore.Map(
		fieldChanges,
		func(fieldChange provider.FieldChange, _ int) *state.ResourceDriftFieldChange {
			return &state.ResourceDriftFieldChange{
				FieldPath:    fieldChange.FieldPath,
				StateValue:   fieldChange.PrevValue,
				DriftedValue: fieldChange.NewValue,
			}
		},
	)
}

func applyLinksToResourceState(
	resourceState *state.ResourceState,
	linksWithResourceDataMappings []state.LinkState,
) (state.ResourceState, error) {
	appliedResourceState := state.ResourceState{
		ResourceID:                 resourceState.ResourceID,
		Name:                       resourceState.Name,
		Type:                       resourceState.Type,
		TemplateName:               resourceState.TemplateName,
		InstanceID:                 resourceState.InstanceID,
		Status:                     resourceState.Status,
		PreciseStatus:              resourceState.PreciseStatus,
		LastStatusUpdateTimestamp:  resourceState.LastStatusUpdateTimestamp,
		LastDeployedTimestamp:      resourceState.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: resourceState.LastDeployAttemptTimestamp,
		SpecData:                   core.CopyMappingNode(resourceState.SpecData),
		Description:                resourceState.Description,
		Metadata:                   resourceState.Metadata,
		DependsOnResources:         resourceState.DependsOnResources,
		DependsOnChildren:          resourceState.DependsOnChildren,
		FailureReasons:             resourceState.FailureReasons,
		Drifted:                    resourceState.Drifted,
		LastDriftDetectedTimestamp: resourceState.LastDriftDetectedTimestamp,
		Durations:                  resourceState.Durations,
	}

	for _, link := range linksWithResourceDataMappings {
		for resourceFieldPath, linkFieldPath := range link.ResourceDataMappings {
			// resourceFieldPath is in the form "resourceName::fieldPath".
			parts := strings.SplitN(resourceFieldPath, "::", 2)
			if len(parts) == 2 {
				linkDataPathWithRoot := core.AddRootToPath(linkFieldPath)
				linkDataValue, _ := core.GetPathValue(
					linkDataPathWithRoot,
					&core.MappingNode{
						Fields: link.Data,
					},
					core.MappingNodeMaxTraverseDepth,
				)

				if linkDataValue != nil {
					fieldPath := core.ReplaceSpecWithRoot(parts[1])
					err := core.InjectPathValueReplaceFields(
						fieldPath,
						linkDataValue,
						appliedResourceState.SpecData,
						core.MappingNodeMaxTraverseDepth,
					)
					if err != nil {
						return state.ResourceState{}, fmt.Errorf(
							"failed to apply link data to resource state: %w",
							err,
						)
					}
				}
			}
		}
	}

	return appliedResourceState, nil
}

func (c *defaultChecker) CheckInterruptedResources(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
) ([]ReconcileResult, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)
	instances := c.stateContainer.Instances()

	instanceLogger.Info("checking for interrupted resources that require reconciliation")
	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		instanceLogger.Debug(
			"failed to fetch instance state for interrupted resource check",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	return c.checkInterruptedResourcesWithState(ctx, &instanceState, params, instanceLogger)
}

func (c *defaultChecker) CheckInterruptedResourcesWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]ReconcileResult, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceState.InstanceID),
	)
	return c.checkInterruptedResourcesWithState(ctx, instanceState, params, instanceLogger)
}

func (c *defaultChecker) checkInterruptedResourcesWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	instanceLogger core.Logger,
) ([]ReconcileResult, error) {
	results := []ReconcileResult{}
	for _, resource := range instanceState.Resources {
		if !isInterruptedStatus(resource.PreciseStatus) {
			continue
		}

		resourceLogger := instanceLogger.WithFields(
			core.StringLogField("resourceId", resource.ResourceID),
			core.StringLogField("resourceName", resource.Name),
		)
		resourceLogger.Info("found interrupted resource, fetching external state")

		result, err := c.checkInterruptedResource(
			ctx,
			resource,
			instanceState.InstanceName,
			params,
			resourceLogger,
		)
		if err != nil {
			resourceLogger.Debug(
				"failed to check interrupted resource",
				core.ErrorLogField("error", err),
			)
			return nil, err
		}

		results = append(results, *result)
	}

	return results, nil
}

func (c *defaultChecker) checkInterruptedResource(
	ctx context.Context,
	resource *state.ResourceState,
	instanceName string,
	params core.BlueprintParams,
	resourceLogger core.Logger,
) (*ReconcileResult, error) {
	providerNamespace := provider.ExtractProviderFromItemType(resource.Type)
	resourceImpl, resourceProvider, err := c.getResourceImplementation(ctx, providerNamespace, resource.Type)
	if err != nil {
		return nil, err
	}

	policy, err := c.getRetryPolicy(ctx, resourceProvider, provider.DefaultRetryPolicy)
	if err != nil {
		return nil, err
	}

	retryCtx := provider.CreateRetryContext(policy)
	providerCtx := provider.NewProviderContextFromParams(providerNamespace, params)

	externalStateOutput, err := c.getResourceExternalState(
		ctx,
		resourceImpl,
		&provider.ResourceGetExternalStateInput{
			InstanceID:              resource.InstanceID,
			InstanceName:            instanceName,
			ResourceID:              resource.ResourceID,
			CurrentResourceSpec:     resource.SpecData,
			CurrentResourceMetadata: resource.Metadata,
			ProviderContext:         providerCtx,
		},
		retryCtx,
		resourceLogger,
	)
	if err != nil {
		return nil, err
	}

	// Determine the new status based on whether the resource exists in the cloud
	newStatus := determineReconcileStatus(resource.PreciseStatus, externalStateOutput)

	result := &ReconcileResult{
		ResourceID:     resource.ResourceID,
		ResourceName:   resource.Name,
		ResourceType:   resource.Type,
		OldStatus:      resource.PreciseStatus,
		NewStatus:      newStatus,
		PersistedState: resource.SpecData,
	}

	// If the resource exists in the cloud, generate the state changes
	if externalStateOutput != nil && externalStateOutput.ResourceSpecState != nil {
		result.ExternalState = externalStateOutput.ResourceSpecState

		driftedResourceInfo := createDriftedResourceInfo(resource, externalStateOutput)
		stateChanges, err := c.changeGenerator.GenerateChanges(
			ctx,
			driftedResourceInfo,
			resourceImpl,
			/* resolveOnDeploy */ []string{},
			params,
		)
		if err != nil {
			resourceLogger.Debug(
				"failed to generate state changes for interrupted resource",
				core.ErrorLogField("error", err),
			)
			// Don't fail the whole reconciliation check if we can't generate changes
			// The user can still see the status transition
		} else {
			result.StateChanges = stateChanges
		}
	}

	return result, nil
}

func (c *defaultChecker) ApplyReconciliation(
	ctx context.Context,
	results []ReconcileResult,
) error {
	resources := c.stateContainer.Resources()
	currentTime := int(c.clock.Now().Unix())

	for _, result := range results {
		resourceLogger := c.logger.WithFields(
			core.StringLogField("resourceId", result.ResourceID),
			core.StringLogField("resourceName", result.ResourceName),
		)

		if result.ResourceExists() {
			// Resource exists in cloud - update both status AND spec data
			// to reflect the actual cloud state
			resourceLogger.Info(
				"applying reconciliation: updating resource state from external state",
				core.StringLogField("oldStatus", result.OldStatus.String()),
				core.StringLogField("newStatus", result.NewStatus.String()),
			)

			// First get the current resource state
			currentState, err := resources.Get(ctx, result.ResourceID)
			if err != nil {
				return fmt.Errorf(
					"failed to get current state for resource %s: %w",
					result.ResourceName,
					err,
				)
			}

			// Update the state with the external state and new status
			currentState.Status = preciseToResourceStatus(result.NewStatus)
			currentState.PreciseStatus = result.NewStatus
			currentState.LastStatusUpdateTimestamp = currentTime
			currentState.SpecData = result.ExternalState
			currentState.FailureReasons = nil // Clear any previous failure reasons

			// Save the updated state
			err = resources.Save(ctx, currentState)
			if err != nil {
				return fmt.Errorf(
					"failed to apply reconciliation for resource %s: %w",
					result.ResourceName,
					err,
				)
			}
		} else {
			// Resource doesn't exist in upstream provider - just update status to failed
			resourceLogger.Info(
				"applying reconciliation: resource not found, marking as failed",
				core.StringLogField("oldStatus", result.OldStatus.String()),
				core.StringLogField("newStatus", result.NewStatus.String()),
			)
			err := resources.UpdateStatus(
				ctx,
				result.ResourceID,
				state.ResourceStatusInfo{
					Status:                    preciseToResourceStatus(result.NewStatus),
					PreciseStatus:             result.NewStatus,
					LastStatusUpdateTimestamp: &currentTime,
					FailureReasons:            []string{"resource not found during reconciliation"},
				},
			)
			if err != nil {
				return fmt.Errorf(
					"failed to apply reconciliation for resource %s: %w",
					result.ResourceName,
					err,
				)
			}
		}
	}

	return nil
}

// isInterruptedStatus returns true if the given precise resource status
// indicates the resource was interrupted.
func isInterruptedStatus(status core.PreciseResourceStatus) bool {
	return status == core.PreciseResourceStatusCreateInterrupted ||
		status == core.PreciseResourceStatusUpdateInterrupted ||
		status == core.PreciseResourceStatusDestroyInterrupted
}

// determineReconcileStatus determines the new status for a resource based on
// its previous interrupted status and whether it exists in the cloud.
func determineReconcileStatus(
	oldStatus core.PreciseResourceStatus,
	externalState *provider.ResourceGetExternalStateOutput,
) core.PreciseResourceStatus {
	resourceExists := externalState != nil && externalState.ResourceSpecState != nil

	switch oldStatus {
	case core.PreciseResourceStatusCreateInterrupted:
		if resourceExists {
			return core.PreciseResourceStatusCreated
		}
		return core.PreciseResourceStatusCreateFailed

	case core.PreciseResourceStatusUpdateInterrupted:
		if resourceExists {
			return core.PreciseResourceStatusUpdated
		}
		// If resource doesn't exist after an update interruption,
		// something went very wrong - treat as update failed
		return core.PreciseResourceStatusUpdateFailed

	case core.PreciseResourceStatusDestroyInterrupted:
		if resourceExists {
			// Resource still exists, destruction didn't complete
			// Revert to previous stable state (updated or created)
			return core.PreciseResourceStatusUpdated
		}
		return core.PreciseResourceStatusDestroyed

	default:
		// Shouldn't happen, but return the old status if we don't recognize it
		return oldStatus
	}
}

// preciseToResourceStatus converts a precise resource status to a regular resource status.
func preciseToResourceStatus(preciseStatus core.PreciseResourceStatus) core.ResourceStatus {
	switch preciseStatus {
	case core.PreciseResourceStatusCreated, core.PreciseResourceStatusConfigComplete:
		return core.ResourceStatusCreated
	case core.PreciseResourceStatusCreateFailed:
		return core.ResourceStatusCreateFailed
	case core.PreciseResourceStatusUpdated, core.PreciseResourceStatusUpdateConfigComplete:
		return core.ResourceStatusUpdated
	case core.PreciseResourceStatusUpdateFailed:
		return core.ResourceStatusUpdateFailed
	case core.PreciseResourceStatusDestroyed:
		return core.ResourceStatusDestroyed
	case core.PreciseResourceStatusDestroyFailed:
		return core.ResourceStatusDestroyFailed
	case core.PreciseResourceStatusCreating:
		return core.ResourceStatusCreating
	case core.PreciseResourceStatusUpdating:
		return core.ResourceStatusUpdating
	case core.PreciseResourceStatusDestroying:
		return core.ResourceStatusDestroying
	default:
		return core.ResourceStatusUnknown
	}
}

func (c *defaultChecker) CheckLinkDrift(
	ctx context.Context,
	instanceID string,
	linkID string,
	params core.BlueprintParams,
) (*state.LinkDriftState, error) {
	linkLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
		core.StringLogField("linkId", linkID),
	)
	links := c.stateContainer.Links()
	instances := c.stateContainer.Instances()

	linkLogger.Info("fetching link state for drift check")
	linkState, err := links.Get(ctx, linkID)
	if err != nil {
		linkLogger.Debug(
			"failed to fetch link state",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		linkLogger.Debug(
			"failed to fetch instance state for link drift check",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	return c.checkLinkDrift(ctx, &linkState, &instanceState, params, linkLogger)
}

func (c *defaultChecker) CheckAllLinkDrift(
	ctx context.Context,
	instanceID string,
	params core.BlueprintParams,
) (map[string]*state.LinkDriftState, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceID),
	)
	instances := c.stateContainer.Instances()

	instanceLogger.Info("fetching instance state for link drift check")
	instanceState, err := instances.Get(ctx, instanceID)
	if err != nil {
		instanceLogger.Debug(
			"failed to fetch instance state for link drift check",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	return c.checkAllLinkDriftWithState(ctx, &instanceState, params, instanceLogger)
}

func (c *defaultChecker) CheckAllLinkDriftWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) (map[string]*state.LinkDriftState, error) {
	instanceLogger := c.logger.WithFields(
		core.StringLogField("instanceId", instanceState.InstanceID),
	)
	return c.checkAllLinkDriftWithState(ctx, instanceState, params, instanceLogger)
}

func (c *defaultChecker) checkAllLinkDriftWithState(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	instanceLogger core.Logger,
) (map[string]*state.LinkDriftState, error) {
	driftResults := map[string]*state.LinkDriftState{}
	for _, link := range instanceState.Links {
		linkLogger := instanceLogger.WithFields(
			core.StringLogField("linkId", link.LinkID),
			core.StringLogField("linkName", link.Name),
		)
		linkLogger.Debug("checking drift for link")

		linkDrift, err := c.checkLinkDrift(ctx, link, instanceState, params, linkLogger)
		if err != nil {
			linkLogger.Debug(
				"failed to check drift for link",
				core.ErrorLogField("error", err),
			)
			return nil, err
		}

		if linkDrift != nil {
			driftResults[link.LinkID] = linkDrift
		}
	}

	return driftResults, nil
}

func (c *defaultChecker) checkLinkDrift(
	ctx context.Context,
	link *state.LinkState,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	linkLogger core.Logger,
) (*state.LinkDriftState, error) {
	var resourceADrift *state.LinkResourceDrift
	var resourceBDrift *state.LinkResourceDrift
	var err error

	// Check for drift via ResourceDataMappings
	if len(link.ResourceDataMappings) > 0 {
		resourceADrift, resourceBDrift, err = c.checkLinkDriftViaResourceDataMappings(
			ctx,
			link,
			instanceState,
			params,
			linkLogger,
		)
		if err != nil {
			return nil, err
		}
	}

	// Check for drift in intermediary resources
	var intermediaryDrift map[string]*state.IntermediaryDriftState
	if len(link.IntermediaryResourceStates) > 0 {
		intermediaryDrift, err = c.checkIntermediaryResourceDrift(
			ctx,
			link,
			instanceState,
			params,
			linkLogger,
		)
		if err != nil {
			return nil, err
		}
	}

	// If no drift detected, remove any existing drift state and return nil
	if resourceADrift == nil && resourceBDrift == nil && len(intermediaryDrift) == 0 {
		linkLogger.Debug("no link drift detected, removing any existing drift state")
		_, err = c.stateContainer.Links().RemoveDrift(ctx, link.LinkID)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Drift detected - create and persist the drift state
	linkLogger.Debug("link drift detected, persisting drift state")
	currentTime := int(c.clock.Now().Unix())
	driftState := state.LinkDriftState{
		LinkID:            link.LinkID,
		LinkName:          link.Name,
		ResourceADrift:    resourceADrift,
		ResourceBDrift:    resourceBDrift,
		IntermediaryDrift: intermediaryDrift,
		Timestamp:         &currentTime,
	}

	err = c.stateContainer.Links().SaveDrift(ctx, driftState)
	if err != nil {
		return nil, err
	}

	return &driftState, nil
}

func (c *defaultChecker) checkLinkDriftViaResourceDataMappings(
	ctx context.Context,
	link *state.LinkState,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	linkLogger core.Logger,
) (*state.LinkResourceDrift, *state.LinkResourceDrift, error) {
	// Parse link name to get resource names (format: "resourceA::resourceB")
	resourceAName, resourceBName := parseLinkName(link.Name)

	// Group mappings by resource
	resourceAMappings := map[string]string{}
	resourceBMappings := map[string]string{}

	for resourceFieldPath, linkDataPath := range link.ResourceDataMappings {
		parts := strings.SplitN(resourceFieldPath, "::", 2)
		if len(parts) != 2 {
			continue
		}
		resourceName, fieldPath := parts[0], parts[1]

		switch resourceName {
		case resourceAName:
			resourceAMappings[fieldPath] = linkDataPath
		case resourceBName:
			resourceBMappings[fieldPath] = linkDataPath
		}
	}

	var resourceADrift *state.LinkResourceDrift
	var resourceBDrift *state.LinkResourceDrift

	// Check ResourceA drift
	if len(resourceAMappings) > 0 {
		resourceA := findResourceByName(instanceState, resourceAName)
		if resourceA != nil {
			drift, err := c.checkResourceFieldsForLinkDrift(
				ctx,
				resourceA,
				resourceAMappings,
				link.Data,
				instanceState.InstanceName,
				params,
				linkLogger,
			)
			if err != nil {
				return nil, nil, err
			}
			resourceADrift = drift
		}
	}

	// Check ResourceB drift
	if len(resourceBMappings) > 0 {
		resourceB := findResourceByName(instanceState, resourceBName)
		if resourceB != nil {
			drift, err := c.checkResourceFieldsForLinkDrift(
				ctx,
				resourceB,
				resourceBMappings,
				link.Data,
				instanceState.InstanceName,
				params,
				linkLogger,
			)
			if err != nil {
				return nil, nil, err
			}
			resourceBDrift = drift
		}
	}

	return resourceADrift, resourceBDrift, nil
}

func (c *defaultChecker) checkResourceFieldsForLinkDrift(
	ctx context.Context,
	resource *state.ResourceState,
	// resourceFieldPath -> linkDataPath
	mappings map[string]string,
	linkData map[string]*core.MappingNode,
	instanceName string,
	params core.BlueprintParams,
	linkLogger core.Logger,
) (*state.LinkResourceDrift, error) {
	providerNamespace := provider.ExtractProviderFromItemType(resource.Type)
	resourceImpl, resourceProvider, err := c.getResourceImplementation(ctx, providerNamespace, resource.Type)
	if err != nil {
		return nil, err
	}

	policy, err := c.getRetryPolicy(ctx, resourceProvider, provider.DefaultRetryPolicy)
	if err != nil {
		return nil, err
	}

	retryCtx := provider.CreateRetryContext(policy)
	providerCtx := provider.NewProviderContextFromParams(providerNamespace, params)

	externalStateOutput, err := c.getResourceExternalState(
		ctx,
		resourceImpl,
		&provider.ResourceGetExternalStateInput{
			InstanceID:              resource.InstanceID,
			InstanceName:            instanceName,
			ResourceID:              resource.ResourceID,
			CurrentResourceSpec:     resource.SpecData,
			CurrentResourceMetadata: resource.Metadata,
			ProviderContext:         providerCtx,
		},
		retryCtx,
		linkLogger,
	)
	if err != nil {
		return nil, err
	}

	if externalStateOutput == nil || externalStateOutput.ResourceSpecState == nil {
		// Resource doesn't exist externally, can't check for link drift
		return nil, nil
	}

	// Compare the mapped fields
	changes := c.compareLinkedFields(mappings, linkData, externalStateOutput.ResourceSpecState)
	if len(changes) == 0 {
		return nil, nil
	}

	return &state.LinkResourceDrift{
		ResourceID:         resource.ResourceID,
		ResourceName:       resource.Name,
		MappedFieldChanges: changes,
	}, nil
}

func (c *defaultChecker) compareLinkedFields(
	mappings map[string]string, // resourceFieldPath -> linkDataPath
	linkData map[string]*core.MappingNode,
	externalResourceState *core.MappingNode,
) []*state.LinkDriftFieldChange {
	var changes []*state.LinkDriftFieldChange

	for resourceFieldPath, linkDataPath := range mappings {
		// Get value from link.Data
		linkDataPathWithRoot := core.AddRootToPath(linkDataPath)
		linkValue, _ := core.GetPathValue(
			linkDataPathWithRoot,
			&core.MappingNode{
				Fields: linkData,
			},
			core.MappingNodeMaxTraverseDepth,
		)

		// Get value from external resource state
		// resourceFieldPath is already in "spec.xxx" format
		externalFieldPath := core.ReplaceSpecWithRoot(resourceFieldPath)
		externalValue, _ := core.GetPathValue(
			externalFieldPath,
			externalResourceState,
			core.MappingNodeMaxTraverseDepth,
		)

		// Compare the values
		if !core.MappingNodeEqual(linkValue, externalValue) {
			changes = append(changes, &state.LinkDriftFieldChange{
				ResourceFieldPath: resourceFieldPath,
				LinkDataPath:      linkDataPath,
				LinkDataValue:     linkValue,
				ExternalValue:     externalValue,
			})
		}
	}

	return changes
}

func (c *defaultChecker) checkIntermediaryResourceDrift(
	ctx context.Context,
	link *state.LinkState,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	linkLogger core.Logger,
) (map[string]*state.IntermediaryDriftState, error) {
	// Get the link implementation to call GetIntermediaryExternalState
	resourceAName, resourceBName := parseLinkName(link.Name)
	resourceA := findResourceByName(instanceState, resourceAName)
	resourceB := findResourceByName(instanceState, resourceBName)

	if resourceA == nil || resourceB == nil {
		linkLogger.Debug("could not find linked resources for intermediary drift check")
		return nil, nil
	}

	// Get the provider namespace from one of the resources
	providerNamespace := provider.ExtractProviderFromItemType(resourceA.Type)
	prov, ok := c.providers[providerNamespace]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", providerNamespace)
	}

	linkImpl, err := prov.Link(ctx, resourceA.Type, resourceB.Type)
	if err != nil {
		linkLogger.Debug(
			"failed to get link implementation for intermediary drift check",
			core.ErrorLogField("error", err),
		)
		return nil, err
	}

	linkCtx := provider.NewLinkContextFromParams(params)

	// Call GetIntermediaryExternalState on the link implementation
	externalOutput, err := linkImpl.GetIntermediaryExternalState(ctx, &provider.LinkGetIntermediaryExternalStateInput{
		InstanceID:       instanceState.InstanceID,
		InstanceName:     instanceState.InstanceName,
		LinkID:           link.LinkID,
		LinkName:         link.Name,
		ResourceAInfo:    createResourceInfoFromState(resourceA),
		ResourceBInfo:    createResourceInfoFromState(resourceB),
		CurrentLinkState: link,
		LinkContext:      linkCtx,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get intermediary external state for link %s: %w", link.Name, err)
	}

	if externalOutput == nil || len(externalOutput.IntermediaryStates) == 0 {
		return nil, nil
	}

	// Compare external state to persisted state
	currentTime := int(c.clock.Now().Unix())
	results := make(map[string]*state.IntermediaryDriftState)

	for _, intermediary := range link.IntermediaryResourceStates {
		externalState := externalOutput.IntermediaryStates[intermediary.ResourceID]
		if externalState == nil {
			// Intermediary exists in state but not returned by provider
			// This could mean it was deleted externally
			changes := generateIntermediaryChangesForDeleted(intermediary.ResourceSpecData)
			results[intermediary.ResourceID] = &state.IntermediaryDriftState{
				ResourceID:     intermediary.ResourceID,
				ResourceType:   intermediary.ResourceType,
				PersistedState: intermediary.ResourceSpecData,
				ExternalState:  nil,
				Changes:        changes,
				Exists:         false,
				Timestamp:      &currentTime,
			}
			continue
		}

		// Compare the states
		if !core.MappingNodeEqual(intermediary.ResourceSpecData, externalState.SpecData) {
			changes := generateIntermediaryChanges(intermediary.ResourceSpecData, externalState.SpecData)
			results[intermediary.ResourceID] = &state.IntermediaryDriftState{
				ResourceID:     intermediary.ResourceID,
				ResourceType:   intermediary.ResourceType,
				PersistedState: intermediary.ResourceSpecData,
				ExternalState:  externalState.SpecData,
				Changes:        changes,
				Exists:         externalState.Exists,
				Timestamp:      &currentTime,
			}
		}
	}

	return results, nil
}

func generateIntermediaryChanges(
	persisted *core.MappingNode,
	external *core.MappingNode,
) *state.IntermediaryDriftChanges {
	if persisted == nil && external == nil {
		return nil
	}

	changes := &state.IntermediaryDriftChanges{}

	persistedFields := extractTopLevelFields(persisted)
	externalFields := extractTopLevelFields(external)

	// Find modified and removed fields
	for fieldPath, persistedValue := range persistedFields {
		externalValue, exists := externalFields[fieldPath]
		if !exists {
			changes.RemovedFields = append(changes.RemovedFields, state.IntermediaryFieldChange{
				FieldPath: fieldPath,
				PrevValue: persistedValue,
			})
		} else if !core.MappingNodeEqual(persistedValue, externalValue) {
			changes.ModifiedFields = append(changes.ModifiedFields, state.IntermediaryFieldChange{
				FieldPath: fieldPath,
				PrevValue: persistedValue,
				NewValue:  externalValue,
			})
		}
	}

	// Find new fields
	for fieldPath, externalValue := range externalFields {
		if _, exists := persistedFields[fieldPath]; !exists {
			changes.NewFields = append(changes.NewFields, state.IntermediaryFieldChange{
				FieldPath: fieldPath,
				NewValue:  externalValue,
			})
		}
	}

	if len(changes.ModifiedFields) == 0 && len(changes.NewFields) == 0 && len(changes.RemovedFields) == 0 {
		return nil
	}

	return changes
}

func generateIntermediaryChangesForDeleted(persisted *core.MappingNode) *state.IntermediaryDriftChanges {
	if persisted == nil {
		return nil
	}

	changes := &state.IntermediaryDriftChanges{}
	persistedFields := extractTopLevelFields(persisted)

	for fieldPath, persistedValue := range persistedFields {
		changes.RemovedFields = append(changes.RemovedFields, state.IntermediaryFieldChange{
			FieldPath: fieldPath,
			PrevValue: persistedValue,
		})
	}

	if len(changes.RemovedFields) == 0 {
		return nil
	}

	return changes
}

func extractTopLevelFields(node *core.MappingNode) map[string]*core.MappingNode {
	if node == nil || node.Fields == nil {
		return map[string]*core.MappingNode{}
	}
	return node.Fields
}

// parseLinkName parses a link name in the format "resourceA::resourceB"
// and returns the two resource names.
func parseLinkName(linkName string) (string, string) {
	parts := strings.SplitN(linkName, "::", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// findResourceByName finds a resource in the instance state by its logical name.
func findResourceByName(instanceState *state.InstanceState, resourceName string) *state.ResourceState {
	for _, resource := range instanceState.Resources {
		if resource.Name == resourceName {
			return resource
		}
	}
	return nil
}

// createResourceInfoFromState creates a provider.ResourceInfo from a resource state.
func createResourceInfoFromState(resource *state.ResourceState) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceID:           resource.ResourceID,
		ResourceName:         resource.Name,
		CurrentResourceState: resource,
	}
}
