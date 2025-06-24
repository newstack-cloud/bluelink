# blueprint state

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint-state&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint-state)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint-state&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint-state)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint-state&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint-state)

A library that provides a collection of state container implementations to be used in the deploy engine or other applications built on top of the blueprint framework.
These state containers implement the following interfaces:

- `state.Container` - The main interface for the state container. This interface is used to manage the state of the blueprint entities and their relationships. This is the interface used by the blueprint framework.
- `manage.Validation` - An interface that manages a blueprint validation request as a resource. This is useful for allowing users to initiate validation of a blueprint and retrieve the results of validation separately, this is especially useful for streaming events when validation takes a while to complete.
- `manage.Changesets` - An interface that manages blueprint change sets as a resource. This is useful for allowing users to initiate change staging, stream events and retrieve the full change set separately. This is especially useful for streaming events when change staging takes a while to complete.
- `manage.Events` - An interface that manages persistence for events that are emitted during validation, change staging and deployment. This is useful for allowing clients of a host application (such as the deploy engine) to recover missed events upon disconnection when streaming events.

## Implementations

- Postgres - A state container backed by a Postgres database that is modelled in a normalised, relational way.
- In-memory with file persistence - A state container backed by an in-process, in-memory store that uses files on disk for persistence. This implementation is mostly useful for single node deployments of the deploy engine or for managing deployments from a developer's machine.

## Usage

### Postgres

A set of database migrations are provided to manage the schema of the database required for the Postgres state container.
See [Postgres migrations](./docs/POSTGRES_MIGRATIONS.md) for more information.

#### Requirements

- A postgres database/cluster that is using Postgres 17.0 and above.
- Only UUIDs are supported for blueprint entity IDs, this means only `core.IDGenerator` imlpementations that generate UUIDs can be used.
- Events must use an ID generator that produces IDs that are time-sortable, this is required to be able to efficiently use an event ID as a starting point for streaming events when a client reconnects to the host application server. For postgres, given event IDs must be UUIDs, UUIDv7 is recommended as it is time-sortable. See [UUIDv7](https://uuid7.com/) for more information.

#### Example

```go
package main

import (
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/newstack-cloud/bluelink/libs/blueprint/core"
    "github.com/newstack-cloud/bluelink/libs/blueprint/state"
    "github.com/newstack-cloud/bluelink/libs/blueprint-state/postgres"
    "github.com/newstack-cloud/bluelink/libs/blueprint/container"
    "github.com/example-org/example-project/changes"
    "github.com/example-org/example-project/validation"
    "github.com/example-org/example-project/events"
)

func main() {
    stateContainer, err := setupStateContainer()
    if err != nil {
        panic(err)
    }

    // Initialise other blueprint loader dependencies ...

    blueprintLoader := container.NewDefaultLoader(
		providers,
		transformers,
		stateContainer,
		childResolver,
	)
    // An example of a service that uses the `manage.Validation` interface
    // to manage blueprint validation requests as a resource.
    validationService := validation.NewService(
        stateContainer.Validation(),
    )
    // An example of a service that uses the `manage.Changesets` interface
    // to manage blueprint change sets as a resource.
    changesetService := changesets.NewService(
        stateContainer.Changesets(),
    )
    // An example of a service that uses the `manage.Events` interface
    // to manage events that are emitted during validation, change staging
    // and deployment.
    eventService := events.NewService(
        stateContainer.Events(),
    )
}

func setupStateContainer() (*postgres.StateContainer, error) {
    connPool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
    if err != nil {
        panic(err)
    }
    logger := core.NewNopLogger()
    // You'll generally want to name the logger to allow for filtering logs
    // that get displayed based on scope or better debugging when an error occurs.
    return postgres.LoadStateContainer(context.Background(), connPool, logger.Named("state"))
}
```

### In-memory with file persistence

#### Requirements

- Events must use an ID generator that generates IDs that are time-sortable, this is required to be able to efficiently use an event ID as a starting point for streaming events when a client reconnects to the host application server. This could be a UUIDv7 or any other timestamp-based or sequential ID. When opting for a simple sequential ID approach, there usually isn't a guarantee that the IDs will be created in the correct time-based order every time where multiple concurrent calls are made to generate IDs for events across multiple threads assuming standard synchronisation mechanisms are being used.

#### Example

```go
package main

import (
    "github.com/spf13/afero"
    "github.com/newstack-cloud/bluelink/libs/blueprint/core"
    "github.com/newstack-cloud/bluelink/libs/blueprint/state"
    "github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
    "github.com/newstack-cloud/bluelink/libs/blueprint/container"
    "github.com/example-org/example-project/changes"
    "github.com/example-org/example-project/validation"
    "github.com/example-org/example-project/events"
)

func main() {
    stateContainer, err := setupStateContainer()
    if err != nil {
        panic(err)
    }

    // Initialise other blueprint loader dependencies ...

    blueprintLoader := container.NewDefaultLoader(
		providers,
		transformers,
		stateContainer,
		childResolver,
	)
    // An example of a service that uses the `manage.Validation` interface
    // to manage blueprint validation requests as a resource.
    validationService := validation.NewService(
        stateContainer.Validation(),
    )
    // An example of a service that uses the `manage.Changesets` interface
    // to manage blueprint change sets as a resource.
    changesetService := changesets.NewService(
        stateContainer.Changesets(),
    )
    // An example of a service that uses the `manage.Events` interface
    // to manage events that are emitted during validation, change staging
    // and deployment.
    eventService := events.NewService(
        stateContainer.Events(),
    )
}

func setupStateContainer() (*memfile.StateContainer, error) {
    fs := afero.NewOsFs()
    logger := core.NewNopLogger()
    // You'll generally want to name the logger to allow for filtering logs
    // that get displayed based on scope or better debugging when an error occurs.
    return memfile.LoadStateContainer(".deploy_state", fs, logger.Named("state"))
}
```

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
