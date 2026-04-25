package mockservice

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
)

type entry struct {
	data []byte
	etag string
}

// Service is an in-memory objectstore.Service implementation.
//
// ETags are simulated with a monotonically-increasing counter per service
// instance: every successful Put yields a new ETag, so clients holding a stale
// ETag will fail IfMatch checks.
//
// Hooks must be configured via SetHooks before any concurrent operations start;
// SetHooks does not synchronise with in-flight calls.
type Service struct {
	mu      sync.Mutex
	objects map[string]*entry
	etagSeq int64
	hooks   Hooks
}

// Hooks lets tests inject errors for specific operations. Each hook runs
// before the corresponding Service method touches state. If a hook returns
// a non-nil error, that error is returned verbatim and no state changes.
// Nil hook fields are skipped.
type Hooks struct {
	BeforeGet    func(ctx context.Context, key string) error
	BeforePut    func(ctx context.Context, key string, data []byte, opts *objectstore.PutOptions) error
	BeforeDelete func(ctx context.Context, key string) error
	BeforeHead   func(ctx context.Context, key string) error
	BeforeList   func(ctx context.Context, prefix string) error
}

// New returns an empty in-memory Service.
func New() *Service {
	return &Service{objects: map[string]*entry{}}
}

// SetHooks configures error-injection hooks. Call before concurrent operations.
func (s *Service) SetHooks(h Hooks) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks = h
}

// Reset clears all stored objects and resets the ETag counter. Hooks are kept.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects = map[string]*entry{}
	s.etagSeq = 0
}

func (s *Service) Get(ctx context.Context, key string) ([]byte, string, error) {
	if hook := s.hooks.BeforeGet; hook != nil {
		if err := hook(ctx, key); err != nil {
			return nil, "", err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	obj, ok := s.objects[key]
	if !ok {
		return nil, "", notFoundError(key)
	}

	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)

	return dataCopy, obj.etag, nil
}

func (s *Service) Put(
	ctx context.Context,
	key string,
	data []byte,
	opts *objectstore.PutOptions,
) (string, error) {
	if hook := s.hooks.BeforePut; hook != nil {
		if err := hook(ctx, key, data, opts); err != nil {
			return "", err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.objects[key]

	if opts != nil {
		if err := checkPreconditions(key, existing, exists, opts); err != nil {
			return "", err
		}
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	newETag := s.nextETagLocked()
	s.objects[key] = &entry{data: dataCopy, etag: newETag}
	return newETag, nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	if hook := s.hooks.BeforeDelete; hook != nil {
		if err := hook(ctx, key); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.objects[key]; !ok {
		return notFoundError(key)
	}
	delete(s.objects, key)
	return nil
}

func (s *Service) Head(ctx context.Context, key string) (*objectstore.ObjectInfo, error) {
	if hook := s.hooks.BeforeHead; hook != nil {
		if err := hook(ctx, key); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	obj, ok := s.objects[key]
	if !ok {
		return nil, notFoundError(key)
	}

	return &objectstore.ObjectInfo{
		Key:  key,
		Size: int64(len(obj.data)),
		ETag: obj.etag,
	}, nil
}

func (s *Service) List(ctx context.Context, prefix string) ([]*objectstore.ObjectInfo, error) {
	if hook := s.hooks.BeforeList; hook != nil {
		if err := hook(ctx, prefix); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var out []*objectstore.ObjectInfo
	for k, obj := range s.objects {
		if strings.HasPrefix(k, prefix) {
			out = append(out, &objectstore.ObjectInfo{
				Key:  k,
				Size: int64(len(obj.data)),
				ETag: obj.etag,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })

	return out, nil
}

func (s *Service) nextETagLocked() string {
	s.etagSeq++
	return strconv.FormatInt(s.etagSeq, 10)
}

func checkPreconditions(
	key string,
	existing *entry,
	exists bool,
	opts *objectstore.PutOptions,
) error {
	if opts.IfNoneMatch == "*" && exists {
		return preconditionError(
			fmt.Sprintf("object %s already exists (IfNoneMatch=*)", key),
		)
	}

	if opts.IfNoneMatch != "" && opts.IfNoneMatch != "*" &&
		exists && existing.etag == opts.IfNoneMatch {
		return preconditionError(
			fmt.Sprintf("etag matches IfNoneMatch=%s", opts.IfNoneMatch),
		)
	}

	if opts.IfMatch != "" {
		if !exists || existing.etag != opts.IfMatch {
			return preconditionError(
				fmt.Sprintf("IfMatch=%s does not match current etag", opts.IfMatch),
			)
		}
	}

	return nil
}

func notFoundError(key string) error {
	return &objectstore.Error{
		ReasonCode: objectstore.ErrorReasonCodeObjectNotFound,
		Err:        fmt.Errorf("object not found: %s", key),
	}
}

func preconditionError(message string) error {
	return &objectstore.Error{
		ReasonCode: objectstore.ErrorReasonCodePreconditionFailed,
		Err:        fmt.Errorf("precondition failed: %s", message),
	}
}
