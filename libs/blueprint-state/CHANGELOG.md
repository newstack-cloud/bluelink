# Changelog

All notable changes to this project will be documented in this file.

## [0.5.0](https://github.com/newstack-cloud/bluelink/compare/blueprint-state/v0.4.2...blueprint-state/v0.5.0) (2026-01-06)


### Features

* **blueprint-state:** add computed fields to resource state ([0811266](https://github.com/newstack-cloud/bluelink/commit/0811266adf3c7c81b41e37c016c3e060324c19df))
* **blueprint-state:** add functionality to list blueprint instances ([377139a](https://github.com/newstack-cloud/bluelink/commit/377139a1f1343c12e7f3e5d2fa573860b38ffbde))
* **blueprint-state:** add persistence for link drift ([a778aa6](https://github.com/newstack-cloud/bluelink/commit/a778aa6dad6b4abdb953413f58ddeb2419a179d2))
* **blueprint-state:** add persistence for reconciliation results ([9f16d59](https://github.com/newstack-cloud/bluelink/commit/9f16d59772aab7f09a3b109042684aa6d89d0015))
* **blueprint-state:** add support for getting the last event id for a given channel ([f4dace1](https://github.com/newstack-cloud/bluelink/commit/f4dace1eb321d6a17830249e97f6d6f28a63ba9d))
* **blueprint-state:** add support for saving system metadata for resources ([7b4a455](https://github.com/newstack-cloud/bluelink/commit/7b4a4554ea4db5ab6738ef334e58cf6d943065d0))


### Bug Fixes

* **blueprint-state:** add fix for memfile drift persistence when drift files are not present ([8c1519e](https://github.com/newstack-cloud/bluelink/commit/8c1519ed576a9728ac4632b8e3808d12bc5c30c5))
* **blueprint-state:** add fix to make sure starting event id is exclusive in memfile ([6966017](https://github.com/newstack-cloud/bluelink/commit/6966017ebf166e5c132ce48c0e079f6815746809))


### Dependencies

* **blueprint-state:** update blueprint core lib ([3f33990](https://github.com/newstack-cloud/bluelink/commit/3f3399056c6e4a0b88215f1cb477f7325e0b14b4))

## [0.4.2](https://github.com/newstack-cloud/bluelink/compare/blueprint-state/v0.4.1...blueprint-state/v0.4.2) (2025-12-16)


### Bug Fixes

* **blueprint-state:** add nil check for child blueprints when attaching to parent ([f01dbf5](https://github.com/newstack-cloud/bluelink/commit/f01dbf5eec3a6d3b2be1964faff3c22d90ecc679))


### Dependencies

* **blueprint-state:** update blueprint lib to 0.36.4 ([e87f069](https://github.com/newstack-cloud/bluelink/commit/e87f069bca1b953f73e723ca8c402c90b2204acc))

## [0.4.1](https://github.com/newstack-cloud/bluelink/compare/blueprint-state/v0.4.0...blueprint-state/v0.4.1) (2025-12-13)


### Bug Fixes

* **blueprint-state:** add fix to gracefully handle when instances chunk file is missing ([36266e7](https://github.com/newstack-cloud/bluelink/commit/36266e77d24d374c38a493a01c9cd1c099c32044))
* **blueprint-state:** initialise empty maps for instance state in memfile state container ([32dadf1](https://github.com/newstack-cloud/bluelink/commit/32dadf1ec5d56015b1663c0d37eafb529ce87bac))


### Dependencies

* **blueprint-state:** bump the go-deps group ([1a530f9](https://github.com/newstack-cloud/bluelink/commit/1a530f972ad22fb5705dbb3dfd1798088db7b36b))
* **blueprint-state:** bump up blueprint lib to 0.36.3 ([ee669cb](https://github.com/newstack-cloud/bluelink/commit/ee669cb4b2ad2ad44ae358aababc97cf2f4d2841))
* **blueprint-state:** update blueprint lib and snapshots with mapping node updates ([45a3650](https://github.com/newstack-cloud/bluelink/commit/45a36507ff3b38b9477a9245d4e00fe732d211b6))

## [0.4.0] - 2025-07-20

### Added

- Adds support for storing resource type in link intermediary resource state. The change made to the `memfile` state container implementation ensures that the `ResourceType` field is copied when copying intermediary resource states. The new field will be persisted automatically for both the `memfile` and `postgres` state container implementations as long as it is set, the postgres state container stores the intermediary resource state for a link in a `jsonb` column that is populated by marshalling the `LinkIntermediaryResourceState` struct.

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
