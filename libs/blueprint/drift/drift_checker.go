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

		// A nil resource drift means that the resource has not drifted.
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
	if schema == nil || depth >= core.MappingNodeMaxTraverseDepth || schema.Computed {
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
