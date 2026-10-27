package statestore

import "context"

// Category-wide queries answer from the cache, so they call these preloaders first. Under
// ModeLazy the loader is an EntityLister and every entity in the category is loaded. Under
// ModeEager it is not, and the cache already holds everything.
//
// None of these may be called while holding the State lock.

func (s *State) preloadAllInstances(ctx context.Context) error {
	return preloadListed(ctx, s, EntityLister.ListInstanceIDs, s.LookupInstance)
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
