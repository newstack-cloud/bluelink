package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/newstack-cloud/bluelink/libs/blueprint-resolvers/utils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

// AzureBlobClientFactory is a function that creates an azblob.Client.
// It takes the include name, include metadata, and blueprint parameters
// that may contain configuration for the client.
type AzureBlobClientFactory func(
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*azblob.Client, error)

type azureBlobStorageChildResolver struct {
	createClient AzureBlobClientFactory
}

// NewResolver creates a new instance of a ChildResolver
// that resolves child blueprints from an Azure Blob Storage container.
// This relies on fields defined in the include metadata,
// specifically the `container` field.
// clientFactory is a function that creates an azblob.Client,
// this will be called on each Resolve request.
// If you provide nil for the clientFactory, one will be created using
// the standard Azure SDK mechanisms, sourcing credentials from
// the current environment.
// The default client factory will need to source the azure storage account name
// to build the service URL.
// The default client factory expects either the "storageAccountName" field
// to be present in the include metadata, the "azureStorageAccountName" context variable
// to be set, or the "AZURE_STORAGE_ACCOUNT_NAME" environment variable to be set.
func NewResolver(clientFactory AzureBlobClientFactory) includes.ChildResolver {
	finalClientFactory := clientFactory
	if finalClientFactory == nil {
		finalClientFactory = createAzureBlobClientFromEnv
	}
	return &azureBlobStorageChildResolver{
		createClient: clientFactory,
	}
}

func (r *azureBlobStorageChildResolver) Resolve(
	ctx context.Context,
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*includes.ChildBlueprintInfo, error) {

	err := utils.ValidateInclude(
		include,
		includeName,
		[]string{"container"},
		"Azure Blob Storage",
		"azure blob storage",
	)
	if err != nil {
		return nil, err
	}

	path := core.StringValue(include.Path)
	container := core.StringValue(include.Metadata.Fields["container"])

	client, err := r.createClient(includeName, include, params)
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}

	stream, err := client.DownloadStream(ctx, container, path, nil)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == 404 {
			return nil, includes.ErrBlueprintNotFound(
				includeName,
				fmt.Sprintf("azure-blob-storage://%s/%s", container, path),
			)
		}

		return nil, includes.ErrResolveFailure(includeName, err)
	}

	downloadedData := bytes.Buffer{}
	retryReader := stream.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(retryReader)
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}

	downloadedDataStr := downloadedData.String()
	return &includes.ChildBlueprintInfo{
		BlueprintSource: &downloadedDataStr,
	}, nil
}

func createAzureBlobClientFromEnv(
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*azblob.Client, error) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}

	storageAccountName := resolveStorageAccountName(
		include.Metadata.Fields,
		params,
	)
	if storageAccountName == "" {
		return nil, includes.ErrResolveFailure(
			includeName,
			errors.New("missing azure storage account name, see resolver docs for details"),
		)
	}

	client, err := azblob.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName),
		credential,
		&azblob.ClientOptions{},
	)

	if err != nil {
		return nil, includes.ErrResolveFailure(includeName, err)
	}

	return client, nil
}

func resolveStorageAccountName(
	fields map[string]*core.MappingNode,
	params core.BlueprintParams,
) string {
	if fields["storageAccountName"] != nil {
		return core.StringValue(fields["storageAccountName"])
	}

	fromContextVars := params.ContextVariable("azureStorageAccountName")
	if fromContextVars != nil {
		return core.StringValueFromScalar(fromContextVars)
	}

	return os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
}
