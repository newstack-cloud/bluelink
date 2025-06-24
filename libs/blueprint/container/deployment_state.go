package container

import (
	"sync"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// DeploymentState provides functionality for tracking and setting the state of a deployment.
// In most cases, this is to be treated as ephemeral state that lasts for the duration
// of a deployment operation.
// This is not to be confused with the state of blueprint instances, which is persisted
// with implementations of the state.Container interface.
type DeploymentState interface {
	// SetDestroyedElement marks an element in the current
	// deployment as destroyed.
	SetDestroyedElement(element state.Element)
	// SetUpdatedEleemnt marks an element in the current
	// deployment as updated.
	SetUpdatedElement(element state.Element)
	// SetCreatedElement marks an element in the current
	// deployment as created.
	SetCreatedElement(element state.Element)
	// SetElementInProgress marks an element in the current
	// deployment as in progress.
	SetElementInProgress(element state.Element)
	// IsElementInProgress checks if an element is currently in progress
	// in the deployment process.
	IsElementInProgress(element state.Element) bool
	// SetElementConfigComplete marks an element in the current
	// deployment as having its configuration completed.
	SetElementConfigComplete(element state.Element)
	// IsElementConfigComplete checks if an element has had its configuration
	// completed in the deployment process.
	IsElementConfigComplete(element state.Element) bool
	// CheckUpdateElementDeploymentStarted checks if the deployment of an element has already started
	// and updates the deployment state at the same time.
	// This makes checking and writing the deployment started state atomic while taking into account
	// other factors that should be considered when checking if the deployment should be started.
	// This is primarily useful for avoiding starting the deployment of an element
	// multiple times.
	CheckUpdateElementDeploymentStarted(element state.Element, otherConditionToStart bool) bool
	// SetLinkDurationInfo stores the duration information for multiple stages
	// of the deployment of a link.
	SetLinkDurationInfo(linkName string, durations *state.LinkCompletionDurations)
	// GetLinkDurationInfo returns the duration information for multiple stages
	// of the deployment of a link.
	GetLinkDurationInfo(linkName string) *state.LinkCompletionDurations
	// SetPrepareDuration sets the duration of the preparation phase for the deployment
	// of a blueprint instance.
	SetPrepareDuration(prepareDuration time.Duration)
	// SetResourceDurationInfo sets the duration information for the "config completion"
	// stage of the deployment of a resource.
	SetResourceDurationInfo(resourceName string, durations *state.ResourceCompletionDurations)
	// GetResourceDurationInfo returns the duration information for the "config completion"
	// stage of the deployment of a resource.
	GetResourceDurationInfo(resourceName string) *state.ResourceCompletionDurations
	// GetPrepareDuration returns the duration of the preparation phase for the deployment
	// of a blueprint instance.
	GetPrepareDuration() *time.Duration
	// SetResourceData sets the spec state and metadata for a resource that has been created
	// or updated.
	SetResourceData(resourceName string, specState *CollectedResourceData)
	// GetResourceData returns the spec state and metadata for a resource that has been created
	// or updated.
	GetResourceData(resourceName string) *CollectedResourceData
	// UpdateLinkDeploymentState updates the state of links that are pending completion
	// and returns a list of links that are ready to be deployed or updated.
	UpdateLinkDeploymentState(node *links.ChainLinkNode) []*LinkPendingCompletion
	// SetLinkDeployResult sets the result of the deployment of a link
	// to be used for persisting.
	// This is primarily meant to store the result
	// in ephemeral state straight after a link has been deployed to be
	// persisted later.
	SetLinkDeployResult(linkName string, result *LinkDeployResult)
	// GetLinkDeployResult returns the result of the deployment of a link.
	GetLinkDeployResult(linkName string) *LinkDeployResult
	// SetElementDependencies sets the dependencies of a resource or child blueprint
	// element in the deployment state.
	SetElementDependencies(element state.Element, dependencies *state.DependencyInfo)
	// GetElementDependencies returns the dependencies of a resource or child blueprint
	// element in the deployment state.
	GetElementDependencies(element state.Element) *state.DependencyInfo
}

// CollectedResourceData holds the spec state and metadata for a resource that is being deployed,
// this structure is primarily used to temporarily store the result of resolving substitutions
// and deploying the resource to be persisted shortly after.
type CollectedResourceData struct {
	Spec         *core.MappingNode
	Metadata     *state.ResourceMetadataState
	TemplateName string
	Description  string
}

// NewDefaultDeploymentState creates a new instance of the default
// implementation for tracking and setting the state of a deployment.
// The default implementation is a thread-safe, ephemeral store for deployment state.
func NewDefaultDeploymentState() DeploymentState {
	return &defaultDeploymentState{
		pendingLinks:               make(map[string]*LinkPendingCompletion),
		resourceNamePendingLinkMap: make(map[string][]string),
		inProgress:                 make(map[string]state.Element),
		configComplete:             make(map[string]state.Element),
		destroyed:                  make(map[string]state.Element),
		created:                    make(map[string]state.Element),
		deploymentStarted:          make(map[string]state.Element),
		resourceData:               make(map[string]*CollectedResourceData),
		linkDeploymentResults:      make(map[string]*LinkDeployResult),
		updated:                    make(map[string]state.Element),
		linkDurationInfo:           make(map[string]*state.LinkCompletionDurations),
		resourceDurationInfo:       make(map[string]*state.ResourceCompletionDurations),
		elementDependencies:        make(map[string]*state.DependencyInfo),
	}
}

// Keeps track of state regarding when links are ready to be processed
// along with elements that have been successfully processed.
// All instance state including statuses of resources, links and child blueprints
// are stored in the state container.
// This is a temporary representation of the state of the deployment
// that is not persisted.
type defaultDeploymentState struct {
	// A mapping of a logical link name to the pending link completion state for links
	// that need to be deployed or updated.
	// A link ID in this context is made up of the resource names of the two resources
	// that are linked together.
	// For example, if resource A is linked to resource B, the link name would be "A::B".
	// This is used to keep track of when links are ready to be deployed or updated.
	// This does not hold the state of the link, only information about when the link is ready
	// to be deployed or updated.
	// Link removals are not tracked here as they do not depend on resource state changes,
	// removal of links happens before resources in the link relationship are processed.
	pendingLinks map[string]*LinkPendingCompletion
	// A mapping of resource names to pending links that include the resource.
	resourceNamePendingLinkMap map[string][]string
	// Elements that have been successfully destroyed.
	// This is a mapping of namespaced logical names (e.g. resources.resourceA) to an element
	// representing identifiers and the kind of the element.
	destroyed map[string]state.Element
	// Elements that have been successfully created/deployed.
	// This is a mapping of namespaced logical names (e.g. resources.resourceA) to an element
	// representing identifiers and the kind of the element.
	created map[string]state.Element
	// Elements that have had their configuration completed.
	// This is a mapping of namespaced logical names (e.g. resources.resourceA) to an element
	// representing identifiers and the kind of the element.
	// This should only be used for resources as child blueprints and links do not have
	// a "config complete" stage.
	configComplete map[string]state.Element
	// Elements that are currently in progress.
	// This is a mapping of namespaced logical names (e.g. resources.resourceA) to an element
	// representing identifiers and the kind of the element.
	inProgress map[string]state.Element
	// Elements for which the deployment process has started.
	// Elements should be added to this map when a deployment operation has started
	// for the element.
	deploymentStarted map[string]state.Element
	// Holds the resource spec and metadata for resources that have been deployed.
	resourceData map[string]*CollectedResourceData
	// A mapping of logical link names to the result of the deployment of links.
	linkDeploymentResults map[string]*LinkDeployResult
	// Elements that have been successfully updated.
	// This is a mapping of namespaced logical names (e.g. resources.resourceA) to an element
	// representing identifiers and the kind of the element.
	updated map[string]state.Element
	// The duration of the preparation phase for the deployment of a blueprint instance.
	prepareDuration *time.Duration
	// A mapping of logical link names to the current duration information for the progress
	// of the link deployment.
	linkDurationInfo map[string]*state.LinkCompletionDurations
	// A mapping of logical resource names to the current duration information for a resource
	// that has reached the "config completion" stage of deployment.
	resourceDurationInfo map[string]*state.ResourceCompletionDurations
	// A mapping of namespaced logical names (e.g. resources.resourceA) to the dependencies
	// of the element.
	elementDependencies map[string]*state.DependencyInfo
	// Mutex is required as resources can be deployed concurrently.
	mu sync.Mutex
}

func (d *defaultDeploymentState) SetDestroyedElement(element state.Element) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.inProgress, getNamespacedLogicalName(element))
	d.destroyed[getNamespacedLogicalName(element)] = element
}

