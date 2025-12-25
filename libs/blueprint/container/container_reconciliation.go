package container

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/drift"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// maxLinkDataUpdatePathDepth is the maximum depth for traversing paths
// when extracting values from external state for link data updates.
const maxLinkDataUpdatePathDepth = 10

func (c *defaultBlueprintContainer) CheckReconciliation(
	ctx context.Context,
	input *CheckReconciliationInput,
	paramOverrides core.BlueprintParams,
) (*ReconciliationCheckResult, error) {
	if input == nil {
		return nil, fmt.Errorf("check reconciliation input is required")
	}

	if input.InstanceID == "" {
		return nil, fmt.Errorf("instance ID is required for reconciliation check")
	}

	instanceState, err := c.stateContainer.Instances().Get(ctx, input.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance state: %w", err)
	}

	result := &ReconciliationCheckResult{
		InstanceID: input.InstanceID,
		Resources:  []ResourceReconcileResult{},
		Links:      []LinkReconcileResult{},
	}

	resourceResults, err := c.checkResourceReconciliation(ctx, input, &instanceState, paramOverrides)
	if err != nil {
		return nil, err
	}
	result.Resources = resourceResults

	linkResults, err := c.checkLinkReconciliation(ctx, input, &instanceState, paramOverrides)
	if err != nil {
		return nil, err
	}
	result.Links = linkResults

	for _, r := range result.Resources {
		if r.Type == ReconciliationTypeInterrupted {
			result.HasInterrupted = true
		}
		if r.Type == ReconciliationTypeDrift {
			result.HasDrift = true
		}
		// Track if any issues are in child blueprints
		if r.ChildPath != "" {
			result.HasChildIssues = true
		}
	}
	for _, l := range result.Links {
		if l.Type == ReconciliationTypeInterrupted {
			result.HasInterrupted = true
		}
		if l.Type == ReconciliationTypeDrift {
			result.HasDrift = true
		}
		// Track if any issues are in child blueprints
		if l.ChildPath != "" {
			result.HasChildIssues = true
		}
	}

	return result, nil
}

func (c *defaultBlueprintContainer) checkResourceReconciliation(
	ctx context.Context,
	input *CheckReconciliationInput,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]ResourceReconcileResult, error) {
	includeChildren := shouldIncludeChildren(input)

	switch input.Scope {
	case ReconciliationScopeInterrupted:
		return c.checkInterruptedResourceReconciliationWithChildren(
			ctx, instanceState, params, includeChildren, input.ChildPath,
		)
	case ReconciliationScopeSpecific:
		return c.checkSpecificResourceReconciliation(ctx, input, instanceState, params)
	case ReconciliationScopeAll:
		return c.checkAllResourceReconciliationWithChildren(
			ctx, instanceState, params, includeChildren, input.ChildPath,
		)
	default:
		return c.checkInterruptedResourceReconciliationWithChildren(
			ctx, instanceState, params, includeChildren, input.ChildPath,
		)
	}
}

func (c *defaultBlueprintContainer) checkInterruptedResourceReconciliation(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]ResourceReconcileResult, error) {
	driftResults, err := c.driftChecker.CheckInterruptedResourcesWithState(ctx, instanceState, params)
	if err != nil {
		return nil, err
	}

	return convertDriftResultsToResourceReconcileResults(
		driftResults,
		ReconciliationTypeInterrupted,
	), nil
}

// checkInterruptedResourceReconciliationWithChildren checks for interrupted resources
// across the instance and optionally its child blueprints.
func (c *defaultBlueprintContainer) checkInterruptedResourceReconciliationWithChildren(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	includeChildren bool,
	childPathFilter string,
) ([]ResourceReconcileResult, error) {
	if !includeChildren {
		// Only check parent instance, apply filter if set
		if childPathFilter != "" {
			// Filter is set but we're not including children - only match parent (empty path).
			// Parent won't match the filter, so we return an empty result.
			if !matchesChildPathFilter("", childPathFilter) {
				return []ResourceReconcileResult{}, nil
			}
		}
		return c.checkInterruptedResourceReconciliation(ctx, instanceState, params)
	}

	// Flatten all resources from instance hierarchy
	flattenedResources, err := flattenInstanceResources(instanceState, "", 1)
	if err != nil {
		return nil, err
	}

	results := []ResourceReconcileResult{}

	// Check each flattened resource for interrupted state
	for _, fr := range flattenedResources {
		// Apply child path filter
		if !matchesChildPathFilter(fr.ChildPath, childPathFilter) {
			continue
		}

		if !isInterruptedPreciseResourceStatus(fr.Resource.PreciseStatus) {
			continue
		}

		// Check the interrupted resource using the drift checker
		driftResults, err := c.driftChecker.CheckInterruptedResourcesWithState(
			ctx, fr.InstanceState, params,
		)
		if err != nil {
			return nil, err
		}

		// Find and add result for this specific resource
		for _, dr := range driftResults {
			if dr.ResourceID == fr.Resource.ResourceID {
				result := convertDriftResultToResourceReconcileResult(dr, ReconciliationTypeInterrupted)
				result.ChildPath = fr.ChildPath
				results = append(results, result)
				break
			}
		}
	}

	return results, nil
}

// checkAllResourceReconciliationWithChildren checks all resources for reconciliation
// across the instance and optionally its child blueprints.
func (c *defaultBlueprintContainer) checkAllResourceReconciliationWithChildren(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	includeChildren bool,
	childPathFilter string,
) ([]ResourceReconcileResult, error) {
	if !includeChildren {
		// Only check parent instance
		if childPathFilter != "" {
			// Filter is set but we're not including children - only match parent (empty path)
			if !matchesChildPathFilter("", childPathFilter) {
				return []ResourceReconcileResult{}, nil
			}
		}
		return c.checkAllResourceReconciliation(ctx, instanceState, params)
	}

	// Flatten all resources from instance hierarchy
	flattenedResources, err := flattenInstanceResources(instanceState, "", 1)
	if err != nil {
		return nil, err
	}

	results := []ResourceReconcileResult{}
	checkedInstanceIDs := make(map[string]bool)

	// Group resources by instance for efficient batch checking
	for _, fr := range flattenedResources {
		// Apply child path filter
		if !matchesChildPathFilter(fr.ChildPath, childPathFilter) {
			continue
		}

		// Check each instance only once for interrupted and drift
		if !checkedInstanceIDs[fr.InstanceState.InstanceID] {
			checkedInstanceIDs[fr.InstanceState.InstanceID] = true

			// Get interrupted results for this instance
			interruptedResults, err := c.checkInterruptedResourceReconciliation(
				ctx, fr.InstanceState, params,
			)
			if err != nil {
				return nil, err
			}

			// Set child path and add to results
			for i := range interruptedResults {
				interruptedResults[i].ChildPath = fr.ChildPath
			}
			results = append(results, interruptedResults...)

			// Get drift results for this instance
			driftResults, err := c.driftChecker.CheckDriftWithState(ctx, fr.InstanceState, params)
			if err != nil {
				return nil, err
			}

			// Build set of interrupted resource IDs to exclude from drift results
			interruptedResourceIDs := collectInterruptedResourceIDs(interruptedResults)

			// Convert drift results, excluding interrupted ones
			driftReconcileResults := c.convertDriftStatesToReconcileResults(
				driftResults, interruptedResourceIDs, fr.InstanceState.Resources,
			)

			// Set child path and add to results
			for i := range driftReconcileResults {
				driftReconcileResults[i].ChildPath = fr.ChildPath
			}
			results = append(results, driftReconcileResults...)
		}
	}

	return results, nil
}

