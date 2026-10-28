package statestore

import "context"

// Category-wide queries answer from the cache, so they call these preloaders first. Under
// ModeLazy the loader is an EntityLister and every entity in the category is loaded. Under
// ModeEager it is not, and the cache already holds everything.
//
// None of these may be called while holding the State lock.

// With revalidation, a cached instance that is no longer listed was removed by another process
// and is dropped, so a listing does not keep returning it.
func (s *State) preloadAllInstances(ctx context.Context) error {
	lister, isLister := s.lister()
	if !isLister {
		return nil
	}

	ids, err := lister.ListInstanceIDs(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if _, _, err := s.LookupInstance(ctx, id); err != nil {
			return err
		}
	}

	if s.revalidation != nil {
		s.uncacheUnlisted(ids)
	}
	return nil
}

func (s *State) uncacheUnlisted(listedIDs []string) {
	listed := make(map[string]bool, len(listedIDs))
	for _, id := range listedIDs {
		listed[id] = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.instances {
		if !listed[id] {
			s.uncacheInstance(id)
		}
	}
}

func (s *State) preloadAllReconciliationResults(ctx context.Context) error {
	return preloadListed(ctx, s, EntityLister.ListReconciliationResultIDs, s.LookupReconciliation)
}

func (s *State) preloadAllCleanupOperations(ctx context.Context) error {
	return preloadListed(ctx, s, EntityLister.ListCleanupOperationIDs, s.LookupCleanupOperation)
}

func preloadListed[T any](
	ctx context.Context,
	s *State,
	list func(EntityLister, context.Context) ([]string, error),
	lookup func(context.Context, string) (T, bool, error),
) error {
	lister, isLister := s.lister()
	if !isLister {
		return nil
	}

	ids, err := list(lister, ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if _, _, err := lookup(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) lister() (EntityLister, bool) {
	lister, isLister := s.loader.(EntityLister)
	return lister, isLister
}

func (s *State) isLazilyListed() bool {
	_, isLister := s.lister()
	return isLister
}