func (d *defaultDeploymentState) SetUpdatedElement(element state.Element) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Elements can be child blueprints, resources or links so we need to make sure
	// we remove the item from both the inProgress and configComplete maps as "config complete"
	// does not apply to child blueprints and links.
	delete(d.inProgress, getNamespacedLogicalName(element))
	delete(d.configComplete, getNamespacedLogicalName(element))
	d.updated[getNamespacedLogicalName(element)] = element
}

func (d *defaultDeploymentState) SetCreatedElement(element state.Element) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Elements can be child blueprints, resources or links so we need to make sure
	// we remove the item from both the inProgress and configComplete maps as "config complete"
	// does not apply to child blueprints and links.
	delete(d.inProgress, getNamespacedLogicalName(element))
	delete(d.configComplete, getNamespacedLogicalName(element))
	d.created[getNamespacedLogicalName(element)] = element
}

func (d *defaultDeploymentState) SetElementInProgress(element state.Element) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.inProgress[getNamespacedLogicalName(element)] = element
	// To create a clean interface for setting elements in progress,
	// also set the deployment started state when setting in progress state.
	// The difference between the two states is that deployment started state
	// will only be set once and remain for the duration of the deployment
	// while an element is no longer considered in progress once the element
	// has finished deploying.
	//
	// This may lead to duplicate writes of the deployment started state
	// as for nodes that depend on others, it will also be set as a part of
	// the atomic CheckUpdateElementDeploymentStarted operation.
	// This is acceptable as the cost of duplicate writes to a hash map is low
	// and allows for a cleaner interface without callers having to worry about
	// calling numerous methods to update in progress and deployment started states
	// for an element.
	d.deploymentStarted[getNamespacedLogicalName(element)] = element
}