func (c *defaultBlueprintContainer) checkSpecificResourceReconciliation(
	ctx context.Context,
	input *CheckReconciliationInput,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]ResourceReconcileResult, error) {
	if len(input.ResourceNames) == 0 {
		return []ResourceReconcileResult{}, nil
	}

	includeChildren := shouldIncludeChildren(input)

	// If a specific ChildPath is provided, navigate to that child instance first
	if input.ChildPath != "" {
		targetInstance := getInstanceStateByChildPath(instanceState, input.ChildPath)
		if targetInstance == nil {
			return []ResourceReconcileResult{}, nil
		}
		return c.checkSpecificResourcesInInstance(
			ctx, input.ResourceNames, targetInstance, params, input.ChildPath,
		)
	}

	// If not including children, just check the parent instance
	if !includeChildren {
		return c.checkSpecificResourcesInInstance(
			ctx, input.ResourceNames, instanceState, params, "",
		)
	}

	// Flatten and search across all instances
	flattenedResources, err := flattenInstanceResources(instanceState, "", 1)
	if err != nil {
		return nil, err
	}

	results := []ResourceReconcileResult{}
	resourceNamesSet := make(map[string]bool)
	for _, name := range input.ResourceNames {
		resourceNamesSet[name] = true
	}

	for _, fr := range flattenedResources {
		if !resourceNamesSet[fr.Resource.Name] {
			continue
		}

		result, err := c.checkSingleResourceReconciliation(
			ctx, fr.InstanceState, fr.Resource, params,
		)
		if err != nil {
			return nil, err
		}
		if result != nil {
			result.ChildPath = fr.ChildPath
			results = append(results, *result)
		}
	}

	return results, nil
}

// checkSpecificResourcesInInstance checks specific resources within a single instance.
func (c *defaultBlueprintContainer) checkSpecificResourcesInInstance(
	ctx context.Context,
	resourceNames []string,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	childPath string,
) ([]ResourceReconcileResult, error) {
	results := []ResourceReconcileResult{}
	for _, resourceName := range resourceNames {
		targetResource := findResourceByName(instanceState.Resources, resourceName)
		if targetResource == nil {
			continue
		}

		result, err := c.checkSingleResourceReconciliation(
			ctx, instanceState, targetResource, params,
		)
		if err != nil {
			return nil, err
		}
		if result != nil {
			result.ChildPath = childPath
			results = append(results, *result)
		}
	}

	return results, nil
}

func (c *defaultBlueprintContainer) checkSingleResourceReconciliation(
	ctx context.Context,
	instanceState *state.InstanceState,
	resource *state.ResourceState,
	params core.BlueprintParams,
) (*ResourceReconcileResult, error) {
	if isInterruptedPreciseResourceStatus(resource.PreciseStatus) {
		return c.checkInterruptedResourceByName(ctx, instanceState, resource.Name, params)
	}
	return c.checkResourceDriftReconciliation(ctx, instanceState, resource, params)
}

func (c *defaultBlueprintContainer) checkInterruptedResourceByName(
	ctx context.Context,
	instanceState *state.InstanceState,
	resourceName string,
	params core.BlueprintParams,
) (*ResourceReconcileResult, error) {
	driftResults, err := c.driftChecker.CheckInterruptedResourcesWithState(ctx, instanceState, params)
	if err != nil {
		return nil, err
	}

	for _, dr := range driftResults {
		if dr.ResourceName == resourceName {
			result := convertDriftResultToResourceReconcileResult(dr, ReconciliationTypeInterrupted)
			return &result, nil
		}
	}
	return nil, nil
}

func (c *defaultBlueprintContainer) checkResourceDriftReconciliation(
	ctx context.Context,
	instanceState *state.InstanceState,
	resource *state.ResourceState,
	params core.BlueprintParams,
) (*ResourceReconcileResult, error) {
	driftState, err := c.driftChecker.CheckResourceDrift(
		ctx, instanceState.InstanceID, instanceState.InstanceName, resource.ResourceID, params,
	)
	if err != nil {
		return nil, err
	}

	if driftState == nil {
		return nil, nil
	}

	return &ResourceReconcileResult{
		ResourceID:        resource.ResourceID,
		ResourceName:      resource.Name,
		ResourceType:      resource.Type,
		Type:              ReconciliationTypeDrift,
		OldStatus:         resource.PreciseStatus,
		NewStatus:         resource.PreciseStatus,
		ExternalState:     driftState.SpecData,
		PersistedState:    resource.SpecData,
		Changes:           convertResourceDriftChangesToProviderChanges(driftState.Difference),
		ResourceExists:    true,
		RecommendedAction: ReconciliationActionAcceptExternal,
	}, nil
}

