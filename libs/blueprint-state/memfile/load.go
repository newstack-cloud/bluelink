package memfile

import (
	"encoding/json"
	"os"
	"path"
	"regexp"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

type internalState struct {
	instances                map[string]*state.InstanceState
	resources                map[string]*state.ResourceState
	resourceDrift            map[string]*state.ResourceDriftState
	links                    map[string]*state.LinkState
	events                   map[string]*manage.Event
	partitionEvents          map[string][]*manage.Event
	changesets               map[string]*manage.Changeset
	blueprintValidations     map[string]*manage.BlueprintValidation
	instanceIndex            map[string]*indexLocation
	resourceDriftIndex       map[string]*indexLocation
	eventIndex               map[string]*eventIndexLocation
	changesetIndex           map[string]*indexLocation
	blueprintValidationIndex map[string]*indexLocation
}

type indexLocation struct {
	ChunkNumber  int `json:"chunkNumber"`
	IndexInChunk int `json:"indexInChunk"`
}

type eventIndexLocation struct {
	Partition        string `json:"partition"`
	IndexInPartition int    `json:"indexInPartition"`
}

// Provides a slightly different structure than state.InstanceState to persist only the relationships
// between parent and child blueprints instead of embedding the entire child blueprint instance state.
type persistedInstanceState struct {
	InstanceID                 string                          `json:"id"`
	InstanceName               string                          `json:"name"`
	Status                     core.InstanceStatus             `json:"status"`
	LastStatusUpdateTimestamp  int                             `json:"lastStatusUpdateTimestamp,omitempty"`
	LastDeployedTimestamp      int                             `json:"lastDeployedTimestamp"`
	LastDeployAttemptTimestamp int                             `json:"lastDeployAttemptTimestamp"`
	ResourceIDs                map[string]string               `json:"resourceIds"`
	Resources                  map[string]*state.ResourceState `json:"resources"`
	Links                      map[string]*state.LinkState     `json:"links"`
	Metadata                   map[string]*core.MappingNode    `json:"metadata"`
	Exports                    map[string]*state.ExportState   `json:"exports"`
	// A mapping of child blueprint names to their blueprint instance IDs.
	ChildBlueprints   map[string]string                 `json:"childBlueprints"`
	ChildDependencies map[string]*state.DependencyInfo  `json:"childDependencies,omitempty"`
	Durations         *state.InstanceCompletionDuration `json:"durations,omitempty"`
}

type childInstanceInfo struct {
	childName       string
	childInstanceID string
}

func loadStateFromDir(stateDir string, fs afero.Fs) (*internalState, error) {
	currentState := &internalState{
		instances:                map[string]*state.InstanceState{},
		resources:                map[string]*state.ResourceState{},
		resourceDrift:            map[string]*state.ResourceDriftState{},
		links:                    map[string]*state.LinkState{},
		events:                   map[string]*manage.Event{},
		partitionEvents:          map[string][]*manage.Event{},
		changesets:               map[string]*manage.Changeset{},
		blueprintValidations:     map[string]*manage.BlueprintValidation{},
		resourceDriftIndex:       map[string]*indexLocation{},
		instanceIndex:            map[string]*indexLocation{},
		eventIndex:               map[string]*eventIndexLocation{},
		changesetIndex:           map[string]*indexLocation{},
		blueprintValidationIndex: map[string]*indexLocation{},
	}

	parentChildMapping := map[string][]*childInstanceInfo{}

	entries, err := afero.ReadDir(fs, stateDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		err := loadStateFromFileEntry(fs, stateDir, entry, currentState, parentChildMapping)
		if err != nil {
			return nil, err
		}
	}

	for parentInstanceID, childInstanceInfo := range parentChildMapping {
		parentInstance := currentState.instances[parentInstanceID]
		for _, childInstanceInfo := range childInstanceInfo {
			childInstance := currentState.instances[childInstanceInfo.childInstanceID]
			parentInstance.ChildBlueprints[childInstanceInfo.childName] = childInstance
		}
	}

	return currentState, nil
}

func loadStateFromFileEntry(
	fs afero.Fs,
	stateDir string,
	entry os.FileInfo,
	targetState *internalState,
	parentChildMapping map[string][]*childInstanceInfo,
) error {
	if entry.IsDir() {
		return nil
	}

	entryName := entry.Name()
	if isInstanceFile(entryName) {
		return loadInstanceStateFromFile(
			fs,
			stateDir,
			entryName,
			targetState,
			parentChildMapping,
		)
	}

	if isResourceDriftFile(entryName) {
		return loadResourceDriftFromFile(fs, stateDir, entryName, targetState)
	}

	if isEventPartitionFile(entryName) {
		return loadEventPartitionFromFile(fs, stateDir, entryName, targetState)
	}

	if isChangesetFile(entryName) {
		return loadChangesetsFromFile(fs, stateDir, entryName, targetState)
	}

	if isBlueprintValidationFile(entryName) {
		return loadBlueprintValidationFromFile(fs, stateDir, entryName, targetState)
	}

	if isInstanceIndexFile(entryName) {
		return loadInstanceIndexFromFile(fs, stateDir, entryName, targetState)
	}

	if isResourceDriftIndexFile(entryName) {
		return loadResourceDriftIndexFromFile(fs, stateDir, entryName, targetState)
	}

	if isEventIndexFile(entryName) {
		return loadEventIndexFromFile(fs, stateDir, entryName, targetState)
	}

	if isChangesetIndexFile(entryName) {
		return loadChangesetIndexFromFile(fs, stateDir, entryName, targetState)
	}

	if isBlueprintValidationIndexFile(entryName) {
		return loadBlueprintValidationIndexFromFile(fs, stateDir, entryName, targetState)
	}

	return nil
}

func loadInstanceStateFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
	parentChildMapping map[string][]*childInstanceInfo,
) error {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return err
	}

	persistedInstances := []*persistedInstanceState{}
	err = json.Unmarshal(data, &persistedInstances)
	if err != nil {
		return err
	}

	for _, persistedInstance := range persistedInstances {
		targetState.instances[persistedInstance.InstanceID] = persistedToInstanceStateWithoutChildren(
			persistedInstance,
		)

		for _, resource := range persistedInstance.Resources {
			targetState.resources[resource.ResourceID] = resource
		}

		for _, link := range persistedInstance.Links {
			targetState.links[link.LinkID] = link
		}

		parentChildMapping[persistedInstance.InstanceID] = getChildBlueprintValues(
			persistedInstance.ChildBlueprints,
		)
	}

	return nil
}

