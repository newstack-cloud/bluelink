package manage

import (
	"context"
	"time"
)

// Events is an interface that represents a service that manages
// state for events that are produced and streamed for blueprint validation,
// change staging and deployments.
type Events interface {
	// Get an event by ID.
	Get(ctx context.Context, id string) (Event, error)

	// Save a new event for blueprint validation, change staging or deployment.
	Save(ctx context.Context, event *Event) error

	// Stream events for a given channel, starting at an optional event ID.
	// The channel ID is used to identify the event stream
	// for a specific blueprint validation, change staging process or deployment.
	// A channel is returned that can be used to stop the stream.
	// Once the caller knows that they are done with the stream,
	// it must send an empty struct to the channel to stop the stream.
	Stream(
		ctx context.Context,
		params *EventStreamParams,
		streamTo chan Event,
		errChan chan error,
	) (chan struct{}, error)

	// Cleanup removes all events that are older than the given threshold date.
	// Returns the number of events deleted.
	Cleanup(ctx context.Context, thresholdDate time.Time) (int64, error)

	// GetLastEventID returns the ID of the last event for a given channel.
	// Returns empty string if no events exist for the channel.
	GetLastEventID(ctx context.Context, channelType, channelID string) (string, error)
}

// Event represents an event during blueprint validation,
// change staging or deployment processes.
type Event struct {
	// A unique ID for the event.
	ID string `json:"id"`
	// The type of event that occurred.
	Type string `json:"type"`
	// The type of channel that the event is associated with.
	ChannelType string `json:"channelType"`
	// The ID of the channel that the event is associated with.
	ChannelID string `json:"channelId"`
	// Data is a JSON encoded string that contains the event data.
	Data string `json:"data"`
	// The unix timestamp in seconds when the event was created.
	Timestamp int64 `json:"timestamp"`
	// Whether or not the current event represents the end of a stream.
	End bool `json:"end"`
}

////////////////////////////////////////////////////////////////////////////////////
// Helper method that implements the `manage.Entity` interface
// used to get common members of multiple entity types.
////////////////////////////////////////////////////////////////////////////////////

func (c *Event) GetID() string {
	return c.ID
}

func (c *Event) GetCreated() int64 {
	return c.Timestamp
}

type EventStreamParams struct {
	// The type of channel to listen to.
	ChannelType string `json:"channelType"`
	// The ID of the channel to listen to.
	// This is used to identify the event stream
	// for a specific blueprint validation, change staging process or deployment.
	ChannelID string `json:"channelId"`
	// The ID of the event to start listening from.
	// If this is not provided, the stream will start from the earliest retained
	// event in the channel.
	// This is exclusive, meaning that the event with this ID will not be included
	// in the stream.
	StartingEventID string `json:"startingEventId,omitempty"`
}
