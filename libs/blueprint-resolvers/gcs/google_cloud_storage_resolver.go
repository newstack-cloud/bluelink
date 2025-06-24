package gcs

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"google.golang.org/api/option"
)

type gcsChildResolver struct {
	endpoint string
}

// NewResolver creates a new instance of a ChildResolver
// that resolves child blueprints from a Google Cloud Storage bucket.
// This relies on fields defined in the include metadata,
// specifically the `bucket` and `region` fields.
// Credentials are loaded via the standard Google Cloud SDK mechanisms.
func NewResolver(endpoint string) includes.ChildResolver {
	return &gcsChildResolver{
		endpoint,
	}
}

func (r *gcsChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {

	err := utils.ValidateInclude(
		include,
		includeName,
		[]string{"bucket"},
		"Google Cloud Storage",
		"google cloud storage",
	)
	if err != nil {
		return nil, err
	}

	path := core.StringValue(include.Path)
	bucket := core.StringValue(include.Metadata.Fields["bucket"])

	client, err := createClient(ctx, r.endpoint)
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}

	bucketHandle := client.Bucket(bucket)
	objectHandle := bucketHandle.Object(path)
	reader, err := objectHandle.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, includes.ErrBlueprintNotFound(
				includeName,
				fmt.Sprintf("gcs://%s/%s", bucket, path),
			)
		}

		return nil, includes.ErrResolveFailure(includeName, err)
	}
	defer reader.Close()

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}
	bodyStr := string(bodyBytes)

	return &includes.ChildBlueprintInfo{
		BlueprintSource: &bodyStr,
	}, nil
}

func createClient(ctx context.Context, endpoint string) (*storage.Client, error) {
	if endpoint == "" {
		return storage.NewClient(ctx)
	}

	return storage.NewClient(ctx, option.WithEndpoint(endpoint))
}
