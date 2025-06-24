# blueprint resolvers

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint-resolvers&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint-resolvers)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint-resolvers&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint-resolvers)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint-resolvers&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint-resolvers)

A library that provides a collection of blueprint framework `ChildResolver` implementations for sourcing child blueprints referenced through the use of the `include` property in a parent blueprint.

## Implementations

- File system - Resolves child blueprints from a provided file system.
- S3 - Resolves child blueprints from an S3 bucket.
- Google Cloud Storage - Resolves child blueprints from a Google Cloud Storage bucket.
- Azure Blob Storage - Resolves child blueprints from Azure Blob Storage.
- HTTPS - Resolves child blueprints from a public URL over HTTPS.

## Usage

```go
import (
    "context"

	"github.com/spf13/afero"
    "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/router"
    "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/fs"
    "github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/s3"
    "github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
    "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

func main() {
    osfs := afero.NewOsFs()
    fsResolver := fs.NewResolver(osfs)

    // An empty endpoint will lead the resolver to use the default endpoint,
    // credentials will be configured using the default AWS SDK
    // credential chain that will pick up environment variables,
    // shared credentials file, or IAM role.
    s3Resolver := s3.NewResolver( /* endpoint */ "", /* usePathStyle */ false)

    // Create a new router that allows for multiple resolvers
    // to be used to resolve child blueprints based on a `sourceType`
    // property in the metadata of an include.
    r := router.NewResolver(
        fsResolver,
        router.WithRoute("aws/s3", s3Resolver),
    )

    // Resolve a child blueprint
    path := "core-infra/blueprints/childBlueprint1.yaml"
    sourceType := "aws/s3"
    bucket := "my-bucket"
    child, err := r.Resolve(
        context.Background(),
        "childBlueprint1",
        &subengine.ResolvedInclude{
            Path: &core.MappingNode{
                Literal: &core.ScalarValue{
                    StringValue: &path,
                },
            },
            Metadata: &core.MappingNode{
                Fields: map[string]*core.MappingNode{
                    "sourceType": {
                        Literal: &core.ScalarValue{
                            StringValue: &sourceType,
                        },
                    },
                    "bucket": {
                        Literal: &core.ScalarValue{
                            StringValue: &bucket,
                        },
                    },
                },
            },
        },
    )
    if err != nil {
        // Handle error
    }

    // Process the resolved child blueprint ...
}
```

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
