package integratedtestsuites

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

type ServerV1WithPGStorageEngineTestSuite struct {
	suite.Suite
	server  *httptest.Server
	client  *http.Client
	cleanup func()
}

func (s *ServerV1WithPGStorageEngineTestSuite) SetupSuite() {
	config, err := core.LoadConfig()
	s.Require().NoError(err, "error loading config")

	config.State.StorageEngine = "postgres"
	pluginPath, logFileRootDir, err := testPluginPaths()
	s.Require().NoError(err, "error getting plugin path")
	config.PluginsV1.PluginPath = pluginPath
	config.PluginsV1.LogFileRootDir = logFileRootDir

	router := mux.NewRouter().PathPrefix("/v1").Subrouter()

	// Listen on port 43045 for the plugin service to not conflict
	// with the default plugin service TCP port.
	pluginServiceListener, err := net.Listen("tcp", ":43045")
	s.Require().NoError(err, "error creating plugin service listener")

	_, cleanup, err := enginev1.Setup(router, &config, pluginServiceListener)
	s.cleanup = cleanup
	s.Require().NoError(err, "error setting up Deploy Engine API server")

	s.server = httptest.NewServer(router)
	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (s *ServerV1WithPGStorageEngineTestSuite) Test_server_endpoint_request() {
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

func (s *ServerV1WithPGStorageEngineTestSuite) TearDownSuite() {
	if s.cleanup != nil {
		s.cleanup()
	}
	s.server.Close()
}

func TestServerV1WithPGStorageEngineTestSuite(t *testing.T) {
	suite.Run(t, new(ServerV1WithPGStorageEngineTestSuite))
}
