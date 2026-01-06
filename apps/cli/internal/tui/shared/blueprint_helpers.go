package shared

import (
	"path"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
)

// BlueprintSourceFromPath determines the blueprint source type from a file path.
// It checks for URL schemes (https://, s3://, gcs://, azureblob://) and returns
// the appropriate source constant, defaulting to file source for local paths.
func BlueprintSourceFromPath(blueprintFile string) string {
	if strings.HasPrefix(blueprintFile, "https://") {
		return consts.BlueprintSourceHTTPS
	}
	if strings.HasPrefix(blueprintFile, "s3://") {
		return consts.BlueprintSourceS3
	}
	if strings.HasPrefix(blueprintFile, "gcs://") {
		return consts.BlueprintSourceGCS
	}
	if strings.HasPrefix(blueprintFile, "azureblob://") {
		return consts.BlueprintSourceAzureBlob
	}
	return consts.BlueprintSourceFile
}

// StripObjectStorageScheme removes the scheme prefix from an object storage path.
// For example, "s3://bucket/file.yaml" becomes "bucket/file.yaml".
// If the path doesn't have the expected scheme prefix, it's returned unchanged.
func StripObjectStorageScheme(blueprintFile, scheme string) string {
	prefix := scheme + "://"
	if strings.HasPrefix(blueprintFile, prefix) {
		return strings.TrimPrefix(blueprintFile, prefix)
	}
	return blueprintFile
}

// BuildObjectStorageDocumentInfo creates a BlueprintDocumentInfo for object storage sources.
// It strips the scheme prefix before extracting directory and file components,
// since path.Dir doesn't handle scheme-prefixed paths correctly.
func BuildObjectStorageDocumentInfo(blueprintFile, scheme string) types.BlueprintDocumentInfo {
	pathWithoutScheme := StripObjectStorageScheme(blueprintFile, scheme)
	return types.BlueprintDocumentInfo{
		FileSourceScheme: scheme,
		Directory:        path.Dir(pathWithoutScheme),
		BlueprintFile:    path.Base(pathWithoutScheme),
	}
}
