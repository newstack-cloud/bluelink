package azure

import (
	"context"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

type AzureBlobStorageChildResolverSuite struct {
	resolver                includes.ChildResolver
	expectedBlueprintSource string
	client                  *azblob.Client
	suite.Suite
}

func (s *AzureBlobStorageChildResolverSuite) SetupSuite() {
	fileBytes, err := os.ReadFile("../__testdata/azure/data/test-container/azure.test.blueprint.yml")
	s.Require().NoError(err)
	s.expectedBlueprintSource = string(fileBytes)
	// Create a client to handle tear down.
	client, err := createAzuriteBlobStorageClient(
		"test",
		nil,
		createEmptyBlueprintParams(),
	)
	s.Require().NoError(err)
	s.client = client
	// The resolver takes in a factory that creates clients
	// on the fly based on the current environment and user-provided
	// configuration.
	s.resolver = NewResolver(createAzuriteBlobStorageClient)
	err = createTestContainer(client, "test-container")
	s.Require().NoError(err)
	err = uploadTestFile(client, "test-container", "azure.test.blueprint.yml", fileBytes)
	s.Require().NoError(err)
}

func (s *AzureBlobStorageChildResolverSuite) TearDownSuite() {
	ctx := context.Background()
	_, err := s.client.DeleteContainer(ctx, "test-container", &container.DeleteOptions{})
	s.Require().NoError(err)
}

func (s *AzureBlobStorageChildResolverSuite) Test_resolves_blueprint_file() {
	path := "azure.test.blueprint.yml"
	container := "test-container"
	sourceType := "azure/blob-storage"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
				"container": {
					Scalar: &core.ScalarValue{
						StringValue: &container,
					},
				},
			},
		},
	}
	resolvedInfo, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().NoError(err)
	s.Assert().NotNil(resolvedInfo)
	s.Assert().NotNil(resolvedInfo.BlueprintSource)
	s.Assert().Equal(s.expectedBlueprintSource, *resolvedInfo.BlueprintSource)
}

func (s *AzureBlobStorageChildResolverSuite) Test_returns_error_when_path_is_empty() {
	path := ""
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidPath, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: invalid path found, path value must be a string "+
			"for the azure blob storage child resolver, the provided value is either empty or not a string",
		runErr.Err.Error(),
	)
}

func (s *AzureBlobStorageChildResolverSuite) Test_returns_error_when_metadata_is_not_set() {
	path := "azure.test.blueprint.yml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidMetadata, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: invalid metadata provided for the Azure Blob Storage include",
		runErr.Err.Error(),
	)
}

func (s *AzureBlobStorageChildResolverSuite) Test_returns_error_when_bucket_is_missing_from_metadata() {
	path := "azure.test.blueprint.yml"
	sourceType := "azure/blob-storage"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidMetadata, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: missing container field in metadata for the Azure Blob Storage include",
		runErr.Err.Error(),
	)
}

func (s *AzureBlobStorageChildResolverSuite) Test_returns_error_when_file_does_not_exist() {
	path := "azure.missing.blueprint.yml"
	container := "test-container"
	sourceType := "azure/blob-storage"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"sourceType": {
					Scalar: &core.ScalarValue{
						StringValue: &sourceType,
					},
				},
				"container": {
					Scalar: &core.ScalarValue{
						StringValue: &container,
					},
				},
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeBlueprintNotFound, runErr.ReasonCode)
	s.Assert().Equal(
		"[include.test]: blueprint not found at path: azure-blob-storage://test-container/azure.missing.blueprint.yml",
		runErr.Err.Error(),
	)
}

func createTestContainer(client *azblob.Client, name string) error {
	ctx := context.Background()
	_, err := client.CreateContainer(ctx, name, &container.CreateOptions{})
	return err
}

func uploadTestFile(client *azblob.Client, container string, path string, fileBytes []byte) error {
	ctx := context.Background()
	_, err := client.UploadBuffer(ctx, container, path, fileBytes, &azblob.UploadBufferOptions{})
	return err
}

func createAzuriteBlobStorageClient(
	includeName string,
	include *subengine.ResolvedInclude,
	params core.BlueprintParams,
) (*azblob.Client, error) {
	return azblob.NewClientFromConnectionString(
		// This is the default connection string for Blob storage in Azurite,
		// a local Azure Blob Storage emulator.
		// See: https://learn.microsoft.com/en-us/azure/storage/common/storage-use-azurite?tabs=visual-studio%2Cblob-storage#http-connection-strings
		"DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;",
		&azblob.ClientOptions{},
	)
}

func createEmptyBlueprintParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

func TestAzureBlobStorageChildResolverSuite(t *testing.T) {
	suite.Run(t, new(AzureBlobStorageChildResolverSuite))
}
