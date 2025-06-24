package testutils

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/r3labs/sse/v2"
)

func SSEToManageEvent(
	event *sse.Event,
) *manage.Event {
	return &manage.Event{
		ID:   string(event.ID),
		Type: string(event.Event),
		Data: string(event.Data),
	}
}
