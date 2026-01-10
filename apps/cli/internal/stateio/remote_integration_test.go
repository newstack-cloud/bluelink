package stateio

import (
	"context"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/stretchr/testify/suite"
)

// S3 Integration Tests

type S3RemoteIntegrationSuite struct {
	suite.Suite
	expectedInstancesJSON []byte
}

func TestS3RemoteIntegrationSuite(t *testing.T) {
	suite.Run(t, new(S3RemoteIntegrationSuite))
}

func (s *S3RemoteIntegrationSuite) SetupTest() {
	var err error
	s.expectedInstancesJSON, err = os.ReadFile("__testdata/s3/instances.json")
	s.Require().NoError(err)
}

func (s *S3RemoteIntegrationSuite) Test_downloads_instances_json_from_s3() {
	opts := &RemoteDownloadOptions{
		S3Endpoint:     "http://localhost:4580",
		S3UsePathStyle: true,
	}
	data, err := DownloadRemoteFile(context.Background(), "s3://test-bucket/instances.json", opts)
	s.Require().NoError(err)
	s.Equal(s.expectedInstancesJSON, data)
}

func (s *S3RemoteIntegrationSuite) Test_returns_not_found_error_for_missing_s3_file() {
	opts := &RemoteDownloadOptions{
		S3Endpoint:     "http://localhost:4580",
		S3UsePathStyle: true,
	}
	_, err := DownloadRemoteFile(context.Background(), "s3://test-bucket/nonexistent.json", opts)
	s.Require().Error(err)

	importErr, ok := err.(*ImportError)
	s.True(ok)
	s.Equal(ErrCodeFileNotFound, importErr.Code)
}

// GCS Integration Tests

type GCSRemoteIntegrationSuite struct {
	suite.Suite
	expectedInstancesJSON []byte
}

func TestGCSRemoteIntegrationSuite(t *testing.T) {
	suite.Run(t, new(GCSRemoteIntegrationSuite))
}

func (s *GCSRemoteIntegrationSuite) SetupTest() {
	var err error
	s.expectedInstancesJSON, err = os.ReadFile("__testdata/gcs/test-bucket/instances.json")
	s.Require().NoError(err)
}

func (s *GCSRemoteIntegrationSuite) Test_downloads_instances_json_from_gcs() {
	opts := &RemoteDownloadOptions{
		GCSEndpoint: "http://localhost:8185/storage/v1/",
	}
	data, err := DownloadRemoteFile(context.Background(), "gcs://test-bucket/instances.json", opts)
	s.Require().NoError(err)
	s.Equal(s.expectedInstancesJSON, data)
}

func (s *GCSRemoteIntegrationSuite) Test_returns_not_found_error_for_missing_gcs_file() {
	opts := &RemoteDownloadOptions{
		GCSEndpoint: "http://localhost:8185/storage/v1/",
	}
	_, err := DownloadRemoteFile(context.Background(), "gcs://test-bucket/nonexistent.json", opts)
	s.Require().Error(err)

	importErr, ok := err.(*ImportError)
	s.True(ok)
	s.Equal(ErrCodeFileNotFound, importErr.Code)
}

// Azure Blob Integration Tests

type AzureBlobRemoteIntegrationSuite struct {
	suite.Suite
	client                *azblob.Client
	expectedInstancesJSON []byte
}

func TestAzureBlobRemoteIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AzureBlobRemoteIntegrationSuite))
}

func (s *AzureBlobRemoteIntegrationSuite) SetupSuite() {
	connectionString := "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10003/devstoreaccount1;"

	var err error
	s.client, err = azblob.NewClientFromConnectionString(connectionString, nil)
	s.Require().NoError(err)

	// Create test container
	ctx := context.Background()
	_, err = s.client.CreateContainer(ctx, "test-container", nil)
	if err != nil {
		// Container might already exist, ignore error
	}

	// Upload test file
	s.expectedInstancesJSON, err = os.ReadFile("__testdata/azure/instances.json")
	s.Require().NoError(err)
	_, err = s.client.UploadBuffer(ctx, "test-container", "instances.json", s.expectedInstancesJSON, nil)
	s.Require().NoError(err)
}

func (s *AzureBlobRemoteIntegrationSuite) TearDownSuite() {
	ctx := context.Background()
	_, _ = s.client.DeleteContainer(ctx, "test-container", nil)
}

func (s *AzureBlobRemoteIntegrationSuite) Test_downloads_instances_json_from_azure_blob() {
	opts := &RemoteDownloadOptions{
		AzureConnectionString: "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10003/devstoreaccount1;",
	}
	data, err := DownloadRemoteFile(context.Background(), "azureblob://test-container/instances.json", opts)
	s.Require().NoError(err)
	s.Equal(s.expectedInstancesJSON, data)
}

func (s *AzureBlobRemoteIntegrationSuite) Test_returns_not_found_error_for_missing_azure_blob() {
	opts := &RemoteDownloadOptions{
		AzureConnectionString: "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10003/devstoreaccount1;",
	}
	_, err := DownloadRemoteFile(context.Background(), "azureblob://test-container/nonexistent.json", opts)
	s.Require().Error(err)

	importErr, ok := err.(*ImportError)
	s.True(ok)
	s.Equal(ErrCodeFileNotFound, importErr.Code)
}
