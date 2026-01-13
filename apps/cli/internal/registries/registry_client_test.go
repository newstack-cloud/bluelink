package registries

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RegistryClientSuite struct {
	suite.Suite
	tempDir string
}

func TestRegistryClientSuite(t *testing.T) {
	suite.Run(t, new(RegistryClientSuite))
}

func (s *RegistryClientSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "registry-client-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *RegistryClientSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *RegistryClientSuite) TestListVersions_success() {
	versionsResponse := PluginVersionsResponse{
		Versions: []PluginVersionInfo{
			{Version: "1.0.0"},
			{Version: "1.1.0"},
			{Version: "2.0.0"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/aws/versions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionsResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	resp, err := client.ListVersions(context.Background(), server.URL, "bluelink", "aws")
	s.NoError(err)
	s.Require().NotNil(resp)
	s.Len(resp.Versions, 3)
	s.Equal("1.0.0", resp.Versions[0].Version)
	s.Equal("2.0.0", resp.Versions[2].Version)
}

func (s *RegistryClientSuite) TestListVersions_plugin_not_found() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	_, err := client.ListVersions(context.Background(), server.URL, "bluelink", "nonexistent")
	s.ErrorIs(err, ErrPluginNotFound)
}

func (s *RegistryClientSuite) TestListVersions_unauthorized() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	_, err := client.ListVersions(context.Background(), server.URL, "bluelink", "aws")
	s.ErrorIs(err, ErrNoCredentials)
}

func (s *RegistryClientSuite) TestGetPackageMetadata_success() {
	metadata := PluginPackageMetadata{
		Filename:            "test-provider_1.0.0_darwin_arm64.tar.gz",
		DownloadURL:         "/download/test-provider_1.0.0_darwin_arm64.tar.gz",
		OS:                  "darwin",
		Arch:                "arm64",
		Shasum:              "abc123def456",
		ShasumsURL:          "/download/test-provider_1.0.0_SHA256SUMS",
		ShasumsSignatureURL: "/download/test-provider_1.0.0_SHA256SUMS.sig",
		SigningKeys: map[string]string{
			"gpg_public_key": "-----BEGIN PGP PUBLIC KEY BLOCK-----...",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/test-provider/1.0.0/package/darwin/arm64":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metadata)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	result, err := client.GetPackageMetadata(
		context.Background(),
		server.URL,
		"bluelink",
		"test-provider",
		"1.0.0",
		"darwin",
		"arm64",
	)
	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(metadata.Filename, result.Filename)
	s.Equal(metadata.DownloadURL, result.DownloadURL)
	s.Equal(metadata.Shasum, result.Shasum)
	s.Equal(metadata.ShasumsURL, result.ShasumsURL)
	s.Equal(metadata.ShasumsSignatureURL, result.ShasumsSignatureURL)
	s.Require().Contains(result.SigningKeys, "gpg_public_key")
}

func (s *RegistryClientSuite) TestGetPackageMetadata_version_not_found() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	_, err := client.GetPackageMetadata(
		context.Background(),
		server.URL,
		"bluelink",
		"test-provider",
		"99.99.99",
		"darwin",
		"arm64",
	)
	s.ErrorIs(err, ErrVersionNotFound)
}

func (s *RegistryClientSuite) TestDownloadPackage_success() {
	testContent := []byte("test archive content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/download/test-provider.tar.gz":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "20")
			w.Write(testContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	metadata := &PluginPackageMetadata{
		DownloadURL: "/download/test-provider.tar.gz",
	}

	destPath := filepath.Join(s.tempDir, "downloaded.tar.gz")

	var progressCalled bool
	err := client.DownloadPackage(
		context.Background(),
		server.URL,
		metadata,
		destPath,
		func(downloaded, total int64) {
			progressCalled = true
		},
	)

	s.NoError(err)
	s.True(progressCalled)

	content, err := os.ReadFile(destPath)
	s.NoError(err)
	s.Equal(testContent, content)
}

func (s *RegistryClientSuite) TestDownloadPackage_full_url() {
	testContent := []byte("test archive content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/download/test-provider.tar.gz":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(testContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	metadata := &PluginPackageMetadata{
		DownloadURL: server.URL + "/download/test-provider.tar.gz",
	}

	destPath := filepath.Join(s.tempDir, "downloaded.tar.gz")

	err := client.DownloadPackage(
		context.Background(),
		server.URL,
		metadata,
		destPath,
		nil,
	)

	s.NoError(err)

	content, err := os.ReadFile(destPath)
	s.NoError(err)
	s.Equal(testContent, content)
}

func (s *RegistryClientSuite) TestDownloadShasums_success() {
	shasumsContent := []byte("abc123  test-provider_1.0.0_darwin_arm64.tar.gz\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/download/SHA256SUMS":
			w.Header().Set("Content-Type", "text/plain")
			w.Write(shasumsContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	result, err := client.DownloadShasums(context.Background(), server.URL, "/download/SHA256SUMS")
	s.NoError(err)
	s.Equal(shasumsContent, result)
}

func (s *RegistryClientSuite) TestDownloadSignature_success() {
	sigContent := []byte("-----BEGIN PGP SIGNATURE-----\ntest signature\n-----END PGP SIGNATURE-----")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/download/SHA256SUMS.sig":
			w.Header().Set("Content-Type", "application/pgp-signature")
			w.Write(sigContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	result, err := client.DownloadSignature(context.Background(), server.URL, "/download/SHA256SUMS.sig")
	s.NoError(err)
	s.Equal(sigContent, result)
}

func (s *RegistryClientSuite) TestAuthHeader_with_api_key() {
	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				Auth: &AuthV1Config{
					APIKeyHeader: "X-Api-Key",
				},
				ProviderV1: &PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/aws/versions":
			receivedAuthHeader = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(PluginVersionsResponse{})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	authConfigPath := filepath.Join(s.tempDir, "plugins.auth.json")
	authStore := NewAuthConfigStoreWithPath(authConfigPath)
	err := authStore.SaveRegistryAuth(server.URL, &RegistryAuthConfig{
		APIKey: "test-api-key-12345",
	})
	s.Require().NoError(err)

	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")
	tokenStore := NewTokenStoreWithPath(tokenPath)

	discoveryClient := NewServiceDiscoveryClientWithHTTPClient(server.Client())
	client := NewRegistryClientWithHTTPClient(server.Client(), authStore, tokenStore, discoveryClient)

	_, err = client.ListVersions(context.Background(), server.URL, "bluelink", "aws")
	s.NoError(err)
	s.Equal("Bearer test-api-key-12345", receivedAuthHeader)
}

func (s *RegistryClientSuite) TestNoPluginServiceConfigured() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serviceDiscoveryPath:
			doc := ServiceDiscoveryDocument{
				Auth: &AuthV1Config{
					APIKeyHeader: "X-Api-Key",
				},
				// No ProviderV1 or TransformerV1
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := s.createClient(server)

	_, err := client.ListVersions(context.Background(), server.URL, "bluelink", "aws")
	s.Error(err)
	s.Contains(err.Error(), "no plugin service configured")
}

func (s *RegistryClientSuite) createClient(server *httptest.Server) *RegistryClient {
	authConfigPath := filepath.Join(s.tempDir, "plugins.auth.json")
	authStore := NewAuthConfigStoreWithPath(authConfigPath)

	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")
	tokenStore := NewTokenStoreWithPath(tokenPath)

	discoveryClient := NewServiceDiscoveryClientWithHTTPClient(server.Client())

	return NewRegistryClientWithHTTPClient(server.Client(), authStore, tokenStore, discoveryClient)
}