func (d *defaultDeploymentState) IsElementInProgress(element state.Element) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, inProgress := d.inProgress[getNamespacedLogicalName(element)]
	return inProgress
}

func (d *defaultDeploymentState) SetElementConfigComplete(element state.Element) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.inProgress, getNamespacedLogicalName(element))
	d.configComplete[getNamespacedLogicalName(element)] = element
}

func (d *defaultDeploymentState) IsElementConfigComplete(element state.Element) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, isConfigComplete := d.configComplete[getNamespacedLogicalName(element)]
	return isConfigComplete
}

func (d *defaultDeploymentState) CheckUpdateElementDeploymentStarted(
	element state.Element,
	otherConditionToStart bool,
) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, hasDeploymentStarted := d.deploymentStarted[getNamespacedLogicalName(element)]

	if !hasDeploymentStarted && otherConditionToStart {
		d.deploymentStarted[getNamespacedLogicalName(element)] = element
		// We need to report the state before the update
		// as this is an atomic operation to check if an element is already
		// deploying and update the state at the same time to avoid
		// starting the deployment of an element multiple times.
		return false
	}

	return hasDeploymentStarted
}

func (d *defaultDeploymentState) SetLinkDurationInfo(linkName string, durations *state.LinkCompletionDurations) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.linkDurationInfo[linkName] = copyLinkCompletionDurations(durations)
}

