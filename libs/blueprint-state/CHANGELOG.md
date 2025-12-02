# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.1](https://github.com/newstack-cloud/bluelink/compare/blueprint-state/v0.4.0...blueprint-state/v0.4.1) (2025-12-02)


### Features

* **blueprint-state:** add blueprint state library with memfile implementation ([31beba3](https://github.com/newstack-cloud/bluelink/commit/31beba325799acb4611b0447f618c8364e012d94))
* **blueprint-state:** add blueprint validation state implementation to memfile ([9a9cab3](https://github.com/newstack-cloud/bluelink/commit/9a9cab385672ee73e2c8875815c2571ae5d1cba7))
* **blueprint-state:** add change set state implementation to memfile ([676e372](https://github.com/newstack-cloud/bluelink/commit/676e37287aaec598d88fceed3dd73540aa2a2d9a))
* **blueprint-state:** add event state implementation to memfile ([16e55e2](https://github.com/newstack-cloud/bluelink/commit/16e55e2989d98c95e6e63657bc6688ee54217178))
* **blueprint-state:** add functionality to clean up events before given time ([896bba3](https://github.com/newstack-cloud/bluelink/commit/896bba3665ad188e1c8b6f040096da3c93e702a8))
* **blueprint-state:** add instance name column and behaviour to lookup instance by name ([8957ca8](https://github.com/newstack-cloud/bluelink/commit/8957ca89c9b4dfcd2aacd906d3e74cfbb9ec39c0))
* **blueprint-state:** add pg implementation for streaming events ([979ff33](https://github.com/newstack-cloud/bluelink/commit/979ff33faf8fd12d54e6b7ba36927484497ebfe6))
* **blueprint-state:** add postgres implementation for blueprint validations ([ed096cb](https://github.com/newstack-cloud/bluelink/commit/ed096cb89aa0bb84a66cc9ad69468b92bb978439))
* **blueprint-state:** add postgres implementation for change sets ([74b99c3](https://github.com/newstack-cloud/bluelink/commit/74b99c3bf2217b75046de37b6d77600049fc3df6))
* **blueprint-state:** add postgres state container implementation ([3600a1d](https://github.com/newstack-cloud/bluelink/commit/3600a1d0ece1ecf3d030ecb3dd55cbb9ceab6457))
* **blueprint-state:** add support for link resource data mappings ([7b164fc](https://github.com/newstack-cloud/bluelink/commit/7b164fc6b17e59c010a3334901a407cc3735b755))
* **blueprint-state:** improve memfile event streaming behaviour ([5e3771d](https://github.com/newstack-cloud/bluelink/commit/5e3771dbc88bd34632e75c546991bec083502962))
* **blueprint-state:** improve postgres event streaming behaviour ([56780dd](https://github.com/newstack-cloud/bluelink/commit/56780ddf5bcf6c9c305d4f5ce7ed84a95a50ef4c))
* **blueprint-state:** update blueprint framework version and add support for resource type ([eee53b0](https://github.com/newstack-cloud/bluelink/commit/eee53b055dce26e353c74c4363927c31b0ea96dc))


### Bug Fixes

* **blueprint-state:** add correction to postgres state container to update on conflict ([27a330f](https://github.com/newstack-cloud/bluelink/commit/27a330fe19f77face6b2b5e9207a49562f10dcad))
* **blueprint-state:** add memfile fix for saving events in-memory ([e74ebe8](https://github.com/newstack-cloud/bluelink/commit/e74ebe89b99c3d14a0cce3c238986759133844d5))
* **blueprint-state:** add memfile fix for streaming queued events ([a251dfc](https://github.com/newstack-cloud/bluelink/commit/a251dfc4178f419cbf5abd4b83c3b8a3746e3085))
* **blueprint-state:** add postgres fix for streaming queued events ([f1fe30d](https://github.com/newstack-cloud/bluelink/commit/f1fe30d453c6f428c01f5844ee6e98885ed72812))
* **blueprint-state:** make sure change set is added to chunk on update ([fa2e9a9](https://github.com/newstack-cloud/bluelink/commit/fa2e9a9cd7079a09c9a671d73e892f14e2128c62))
* **blueprint-state:** make sure end field is copied for events ([b350b35](https://github.com/newstack-cloud/bluelink/commit/b350b3540487b6546b9db1d07a4015d7f21d8122))
* **blueprint-state:** update dependencies to use versions in new org ([82bec4a](https://github.com/newstack-cloud/bluelink/commit/82bec4a4b1cabbc1f6824b83a3587b204a277060))

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
