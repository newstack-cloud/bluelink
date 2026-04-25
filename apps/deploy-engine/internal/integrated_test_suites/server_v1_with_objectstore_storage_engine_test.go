package integratedtestsuites

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/auth"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/helpersv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1/validationv1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/stretchr/testify/suite"
)

// ServerV1WithObjectstoreStorageEngineTestSuite exercises the same
// validation-endpoint golden path as the memfile and postgres suites,
// but against the objectstore state backend (S3 provider) wired up to
// the LocalStack emulator brought up by docker-compose.test-deps.yml.
// Proves deploy-engine's state-setup switch and the objectstore Service
// adapter work end-to-end through the HTTP API.
type ServerV1WithObjectstoreStorageEngineTestSuite struct {
	suite.Suite
	server    *httptest.Server
	client    *http.Client
	cleanup   func()
	s3Client  *s3sdk.Client
	s3Bucket  string
	s3Region  string
	s3EndPt   string
}

func (s *ServerV1WithObjectstoreStorageEngineTestSuite) SetupSuite() {
	config, err := core.LoadConfig()
	s.Require().NoError(err, "error loading config")

	s.Require().Equal(
		"s3", config.State.ObjectStore.Provider,
		"objectstore provider env var must be set to s3 by .env.test",
	)
	s.Require().NotEmpty(
		config.State.ObjectStore.S3.Bucket,
		"objectstore s3 bucket env var must be set by .env.test",
	)

	config.State.StorageEngine = "objectstore"
	pluginPath, logFileRootDir, err := testPluginPaths()
	s.Require().NoError(err, "error getting plugin path")
	config.PluginsV1.PluginPath = pluginPath
	config.PluginsV1.LogFileRootDir = logFileRootDir

	s.s3Bucket = config.State.ObjectStore.S3.Bucket
	s.s3Region = config.State.ObjectStore.S3.Region
	s.s3EndPt = config.State.ObjectStore.S3.Endpoint
	s.ensureTestBucket(config.State.ObjectStore.S3)

	router := mux.NewRouter().PathPrefix("/v1").Subrouter()

	// Listen on port 43046 for the plugin service to not conflict with
	// the default port or the ports used by the other suites.
	pluginServiceListener, err := net.Listen("tcp", ":43046")
	s.Require().NoError(err, "error creating plugin service listener")

	_, cleanup, err := enginev1.Setup(router, &config, pluginServiceListener)
	s.cleanup = cleanup
	s.Require().NoError(err, "error setting up Deploy Engine API server")

	s.server = httptest.NewServer(router)
	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (s *ServerV1WithObjectstoreStorageEngineTestSuite) Test_server_endpoint_request() {
	testBlueprintDir, err := testBlueprintDirectory()
	s.Require().NoError(err, "error getting test blueprint dir")
	bodyBytes, err := json.Marshal(
		&validationv1.CreateValidationRequestPayload{
			BlueprintDocumentInfo: resolve.BlueprintDocumentInfo{
				FileSourceScheme: "file",
				Directory:        testBlueprintDir,
				BlueprintFile:    "test-blueprint.yml",
			},
		},
	)
	s.Require().NoError(err, "error marshalling request payload")
	bodyReader := bytes.NewReader(bodyBytes)
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/validations", s.server.URL),
		bodyReader,
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(auth.BluelinkAPIKeyHeaderName, "test-api-key")
	s.Require().NoError(err, "error creating request")

	response, err := s.client.Do(req)
	s.Require().NoError(err, "error making request")
	defer response.Body.Close()
	s.Assert().Equal(http.StatusAccepted, response.StatusCode, "unexpected status code")

	wrappedResponse := &helpersv1.AsyncOperationResponse[*manage.BlueprintValidation]{}
	respBytes, err := io.ReadAll(response.Body)
	s.Require().NoError(err, "error reading response body")

	err = json.Unmarshal(respBytes, wrappedResponse)
	s.Require().NoError(err, "error unmarshalling response body")

	blueprintValidation := wrappedResponse.Data
	s.Assert().Equal(
		fmt.Sprintf(
			"file://%s/test-blueprint.yml",
			testBlueprintDir,
		),
		blueprintValidation.BlueprintLocation,
	)
	s.Assert().Equal(
		manage.BlueprintValidationStatusStarting,
		blueprintValidation.Status,
	)
	s.Assert().Greater(
		blueprintValidation.Created,
		int64(0),
		"created timestamp should be greater than 0",
	)
	s.Assert().True(
		len(blueprintValidation.ID) > 0,
	)
}

func (s *ServerV1WithObjectstoreStorageEngineTestSuite) TearDownSuite() {
	if s.cleanup != nil {
		s.cleanup()
	}
	if s.server != nil {
		s.server.Close()
	}
	s.emptyAndDeleteTestBucket()
}

// ensureTestBucket creates the LocalStack bucket the suite writes state
// into. Constructs an S3 client mirroring the deploy-engine's own
// builder so credential / addressing behaviour matches production wiring.
func (s *ServerV1WithObjectstoreStorageEngineTestSuite) ensureTestBucket(cfg core.ObjectStoreS3Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conf, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	s.Require().NoError(err, "error loading aws config for test bucket setup")

	s.s3Client = s3sdk.NewFromConfig(conf, func(o *s3sdk.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		if cfg.Endpoint != "" {
			o.BaseEndpoint = awssdk.String(cfg.Endpoint)
		}
	})

	_, err = s.s3Client.CreateBucket(ctx, &s3sdk.CreateBucketInput{
		Bucket: awssdk.String(cfg.Bucket),
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(cfg.Region),
		},
	})
	if err != nil {
		var already *s3types.BucketAlreadyOwnedByYou
		var exists *s3types.BucketAlreadyExists
		if errors.As(err, &already) || errors.As(err, &exists) {
			return
		}
		s.Require().NoError(err, "error creating test bucket")
	}
}

// emptyAndDeleteTestBucket cleans up after the suite so repeated local
// runs against a persistent LocalStack container start from a clean
// slate. Retries list+delete a handful of times to paper over the small
// window where async validation writes trickle in after the server
// closes. Best-effort: failures are logged but don't fail the teardown.
func (s *ServerV1WithObjectstoreStorageEngineTestSuite) emptyAndDeleteTestBucket() {
	if s.s3Client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const maxAttempts = 5
	for attempt := range maxAttempts {
		s.drainBucketObjects(ctx)
		_, err := s.s3Client.DeleteBucket(ctx, &s3sdk.DeleteBucketInput{
			Bucket: awssdk.String(s.s3Bucket),
		})
		if err == nil {
			return
		}
		if attempt == maxAttempts-1 {
			s.T().Logf("delete bucket %s: %v", s.s3Bucket, err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *ServerV1WithObjectstoreStorageEngineTestSuite) drainBucketObjects(ctx context.Context) {
	paginator := s3sdk.NewListObjectsV2Paginator(s.s3Client, &s3sdk.ListObjectsV2Input{
		Bucket: awssdk.String(s.s3Bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.T().Logf("list bucket %s for teardown: %v", s.s3Bucket, err)
			return
		}
		for _, obj := range page.Contents {
			_, _ = s.s3Client.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
				Bucket: awssdk.String(s.s3Bucket),
				Key:    obj.Key,
			})
		}
	}
}

func TestServerV1WithObjectstoreStorageEngineTestSuite(t *testing.T) {
	suite.Run(t, new(ServerV1WithObjectstoreStorageEngineTestSuite))
}