func (d *defaultDeploymentState) GetLinkDurationInfo(linkName string) *state.LinkCompletionDurations {
	d.mu.Lock()
	defer d.mu.Unlock()

	durationInfo, hasDurationInfo := d.linkDurationInfo[linkName]
	if !hasDurationInfo {
		return &state.LinkCompletionDurations{}
	}

	// Make a copy of durations so any modifications made to the returned value
	// does not affect the value in the deployment state.
	return copyLinkCompletionDurations(durationInfo)
}

func (d *defaultDeploymentState) SetResourceDurationInfo(resourceName string, durations *state.ResourceCompletionDurations) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.resourceDurationInfo[resourceName] = copyResourceCompletionDurations(durations)
}

func (d *defaultDeploymentState) GetResourceDurationInfo(resourceName string) *state.ResourceCompletionDurations {
	d.mu.Lock()
	defer d.mu.Unlock()

	durationInfo, hasDurationInfo := d.resourceDurationInfo[resourceName]
	if !hasDurationInfo {
		return &state.ResourceCompletionDurations{}
	}

	// Make a copy of durations so any modifications made to the returned value
	// does not affect the value in the deployment state.
	return copyResourceCompletionDurations(durationInfo)
}

func (d *defaultDeploymentState) SetPrepareDuration(prepareDuration time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.prepareDuration = &prepareDuration
}

func (d *defaultDeploymentState) GetPrepareDuration() *time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.prepareDuration
}

func (d *defaultDeploymentState) SetResourceData(resourceName string, data *CollectedResourceData) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.resourceData[resourceName] = data
}

func (d *defaultDeploymentState) GetResourceData(resourceName string) *CollectedResourceData {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, hasData := d.resourceData[resourceName]
	if !hasData {
		return nil
	}

	// Copy the resource data so that modifications to the returned value
	// do not affect the value in the deployment state.
	return &CollectedResourceData{
		Spec:         core.CopyMappingNode(data.Spec),
		Metadata:     copyResourceMetadataState(data.Metadata),
		TemplateName: data.TemplateName,
	}
}

func (d *defaultDeploymentState) SetLinkDeployResult(linkName string, result *LinkDeployResult) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.linkDeploymentResults[linkName] = result
}

func (d *defaultDeploymentState) GetLinkDeployResult(linkName string) *LinkDeployResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, hasResult := d.linkDeploymentResults[linkName]
	if !hasResult {
		return nil
	}

	// Copy the result so that modifications to the returned value
	// do not affect the value in the deployment state.
	return copyLinkDeployResult(result)
}

func (d *defaultDeploymentState) UpdateLinkDeploymentState(
	node *links.ChainLinkNode,
) []*LinkPendingCompletion {
	d.mu.Lock()
	defer d.mu.Unlock()

	hasLinks := len(node.LinksTo) > 0 || len(node.LinkedFrom) > 0
	pendingLinkNames := d.resourceNamePendingLinkMap[node.ResourceName]
	if hasLinks {
		addPendingLinksToEphemeralState(
			node,
			pendingLinkNames,
			d.pendingLinks,
			d.resourceNamePendingLinkMap,
		)
	}
	return updatePendingLinksInEphemeralState(
		node,
		pendingLinkNames,
		d.pendingLinks,
	)
}

func (d *defaultDeploymentState) SetElementDependencies(element state.Element, dependencies *state.DependencyInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.elementDependencies[getNamespacedLogicalName(element)] = dependencies
}

func (d *defaultDeploymentState) GetElementDependencies(element state.Element) *state.DependencyInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	dependencies, hasDependencies := d.elementDependencies[getNamespacedLogicalName(element)]
	if !hasDependencies {
		return nil
	}

	// Copy the dependencies so that modifications to the returned value
	// do not affect the value in the deployment state.
	resourceDepsCopy := append([]string{}, dependencies.DependsOnResources...)
	childDepsCopy := append([]string{}, dependencies.DependsOnChildren...)

	return &state.DependencyInfo{
		DependsOnResources: resourceDepsCopy,
		DependsOnChildren:  childDepsCopy,
	}
}