func findResourceByName(resources map[string]*state.ResourceState, name string) *state.ResourceState {
	for _, r := range resources {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func (c *defaultBlueprintContainer) checkAllResourceReconciliation(
	ctx context.Context,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]ResourceReconcileResult, error) {
	interruptedResults, err := c.checkInterruptedResourceReconciliation(ctx, instanceState, params)
	if err != nil {
		return nil, err
	}

	driftResults, err := c.driftChecker.CheckDriftWithState(ctx, instanceState, params)
	if err != nil {
		return nil, err
	}

	interruptedResourceIDs := collectInterruptedResourceIDs(interruptedResults)
	driftReconcileResults := c.convertDriftStatesToReconcileResults(
		driftResults, interruptedResourceIDs, instanceState.Resources,
	)

	return append(interruptedResults, driftReconcileResults...), nil
}

func collectInterruptedResourceIDs(results []ResourceReconcileResult) map[string]bool {
	ids := make(map[string]bool, len(results))
	for _, r := range results {
		ids[r.ResourceID] = true
	}
	return ids
}

func (c *defaultBlueprintContainer) convertDriftStatesToReconcileResults(
	driftResults map[string]*state.ResourceDriftState,
	excludeIDs map[string]bool,
	resources map[string]*state.ResourceState,
) []ResourceReconcileResult {
	results := make([]ResourceReconcileResult, 0, len(driftResults))
	for resourceID, driftState := range driftResults {
		if excludeIDs[resourceID] {
			continue
		}
		results = append(results, createDriftReconcileResult(resourceID, driftState, resources))
	}
	return results
}

func createDriftReconcileResult(
	resourceID string,
	driftState *state.ResourceDriftState,
	resources map[string]*state.ResourceState,
) ResourceReconcileResult {
	result := ResourceReconcileResult{
		ResourceID:        resourceID,
		ResourceName:      driftState.ResourceName,
		Type:              ReconciliationTypeDrift,
		ExternalState:     driftState.SpecData,
		Changes:           convertResourceDriftChangesToProviderChanges(driftState.Difference),
		ResourceExists:    true,
		RecommendedAction: ReconciliationActionAcceptExternal,
	}

	if resource := findResourceByID(resources, resourceID); resource != nil {
		result.ResourceType = resource.Type
		result.OldStatus = resource.PreciseStatus
		result.NewStatus = resource.PreciseStatus
		result.PersistedState = resource.SpecData
	}

	return result
}

func findResourceByID(resources map[string]*state.ResourceState, id string) *state.ResourceState {
	for _, r := range resources {
		if r.ResourceID == id {
			return r
		}
	}
	return nil
}

func (c *defaultBlueprintContainer) checkLinkReconciliation(
	ctx context.Context,
	input *CheckReconciliationInput,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]LinkReconcileResult, error) {
	includeChildren := shouldIncludeChildren(input)

	if input.Scope == ReconciliationScopeInterrupted {
		return c.checkInterruptedLinkReconciliationWithChildren(
			instanceState, includeChildren, input.ChildPath,
		)
	}

	if input.Scope == ReconciliationScopeAll ||
		input.Scope == ReconciliationScopeSpecific {
		return c.checkLinkAndIntermediaryReconciliationWithChildren(
			ctx, input, instanceState, params, includeChildren,
		)
	}

	return []LinkReconcileResult{}, nil
}

func (c *defaultBlueprintContainer) checkInterruptedLinkReconciliation(
	instanceState *state.InstanceState,
) ([]LinkReconcileResult, error) {
	results := []LinkReconcileResult{}
	for _, link := range instanceState.Links {
		if !isInterruptedLinkStatus(link.PreciseStatus) {
			continue
		}

		newStatus := c.deriveLinkStatusFromResources(link, instanceState)

		result := LinkReconcileResult{
			LinkID:            link.LinkID,
			LinkName:          link.Name,
			Type:              ReconciliationTypeInterrupted,
			OldStatus:         link.PreciseStatus,
			NewStatus:         newStatus,
			RecommendedAction: determineLinkReconciliationAction(newStatus),
		}

		results = append(results, result)
	}

	return results, nil
}

// checkInterruptedLinkReconciliationWithChildren checks for interrupted links
// across the instance and optionally its child blueprints.
func (c *defaultBlueprintContainer) checkInterruptedLinkReconciliationWithChildren(
	instanceState *state.InstanceState,
	includeChildren bool,
	childPathFilter string,
) ([]LinkReconcileResult, error) {
	if !includeChildren {
		// Only check parent instance
		if childPathFilter != "" {
			if !matchesChildPathFilter("", childPathFilter) {
				return []LinkReconcileResult{}, nil
			}
		}
		return c.checkInterruptedLinkReconciliation(instanceState)
	}

	// Flatten all links from instance hierarchy
	flattenedLinks, err := flattenInstanceLinks(instanceState, "", 1)
	if err != nil {
		return nil, err
	}

	results := []LinkReconcileResult{}

	for _, fl := range flattenedLinks {
		// Apply child path filter
		if !matchesChildPathFilter(fl.ChildPath, childPathFilter) {
			continue
		}

		if !isInterruptedLinkStatus(fl.Link.PreciseStatus) {
			continue
		}

		newStatus := c.deriveLinkStatusFromResources(fl.Link, fl.InstanceState)

		result := LinkReconcileResult{
			LinkID:            fl.Link.LinkID,
			LinkName:          fl.Link.Name,
			ChildPath:         fl.ChildPath,
			Type:              ReconciliationTypeInterrupted,
			OldStatus:         fl.Link.PreciseStatus,
			NewStatus:         newStatus,
			RecommendedAction: determineLinkReconciliationAction(newStatus),
		}

		results = append(results, result)
	}

	return results, nil
}

// checkLinkAndIntermediaryReconciliationWithChildren checks links and intermediary resources
// across the instance and optionally its child blueprints.
func (c *defaultBlueprintContainer) checkLinkAndIntermediaryReconciliationWithChildren(
	ctx context.Context,
	input *CheckReconciliationInput,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
	includeChildren bool,
) ([]LinkReconcileResult, error) {
	if !includeChildren {
		// Only check parent instance
		if input.ChildPath != "" {
			if !matchesChildPathFilter("", input.ChildPath) {
				return []LinkReconcileResult{}, nil
			}
		}
		return c.checkLinkAndIntermediaryReconciliation(ctx, input, instanceState, params)
	}

	// If a specific ChildPath is provided for specific scope, navigate to that child
	if input.Scope == ReconciliationScopeSpecific && input.ChildPath != "" {
		targetInstance := getInstanceStateByChildPath(instanceState, input.ChildPath)
		if targetInstance == nil {
			return []LinkReconcileResult{}, nil
		}
		results, err := c.checkLinkAndIntermediaryReconciliation(ctx, input, targetInstance, params)
		if err != nil {
			return nil, err
		}
		// Set child path on results
		for i := range results {
			results[i].ChildPath = input.ChildPath
		}
		return results, nil
	}

	// Flatten all links from instance hierarchy
	flattenedLinks, err := flattenInstanceLinks(instanceState, "", 1)
	if err != nil {
		return nil, err
	}

	results := []LinkReconcileResult{}

	for _, fl := range flattenedLinks {
		// Apply child path filter
		if !matchesChildPathFilter(fl.ChildPath, input.ChildPath) {
			continue
		}

		// Apply link name filter for specific scope
		if !shouldCheckLink(input, fl.Link.Name) {
			continue
		}

		result, err := c.checkSingleLinkReconciliation(ctx, fl.InstanceState, fl.Link, params)
		if err != nil {
			c.logger.Debug(
				"failed to check link drift",
				core.StringLogField("linkName", fl.Link.Name),
				core.StringLogField("childPath", fl.ChildPath),
				core.ErrorLogField("error", err),
			)
			continue
		}

		if result != nil {
			result.ChildPath = fl.ChildPath
			results = append(results, *result)
		}
	}

	return results, nil
}

