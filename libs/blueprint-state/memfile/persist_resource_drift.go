package memfile

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
)

const (
	malformedResourceDriftStateFileMessage = "resource drift state file is malformed"
)

func (s *statePersister) createResourceDrift(resourceDrift *state.ResourceDriftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastChunkFilePath := resourceDriftChunkFilePath(s.stateDir, s.lastResourceDriftChunk)
	chunkFileInfo, err := s.getFileSizeInfo(lastChunkFilePath)
	if err != nil {
		return err
	}

	chunkFilePath, err := s.prepareChunkFile(
		chunkFileInfo,
		s.lastResourceDriftChunk,
		lastChunkFilePath,
		resourceDriftChunkFilePath,
		func(incrementBy int) {
			s.lastResourceDriftChunk += incrementBy
		},
	)
	if err != nil {
		return err
	}

	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return err
	}

	chunkResourceDriftEntries := []*state.ResourceDriftState{}
	err = json.Unmarshal(existingData, &chunkResourceDriftEntries)
	if err != nil {
		return err
	}

	chunkResourceDriftEntries = append(chunkResourceDriftEntries, resourceDrift)

	updatedData, err := json.Marshal(chunkResourceDriftEntries)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.addToResourceDriftIndex(resourceDrift, len(chunkResourceDriftEntries)-1)
}

func (s *statePersister) addToResourceDriftIndex(resourceDrift *state.ResourceDriftState, indexInFile int) error {
	s.resourceDriftIndex[resourceDrift.ResourceID] = &indexLocation{
		ChunkNumber:  s.lastInstanceChunk,
		IndexInChunk: indexInFile,
	}

	return s.persistResourceDriftIndexFile()
}

func (s *statePersister) persistResourceDriftIndexFile() error {
	indexData, err := json.Marshal(s.resourceDriftIndex)
	if err != nil {
		return err
	}

	indexFilePath := resourceDriftIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func (s *statePersister) updateResourceDrift(resourceDrift *state.ResourceDriftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateResourceDriftEntry(resourceDrift)
	if err != nil {
		return err
	}

	info.chunkResourceDriftEntries[info.entry.IndexInChunk] = resourceDrift

	updatedData, err := json.Marshal(info.chunkResourceDriftEntries)
	if err != nil {
		return err
	}

	return afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
}

func (s *statePersister) removeResourceDrift(resourceDrift *state.ResourceDriftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateResourceDriftEntry(resourceDrift)
	if err != nil {
		return err
	}

	info.chunkResourceDriftEntries = slices.Delete(
		info.chunkResourceDriftEntries,
		info.entry.IndexInChunk,
		info.entry.IndexInChunk+1,
	)

	updatedData, err := json.Marshal(info.chunkResourceDriftEntries)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.removeFromResourceDriftIndex(resourceDrift)
}

type persistedResourceDriftInfo struct {
	chunkResourceDriftEntries []*state.ResourceDriftState
	entry                     *indexLocation
	chunkFilePath             string
}

// A lock must be held when calling this method.
func (s *statePersister) loadAndValidateResourceDriftEntry(
	resourceDrift *state.ResourceDriftState,
) (*persistedResourceDriftInfo, error) {
	entry, hasEntry := s.resourceDriftIndex[resourceDrift.ResourceID]
	if !hasEntry {
		return nil, state.ResourceNotFoundError(resourceDrift.ResourceID)
	}

	chunkFilePath := resourceDriftChunkFilePath(s.stateDir, entry.ChunkNumber)
	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return nil, err
	}

	chunkResourceDriftEntries := []*state.ResourceDriftState{}
	err = json.Unmarshal(existingData, &chunkResourceDriftEntries)
	if err != nil {
		return nil, err
	}

	if entry.IndexInChunk == -1 ||
		entry.IndexInChunk >= len(chunkResourceDriftEntries) {
		return nil, errMalformedStateFile(malformedResourceDriftStateFileMessage)
	}

	return &persistedResourceDriftInfo{
		chunkResourceDriftEntries: chunkResourceDriftEntries,
		entry:                     entry,
		chunkFilePath:             chunkFilePath,
	}, nil
}

func (s *statePersister) removeFromResourceDriftIndex(resourceDrift *state.ResourceDriftState) error {
	delete(s.resourceDriftIndex, resourceDrift.ResourceID)
	return s.persistInstanceIndexFile()
}

func resourceDriftChunkFilePath(stateDir string, chunkIndex int) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("resource_drift_c%d.json", chunkIndex),
	)
}

func resourceDriftIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "resource_drift_index.json")
}
