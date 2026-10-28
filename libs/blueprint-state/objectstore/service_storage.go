package objectstore

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
)

// ServiceStorage adapts an objectstore.Service to statestore.Storage so the
// shared persistence engine can back its writes with an object store. It also
// implements statestore.ConditionalStorage, with object ETags as versions, so
// engines sharing a bucket do not overwrite each other's instance changes.
type ServiceStorage struct {
	svc Service
}

// NewServiceStorage returns a statestore.Storage adapter over svc.
func NewServiceStorage(svc Service) *ServiceStorage {
	return &ServiceStorage{svc: svc}
}

var _ statestore.ConditionalStorage = (*ServiceStorage)(nil)

func (s *ServiceStorage) ReadVersion(ctx context.Context, key string) ([]byte, string, error) {
	data, etag, err := s.svc.Get(ctx, key)
	if err != nil {
		return nil, "", mapStorageErr(err)
	}
	return data, etag, nil
}

func (s *ServiceStorage) WriteIfMatch(
	ctx context.Context,
	key string,
	data []byte,
	version string,
) (string, error) {
	opts := &PutOptions{IfMatch: version}
	if version == "" {
		opts = &PutOptions{IfNoneMatch: "*"}
	}

	etag, err := s.svc.Put(ctx, key, data, opts)
	if err != nil {
		return "", mapStorageErr(err)
	}
	return etag, nil
}

func (s *ServiceStorage) DeleteIfMatch(ctx context.Context, key string, version string) error {
	return mapStorageErr(s.svc.Delete(ctx, key, &DeleteOptions{IfMatch: version}))
}

func mapStorageErr(err error) error {
	switch {
	case isObjectNotFound(err):
		return statestore.ErrNotFound
	case isPreconditionFailed(err):
		return statestore.ErrPreconditionFailed
	}
	return err
}

func (s *ServiceStorage) Read(ctx context.Context, key string) ([]byte, error) {
	data, _, err := s.svc.Get(ctx, key)
	if err != nil {
		if isObjectNotFound(err) {
			return nil, statestore.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (s *ServiceStorage) Write(ctx context.Context, key string, data []byte) error {
	_, err := s.svc.Put(ctx, key, data, nil)
	return err
}

func (s *ServiceStorage) Delete(ctx context.Context, key string) error {
	if err := s.svc.Delete(ctx, key, nil); err != nil {
		if isObjectNotFound(err) {
			return statestore.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ServiceStorage) List(ctx context.Context, prefix string) ([]string, error) {
	infos, err := s.svc.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(infos))
	for _, info := range infos {
		keys = append(keys, info.Key)
	}
	return keys, nil
}

func (s *ServiceStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.svc.Head(ctx, key)
	if err != nil {
		if isObjectNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isObjectNotFound(err error) bool {
	var sErr *Error
	if !errors.As(err, &sErr) {
		return false
	}
	return sErr.ReasonCode == ErrorReasonCodeObjectNotFound
}
