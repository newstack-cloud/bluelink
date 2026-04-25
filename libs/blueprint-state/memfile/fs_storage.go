package memfile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/spf13/afero"
)

// fsStorage adapts an afero.Fs to the statestore.Storage interface.
// Keys are interpreted as forward-slash-separated filesystem paths; parent
// directories are created on Write as needed.
type fsStorage struct {
	fs afero.Fs
}

func newFSStorage(fs afero.Fs) *fsStorage {
	return &fsStorage{
		fs: fs,
	}
}

func (s *fsStorage) Read(ctx context.Context, key string) ([]byte, error) {
	data, err := afero.ReadFile(s.fs, key)
	if err != nil {
		if isFsNotFound(err) {
			return nil, statestore.ErrNotFound
		}

		return nil, err
	}

	return data, nil
}

func (s *fsStorage) Write(ctx context.Context, key string, data []byte) error {
	if dir := path.Dir(key); dir != "." && dir != "/" && dir != "" {
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return afero.WriteFile(s.fs, key, data, 0644)
}

func (s *fsStorage) Delete(ctx context.Context, key string) error {
	err := s.fs.Remove(key)
	if err != nil {
		if isFsNotFound(err) {
			return statestore.ErrNotFound
		}

		return err
	}

	return nil
}

func (s *fsStorage) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	walkErr := afero.Walk(s.fs, ".", func(currentPath string, info fs.FileInfo, err error) error {
		if err != nil {
			if isFsNotFound(err) {
				return nil
			}

			return err
		}

		if info.IsDir() {
			return nil
		}

		normalised := strings.TrimPrefix(currentPath, "./")
		if prefix == "" || strings.HasPrefix(normalised, prefix) {
			keys = append(keys, normalised)
		}

		return nil
	})

	if walkErr != nil && !isFsNotFound(walkErr) {
		return nil, walkErr
	}

	return keys, nil
}

func (s *fsStorage) Exists(ctx context.Context, key string) (bool, error) {
	return afero.Exists(s.fs, key)
}

func isFsNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist)
}