func copyLinkDeployResult(result *LinkDeployResult) *LinkDeployResult {
	return &LinkDeployResult{
		LinkData:                   core.CopyMappingNode(result.LinkData),
		IntermediaryResourceStates: copyLinkIntermediaryResourceStates(result.IntermediaryResourceStates),
	}
}

func copyLinkIntermediaryResourceStates(
	states []*state.LinkIntermediaryResourceState,
) []*state.LinkIntermediaryResourceState {
	if states == nil {
		return []*state.LinkIntermediaryResourceState{}
	}

	statesCopy := make([]*state.LinkIntermediaryResourceState, len(states))
	for i, state := range states {
		statesCopy[i] = copyLinkIntermediaryResourceState(state)
	}

	return statesCopy
}

func copyLinkIntermediaryResourceState(
	linkResourceState *state.LinkIntermediaryResourceState,
) *state.LinkIntermediaryResourceState {
	if linkResourceState == nil {
		return nil
	}

	return &state.LinkIntermediaryResourceState{
		ResourceID:                 linkResourceState.ResourceID,
		InstanceID:                 linkResourceState.InstanceID,
		LastDeployedTimestamp:      linkResourceState.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: linkResourceState.LastDeployAttemptTimestamp,
		ResourceSpecData:           core.CopyMappingNode(linkResourceState.ResourceSpecData),
	}
}

func copyLinkCompletionDurations(durations *state.LinkCompletionDurations) *state.LinkCompletionDurations {
	if durations == nil {
		return &state.LinkCompletionDurations{}
	}

	return &state.LinkCompletionDurations{
		ResourceAUpdate:       copyLinkComponentCompletionDurations(durations.ResourceAUpdate),
		ResourceBUpdate:       copyLinkComponentCompletionDurations(durations.ResourceBUpdate),
		IntermediaryResources: copyLinkComponentCompletionDurations(durations.IntermediaryResources),
	}
}

func copyLinkComponentCompletionDurations(durations *state.LinkComponentCompletionDurations) *state.LinkComponentCompletionDurations {
	if durations == nil {
		return nil
	}

	totalDurationCopy := copyFloatPtr(durations.TotalDuration)

	return &state.LinkComponentCompletionDurations{
		TotalDuration:    totalDurationCopy,
		AttemptDurations: append([]float64{}, durations.AttemptDurations...),
	}
}

func copyResourceCompletionDurations(durations *state.ResourceCompletionDurations) *state.ResourceCompletionDurations {
	if durations == nil {
		return nil
	}

	totalDurationCopy := copyFloatPtr(durations.TotalDuration)
	configCompleteDurationCopy := copyFloatPtr(durations.ConfigCompleteDuration)

	return &state.ResourceCompletionDurations{
		ConfigCompleteDuration: configCompleteDurationCopy,
		TotalDuration:          totalDurationCopy,
		AttemptDurations:       append([]float64{}, durations.AttemptDurations...),
	}
}

func copyResourceMetadataState(metadata *state.ResourceMetadataState) *state.ResourceMetadataState {
	if metadata == nil {
		return nil
	}

	annotationsCopy := map[string]*core.MappingNode{}
	for key, value := range metadata.Annotations {
		annotationsCopy[key] = core.CopyMappingNode(value)
	}

	labelsCopy := map[string]string{}
	for key, value := range metadata.Labels {
		labelsCopy[key] = value
	}

	return &state.ResourceMetadataState{
		DisplayName: metadata.DisplayName,
		Annotations: annotationsCopy,
		Labels:      labelsCopy,
		Custom:      core.CopyMappingNode(metadata.Custom),
	}
}

func copyFloatPtr(value *float64) *float64 {
	if value == nil {
		return nil
	}

	valueCopy := *value
	return &valueCopy
}
