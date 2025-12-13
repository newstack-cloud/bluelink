package memfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

const (
	malformedInstanceStateFileMessage = "instance state file is malformed"
)

func (s *statePersister) createInstance(instance *state.InstanceState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastChunkFilePath := instanceChunkFilePath(s.stateDir, s.lastInstanceChunk)
	chunkFileInfo, err := s.getFileSizeInfo(lastChunkFilePath)
	if err != nil {
		return err
	}

	chunkFilePath, err := s.prepareChunkFile(
		chunkFileInfo,
		s.lastInstanceChunk,
		lastChunkFilePath,
		instanceChunkFilePath,
		func(incrementBy int) {
			s.lastInstanceChunk += incrementBy
		},
	)
	if err != nil {
		return err
	}

	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		if !errors.Is(err, afero.ErrFileNotFound) {
			return err
		}
		existingData = []byte("[]")
	}

	chunkInstances := []*persistedInstanceState{}
	err = json.Unmarshal(existingData, &chunkInstances)
	if err != nil {
		return err
	}

	chunkInstances = append(chunkInstances, toPersistedInstanceState(instance))

	updatedData, err := json.Marshal(chunkInstances)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.addToInstanceIndex(instance, len(chunkInstances)-1)
}

func (s *statePersister) addToInstanceIndex(instance *state.InstanceState, indexInFile int) error {
	s.instanceIndex[instance.InstanceID] = &indexLocation{
		ChunkNumber:  s.lastInstanceChunk,
		IndexInChunk: indexInFile,
	}

	return s.persistInstanceIndexFile()
}

func (s *statePersister) updateInstance(instance *state.InstanceState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateInstanceEntry(instance)
	if err != nil {
		return err
	}

	info.chunkInstances[info.entry.IndexInChunk] = toPersistedInstanceState(instance)

	updatedData, err := json.Marshal(info.chunkInstances)
	if err != nil {
		return err
	}

	return afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
}

func (s *statePersister) removeInstance(instance *state.InstanceState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateInstanceEntry(instance)
	if err != nil {
		return err
	}

	chunkInstances := slices.Delete(
		info.chunkInstances,
		info.entry.IndexInChunk,
		info.entry.IndexInChunk+1,
	)

	updatedData, err := json.Marshal(chunkInstances)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.removeFromInstanceIndex(instance)
}

type persistedInstanceInfo struct {
	chunkInstances []*persistedInstanceState
	entry          *indexLocation
	chunkFilePath  string
}

// A lock must be held when calling this method.
func (s *statePersister) loadAndValidateInstanceEntry(
	instance *state.InstanceState,
) (*persistedInstanceInfo, error) {
	entry, hasEntry := s.instanceIndex[instance.InstanceID]
	if !hasEntry {
		return nil, state.InstanceNotFoundError(instance.InstanceID)
	}

	chunkFilePath := instanceChunkFilePath(s.stateDir, entry.ChunkNumber)
	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return nil, err
	}

	chunkInstances := []*persistedInstanceState{}
	err = json.Unmarshal(existingData, &chunkInstances)
	if err != nil {
		return nil, err
	}

	if entry.IndexInChunk == -1 ||
		entry.IndexInChunk >= len(chunkInstances) {
		return nil, errMalformedStateFile(malformedInstanceStateFileMessage)
	}

	return &persistedInstanceInfo{
		chunkInstances: chunkInstances,
		entry:          entry,
		chunkFilePath:  chunkFilePath,
	}, nil
}

func (s *statePersister) removeFromInstanceIndex(instance *state.InstanceState) error {
	delete(s.instanceIndex, instance.InstanceID)
	return s.persistInstanceIndexFile()
}

func (s *statePersister) persistInstanceIndexFile() error {
	indexData, err := json.Marshal(s.instanceIndex)
	if err != nil {
		return err
	}

	indexFilePath := instanceIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func toPersistedInstanceState(instance *state.InstanceState) *persistedInstanceState {
	childBlueprintsMap := map[string]string{}
	for childName, childBlueprint := range instance.ChildBlueprints {
		childBlueprintsMap[childName] = childBlueprint.InstanceID
	}

	return &persistedInstanceState{
		InstanceID:                 instance.InstanceID,
		InstanceName:               instance.InstanceName,
		Status:                     instance.Status,
		LastStatusUpdateTimestamp:  instance.LastStatusUpdateTimestamp,
		LastDeployedTimestamp:      instance.LastDeployedTimestamp,
		LastDeployAttemptTimestamp: instance.LastDeployAttemptTimestamp,
		ResourceIDs:                instance.ResourceIDs,
		Resources:                  instance.Resources,
		Links:                      instance.Links,
		Metadata:                   instance.Metadata,
		Exports:                    instance.Exports,
		ChildDependencies:          instance.ChildDependencies,
		ChildBlueprints:            childBlueprintsMap,
		Durations:                  instance.Durations,
	}
}

func instanceChunkFilePath(stateDir string, chunkIndex int) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("instances_c%d.json", chunkIndex),
	)
}

func instanceIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "instance_index.json")
}
