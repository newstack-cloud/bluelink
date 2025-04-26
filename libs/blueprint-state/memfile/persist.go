package memfile

import (
	"errors"
	"sync"

	"github.com/spf13/afero"
)

type statePersister struct {
	stateDir               string
	fs                     afero.Fs
	instanceIndex          map[string]*indexLocation
	lastInstanceChunk      int
	maxGuideFileSize       int64
	resourceDriftIndex     map[string]*indexLocation
	lastResourceDriftChunk int
	eventIndex             map[string]*eventIndexLocation
	maxEventPartitionSize  int64
	changesetIndex         map[string]*indexLocation
	lastChangesetChunk     int
	// The persister has its own mutex, separate from
	// the state container's read/write lock.
	mu sync.Mutex
}

func (s *statePersister) prepareChunkFile(
	chunkFileInfo fileSizeInfo,
	lastChunk int,
	lastChunkFilePath string,
	createChunkFilePath func(string, int) string,
	incrementLastChunk func(int),
) (string, error) {
	if chunkFileInfo.Size() >= s.maxGuideFileSize {
		incrementLastChunk(1)
		newChunkFilePath := createChunkFilePath(s.stateDir, lastChunk+1)
		err := afero.WriteFile(s.fs, newChunkFilePath, []byte("[]"), 0644)
		if err != nil {
			return "", err
		}

		return newChunkFilePath, nil
	}

	return lastChunkFilePath, nil
}

func (s *statePersister) getFileSizeInfo(
	filePath string,
) (fileSizeInfo, error) {
	var chunkFileInfo fileSizeInfo
	var err error
	chunkFileInfo, err = s.fs.Stat(filePath)
	if err != nil {
		if !errors.Is(err, afero.ErrFileNotFound) {
			return nil, err
		}
		chunkFileInfo = &emptyFileInfo{}
	}

	return chunkFileInfo, nil
}

type fileSizeInfo interface {
	Size() int64
}

type emptyFileInfo struct{}

func (emptyFileInfo) Size() int64 {
	return 0
}
