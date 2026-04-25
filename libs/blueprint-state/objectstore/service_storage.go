package objectstore

import (
	"context"
	"errors"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
)

// ServiceStorage adapts an objectstore.Service to statestore.Storage so the
// shared persistence engine can back its writes with an object store. ETag
// semantics are intentionally hidden at this layer — the atomic claim path
// talks to the Service directly for IfMatch / IfNoneMatch.
type ServiceStorage struct {
	svc Service
}

// NewServiceStorage returns a statestore.Storage adapter over svc.
func NewServiceStorage(svc Service) *ServiceStorage {
	return &ServiceStorage{svc: svc}
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
	if err := s.svc.Delete(ctx, key); err != nil {
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
