package engine

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// DeployEngine is an interface used to allow the CLI to interact
// with the deploy engine that implements the v1 API.
type DeployEngine interface {

	// CreateBlueprintValidation creates a new blueprint validation
	// for the provided blueprint document and starts the validation process.
	// This will return an ID that can be used to stream the validation events
	// as they occur.
	CreateBlueprintValidation(
		ctx context.Context,
		payload *types.CreateBlueprintValidationPayload,
		query *types.CreateBlueprintValidationQuery,
	) (*manage.BlueprintValidation, error)

	// GetBlueprintValidation retrieves metadata and status information
	// about a blueprint validation.
	// To get validation events (diagnostics), use the `StreamBlueprintValidationEvents`
	// method.
	GetBlueprintValidation(
		ctx context.Context,
		validationID string,
	) (*manage.BlueprintValidation, error)

	// StreamBlueprintValidationEvents streams events from a blueprint
	// validation process.
	// This will produce a stream of events as they occur or that have
	// recently occurred.
	// Any HTTP errors that occur when estabilishing a connection will be sent
	// to the provided error channel.
	StreamBlueprintValidationEvents(
		ctx context.Context,
		validationID string,
		streamTo chan<- types.BlueprintValidationEvent,
		errChan chan<- error,
	) error

	// CleanupBlueprintValidations cleans up blueprint validation that are
	// older than the retention period configured for the Deploy Engine instance.
	CleanupBlueprintValidations(
		ctx context.Context,
	) error

	// CreateChangeset creates a change set for a blueprint deployment.
	// This will start a change staging process for the provided blueprint
	// document and return an ID that can be used to retrieve the change set
	// or stream change staging events.
	//
	// If a valid instance ID or name is provided, a change set will be created
	// by comparing the provided blueprint document with the current state of the
	// existing blueprint instance.
	//
	// Creating a change set should be carried out in preparation for deploying new blueprint
	// instances or updating existing blueprint instances.
	CreateChangeset(
		ctx context.Context,
		payload *types.CreateChangesetPayload,
	) (*manage.Changeset, error)

	// GetChangeset retrieves a change set for a blueprint deployment.
	// This will return the current status of the change staging process.
	// If complete, the response will include a full set of changes that
	// will be applied when deploying the change set.
	GetChangeset(
		ctx context.Context,
		changesetID string,
	) (*manage.Changeset, error)

	// StreamChangeStagingEvents streams events from the change staging process
	// for the given change set ID.
	// This will produce a stream of events as they occur or that have recently occurred.
	// Any HTTP errors that occur when estabilishing a connection will be sent
	// to the provided error channel.
	StreamChangeStagingEvents(
		ctx context.Context,
		changesetID string,
		streamTo chan<- types.ChangeStagingEvent,
		errChan chan<- error,
	) error

	// CleanupChangesets cleans up change sets that are older than the retention
	// period configured for the Deploy Engine instance.
	CleanupChangesets(
		ctx context.Context,
	) error

	// CreateBlueprintInstance (Deploy New) creates a new blueprint deployment instance.
	// This will start the deployment process for the provided blueprint
	// document and change set.
	// It will return a blueprint instance resource containing an ID that
	// can be used to stream deployment events as they occur.
	CreateBlueprintInstance(
		ctx context.Context,
		payload *types.BlueprintInstancePayload,
	) (*state.InstanceState, error)

	// UpdateBlueprintInstance (Deploy Existing) updates an existing blueprint
	// deployment instance.
	// This will start the deployment process for the provided blueprint
	// document and change set.
	// It will return the current state of the blueprint instance,
	// the same ID provided should be used to stream deployment events as they occur.
	UpdateBlueprintInstance(
		ctx context.Context,
		instanceID string,
		payload *types.BlueprintInstancePayload,
	) (*state.InstanceState, error)

	// GetBlueprintInstance retrieves a blueprint deployment instance.
	// This will return the current status of the deployment along with
	// the current state of the blueprint intance.
	GetBlueprintInstance(
		ctx context.Context,
		instanceID string,
	) (*state.InstanceState, error)

	// GetBlueprintInstanceExports retrieves the exports from a blueprint
	// deployment instance.
	// This will return the exported fields from the blueprint instance.
	GetBlueprintInstanceExports(
		ctx context.Context,
		instanceID string,
	) (map[string]*state.ExportState, error)

	// DestroyBlueprintInstance destroys a blueprint deployment instance.
	// This will start the destroy process for the provided change set.
	// It will return the current state of the blueprint instance,
	// the same ID provided should be used to stream destroy events as they occur.
	DestroyBlueprintInstance(
		ctx context.Context,
		instanceID string,
		payload *types.DestroyBlueprintInstancePayload,
	) (*state.InstanceState, error)

	// StreamBlueprintInstanceEvents streams events from the current deployment
	// process for the given blueprint instance ID.
	//
	// This will stream events for new deployments, updates and for destroying
	// a blueprint instance.
	//
	// This will produce a stream of events as they occur or that have recently occurred.
	//
	// For a blueprint instance that has been destroyed, this stream will no longer be available
	// to new connections once the destroy process has been successfully completed.
	//
	// Any HTTP errors that occur when estabilishing a connection or unexpected failures
	// in the deployment process will be sent to the provided error channel.
	StreamBlueprintInstanceEvents(
		ctx context.Context,
		instanceID string,
		streamTo chan<- types.BlueprintInstanceEvent,
		errChan chan<- error,
	) error

	// CleanupEvents cleans up events that are older than the retention
	// period configured for the Deploy Engine instance.
	//
	// This will clean up events for all processes including blueprint validations,
	// change staging and deployments. This will not clean up the resources themselves,
	// only the events that are associated with them.
	// You can clean up change sets and blueprint validations using the dedicated methods.
	CleanupEvents(
		ctx context.Context,
	) error
}
