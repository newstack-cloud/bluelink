package deployengine

import (
	"encoding/json"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/r3labs/sse/v2"
)

func sseToBlueprintValidationEvent(
	event *sse.Event,
) types.BlueprintValidationEvent {
	target := &types.BlueprintValidationEvent{}
	err := json.Unmarshal(event.Data, target)
	if err != nil {
		// Return an empty event if we can't unmarshal to
		// a blueprint validation event.
		// This means that if a client receives empty blueprint
		// validation events, it is most likely because the server
		// sent an event that was not in the expected shape.
		return types.BlueprintValidationEvent{}
	}

	target.ID = string(event.ID)
	return *target
}

func checkIsValidationStreamEnd(
	event types.BlueprintValidationEvent,
) bool {
	return event.End
}

func sseToChangeStagingEvent(
	event *sse.Event,
) types.ChangeStagingEvent {
	switch types.ChangeStagingEventType(event.Event) {
	case types.ChangeStagingEventTypeResourceChanges:
		resourceChanges := &types.ResourceChangesEventData{}
		err := json.Unmarshal(event.Data, resourceChanges)
		if err != nil {
			return types.ChangeStagingEvent{}
		}
		return types.ChangeStagingEvent{
			ID:              string(event.ID),
			ResourceChanges: resourceChanges,
		}
	case types.ChangeStagingEventTypeChildChanges:
		childChanges := &types.ChildChangesEventData{}
		err := json.Unmarshal(event.Data, childChanges)
		if err != nil {
			return types.ChangeStagingEvent{}
		}
		return types.ChangeStagingEvent{
			ID:           string(event.ID),
			ChildChanges: childChanges,
		}
	case types.ChangeStagingEventTypeLinkChanges:
		linkChanges := &types.LinkChangesEventData{}
		err := json.Unmarshal(event.Data, linkChanges)
		if err != nil {
			return types.ChangeStagingEvent{}
		}
		return types.ChangeStagingEvent{
			ID:          string(event.ID),
			LinkChanges: linkChanges,
		}
	case types.ChangeStagingEventTypeCompleteChanges:
		completeChanges := &types.CompleteChangesEventData{}
		err := json.Unmarshal(event.Data, completeChanges)
		if err != nil {
			return types.ChangeStagingEvent{}
		}
		return types.ChangeStagingEvent{
			ID:              string(event.ID),
			CompleteChanges: completeChanges,
		}
	}

	return types.ChangeStagingEvent{}
}

func checkIsChangeStagingStreamEnd(
	event types.ChangeStagingEvent,
) bool {
	return event.GetType() == types.ChangeStagingEventTypeCompleteChanges
}

func sseToBlueprintInstanceEvent(
	event *sse.Event,
) types.BlueprintInstanceEvent {
	switch types.BlueprintInstanceEventType(event.Event) {
	case types.BlueprintInstanceEventTypeResourceUpdate:
		resourceUpdateMessage := &container.ResourceDeployUpdateMessage{}
		err := json.Unmarshal(event.Data, resourceUpdateMessage)
		if err != nil {
			return types.BlueprintInstanceEvent{}
		}
		return types.BlueprintInstanceEvent{
			ID: string(event.ID),
			DeployEvent: container.DeployEvent{
				ResourceUpdateEvent: resourceUpdateMessage,
			},
		}
	case types.BlueprintInstanceEventTypeChildUpdate:
		childUpdateMessage := &container.ChildDeployUpdateMessage{}
		err := json.Unmarshal(event.Data, childUpdateMessage)
		if err != nil {
			return types.BlueprintInstanceEvent{}
		}
		return types.BlueprintInstanceEvent{
			ID: string(event.ID),
			DeployEvent: container.DeployEvent{
				ChildUpdateEvent: childUpdateMessage,
			},
		}
	case types.BlueprintInstanceEventTypeLinkUpdate:
		linkUpdateMessage := &container.LinkDeployUpdateMessage{}
		err := json.Unmarshal(event.Data, linkUpdateMessage)
		if err != nil {
			return types.BlueprintInstanceEvent{}
		}
		return types.BlueprintInstanceEvent{
			ID: string(event.ID),
			DeployEvent: container.DeployEvent{
				LinkUpdateEvent: linkUpdateMessage,
			},
		}
	case types.BlueprintInstanceEventTypeInstanceUpdate:
		instanceUpdateMessage := &container.DeploymentUpdateMessage{}
		err := json.Unmarshal(event.Data, instanceUpdateMessage)
		if err != nil {
			return types.BlueprintInstanceEvent{}
		}
		return types.BlueprintInstanceEvent{
			ID: string(event.ID),
			DeployEvent: container.DeployEvent{
				DeploymentUpdateEvent: instanceUpdateMessage,
			},
		}
	case types.BlueprintInstanceEventTypeDeployFinished:
		finishedMessage := &container.DeploymentFinishedMessage{}
		err := json.Unmarshal(event.Data, finishedMessage)
		if err != nil {
			return types.BlueprintInstanceEvent{}
		}
		return types.BlueprintInstanceEvent{
			ID: string(event.ID),
			DeployEvent: container.DeployEvent{
				FinishEvent: finishedMessage,
			},
		}
	}

	return types.BlueprintInstanceEvent{}
}

func checkIsBlueprintInstanceStreamEnd(
	event types.BlueprintInstanceEvent,
) bool {
	return event.GetType() == types.BlueprintInstanceEventTypeDeployFinished
}
