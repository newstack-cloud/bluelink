package resolverhttps

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/stretchr/testify/suite"
)

type HTTPSChildResolverSuite struct {
	server                  *httptest.Server
	client                  *http.Client
	resolver                includes.ChildResolver
	expectedBlueprintSource string
	suite.Suite
}

func (s *HTTPSChildResolverSuite) SetupTest() {
	router := mux.NewRouter()
	router.PathPrefix("/public/").Handler(
		http.StripPrefix(
			"/public/",
			http.FileServer(http.Dir("__testdata")),
		),
	)
	router.PathPrefix("/private/").Handler(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("{\"message\":\"Unauthorized\"}"))
			},
		),
	)
	server := httptest.NewTLSServer(
		router,
	)
	s.server = server
	s.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	expectedBytes, err := os.ReadFile("__testdata/https.test.blueprint.yml")
	s.Require().NoError(err)
	s.expectedBlueprintSource = string(expectedBytes)
	s.resolver = NewResolver(s.client)
}

func (s *HTTPSChildResolverSuite) Test_resolves_blueprint_file() {
	host := strings.TrimPrefix(s.server.URL, "https://")
	path := "/public/https.test.blueprint.yml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: core.ScalarFromString(path),
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"host": core.MappingNodeFromString(host),
			},
		},
	}
	resolvedInfo, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().NoError(err)
	s.Assert().NotNil(resolvedInfo)
	s.Assert().NotNil(resolvedInfo.BlueprintSource)
	s.Assert().Equal(s.expectedBlueprintSource, *resolvedInfo.BlueprintSource)
}

func (s *HTTPSChildResolverSuite) Test_returns_error_when_path_is_empty() {
	host := strings.TrimPrefix(s.server.URL, "https://")
	path := ""
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"host": core.MappingNodeFromString(host),
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidPath, runErr.ReasonCode)
	s.Assert().Equal(
		runErr.Err.Error(),
		"[include.test]: invalid path found, path value must be a string "+
			"for the https child resolver, the provided value is either empty or not a string",
	)
}

func (s *HTTPSChildResolverSuite) Test_returns_error_when_host_is_not_provided() {
	path := "/public/https.test.blueprint.yml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeInvalidMetadata, runErr.ReasonCode)
	s.Assert().Equal(
		runErr.Err.Error(),
		"[include.test]: missing host field in metadata for the HTTPS include",
	)
}

func (s *HTTPSChildResolverSuite) Test_returns_error_when_blueprint_url_does_not_exist() {
	host := strings.TrimPrefix(s.server.URL, "https://")
	path := "/public/https.missing.test.blueprint.yml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"host": core.MappingNodeFromString(host),
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodeBlueprintNotFound, runErr.ReasonCode)
	s.Assert().Equal(
		runErr.Err.Error(),
		fmt.Sprintf("[include.test]: blueprint not found at path: https://%s%s", host, path),
	)
}

func (s *HTTPSChildResolverSuite) Test_returns_error_when_url_is_protected() {
	host := strings.TrimPrefix(s.server.URL, "https://")
	path := "/private/https.missing.test.blueprint.yml"
	include := &subengine.ResolvedInclude{
		Path: &core.MappingNode{
			Scalar: &core.ScalarValue{
				StringValue: &path,
			},
		},
		Metadata: &core.MappingNode{
			Fields: map[string]*core.MappingNode{
				"host": core.MappingNodeFromString(host),
			},
		},
	}
	_, err := s.resolver.Resolve(context.TODO(), "test", include, nil)
	s.Require().Error(err)
	runErr, isRunError := err.(*errors.RunError)
	s.Require().True(isRunError)
	s.Assert().Equal(includes.ErrorReasonCodePermissions, runErr.ReasonCode)
	s.Assert().Equal(
		runErr.Err.Error(),
		fmt.Sprintf(
			"[include.test]: permission error encountered while "+
				"reading blueprint at path: https://%s%s: HTTP status code: 401",
			host,
			path,
		),
	)
}

func TestHTTPSChildResolverSuite(t *testing.T) {
	suite.Run(t, new(HTTPSChildResolverSuite))
}
