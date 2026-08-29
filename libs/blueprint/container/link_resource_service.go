package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Attributes every lock a link takes to that link, whatever the link implementation
// asked for.
//
// The deployer releases a link's locks with ReleaseResourceLocksAcquiredBy, keyed on
// the link ID, so a lock recorded under any other acquirer is never released by it. A
// link implementation acquiring a lock on a shared intermediary has no reason to know
// that, and in-process it holds the registry directly, with nothing between it and the
// lock table to fill the field in. Left unset, the lock outlived the link and the next
// link to want that resource waited on a holder that had already finished.
//
// Stamping it here makes acquire and release agree by construction rather than by every
// provider remembering.
type linkScopedResourceService struct {
	provider.ResourceService
	linkID string
}

func newLinkScopedResourceService(
	resourceService provider.ResourceService,
	linkID string,
) provider.ResourceService {
	return &linkScopedResourceService{
		ResourceService: resourceService,
		linkID:          linkID,
	}
}

func (s *linkScopedResourceService) AcquireResourceLock(
	ctx context.Context,
	input *provider.AcquireResourceLockInput,
) error {
	scoped := *input
	scoped.AcquiredBy = s.linkID

	return s.ResourceService.AcquireResourceLock(ctx, &scoped)
}

// Stamped here for the same reason as on acquire, and it matters more.
// This is because a release only takes effect on a lock recorded under the same acquirer,
// so an unstamped release would silently do nothing and the caller would keep a lock it believes
// it gave up.
func (s *linkScopedResourceService) ReleaseResourceLock(
	ctx context.Context,
	input *provider.ReleaseResourceLockInput,
) error {
	scoped := *input
	scoped.AcquiredBy = s.linkID

	return s.ResourceService.ReleaseResourceLock(ctx, &scoped)
}
