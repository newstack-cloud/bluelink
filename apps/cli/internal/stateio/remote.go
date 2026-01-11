package stateio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/shared"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	"google.golang.org/api/option"
)

// RemoteDownloadOptions contains options for downloading files from remote storage.
type RemoteDownloadOptions struct {
	// S3Endpoint overrides the default S3 endpoint (useful for testing with LocalStack).
	S3Endpoint string
	// S3UsePathStyle enables path-style addressing for S3 (required for LocalStack).
	S3UsePathStyle bool
	// GCSEndpoint overrides the default GCS endpoint (useful for testing with fake-gcs-server).
	GCSEndpoint string
	// AzureConnectionString is the connection string for Azure Blob Storage.
	// If empty, DefaultAzureCredential will be used.
	AzureConnectionString string
}

// DownloadRemoteFile downloads a file from a remote storage location.
// Supports s3://, gcs://, and azureblob:// URL schemes.
func DownloadRemoteFile(ctx context.Context, filePath string, opts *RemoteDownloadOptions) ([]byte, error) {
	if opts == nil {
		opts = &RemoteDownloadOptions{}
	}

	source := shared.BlueprintSourceFromPath(filePath)
	switch source {
	case consts.BlueprintSourceS3:
		return downloadFromS3(ctx, filePath, opts)
	case consts.BlueprintSourceGCS:
		return downloadFromGCS(ctx, filePath, opts)
	case consts.BlueprintSourceAzureBlob:
		return downloadFromAzureBlob(ctx, filePath, opts)
	default:
		return nil, &ImportError{
			Code:    ErrCodeFileNotFound,
			Message: fmt.Sprintf("unsupported remote source type for path: %s", filePath),
		}
	}
}

// IsRemoteFile returns true if the file path refers to a remote storage location.
func IsRemoteFile(filePath string) bool {
	source := shared.BlueprintSourceFromPath(filePath)
	return source == consts.BlueprintSourceS3 ||
		source == consts.BlueprintSourceGCS ||
		source == consts.BlueprintSourceAzureBlob
}

func downloadFromS3(ctx context.Context, filePath string, opts *RemoteDownloadOptions) ([]byte, error) {
	pathWithoutScheme := shared.StripObjectStorageScheme(filePath, "s3")
	bucket, key, err := parseS3Path(pathWithoutScheme)
	if err != nil {
		return nil, err
	}

	configOpts := []func(*awsconfig.LoadOptions) error{}
	if opts.S3Endpoint != "" {
		// When using a custom endpoint (e.g., LocalStack), set a default region
		configOpts = append(configOpts, awsconfig.WithRegion("us-east-1"))
	}

	conf, err := awsconfig.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to load AWS config",
			Err:     err,
		}
	}

	client := createS3Client(conf, opts.S3Endpoint, opts.S3UsePathStyle)
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		var noSuchKeyErr *s3types.NoSuchKey
		if errors.As(err, &noSuchKeyErr) {
			return nil, &ImportError{
				Code:    ErrCodeFileNotFound,
				Message: fmt.Sprintf("file not found: s3://%s/%s", bucket, key),
			}
		}
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to download from S3",
			Err:     err,
		}
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}

func createS3Client(conf aws.Config, endpoint string, usePathStyle bool) *s3.Client {
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

func parseS3Path(pathWithoutScheme string) (bucket, key string, err error) {
	parts := strings.SplitN(pathWithoutScheme, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: fmt.Sprintf("invalid S3 path: %s", pathWithoutScheme),
		}
	}
	return parts[0], parts[1], nil
}

func downloadFromGCS(ctx context.Context, filePath string, opts *RemoteDownloadOptions) ([]byte, error) {
	pathWithoutScheme := shared.StripObjectStorageScheme(filePath, "gcs")
	bucket, object, err := parseGCSPath(pathWithoutScheme)
	if err != nil {
		return nil, err
	}

	client, err := createGCSClient(ctx, opts.GCSEndpoint)
	if err != nil {
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to create GCS client",
			Err:     err,
		}
	}
	defer client.Close()

	reader, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, &ImportError{
				Code:    ErrCodeFileNotFound,
				Message: fmt.Sprintf("file not found: gcs://%s/%s", bucket, object),
			}
		}
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to download from GCS",
			Err:     err,
		}
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func createGCSClient(ctx context.Context, endpoint string) (*storage.Client, error) {
	if endpoint == "" {
		return storage.NewClient(ctx)
	}
	return storage.NewClient(ctx, option.WithEndpoint(endpoint))
}

func parseGCSPath(pathWithoutScheme string) (bucket, object string, err error) {
	parts := strings.SplitN(pathWithoutScheme, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: fmt.Sprintf("invalid GCS path: %s", pathWithoutScheme),
		}
	}
	return parts[0], parts[1], nil
}

