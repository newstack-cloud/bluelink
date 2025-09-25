package resolve

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

const (
	// S3SourceType is the `sourceType` field for resolving
	// blueprints from AWS S3.
	S3SourceType = "aws/s3"
	// AzureBlobStorageSourceType is the `sourceType` field for resolving
	// blueprints from Azure Blob Storage.
	AzureBlobStorageSourceType = "azure/blob"
	// GoogleCloudStorageSourceType is the `sourceType` field for resolving
	// blueprints from Google Cloud Storage.
	GoogleCloudStorageSourceType = "googlecloud/storage"
	// HTTPSSourceType is the `sourceType` field for resolving
	// blueprints from a public HTTPS URL.
	HTTPSSourceType = "https"
)

// BlueprintDocumentInfo is a type that provides
// information about the location of a source blueprint document.
// This is mostly used to be able to use the blueprint resolvers
// package to load blueprint documents from multiple sources
// without them needing to be child blueprints.
type BlueprintDocumentInfo struct {
	// FileSourceScheme is the file source scheme
	// to determine where the blueprint document is located.
	// This can one of the following:
	//
	// `file`: The blueprint document is located on the local file system of the Deploy Engine server.
	// `s3`: The blueprint document is located in an S3 bucket.
	// `gcs`: The blueprint document is located in a Google Cloud Storage bucket.
	// `azureblob`: The blueprint document is located in an Azure Blob Storage container.
	// `https`: The blueprint document is located via a public HTTPS URL.
	//
	// For remote source authentication, the Deploy Engine server will need to be configured
	// with the appropriate credentials to access the remote source.
	// Authentication is not supported `https` sources.
	//
	// If not provided, the default value of `file` will be used.
	FileSourceScheme string `json:"fileSourceScheme" validate:"oneof=file s3 gcs azureblob https"`
	// Directory where the blueprint document is located.
	// For `file` sources, this must be an absolute path to the directory
	// on the local file system of the Deploy Engine server.
	// An example for a `file` source would be `/path/to/blueprint-directory`.
	// For `s3`, `gcs` and `azureblob` sources, this must be the path to the
	// virtual directory where the first path segment is the bucket/container name
	// and the rest of the path is the path to the virtual directory.
	//
	// An example for a remote object storage source would be
	/// `bucket-name/path/to/blueprint-directory`.
	// For `https` sources, this must be the base URL to the blueprint document
	// excluding the scheme.
	// An example for a `https` source would be `example.com/path/to/blueprint-directory`.
	Directory string `json:"directory" validate:"required"`
	// BlueprintFile is the name of the blueprint file to validate.
	//
	// If not provided, the default value of `project.blueprint.yml` will be used.
	BlueprintFile string `json:"blueprintFile"`
	// BlueprintLocationMetadata is a mapping of string keys to
	// scalar values that hold additional information about the location
	// of the blueprint document.
	// For example, this can be used to specify the region of the bucket/container
	// where the blueprint document is located in a cloud storage service.
	BlueprintLocationMetadata map[string]any `json:"blueprintLocationMetadata"`
}

// BlueprintDocumentInfoToInclude generates an include
// from the given blueprint document info derived from a request
// payload.
func BlueprintDocumentInfoToInclude(
	blueprintDocInfo *BlueprintDocumentInfo,
) (*subengine.ResolvedInclude, error) {
	fileSourceScheme := blueprintDocInfo.FileSourceScheme
	fmt.Println("fileSourceScheme", fileSourceScheme)
	if fileSourceScheme == "file" {
		return createOSFileInclude(blueprintDocInfo), nil
	}

	if fileSourceScheme == "https" {
		return createHTTPSInclude(blueprintDocInfo)
	}

	bucketName, pathPrefix := splitDirectoryForRemoteSource(
		blueprintDocInfo.Directory,
	)
	objectPath := strings.Join(
		[]string{
			pathPrefix,
			blueprintDocInfo.BlueprintFile,
		},
		"/",
	)
	return &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(objectPath),
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
		Metadata: createMetadataForRemoteSource(
			fileSourceScheme,
			bucketName,
			blueprintDocInfo.BlueprintLocationMetadata,
		),
	}, nil
}

func createOSFileInclude(
	blueprintDocInfo *BlueprintDocumentInfo,
) *subengine.ResolvedInclude {
	filePath := createFilePathForOS(blueprintDocInfo)
	return &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(filePath),
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}
}

func createHTTPSInclude(
	blueprintDocInfo *BlueprintDocumentInfo,
) (*subengine.ResolvedInclude, error) {
	urlPath := strings.Join(
		[]string{
			blueprintDocInfo.Directory,
			blueprintDocInfo.BlueprintFile,
		},
		"/",
	)

	locationMetadata := blueprintDocInfo.BlueprintLocationMetadata

	host := ""
	var hasHost bool
	if host, hasHost = locationMetadata["host"].(string); !hasHost {
		return nil, &InvalidLocationMetadataError{
			Reason: "missing host in location metadata which is " +
				"required for the https file source scheme",
		}
	}

	return &subengine.ResolvedInclude{
		Path: core.MappingNodeFromString(
			urlPath,
		),
		Variables: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": core.MappingNodeFromString(
					HTTPSSourceType,
				),
				"host": core.MappingNodeFromString(host),
			},
		},
	}, nil
}

func createMetadataForRemoteSource(
	fileSourceScheme string,
	bucketName string,
	locationMetadata map[string]any,
) *core.MappingNode {
	metadata := &core.MappingNode{
		Fields: map[string]*core.MappingNode{},
	}

	if fileSourceScheme == "s3" {
		metadata.Fields["sourceType"] = core.MappingNodeFromString(S3SourceType)
		metadata.Fields["bucket"] = core.MappingNodeFromString(bucketName)
		if region, ok := locationMetadata["region"].(string); ok {
			metadata.Fields["region"] = core.MappingNodeFromString(region)
		}
	}

	if fileSourceScheme == "gcs" {
		metadata.Fields["sourceType"] = core.MappingNodeFromString(
			GoogleCloudStorageSourceType,
		)
		metadata.Fields["bucket"] = core.MappingNodeFromString(bucketName)
	}

	if fileSourceScheme == "azureblob" {
		metadata.Fields["sourceType"] = core.MappingNodeFromString(
			AzureBlobStorageSourceType,
		)
		metadata.Fields["container"] = core.MappingNodeFromString(bucketName)
	}

	return metadata
}

func splitDirectoryForRemoteSource(dirPath string) (string, string) {
	// The first path segment is the bucket/container name
	// and the rest of the path is the path to the virtual directory.
	// For example, `bucket-name/path/to/blueprint-directory` would be split into
	// `bucket-name` and `path/to/blueprint-directory`.
	trimmedDirPath := strings.TrimPrefix(dirPath, "/")
	pathSegments := strings.SplitN(trimmedDirPath, "/", 2)
	if len(pathSegments) == 1 {
		return pathSegments[0], ""
	}
	return pathSegments[0], pathSegments[1]
}

func createFilePathForOS(
	blueprintDocInfo *BlueprintDocumentInfo,
) string {
	// Normalise the path for the current operating system,
	// even if the user provides a separator for a different OS.
	dirPath := blueprintDocInfo.Directory
	toReplace := "\\"
	if os.PathSeparator == '\\' {
		toReplace = "/"
	}
	normalisedDirPath := strings.ReplaceAll(dirPath, toReplace, string(os.PathSeparator))

	return path.Join(
		normalisedDirPath,
		blueprintDocInfo.BlueprintFile,
	)
}
