# deploy engine client library

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine-client&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine-client)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine-client&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine-client)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-deploy-engine-client&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-deploy-engine-client)

The deploy engine client library for Go that provides a client for the deploy engine API.

## Installation

```bash
go get github.com/newstack-cloud/bluelink/libs/deploy-engine-client
```

## Usage

### HTTP request

```go
package main

import (
    "github.com/newstack-cloud/bluelink/libs/deploy-engine-client"
)

func main() {
    // Create a new client
    client := deployengine.NewClient(
        // Configure the client with a custom endpoint and credentials.
        deployengine.WithClientEndpoint("https://deploy.my-service.io"),
        deployengine.WithClientBluelinkSigv1KeyPair(
            &deployengine.BluelinkSignatureKeyPair{
                KeyID: os.Getenv("DEPLOY_ENGINE_KEY_ID"),
                SecretKey: os.Getenv("DEPLOY_ENGINE_SECRET_KEY"),
            },
        ),
    )

    bpValidation, err := client.GetBlueprintValidation(
        context.Background(),
        "c6f69b85-a6e8-4374-8c6f-8b4539d1142b",
    )
    // handle error and use the retrieved blueprint validation ...
}
```

### Stream events

```go
package main

import (
    "github.com/newstack-cloud/bluelink/libs/deploy-engine-client"
)

func main() {
    // Set up client ...
    streamTo := make(chan types.BlueprintValidationEvent)
    errChan := make(chan error)
    err := client.StreamBlueprintValidationEvents(
        context.Background(),
        "c6f69b85-a6e8-4374-8c6f-8b4539d1142b",
        streamTo,
        errChan,
    )
    if err != nil {
        // handle error
    }

    // You'll most likely want to spawn a goroutine
    // to collect and handle events from the stream.
    for {
        select {
        case event, ok := <-streamTo:
            if !ok {
                // channel closed
                return
            }
            // handle event
        case err := <-errChan:
            // handle error
            return
        }
    }
}
```

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
