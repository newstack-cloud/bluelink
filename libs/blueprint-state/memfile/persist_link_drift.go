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
	malformedLinkDriftStateFileMessage = "link drift state file is malformed"
)

func (s *statePersister) createLinkDrift(linkDrift *state.LinkDriftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastChunkFilePath := linkDriftChunkFilePath(s.stateDir, s.lastLinkDriftChunk)
	chunkFileInfo, err := s.getFileSizeInfo(lastChunkFilePath)
	if err != nil {
		return err
	}

	chunkFilePath, err := s.prepareChunkFile(
		chunkFileInfo,
		s.lastLinkDriftChunk,
		lastChunkFilePath,
		linkDriftChunkFilePath,
		func(incrementBy int) {
			s.lastLinkDriftChunk += incrementBy
		},
	)
	if err != nil {
		return err
	}

	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return err
	}

	chunkLinkDriftEntries := []*state.LinkDriftState{}
	err = json.Unmarshal(existingData, &chunkLinkDriftEntries)
	if err != nil {
		return err
	}

	chunkLinkDriftEntries = append(chunkLinkDriftEntries, linkDrift)

	updatedData, err := json.Marshal(chunkLinkDriftEntries)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.addToLinkDriftIndex(linkDrift, len(chunkLinkDriftEntries)-1)
}

func (s *statePersister) addToLinkDriftIndex(linkDrift *state.LinkDriftState, indexInFile int) error {
	s.linkDriftIndex[linkDrift.LinkID] = &indexLocation{
		ChunkNumber:  s.lastLinkDriftChunk,
		IndexInChunk: indexInFile,
	}

	return s.persistLinkDriftIndexFile()
}

func (s *statePersister) persistLinkDriftIndexFile() error {
	indexData, err := json.Marshal(s.linkDriftIndex)
	if err != nil {
		return err
	}

	indexFilePath := linkDriftIndexFilePath(s.stateDir)
	return afero.WriteFile(s.fs, indexFilePath, indexData, 0644)
}

func (s *statePersister) updateLinkDrift(linkDrift *state.LinkDriftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateLinkDriftEntry(linkDrift)
	if err != nil {
		return err
	}

	info.chunkLinkDriftEntries[info.entry.IndexInChunk] = linkDrift

	updatedData, err := json.Marshal(info.chunkLinkDriftEntries)
	if err != nil {
		return err
	}

	return afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
}

func (s *statePersister) removeLinkDrift(linkDrift *state.LinkDriftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.loadAndValidateLinkDriftEntry(linkDrift)
	if err != nil {
		return err
	}

	info.chunkLinkDriftEntries = slices.Delete(
		info.chunkLinkDriftEntries,
		info.entry.IndexInChunk,
		info.entry.IndexInChunk+1,
	)

	updatedData, err := json.Marshal(info.chunkLinkDriftEntries)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, info.chunkFilePath, updatedData, 0644)
	if err != nil {
		return err
	}

	return s.removeFromLinkDriftIndex(linkDrift)
}

type persistedLinkDriftInfo struct {
	chunkLinkDriftEntries []*state.LinkDriftState
	entry                 *indexLocation
	chunkFilePath         string
}

// A lock must be held when calling this method.
func (s *statePersister) loadAndValidateLinkDriftEntry(
	linkDrift *state.LinkDriftState,
) (*persistedLinkDriftInfo, error) {
	entry, hasEntry := s.linkDriftIndex[linkDrift.LinkID]
	if !hasEntry {
		return nil, state.LinkNotFoundError(linkDrift.LinkID)
	}

	chunkFilePath := linkDriftChunkFilePath(s.stateDir, entry.ChunkNumber)
	existingData, err := afero.ReadFile(s.fs, chunkFilePath)
	if err != nil {
		return nil, err
	}

	chunkLinkDriftEntries := []*state.LinkDriftState{}
	err = json.Unmarshal(existingData, &chunkLinkDriftEntries)
	if err != nil {
		return nil, err
	}

	if entry.IndexInChunk == -1 ||
		entry.IndexInChunk >= len(chunkLinkDriftEntries) {
		return nil, errMalformedStateFile(malformedLinkDriftStateFileMessage)
	}

	return &persistedLinkDriftInfo{
		chunkLinkDriftEntries: chunkLinkDriftEntries,
		entry:                 entry,
		chunkFilePath:         chunkFilePath,
	}, nil
}

func (s *statePersister) removeFromLinkDriftIndex(linkDrift *state.LinkDriftState) error {
	delete(s.linkDriftIndex, linkDrift.LinkID)
	return s.persistLinkDriftIndexFile()
}

func linkDriftChunkFilePath(stateDir string, chunkIndex int) string {
	return path.Join(
		stateDir,
		fmt.Sprintf("link_drift_c%d.json", chunkIndex),
	)
}

func linkDriftIndexFilePath(stateDir string) string {
	return path.Join(stateDir, "link_drift_index.json")
}