func downloadFromAzureBlob(ctx context.Context, filePath string, opts *RemoteDownloadOptions) ([]byte, error) {
	pathWithoutScheme := shared.StripObjectStorageScheme(filePath, "azureblob")
	container, blob, err := parseAzureBlobPath(pathWithoutScheme)
	if err != nil {
		return nil, err
	}

	client, err := createAzureBlobClient(opts.AzureConnectionString)
	if err != nil {
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to create Azure Blob client",
			Err:     err,
		}
	}

	stream, err := client.DownloadStream(ctx, container, blob, nil)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == 404 {
			return nil, &ImportError{
				Code:    ErrCodeFileNotFound,
				Message: fmt.Sprintf("file not found: azureblob://%s/%s", container, blob),
			}
		}
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to download from Azure Blob Storage",
			Err:     err,
		}
	}

	downloadedData := bytes.Buffer{}
	retryReader := stream.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(retryReader)
	if err != nil {
		return nil, &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: "failed to read Azure Blob data",
			Err:     err,
		}
	}

	return downloadedData.Bytes(), nil
}

func createAzureBlobClient(connectionString string) (*azblob.Client, error) {
	if connectionString != "" {
		return azblob.NewClientFromConnectionString(connectionString, nil)
	}

	storageAccountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	if storageAccountName == "" {
		return nil, errors.New("AZURE_STORAGE_ACCOUNT_NAME environment variable not set")
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	return azblob.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName),
		credential,
		nil,
	)
}

func parseAzureBlobPath(pathWithoutScheme string) (container, blob string, err error) {
	parts := strings.SplitN(pathWithoutScheme, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", &ImportError{
			Code:    ErrCodeRemoteAccessFail,
			Message: fmt.Sprintf("invalid Azure Blob path: %s", pathWithoutScheme),
		}
	}
	return parts[0], parts[1], nil
}

// RemoteUploadOptions contains options for uploading files to remote storage.
type RemoteUploadOptions struct {
	// S3Endpoint overrides the default S3 endpoint (useful for testing with LocalStack).
	S3Endpoint string
	// S3UsePathStyle enables path-style addressing for S3 (required for LocalStack).
	S3UsePathStyle bool
	// GCSEndpoint overrides the default GCS endpoint (useful for testing with fake-gcs-server).
	GCSEndpoint string
	// AzureConnectionString is the connection string for Azure Blob Storage.
	// If empty, DefaultAzureCredential will be used.
	AzureConnectionString string
}

// UploadRemoteFile uploads a file to a remote storage location.
// Supports s3://, gcs://, and azureblob:// URL schemes.
func UploadRemoteFile(ctx context.Context, filePath string, data []byte, opts *RemoteUploadOptions) error {
	if opts == nil {
		opts = &RemoteUploadOptions{}
	}

	source := shared.BlueprintSourceFromPath(filePath)
	switch source {
	case consts.BlueprintSourceS3:
		return uploadToS3(ctx, filePath, data, opts)
	case consts.BlueprintSourceGCS:
		return uploadToGCS(ctx, filePath, data, opts)
	case consts.BlueprintSourceAzureBlob:
		return uploadToAzureBlob(ctx, filePath, data, opts)
	default:
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: fmt.Sprintf("unsupported remote source type for path: %s", filePath),
		}
	}
}

func uploadToS3(ctx context.Context, filePath string, data []byte, opts *RemoteUploadOptions) error {
	pathWithoutScheme := shared.StripObjectStorageScheme(filePath, "s3")
	bucket, key, err := parseS3Path(pathWithoutScheme)
	if err != nil {
		return err
	}

	configOpts := []func(*awsconfig.LoadOptions) error{}
	if opts.S3Endpoint != "" {
		configOpts = append(configOpts, awsconfig.WithRegion("us-east-1"))
	}

	conf, err := awsconfig.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to load AWS config",
			Err:     err,
		}
	}

	client := createS3Client(conf, opts.S3Endpoint, opts.S3UsePathStyle)
	contentType := "application/json"
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to upload to S3",
			Err:     err,
		}
	}

	return nil
}

func uploadToGCS(ctx context.Context, filePath string, data []byte, opts *RemoteUploadOptions) error {
	pathWithoutScheme := shared.StripObjectStorageScheme(filePath, "gcs")
	bucket, object, err := parseGCSPath(pathWithoutScheme)
	if err != nil {
		return err
	}

	client, err := createGCSClient(ctx, opts.GCSEndpoint)
	if err != nil {
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to create GCS client",
			Err:     err,
		}
	}
	defer client.Close()

	writer := client.Bucket(bucket).Object(object).NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to write to GCS",
			Err:     err,
		}
	}

	if err := writer.Close(); err != nil {
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to close GCS writer",
			Err:     err,
		}
	}

	return nil
}

func uploadToAzureBlob(ctx context.Context, filePath string, data []byte, opts *RemoteUploadOptions) error {
	pathWithoutScheme := shared.StripObjectStorageScheme(filePath, "azureblob")
	container, blobPath, err := parseAzureBlobPath(pathWithoutScheme)
	if err != nil {
		return err
	}

	client, err := createAzureBlobClient(opts.AzureConnectionString)
	if err != nil {
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to create Azure Blob client",
			Err:     err,
		}
	}

	_, err = client.UploadBuffer(ctx, container, blobPath, data, nil)
	if err != nil {
		return &ExportError{
			Code:    ErrCodeRemoteUploadFailed,
			Message: "failed to upload to Azure Blob Storage",
			Err:     err,
		}
	}

	return nil
}
