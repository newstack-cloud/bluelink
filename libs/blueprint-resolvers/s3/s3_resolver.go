package s3

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

type s3ChildResolver struct {
	endpoint     string
	usePathStyle bool
}

// NewResolver creates a new instance of a ChildResolver
// that resolves child blueprints from an S3 bucket.
// This relies on fields defined in the include metadata,
// specifically the `bucket` and `region` fields.
// Credentials are loaded via the standard AWS SDK mechanisms, looking for
// environment variables, credential files, or IAM roles.
func NewResolver(endpoint string, usePathStyle bool) includes.ChildResolver {
	return &s3ChildResolver{
		endpoint,
		usePathStyle,
	}
}

func (r *s3ChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {

	err := utils.ValidateInclude(include, includeName, []string{"bucket"}, "S3", "s3")
	if err != nil {
		return nil, err
	}

	path := core.StringValue(include.Path)
	bucket := core.StringValue(include.Metadata.Fields["bucket"])
	region := core.StringValue(include.Metadata.Fields["region"])

	conf, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}

	client := newFromConfig(conf, r.endpoint, r.usePathStyle)
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &path,
	})
	if err != nil {
		var noSuchKeyErr *types.NoSuchKey
		if errors.As(err, &noSuchKeyErr) {
			return nil, includes.ErrBlueprintNotFound(includeName, fmt.Sprintf("s3://%s/%s", bucket, path))
		}

		return nil, includes.ErrResolveFailure(includeName, err)
	}

	bodyBytes, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}
	bodyStr := string(bodyBytes)

	return &includes.ChildBlueprintInfo{
		BlueprintSource: &bodyStr,
	}, nil
}

func newFromConfig(conf aws.Config, endpoint string, usePathStyle bool) *s3.Client {
	if endpoint == "" {
		return s3.NewFromConfig(conf, func(opts *s3.Options) {
			opts.UsePathStyle = usePathStyle
		})
	}

	return s3.NewFromConfig(conf, func(opts *s3.Options) {
		opts.UsePathStyle = usePathStyle
		opts.BaseEndpoint = aws.String(endpoint)
	})
}
