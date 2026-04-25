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
- Object store - A state container backed by a cloud object storage service (Amazon S3, Google Cloud Storage or Azure Blob Storage). State is persisted as discrete objects keyed by entity ID under a configurable prefix, and concurrent writers are serialised by the backend's native conditional-write primitives (S3 / Azure Blob ETag CAS, GCS generation CAS). Suitable for CI/CD pipelines or horizontally scaled deploy engine deployments where many processes need to share state safely without a managed database.

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

### Object store

State is loaded lazily from the underlying bucket / container — each process only materialises the entities it actually touches. Concurrent `InitialiseAndClaim` and `ClaimForDeployment` calls across processes are serialised by the backend's native CAS:

- **S3** uses ETag-based `IfMatch` / `IfNoneMatch` conditional writes.
- **GCS** uses generation-based `GenerationMatch` / `DoesNotExist` conditions.
- **Azure Blob Storage** uses ETag-based `IfMatch` / `IfNoneMatch` access conditions.

Three backend `Service` implementations are provided under `objectstore/stores/`:

- `objectstore/stores/s3` — S3 and S3-compatible gateways (LocalStack, MinIO etc.)
- `objectstore/stores/gcs` — Google Cloud Storage and GCS-compatible emulators (fake-gcs-server)
- `objectstore/stores/azureblob` — Azure Blob Storage and Azurite

Each backend exposes a thin `NewClient` helper that wraps its provider SDK and a `NewService` constructor that binds the SDK client to a bucket or container. The `objectstore.LoadStateContainer` function then composes the chosen `Service` with the shared `statestore` engine.

#### Requirements

- Events must use an ID generator that produces time-sortable IDs (e.g. UUIDv7) so an event ID can be used as a starting point for resuming an event stream after a client reconnects.
- The chosen bucket / container must already exist; the library does not provision storage.
- Credentials are supplied through the underlying provider SDK (default credential chain for AWS / Application Default Credentials for GCS / shared-key credential for Azure). The `NewClient` helpers expose endpoint and emulator-friendly options (`Endpoint`, `WithoutAuthentication`, `UsePathStyle` etc.) when targeting a local emulator.

#### Example (S3)

```go
package main

import (
    "context"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/newstack-cloud/bluelink/libs/blueprint/core"
    "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
    s3store "github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/stores/s3"
)

func setupStateContainer(ctx context.Context) (*objectstore.StateContainer, error) {
    awsConf, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))
    if err != nil {
        return nil, err
    }

    client := s3store.NewClient(awsConf, s3store.ClientOptions{})
    svc := s3store.NewService(client, "bluelink-state")

    logger := core.NewNopLogger()
    return objectstore.LoadStateContainer(
        ctx,
        svc,
        "bluelink-state/",
        logger.Named("state"),
    )
}
```

For GCS and Azure Blob Storage, swap `s3store` for `gcsstore` (`objectstore/stores/gcs`) or `azurestore` (`objectstore/stores/azureblob`) and supply the equivalent client and bucket / container name.

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
