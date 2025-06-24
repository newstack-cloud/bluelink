package container

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type noopLinksContainer struct{}

func (l *noopLinksContainer) Get(ctx context.Context, linkID string) (state.LinkState, error) {
	return state.LinkState{}, nil
}

func (l *noopLinksContainer) GetByName(
	ctx context.Context,
	instanceID string,
	linkName string,
) (state.LinkState, error) {
	return state.LinkState{}, nil
}

func (l *noopLinksContainer) Save(ctx context.Context, linkState state.LinkState) error {
	return nil
}

func (l *noopLinksContainer) UpdateStatus(
	ctx context.Context,
	linkID string,
	statusInfo state.LinkStatusInfo,
) error {
	return nil
}

func (l *noopLinksContainer) Remove(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	return state.LinkState{}, nil
}