func (c *defaultBlueprintContainer) checkLinkAndIntermediaryReconciliation(
	ctx context.Context,
	input *CheckReconciliationInput,
	instanceState *state.InstanceState,
	params core.BlueprintParams,
) ([]LinkReconcileResult, error) {
	results := []LinkReconcileResult{}
	for linkName, link := range instanceState.Links {
		if !shouldCheckLink(input, linkName) {
			continue
		}

		result, err := c.checkSingleLinkReconciliation(ctx, instanceState, link, params)
		if err != nil {
			c.logger.Debug(
				"failed to check link drift",
				core.StringLogField("linkName", link.Name),
				core.ErrorLogField("error", err),
			)
			continue
		}

		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

func shouldCheckLink(input *CheckReconciliationInput, linkName string) bool {
	if input.Scope != ReconciliationScopeSpecific {
		return true
	}
	return slices.Contains(input.LinkNames, linkName)
}

func (c *defaultBlueprintContainer) checkSingleLinkReconciliation(
	ctx context.Context,
	instanceState *state.InstanceState,
	link *state.LinkState,
	params core.BlueprintParams,
) (*LinkReconcileResult, error) {
	if isInterruptedLinkStatus(link.PreciseStatus) {
		return c.createInterruptedLinkResult(link, instanceState), nil
	}
	return c.checkLinkDriftReconciliation(ctx, instanceState.InstanceID, link, params)
}

func (c *defaultBlueprintContainer) createInterruptedLinkResult(
	link *state.LinkState,
	instanceState *state.InstanceState,
) *LinkReconcileResult {
	newStatus := c.deriveLinkStatusFromResources(link, instanceState)
	return &LinkReconcileResult{
		LinkID:            link.LinkID,
		LinkName:          link.Name,
		Type:              ReconciliationTypeInterrupted,
		OldStatus:         link.PreciseStatus,
		NewStatus:         newStatus,
		RecommendedAction: determineLinkReconciliationAction(newStatus),
	}
}

func (c *defaultBlueprintContainer) checkLinkDriftReconciliation(
	ctx context.Context,
	instanceID string,
	link *state.LinkState,
	params core.BlueprintParams,
) (*LinkReconcileResult, error) {
	linkDriftState, err := c.driftChecker.CheckLinkDrift(ctx, instanceID, link.LinkID, params)
	if err != nil {
		return nil, err
	}

	if linkDriftState == nil {
		return nil, nil
	}

	return createLinkDriftResult(link, linkDriftState), nil
}

func createLinkDriftResult(link *state.LinkState, driftState *state.LinkDriftState) *LinkReconcileResult {
	result := &LinkReconcileResult{
		LinkID:            link.LinkID,
		LinkName:          link.Name,
		Type:              ReconciliationTypeDrift,
		OldStatus:         link.PreciseStatus,
		NewStatus:         link.PreciseStatus,
		RecommendedAction: ReconciliationActionAcceptExternal,
	}

	if driftState.ResourceADrift != nil {
		result.ResourceAChanges = convertLinkResourceDriftToProviderChanges(driftState.ResourceADrift)
	}
	if driftState.ResourceBDrift != nil {
		result.ResourceBChanges = convertLinkResourceDriftToProviderChanges(driftState.ResourceBDrift)
	}
	if len(driftState.IntermediaryDrift) > 0 {
		result.IntermediaryChanges = convertIntermediaryDriftToReconcileResult(driftState.IntermediaryDrift)
	}

	// Pre-compute LinkDataUpdates from drift state for easy application
	result.LinkDataUpdates = extractLinkDataUpdatesFromDrift(driftState)

	return result
}

// extractLinkDataUpdatesFromDrift extracts the link data updates needed to
// reconcile drift. This pre-computes the updates so callers can easily
// construct LinkReconcileAction with the correct LinkDataUpdates.
func extractLinkDataUpdatesFromDrift(driftState *state.LinkDriftState) map[string]*core.MappingNode {
	if driftState == nil {
		return nil
	}

	updates := make(map[string]*core.MappingNode)

	// Extract updates from ResourceA drift
	if driftState.ResourceADrift != nil {
		for _, change := range driftState.ResourceADrift.MappedFieldChanges {
			if change.LinkDataPath != "" && change.ExternalValue != nil {
				updates[change.LinkDataPath] = change.ExternalValue
			}
		}
	}

	// Extract updates from ResourceB drift
	if driftState.ResourceBDrift != nil {
		for _, change := range driftState.ResourceBDrift.MappedFieldChanges {
			if change.LinkDataPath != "" && change.ExternalValue != nil {
				updates[change.LinkDataPath] = change.ExternalValue
			}
		}
	}

	if len(updates) == 0 {
		return nil
	}

	return updates
}

func convertLinkResourceDriftToProviderChanges(drift *state.LinkResourceDrift) *provider.Changes {
	if drift == nil || len(drift.MappedFieldChanges) == 0 {
		return nil
	}

	modifiedFields := make([]provider.FieldChange, 0, len(drift.MappedFieldChanges))
	for _, change := range drift.MappedFieldChanges {
		modifiedFields = append(modifiedFields, provider.FieldChange{
			FieldPath: change.ResourceFieldPath,
			PrevValue: change.LinkDataValue,
			NewValue:  change.ExternalValue,
		})
	}

	return &provider.Changes{
		ModifiedFields: modifiedFields,
	}
}

func convertIntermediaryDriftToReconcileResult(
	driftStates map[string]*state.IntermediaryDriftState,
) map[string]*IntermediaryReconcileResult {
	if len(driftStates) == 0 {
		return nil
	}

	results := make(map[string]*IntermediaryReconcileResult)
	for id, driftState := range driftStates {
		results[id] = &IntermediaryReconcileResult{
			Name:           id,
			Type:           driftState.ResourceType,
			ExternalState:  driftState.ExternalState,
			PersistedState: driftState.PersistedState,
			Changes:        convertIntermediaryChangesToProviderChanges(driftState.Changes),
			Exists:         driftState.Exists,
		}
	}
	return results
}

func convertResourceDriftChangesToProviderChanges(
	changes *state.ResourceDriftChanges,
) *provider.Changes {
	if changes == nil {
		return nil
	}

	result := &provider.Changes{}

	for _, change := range changes.ModifiedFields {
		result.ModifiedFields = append(result.ModifiedFields, provider.FieldChange{
			FieldPath: change.FieldPath,
			PrevValue: change.StateValue,
			NewValue:  change.DriftedValue,
		})
	}

	for _, change := range changes.NewFields {
		result.NewFields = append(result.NewFields, provider.FieldChange{
			FieldPath: change.FieldPath,
			NewValue:  change.DriftedValue,
		})
	}

	result.RemovedFields = append(result.RemovedFields, changes.RemovedFields...)

	if len(result.ModifiedFields) == 0 && len(result.NewFields) == 0 && len(result.RemovedFields) == 0 {
		return nil
	}

	return result
}

func convertIntermediaryChangesToProviderChanges(
	changes *state.IntermediaryDriftChanges,
) *provider.Changes {
	if changes == nil {
		return nil
	}

	result := &provider.Changes{}

	for _, change := range changes.ModifiedFields {
		result.ModifiedFields = append(result.ModifiedFields, provider.FieldChange{
			FieldPath: change.FieldPath,
			PrevValue: change.PrevValue,
			NewValue:  change.NewValue,
		})
	}

	for _, change := range changes.NewFields {
		result.NewFields = append(result.NewFields, provider.FieldChange{
			FieldPath: change.FieldPath,
			NewValue:  change.NewValue,
		})
	}

	for _, change := range changes.RemovedFields {
		result.RemovedFields = append(result.RemovedFields, change.FieldPath)
	}

	if len(result.ModifiedFields) == 0 && len(result.NewFields) == 0 && len(result.RemovedFields) == 0 {
		return nil
	}

	return result
}

func (c *defaultBlueprintContainer) deriveLinkStatusFromResources(
	link *state.LinkState,
	instanceState *state.InstanceState,
) core.PreciseLinkStatus {
	resourceAName, resourceBName := parseLinkName(link.Name)
	resourceAState := findResourceByName(instanceState.Resources, resourceAName)
	resourceBState := findResourceByName(instanceState.Resources, resourceBName)

	return deriveLinkStatusFromResourceStates(link.PreciseStatus, resourceAState, resourceBState)
}

func deriveLinkStatusFromResourceStates(
	currentStatus core.PreciseLinkStatus,
	resourceA *state.ResourceState,
	resourceB *state.ResourceState,
) core.PreciseLinkStatus {
	if resourceA == nil || resourceB == nil {
		return deriveLinkFailedStatus(currentStatus)
	}

	if hasFailedResource(resourceA, resourceB) {
		return deriveLinkFailedStatus(currentStatus)
	}

	if hasInterruptedResource(resourceA, resourceB) {
		return currentStatus
	}

	return deriveLinkSuccessStatus(currentStatus)
}

func hasFailedResource(resourceA, resourceB *state.ResourceState) bool {
	return isFailedPreciseResourceStatus(resourceA.PreciseStatus) ||
		isFailedPreciseResourceStatus(resourceB.PreciseStatus)
}

func hasInterruptedResource(resourceA, resourceB *state.ResourceState) bool {
	return isInterruptedPreciseResourceStatus(resourceA.PreciseStatus) ||
		isInterruptedPreciseResourceStatus(resourceB.PreciseStatus)
}

func parseLinkName(linkName string) (string, string) {
	parts := strings.SplitN(linkName, "::", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func (c *defaultBlueprintContainer) ApplyReconciliation(
	ctx context.Context,
	input *ApplyReconciliationInput,
	paramOverrides core.BlueprintParams,
) (*ApplyReconciliationResult, error) {
	if input == nil {
		return nil, fmt.Errorf("apply reconciliation input is required")
	}

	if input.InstanceID == "" {
		return nil, fmt.Errorf("instance ID is required for reconciliation")
	}

	result := &ApplyReconciliationResult{
		InstanceID: input.InstanceID,
		Errors:     []ReconciliationError{},
	}

	for _, action := range input.ResourceActions {
		err := c.applyResourceReconciliation(ctx, action)
		if err != nil {
			elementName := c.getResourceName(ctx, action.ResourceID)
			if action.ChildPath != "" {
				elementName = fmt.Sprintf("%s.%s", action.ChildPath, elementName)
			}
			result.Errors = append(result.Errors, ReconciliationError{
				ElementID:   action.ResourceID,
				ElementName: elementName,
				ElementType: "resource",
				Error:       err.Error(),
			})
		} else {
			result.ResourcesUpdated += 1
		}
	}

	for _, action := range input.LinkActions {
		err := c.applyLinkReconciliation(ctx, action)
		if err != nil {
			elementName := c.getLinkName(ctx, action.LinkID)
			if action.ChildPath != "" {
				elementName = fmt.Sprintf("%s.%s", action.ChildPath, elementName)
			}
			result.Errors = append(result.Errors, ReconciliationError{
				ElementID:   action.LinkID,
				ElementName: elementName,
				ElementType: "link",
				Error:       err.Error(),
			})
		} else {
			result.LinksUpdated += 1
		}
	}

	return result, nil
}

func (c *defaultBlueprintContainer) applyResourceReconciliation(
	ctx context.Context,
	action ResourceReconcileAction,
) error {
	resources := c.stateContainer.Resources()
	currentTime := int(c.clock.Now().Unix())

	switch action.Action {
	case ReconciliationActionAcceptExternal:
		if action.ExternalState == nil {
			return fmt.Errorf(
				"external state is required for action %s on resource %s",
				ReconciliationActionAcceptExternal,
				action.ResourceID,
			)
		}

		currentState, err := resources.Get(ctx, action.ResourceID)
		if err != nil {
			return fmt.Errorf("failed to get current resource state: %w", err)
		}

		currentState.Status = reconcilePreciseToResourceStatus(action.NewStatus)
		currentState.PreciseStatus = action.NewStatus
		currentState.LastStatusUpdateTimestamp = currentTime
		currentState.SpecData = action.ExternalState

		// Update any link.Data that references this resource via ResourceDataMappings
		if err := c.updateAffectedLinkData(ctx, currentState, action.ExternalState); err != nil {
			return fmt.Errorf("failed to update affected link data: %w", err)
		}
		currentState.FailureReasons = nil
		currentState.Drifted = false
		currentState.LastDriftDetectedTimestamp = nil

		// Remove persisted drift state since we've accepted external state
		if _, err := resources.RemoveDrift(ctx, action.ResourceID); err != nil {
			// Log but don't fail - drift state removal is not critical.
			// User can force redeploy or skip drift check if state becomes inconsistent.
			logFields := []core.LogField{
				core.StringLogField("resourceId", action.ResourceID),
				core.ErrorLogField("error", err),
			}
			if action.ChildPath != "" {
				logFields = append(logFields, core.StringLogField("childPath", action.ChildPath))
			}
			c.logger.Warn(
				"failed to remove resource drift state after reconciliation",
				logFields...,
			)
		}

		return resources.Save(ctx, currentState)

	case ReconciliationActionUpdateStatus:
		return resources.UpdateStatus(ctx, action.ResourceID, state.ResourceStatusInfo{
			Status:                    reconcilePreciseToResourceStatus(action.NewStatus),
			PreciseStatus:             action.NewStatus,
			LastStatusUpdateTimestamp: &currentTime,
		})

	case ReconciliationActionMarkFailed:
		return resources.UpdateStatus(ctx, action.ResourceID, state.ResourceStatusInfo{
			Status:                    reconcilePreciseToResourceStatus(action.NewStatus),
			PreciseStatus:             action.NewStatus,
			LastStatusUpdateTimestamp: &currentTime,
			FailureReasons:            []string{"marked as failed during reconciliation"},
		})

	default:
		return fmt.Errorf("unknown reconciliation action: %s", action.Action)
	}
}

func (c *defaultBlueprintContainer) applyLinkReconciliation(
	ctx context.Context,
	action LinkReconcileAction,
) error {
	links := c.stateContainer.Links()
	currentTime := int(c.clock.Now().Unix())

	needsFullSave := len(action.IntermediaryActions) > 0 ||
		len(action.LinkDataUpdates) > 0 ||
		action.Action == ReconciliationActionAcceptExternal

	if !needsFullSave {
		return links.UpdateStatus(ctx, action.LinkID, state.LinkStatusInfo{
			Status:                    reconcilePreciseLinkToLinkStatus(action.NewStatus),
			PreciseStatus:             action.NewStatus,
			LastStatusUpdateTimestamp: &currentTime,
		})
	}

	linkState, err := links.Get(ctx, action.LinkID)
	if err != nil {
		return fmt.Errorf("failed to get link state: %w", err)
	}

	if len(action.LinkDataUpdates) > 0 {
		applyLinkDataUpdates(&linkState, action.LinkDataUpdates)
	}

	for intermediaryID, intermediaryAction := range action.IntermediaryActions {
		err := c.applyIntermediaryReconciliation(
			&linkState,
			intermediaryID,
			intermediaryAction,
			currentTime,
		)
		if err != nil {
			return err
		}
	}

	linkState.Status = reconcilePreciseLinkToLinkStatus(action.NewStatus)
	linkState.PreciseStatus = action.NewStatus
	linkState.LastStatusUpdateTimestamp = currentTime

	// Only clear drift state when accepting external state
	// For UpdateStatus and MarkFailed, the drift still exists
	switch action.Action {
	case ReconciliationActionAcceptExternal:
		linkState.Drifted = false
		linkState.LastDriftDetectedTimestamp = nil
		linkState.FailureReasons = nil

		// Remove persisted drift state since we've accepted external state
		if _, err := links.RemoveDrift(ctx, action.LinkID); err != nil {
			// Log but don't fail - drift state removal is not critical.
			// User can force redeploy or skip drift check if state becomes inconsistent.
			logFields := []core.LogField{
				core.StringLogField("linkId", action.LinkID),
				core.ErrorLogField("error", err),
			}
			if action.ChildPath != "" {
				logFields = append(logFields, core.StringLogField("childPath", action.ChildPath))
			}
			c.logger.Warn(
				"failed to remove link drift state after reconciliation",
				logFields...,
			)
		}
	case ReconciliationActionMarkFailed:
		// MarkFailed adds failure reasons but doesn't clear drift
		linkState.FailureReasons = []string{"marked as failed during reconciliation"}
	}

	return links.Save(ctx, linkState)
}

func applyLinkDataUpdates(linkState *state.LinkState, updates map[string]*core.MappingNode) {
	if linkState.Data == nil {
		linkState.Data = make(map[string]*core.MappingNode)
	}

	for linkDataPath, newValue := range updates {
		setValueAtLinkDataPath(linkState.Data, linkDataPath, newValue)
	}
}

func setValueAtLinkDataPath(data map[string]*core.MappingNode, path string, value *core.MappingNode) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return
	}

	if len(parts) == 1 {
		data[parts[0]] = value
		return
	}

	rootKey := parts[0]
	if data[rootKey] == nil {
		data[rootKey] = &core.MappingNode{Fields: make(map[string]*core.MappingNode)}
	}

	current := data[rootKey]
	for i := 1; i < len(parts)-1; i++ {
		if current.Fields == nil {
			current.Fields = make(map[string]*core.MappingNode)
		}
		if current.Fields[parts[i]] == nil {
			current.Fields[parts[i]] = &core.MappingNode{Fields: make(map[string]*core.MappingNode)}
		}
		current = current.Fields[parts[i]]
	}

	if current.Fields == nil {
		current.Fields = make(map[string]*core.MappingNode)
	}
	current.Fields[parts[len(parts)-1]] = value
}

func (c *defaultBlueprintContainer) applyIntermediaryReconciliation(
	linkState *state.LinkState,
	intermediaryID string,
	action *IntermediaryReconcileAction,
	currentTime int,
) error {
	idx := findIntermediaryIndex(linkState.IntermediaryResourceStates, intermediaryID)
	if idx == -1 {
		return fmt.Errorf("intermediary resource %s not found in link state", intermediaryID)
	}

	return applyIntermediaryAction(linkState.IntermediaryResourceStates[idx], action, currentTime)
}

func findIntermediaryIndex(states []*state.LinkIntermediaryResourceState, id string) int {
	for i, s := range states {
		if s.ResourceID == id {
			return i
		}
	}
	return -1
}

func applyIntermediaryAction(
	intermediary *state.LinkIntermediaryResourceState,
	action *IntermediaryReconcileAction,
	currentTime int,
) error {
	switch action.Action {
	case ReconciliationActionAcceptExternal:
		intermediary.Status = reconcilePreciseToResourceStatus(action.NewStatus)
		intermediary.PreciseStatus = action.NewStatus
		intermediary.LastDeployedTimestamp = currentTime
		if action.ExternalState != nil {
			intermediary.ResourceSpecData = action.ExternalState
		}
		intermediary.FailureReasons = nil

	case ReconciliationActionUpdateStatus:
		intermediary.Status = reconcilePreciseToResourceStatus(action.NewStatus)
		intermediary.PreciseStatus = action.NewStatus

	case ReconciliationActionMarkFailed:
		intermediary.Status = reconcilePreciseToResourceStatus(action.NewStatus)
		intermediary.PreciseStatus = action.NewStatus
		intermediary.FailureReasons = []string{"marked as failed during reconciliation"}

	default:
		return fmt.Errorf("unknown reconciliation action: %s", action.Action)
	}

	return nil
}

func convertDriftResultsToResourceReconcileResults(
	driftResults []drift.ReconcileResult,
	reconciliationType ReconciliationType,
) []ResourceReconcileResult {
	results := make([]ResourceReconcileResult, 0, len(driftResults))
	for _, dr := range driftResults {
		results = append(results, convertDriftResultToResourceReconcileResult(dr, reconciliationType))
	}
	return results
}

func convertDriftResultToResourceReconcileResult(
	dr drift.ReconcileResult,
	reconciliationType ReconciliationType,
) ResourceReconcileResult {
	return ResourceReconcileResult{
		ResourceID:        dr.ResourceID,
		ResourceName:      dr.ResourceName,
		ResourceType:      dr.ResourceType,
		Type:              reconciliationType,
		OldStatus:         dr.OldStatus,
		NewStatus:         dr.NewStatus,
		ExternalState:     dr.ExternalState,
		PersistedState:    dr.PersistedState,
		Changes:           dr.StateChanges,
		ResourceExists:    dr.ResourceExists(),
		RecommendedAction: determineResourceRecommendedAction(dr),
	}
}

func determineResourceRecommendedAction(dr drift.ReconcileResult) ReconciliationAction {
	if dr.ResourceExists() {
		return ReconciliationActionAcceptExternal
	}
	return ReconciliationActionMarkFailed
}

func determineLinkReconciliationAction(newStatus core.PreciseLinkStatus) ReconciliationAction {
	switch newStatus {
	case core.PreciseLinkStatusResourceAUpdateFailed,
		core.PreciseLinkStatusResourceBUpdateFailed,
		core.PreciseLinkStatusIntermediaryResourceUpdateFailed:
		return ReconciliationActionMarkFailed
	default:
		return ReconciliationActionUpdateStatus
	}
}

func isInterruptedPreciseResourceStatus(status core.PreciseResourceStatus) bool {
	return status == core.PreciseResourceStatusCreateInterrupted ||
		status == core.PreciseResourceStatusUpdateInterrupted ||
		status == core.PreciseResourceStatusDestroyInterrupted
}

func isFailedPreciseResourceStatus(status core.PreciseResourceStatus) bool {
	return status == core.PreciseResourceStatusCreateFailed ||
		status == core.PreciseResourceStatusUpdateFailed ||
		status == core.PreciseResourceStatusDestroyFailed
}

func isInterruptedLinkStatus(status core.PreciseLinkStatus) bool {
	return status == core.PreciseLinkStatusResourceAUpdateInterrupted ||
		status == core.PreciseLinkStatusResourceBUpdateInterrupted ||
		status == core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted
}

func deriveLinkFailedStatus(oldStatus core.PreciseLinkStatus) core.PreciseLinkStatus {
	switch oldStatus {
	case core.PreciseLinkStatusResourceAUpdateInterrupted:
		return core.PreciseLinkStatusResourceAUpdateFailed
	case core.PreciseLinkStatusResourceBUpdateInterrupted:
		return core.PreciseLinkStatusResourceBUpdateFailed
	case core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted:
		return core.PreciseLinkStatusIntermediaryResourceUpdateFailed
	default:
		return core.PreciseLinkStatusResourceAUpdateFailed // Default to resource A failed
	}
}

func deriveLinkSuccessStatus(oldStatus core.PreciseLinkStatus) core.PreciseLinkStatus {
	switch oldStatus {
	case core.PreciseLinkStatusResourceAUpdateInterrupted:
		return core.PreciseLinkStatusResourceAUpdated
	case core.PreciseLinkStatusResourceBUpdateInterrupted:
		return core.PreciseLinkStatusResourceBUpdated
	case core.PreciseLinkStatusIntermediaryResourceUpdateInterrupted:
		return core.PreciseLinkStatusIntermediaryResourcesUpdated
	default:
		return core.PreciseLinkStatusResourceAUpdated // Default
	}
}

func reconcilePreciseToResourceStatus(preciseStatus core.PreciseResourceStatus) core.ResourceStatus {
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

func reconcilePreciseLinkToLinkStatus(preciseStatus core.PreciseLinkStatus) core.LinkStatus {
	switch preciseStatus {
	case core.PreciseLinkStatusResourceAUpdated,
		core.PreciseLinkStatusResourceBUpdated,
		core.PreciseLinkStatusIntermediaryResourcesUpdated:
		return core.LinkStatusCreated // Links use Created for successful state
	case core.PreciseLinkStatusResourceAUpdateFailed,
		core.PreciseLinkStatusResourceBUpdateFailed,
		core.PreciseLinkStatusIntermediaryResourceUpdateFailed:
		return core.LinkStatusCreateFailed
	case core.PreciseLinkStatusUpdatingResourceA,
		core.PreciseLinkStatusUpdatingResourceB,
		core.PreciseLinkStatusUpdatingIntermediaryResources:
		return core.LinkStatusCreating
	default:
		return core.LinkStatusUnknown
	}
}

// updateAffectedLinkData updates link.Data for any links that have
// ResourceDataMappings pointing to the given resource. This ensures
// bidirectional consistency when resource state is updated via reconciliation.
func (c *defaultBlueprintContainer) updateAffectedLinkData(
	ctx context.Context,
	resourceState state.ResourceState,
	externalState *core.MappingNode,
) error {
	links := c.stateContainer.Links()

	// Find all links with ResourceDataMappings that reference this resource
	affectedLinks, err := links.ListWithResourceDataMappings(
		ctx,
		resourceState.InstanceID,
		resourceState.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to list links with resource data mappings: %w", err)
	}

	for _, linkState := range affectedLinks {
		linkDataUpdates := extractLinkDataUpdatesFromExternalState(
			linkState.ResourceDataMappings,
			resourceState.Name,
			externalState,
		)

		if len(linkDataUpdates) == 0 {
			continue
		}

		applyLinkDataUpdates(&linkState, linkDataUpdates)

		if err := links.Save(ctx, linkState); err != nil {
			return fmt.Errorf(
				"failed to save link %s after updating data: %w",
				linkState.Name,
				err,
			)
		}
	}

	return nil
}

// extractLinkDataUpdatesFromExternalState extracts values from the external
// resource state that should be used to update link.Data based on
// ResourceDataMappings.
func extractLinkDataUpdatesFromExternalState(
	resourceDataMappings map[string]string,
	resourceName string,
	externalState *core.MappingNode,
) map[string]*core.MappingNode {
	if len(resourceDataMappings) == 0 || externalState == nil {
		return nil
	}

	updates := make(map[string]*core.MappingNode)
	prefix := resourceName + "::"

	for resourceFieldPath, linkDataPath := range resourceDataMappings {
		// Only process mappings for this resource
		if !strings.HasPrefix(resourceFieldPath, prefix) {
			continue
		}

		// Extract the field path within the resource (e.g., "spec.policy.name")
		fieldPath := strings.TrimPrefix(resourceFieldPath, prefix)

		// Get the value from external state at this path using the core helper
		// which supports complex paths including array indices and quoted field names.
		pathWithRoot := core.AddRootToPath(fieldPath)
		externalValue, _ := core.GetPathValue(pathWithRoot, externalState, maxLinkDataUpdatePathDepth)
		if externalValue != nil {
			updates[linkDataPath] = externalValue
		}
	}

	return updates
}

// getResourceName retrieves the name of a resource by ID.
// Returns empty string if the resource cannot be found.
func (c *defaultBlueprintContainer) getResourceName(ctx context.Context, resourceID string) string {
	resource, err := c.stateContainer.Resources().Get(ctx, resourceID)
	if err != nil {
		return ""
	}
	return resource.Name
}

// getLinkName retrieves the name of a link by ID.
// Returns empty string if the link cannot be found.
func (c *defaultBlueprintContainer) getLinkName(ctx context.Context, linkID string) string {
	link, err := c.stateContainer.Links().Get(ctx, linkID)
	if err != nil {
		return ""
	}
	return link.Name
}

// ============================================================================
// Child Blueprint Path Utilities
// ============================================================================

// buildChildPath constructs a hierarchical child path by joining parent and child names.
// Uses dot notation (e.g., "childA.childB.childC").
// If parentPath is empty, returns childName directly.
func buildChildPath(parentPath, childName string) string {
	if parentPath == "" {
		return childName
	}
	return parentPath + "." + childName
}

// parseChildPath splits a hierarchical child path into its components.
// For example, "childA.childB.childC" returns ["childA", "childB", "childC"].
// Returns an empty slice for an empty path.
func parseChildPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ".")
}

// getInstanceStateByChildPath traverses the instance state hierarchy to find
// the child instance at the specified path.
// Returns the root instance state if path is empty.
// Returns nil if any part of the path doesn't exist.
func getInstanceStateByChildPath(
	rootInstance *state.InstanceState,
	childPath string,
) *state.InstanceState {
	if childPath == "" {
		return rootInstance
	}

	parts := parseChildPath(childPath)
	current := rootInstance
	for _, part := range parts {
		if current.ChildBlueprints == nil {
			return nil
		}
		child, exists := current.ChildBlueprints[part]
		if !exists || child == nil {
			return nil
		}
		current = child
	}
	return current
}

// ============================================================================
// Flattened Element Types for Child Blueprint Reconciliation
// ============================================================================

// FlattenedResource represents a resource with its location in the child hierarchy.
// Note: Child instances are stored both as separate top-level instances (accessible
// by their unique InstanceID) and nested under parent via ChildBlueprints. Resources
// and links have globally unique IDs and can be retrieved directly. The ChildPath
// here is for identification/display purposes in reconciliation results.
type FlattenedResource struct {
	// Resource is the resource state.
	Resource *state.ResourceState
	// ChildPath is the path to the child blueprint containing this resource.
	// Empty string for resources in the parent blueprint.
	ChildPath string
	// InstanceState is the instance state containing this resource.
	InstanceState *state.InstanceState
}

// FlattenedLink represents a link with its location in the child hierarchy.
type FlattenedLink struct {
	// Link is the link state.
	Link *state.LinkState
	// ChildPath is the path to the child blueprint containing this link.
	// Empty string for links in the parent blueprint.
	ChildPath string
	// InstanceState is the instance state containing this link.
	InstanceState *state.InstanceState
}

// flattenInstanceResources collects all resources from an instance and its children
// into a flat list with their child paths.
// Respects the max depth limit and returns an error if exceeded.
// The currentDepth parameter starts at 1 for the root instance.
func flattenInstanceResources(
	instanceState *state.InstanceState,
	currentPath string,
	currentDepth int,
) ([]FlattenedResource, error) {
	results := make([]FlattenedResource, 0)
	if err := flattenInstanceResourcesRecursive(instanceState, currentPath, currentDepth, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// flattenInstanceResourcesRecursive recursively collects resources from an instance
// and its children, appending them to the results slice.
func flattenInstanceResourcesRecursive(
	instanceState *state.InstanceState,
	currentPath string,
	currentDepth int,
	results *[]FlattenedResource,
) error {
	// Check max depth (currentDepth starts at 1 for the root)
	if currentDepth > MaxBlueprintDepth {
		return errMaxBlueprintDepthExceeded(currentPath, MaxBlueprintDepth)
	}

	// Add resources from current instance
	for _, resource := range instanceState.Resources {
		*results = append(*results, FlattenedResource{
			Resource:      resource,
			ChildPath:     currentPath,
			InstanceState: instanceState,
		})
	}

	// Recursively add resources from child blueprints
	if instanceState.ChildBlueprints != nil {
		for childName, childState := range instanceState.ChildBlueprints {
			if childState == nil {
				continue
			}
			childPath := buildChildPath(currentPath, childName)
			if err := flattenInstanceResourcesRecursive(childState, childPath, currentDepth+1, results); err != nil {
				return err
			}
		}
	}

	return nil
}

// flattenInstanceLinks collects all links from an instance and its children
// into a flat list with their child paths.
// Respects the max depth limit and returns an error if exceeded.
// The currentDepth parameter starts at 1 for the root instance.
func flattenInstanceLinks(
	instanceState *state.InstanceState,
	currentPath string,
	currentDepth int,
) ([]FlattenedLink, error) {
	results := make([]FlattenedLink, 0)
	if err := flattenInstanceLinksRecursive(instanceState, currentPath, currentDepth, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// flattenInstanceLinksRecursive recursively collects links from an instance
// and its children, appending them to the results slice.
func flattenInstanceLinksRecursive(
	instanceState *state.InstanceState,
	currentPath string,
	currentDepth int,
	results *[]FlattenedLink,
) error {
	// Check max depth (currentDepth starts at 1 for the root)
	if currentDepth > MaxBlueprintDepth {
		return errMaxBlueprintDepthExceeded(currentPath, MaxBlueprintDepth)
	}

	// Add links from current instance
	for _, link := range instanceState.Links {
		*results = append(*results, FlattenedLink{
			Link:          link,
			ChildPath:     currentPath,
			InstanceState: instanceState,
		})
	}

	// Recursively add links from child blueprints
	if instanceState.ChildBlueprints != nil {
		for childName, childState := range instanceState.ChildBlueprints {
			if childState == nil {
				continue
			}
			childPath := buildChildPath(currentPath, childName)
			if err := flattenInstanceLinksRecursive(childState, childPath, currentDepth+1, results); err != nil {
				return err
			}
		}
	}

	return nil
}

// shouldIncludeChildren returns whether child blueprints should be included
// based on the input. Defaults to true if IncludeChildren is nil.
func shouldIncludeChildren(input *CheckReconciliationInput) bool {
	if input.IncludeChildren == nil {
		return true
	}
	return *input.IncludeChildren
}

// matchesChildPathFilter returns whether the element's child path matches the filter.
// If the filter is empty, all paths match.
// If the filter is set, only elements at that exact path or within that subtree match.
func matchesChildPathFilter(elementPath, filterPath string) bool {
	if filterPath == "" {
		return true
	}
	// Exact match
	if elementPath == filterPath {
		return true
	}
	// Element is in a child of the filter path
	if strings.HasPrefix(elementPath, filterPath+".") {
		return true
	}
	return false
}
