# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2025-07-03

### Changed

- Adds support for link resource data mappings as per the latest updates to the blueprint framework to store information about the specific fields in link data that map to fields in resources in the same blueprint, this is used for overlaying link data for drift checks so that the alterations made by links to "activate" the connection between resources is not detected as drift.
    - Updates the postgres schema to include a new `resource_data_mappings` column in the `links` table and a migration to rebuild the `links_json` view to include the new column.
    - Updates the `memfile` state container implementation to support the new `ResourceDataMappings` field for links and implements an internal map to efficiently retrieve mappings given an instance ID and resource name.

## [0.2.6] - 2025-06-24

### Fixed

- Corrects the go module path in the `go.mod` file to `github.com/newstack-cloud/bluelink/libs/blueprint-state` for all future releases.

## [0.2.5] - 2025-06-10

### Fixed

- Corrects the go module path in the `go.mod` file to `github.com/newstack-cloud/celerity/libs/blueprint-state` for all future releases.

## [0.2.4] - 2025-04-28

### Fixed

- Ensures the `End` field of a `manage.Event` struct is copied when making a copy of an event. When copying events for persistence in the `memfile` state container implementation, the `End` value was not being copied meaning that no end of stream marker was added to the in-memory store or persisted to file. This would lead to streams not closing until the client requests to end after waiting for too long without receiving any events.

## [0.2.3] - 2025-04-28

### Fixed

- Adds fix to the `memfile` state container implementation to make sure that the event is written to the in-memory partition slice. This fixes a bug where no events were being sent to the stream after saving an event until the host application was reloaded to re-build the in-memory state from persisted files.

## [0.2.2] - 2025-04-27

### Fixed

- Adds fix to the `postgres` state container implementation so that the status and changes fields are updated on a save request for an existing change set.
- Adds fix to the `postgres` state container implementation so that the status field is updated on a save request for an existing blueprint validation.

## [0.2.1] - 2025-04-27

### Fixed

- Adds fix to the `memfile` state container implementation to make sure that recently queued events are streamed even if the last event for a channel is an end of stream marker.
- Adds fix to the `postgres` state container implementation to make sure that recently queued events are streamed even if the last event for a channel is an end of stream marker.

## [0.2.0] - 2025-04-26

### Changed

- **Breaking change** - Improves event streaming behaviour to include a new `End` field in the `manage.Event` data type. This is used to to indicate the end of a stream of events.
- Updates the `memfile` state container implementation to make use of the new `End` field to end a stream early if the last saved event was an end of stream marker.
- Updates the `memfile` state container implementation to only fetch recently queued events when a starting event ID is not provided.
- Updates the `postgres` state container implementation to make use of the new `End` field to end a stream early if the last saved event was an end of stream marker.
- Updates the `postgres` state container implementation to only fetch recently queued events when a starting event ID is not provided.
- Updates the database migrations for the postgres state container to include the `end` column in the `events` table migration.

_Breaking changes will occur in early 0.x releases of this library._

## [0.1.0] - 2025-04-26

### Added

- Adds the `postgres` state container implementation that implements the `state.Container` interface from the blueprint framework along with the `manage.Validation`, `manage.Changesets` and `manage.Events` interfaces that are designed to be used by host applications such as the deploy engine.
- Adds a set of database migrations for the postgres state container implementation to be used with database migration tools or integrated into an installer for a host application.
- Adds the `memfile` state container implementation that implements the `state.Container` interface from the blueprint framework along with the `manage.Validation`, `manage.Changesets` and `manage.Events` interfaces that are designed to be used by host applications such as the deploy engine. This implementation uses an in-memory store for retrievals and persists writes to files on disk.