func persistedToInstanceStateWithoutChildren(
	persistedInstance *persistedInstanceState,
) *state.InstanceState {
	return &state.InstanceState{
		InstanceID:                 persistedInstance.InstanceID,
		InstanceName:               persistedInstance.InstanceName,
		Status:                     persistedInstance.Status,
		LastStatusUpdateTimestamp:  persistedInstance.LastStatusUpdateTimestamp,
		LastDeployedTimestamp:      persistedInstance.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: persistedInstance.LastDeployAttemptTimestamp,
		ResourceIDs:                persistedInstance.ResourceIDs,
		Resources:                  persistedInstance.Resources,
		Links:                      persistedInstance.Links,
		Metadata:                   persistedInstance.Metadata,
		Exports:                    persistedInstance.Exports,
		ChildDependencies:          persistedInstance.ChildDependencies,
		ChildBlueprints:            map[string]*state.InstanceState{},
		Durations:                  persistedInstance.Durations,
	}
}

func loadResourceDriftFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return err
	}

	resourceDriftEntries := []*state.ResourceDriftState{}
	err = json.Unmarshal(data, &resourceDriftEntries)
	if err != nil {
		return err
	}

	for _, resourceDrift := range resourceDriftEntries {
		targetState.resourceDrift[resourceDrift.ResourceID] = resourceDrift
	}

	return nil
}

func loadEventPartitionFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return err
	}

	partitionEvents := []*manage.Event{}
	err = json.Unmarshal(data, &partitionEvents)
	if err != nil {
		return err
	}

	for _, event := range partitionEvents {
		targetState.events[event.ID] = event
	}

	partitionName := extractPartitionName(name)
	targetState.partitionEvents[partitionName] = partitionEvents

	return nil
}

func loadChangesetsFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return err
	}

	changesets := []*manage.Changeset{}
	err = json.Unmarshal(data, &changesets)
	if err != nil {
		return err
	}

	for _, changeset := range changesets {
		targetState.changesets[changeset.ID] = changeset
	}

	return nil
}

func loadBlueprintValidationFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return err
	}

	blueprintValidations := []*manage.BlueprintValidation{}
	err = json.Unmarshal(data, &blueprintValidations)
	if err != nil {
		return err
	}

	for _, validation := range blueprintValidations {
		targetState.blueprintValidations[validation.ID] = validation
	}

	return nil
}

func loadInstanceIndexFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	instanceIndex, err := loadChunkIndexFromFile(fs, stateDir, name)
	if err != nil {
		return err
	}

	targetState.instanceIndex = instanceIndex

	return nil
}

func loadResourceDriftIndexFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	resourceDriftIndex, err := loadChunkIndexFromFile(fs, stateDir, name)
	if err != nil {
		return err
	}

	targetState.resourceDriftIndex = resourceDriftIndex

	return nil
}

func loadChangesetIndexFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	changesetIndex, err := loadChunkIndexFromFile(fs, stateDir, name)
	if err != nil {
		return err
	}

	targetState.changesetIndex = changesetIndex

	return nil
}

func loadBlueprintValidationIndexFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	blueprintValidationIndex, err := loadChunkIndexFromFile(fs, stateDir, name)
	if err != nil {
		return err
	}

	targetState.blueprintValidationIndex = blueprintValidationIndex

	return nil
}

func loadChunkIndexFromFile(
	fs afero.Fs,
	stateDir, name string,
) (map[string]*indexLocation, error) {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, err
	}

	indexEntries := map[string]*indexLocation{}
	err = json.Unmarshal(data, &indexEntries)
	if err != nil {
		return nil, err
	}

	return indexEntries, nil
}

func loadEventIndexFromFile(
	fs afero.Fs,
	stateDir, name string,
	targetState *internalState,
) error {
	filePath := path.Join(stateDir, name)
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return err
	}

	eventIndex := map[string]*eventIndexLocation{}
	err = json.Unmarshal(data, &eventIndex)
	if err != nil {
		return err
	}

	targetState.eventIndex = eventIndex

	return nil
}

var (
	instancesFilePattern            = regexp.MustCompile(`^instances_c(\d+)\.json$`)
	resourceDriftFilePattern        = regexp.MustCompile(`^resource_drift_c(\d+)\.json$`)
	eventPartitionFilePattern       = regexp.MustCompile(`^events__(.*?)\.json$`)
	changesetsFilePattern           = regexp.MustCompile(`^changesets_c(\d+)\.json$`)
	blueprintValidationsFilePattern = regexp.MustCompile(`^blueprint_validations_c(\d+)\.json$`)
)

func isInstanceFile(name string) bool {
	return instancesFilePattern.Match([]byte(name))
}

func isEventPartitionFile(name string) bool {
	return eventPartitionFilePattern.Match([]byte(name))
}

func isChangesetFile(name string) bool {
	return changesetsFilePattern.Match([]byte(name))
}

func isBlueprintValidationFile(name string) bool {
	return blueprintValidationsFilePattern.Match([]byte(name))
}

func isInstanceIndexFile(name string) bool {
	return name == "instance_index.json"
}

func isResourceDriftFile(name string) bool {
	return resourceDriftFilePattern.Match([]byte(name))
}

func isResourceDriftIndexFile(name string) bool {
	return name == "resource_drift_index.json"
}

func isEventIndexFile(name string) bool {
	return name == "event_index.json"
}

func isChangesetIndexFile(name string) bool {
	return name == "changeset_index.json"
}

func isBlueprintValidationIndexFile(name string) bool {
	return name == "blueprint_validation_index.json"
}

func getChildBlueprintValues(childBlueprintRefs map[string]string) []*childInstanceInfo {
	childInstanceInfos := []*childInstanceInfo{}
	for childBlueprintName, childInstanceID := range childBlueprintRefs {
		childInstanceInfos = append(
			childInstanceInfos,
			&childInstanceInfo{
				childName:       childBlueprintName,
				childInstanceID: childInstanceID,
			},
		)
	}
	return childInstanceInfos
}

func extractPartitionName(fileName string) string {
	matches := eventPartitionFilePattern.FindStringSubmatch(fileName)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
