package plugins

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	"github.com/stretchr/testify/suite"
)

type ManagerSuite struct {
	suite.Suite
	tempDir string
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}

func (s *ManagerSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	s.Require().NoError(err)
	s.tempDir, err = filepath.EvalSymlinks(tempDir)
	s.Require().NoError(err)
}

func (s *ManagerSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *ManagerSuite) TestGetPluginsDir_default() {
	// Unset env var to test default
	originalEnv := os.Getenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH")
	os.Unsetenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH")
	defer func() {
		if originalEnv != "" {
			os.Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", originalEnv)
		}
	}()

	pluginsDir := GetPluginsDir()
	s.NotEmpty(pluginsDir)
	s.Contains(pluginsDir, ".bluelink/engine/plugins")
}

func (s *ManagerSuite) TestGetPluginsDir_env_override() {
	customPath := "/custom/plugins/path"
	os.Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", customPath)
	defer os.Unsetenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH")

	pluginsDir := GetPluginsDir()
	s.Equal(customPath, pluginsDir)
}

func (s *ManagerSuite) TestGetPluginsDir_multiple_paths() {
	// Test that when multiple paths are provided, the first one is used
	firstPath := "/first/plugins/path"
	secondPath := "/second/plugins/path"
	multiPath := firstPath + string(os.PathListSeparator) + secondPath
	os.Setenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH", multiPath)
	defer os.Unsetenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH")

	pluginsDir := GetPluginsDir()
	s.Equal(firstPath, pluginsDir)
}

func (s *ManagerSuite) TestVerifyChecksum_success() {
	// Create a test file
	testContent := []byte("test plugin content")
	testFile := filepath.Join(s.tempDir, "test-plugin.tar.gz")
	err := os.WriteFile(testFile, testContent, 0644)
	s.Require().NoError(err)

	// Calculate expected checksum
	hash := sha256.Sum256(testContent)
	expectedChecksum := hex.EncodeToString(hash[:])

	// Create shasums content
	shasums := fmt.Sprintf("%s  test-plugin.tar.gz\n", expectedChecksum)

	manager := &Manager{pluginsDir: s.tempDir}

	err = manager.VerifyChecksum(testFile, []byte(shasums), "test-plugin.tar.gz")
	s.NoError(err)
}

func (s *ManagerSuite) TestVerifyChecksum_mismatch() {
	// Create a test file
	testContent := []byte("test plugin content")
	testFile := filepath.Join(s.tempDir, "test-plugin.tar.gz")
	err := os.WriteFile(testFile, testContent, 0644)
	s.Require().NoError(err)

	// Use wrong checksum
	shasums := "wrongchecksum123456789abcdef  test-plugin.tar.gz\n"

	manager := &Manager{pluginsDir: s.tempDir}

	err = manager.VerifyChecksum(testFile, []byte(shasums), "test-plugin.tar.gz")
	s.Error(err)
	s.Contains(err.Error(), "checksum mismatch")
}

func (s *ManagerSuite) TestVerifyChecksum_file_not_in_shasums() {
	testContent := []byte("test plugin content")
	testFile := filepath.Join(s.tempDir, "test-plugin.tar.gz")
	err := os.WriteFile(testFile, testContent, 0644)
	s.Require().NoError(err)

	shasums := "abc123  different-file.tar.gz\n"

	manager := &Manager{pluginsDir: s.tempDir}

	err = manager.VerifyChecksum(testFile, []byte(shasums), "test-plugin.tar.gz")
	s.Error(err)
	s.Contains(err.Error(), "checksum not found")
}

func (s *ManagerSuite) TestExtractArchive_success() {
	// Create a test tar.gz archive
	archivePath := filepath.Join(s.tempDir, "test.tar.gz")
	destDir := filepath.Join(s.tempDir, "extracted")

	err := s.createTestArchive(archivePath, map[string]string{
		"plugin/main.go":      "package main",
		"plugin/README.md":    "# Test Plugin",
		"plugin/data/config":  "key=value",
	})
	s.Require().NoError(err)

	manager := &Manager{pluginsDir: s.tempDir}

	err = manager.ExtractArchive(archivePath, destDir)
	s.NoError(err)

	// Verify extracted files
	content, err := os.ReadFile(filepath.Join(destDir, "plugin", "main.go"))
	s.NoError(err)
	s.Equal("package main", string(content))

	content, err = os.ReadFile(filepath.Join(destDir, "plugin", "README.md"))
	s.NoError(err)
	s.Equal("# Test Plugin", string(content))

	content, err = os.ReadFile(filepath.Join(destDir, "plugin", "data", "config"))
	s.NoError(err)
	s.Equal("key=value", string(content))
}

func (s *ManagerSuite) TestExtractArchive_path_traversal() {
	// Create an archive with path traversal attempt
	archivePath := filepath.Join(s.tempDir, "malicious.tar.gz")
	destDir := filepath.Join(s.tempDir, "extracted")

	// Create archive with malicious path
	file, err := os.Create(archivePath)
	s.Require().NoError(err)
	defer file.Close()

	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)

	// Add a file with path traversal
	header := &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0644,
		Size: int64(len("malicious")),
	}
	tw.WriteHeader(header)
	tw.Write([]byte("malicious"))

	tw.Close()
	gw.Close()

	manager := &Manager{pluginsDir: s.tempDir}

	err = manager.ExtractArchive(archivePath, destDir)
	s.Error(err)
	s.Contains(err.Error(), "invalid file path")
}

func (s *ManagerSuite) TestLoadManifest_empty() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	s.Require().NoError(err)

	manager := &Manager{pluginsDir: pluginsDir}

	manifest, err := manager.LoadManifest()
	s.NoError(err)
	s.NotNil(manifest)
	s.Len(manifest.Plugins, 0)
}

func (s *ManagerSuite) TestSaveAndLoadManifest() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}

	err := manager.SaveManifest(manifest)
	s.NoError(err)

	loaded, err := manager.LoadManifest()
	s.NoError(err)
	s.Len(loaded.Plugins, 1)
	s.Equal("bluelink/aws@1.0.0", loaded.Plugins["registry.bluelink.dev/bluelink/aws"].ID)
}

func (s *ManagerSuite) TestIsInstalled_not_installed() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.0.0",
	}

	installed, plugin, err := manager.IsInstalled(pluginID)
	s.NoError(err)
	s.False(installed)
	s.Nil(plugin)
}

func (s *ManagerSuite) TestIsInstalled_installed() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.0.0",
	}

	installed, plugin, err := manager.IsInstalled(pluginID)
	s.NoError(err)
	s.True(installed)
	s.NotNil(plugin)
	s.Equal("1.0.0", plugin.Version)
}

func (s *ManagerSuite) TestIsInstalled_wrong_version() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest with version 1.0.0
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Check for version 2.0.0 (different)
	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "2.0.0",
	}

	installed, _, err := manager.IsInstalled(pluginID)
	s.NoError(err)
	s.False(installed)
}

func (s *ManagerSuite) TestGetMissingPlugins() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest with aws plugin
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Check for aws (installed) and gcp (not installed)
	pluginIDs := []*PluginID{
		{
			RegistryHost: DefaultRegistryHost,
			Namespace:    "bluelink",
			Name:         "aws",
			Version:      "1.0.0",
		},
		{
			RegistryHost: DefaultRegistryHost,
			Namespace:    "bluelink",
			Name:         "gcp",
			Version:      "1.0.0",
		},
	}

	missing, err := manager.GetMissingPlugins(pluginIDs)
	s.NoError(err)
	s.Len(missing, 1)
	s.Equal("gcp", missing[0].Name)
}

func (s *ManagerSuite) TestListInstalled() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
			"registry.bluelink.dev/bluelink/gcp": {
				ID:           "bluelink/gcp@2.0.0",
				Version:      "2.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "def456",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	plugins, err := manager.ListInstalled()
	s.NoError(err)
	s.Len(plugins, 2)

	// Should be sorted by ID
	s.Equal("bluelink/aws@1.0.0", plugins[0].ID)
	s.Equal("bluelink/gcp@2.0.0", plugins[1].ID)
}

func (s *ManagerSuite) TestInstall_missing_signature_urls() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/test/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{
					{Version: "1.0.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/v1/plugins/bluelink/test/1.0.0/package/darwin/arm64",
			"/v1/plugins/bluelink/test/1.0.0/package/darwin/amd64",
			"/v1/plugins/bluelink/test/1.0.0/package/linux/amd64",
			"/v1/plugins/bluelink/test/1.0.0/package/linux/arm64",
			"/v1/plugins/bluelink/test/1.0.0/package/windows/amd64":
			// Missing ShasumsURL and ShasumsSignatureURL
			metadata := registries.PluginPackageMetadata{
				Filename:    "test_1.0.0.tar.gz",
				DownloadURL: "/download/test_1.0.0.tar.gz",
				OS:          "darwin",
				Arch:        "arm64",
				Shasum:      "abc123",
				// ShasumsURL and ShasumsSignatureURL intentionally missing
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metadata)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "test",
		Version:      "1.0.0",
	}

	result, err := manager.Install(context.Background(), pluginID, nil)
	s.NoError(err)
	s.Equal(StatusFailed, result.Status)
	s.ErrorIs(result.Error, registries.ErrSignatureMissing)
}

func (s *ManagerSuite) TestResolveDependencies_no_dependencies() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server with plugin that has no dependencies
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/base/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			if matched, _ := filepath.Match("/v1/plugins/bluelink/base/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:    "base_1.0.0.tar.gz",
					DownloadURL: "/download/base_1.0.0.tar.gz",
					Shasum:      "abc123",
					// No dependencies
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "base",
		Version:      "1.0.0",
	}

	resolved, err := manager.resolveDependencies(context.Background(), []*PluginID{pluginID})
	s.NoError(err)
	s.Len(resolved, 1)
	s.Equal("base", resolved[0].Name)
}

func (s *ManagerSuite) TestResolveDependencies_with_single_dependency() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server with plugin A depending on plugin B
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/plugin-a/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/v1/plugins/bluelink/plugin-b/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-a/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:    "plugin-a_1.0.0.tar.gz",
					DownloadURL: "/download/plugin-a_1.0.0.tar.gz",
					Shasum:      "abc123",
					Dependencies: map[string]string{
						"bluelink/plugin-b": "1.0.0",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-b/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:    "plugin-b_1.0.0.tar.gz",
					DownloadURL: "/download/plugin-b_1.0.0.tar.gz",
					Shasum:      "def456",
					// No dependencies
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "plugin-a",
		Version:      "1.0.0",
	}

	resolved, err := manager.resolveDependencies(context.Background(), []*PluginID{pluginID})
	s.NoError(err)
	s.Len(resolved, 2)
	// Dependencies should come first (topological order)
	s.Equal("plugin-b", resolved[0].Name)
	s.Equal("plugin-a", resolved[1].Name)
}

func (s *ManagerSuite) TestResolveDependencies_transitive() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server: A -> B -> C
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/plugin-a/versions",
			"/v1/plugins/bluelink/plugin-b/versions",
			"/v1/plugins/bluelink/plugin-c/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-a/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:     "plugin-a_1.0.0.tar.gz",
					DownloadURL:  "/download/plugin-a_1.0.0.tar.gz",
					Shasum:       "abc123",
					Dependencies: map[string]string{"bluelink/plugin-b": "1.0.0"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-b/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:     "plugin-b_1.0.0.tar.gz",
					DownloadURL:  "/download/plugin-b_1.0.0.tar.gz",
					Shasum:       "def456",
					Dependencies: map[string]string{"bluelink/plugin-c": "1.0.0"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-c/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:    "plugin-c_1.0.0.tar.gz",
					DownloadURL: "/download/plugin-c_1.0.0.tar.gz",
					Shasum:      "ghi789",
					// No dependencies
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "plugin-a",
		Version:      "1.0.0",
	}

	resolved, err := manager.resolveDependencies(context.Background(), []*PluginID{pluginID})
	s.NoError(err)
	s.Len(resolved, 3)
	// Topological order: C (no deps), B (depends on C), A (depends on B)
	s.Equal("plugin-c", resolved[0].Name)
	s.Equal("plugin-b", resolved[1].Name)
	s.Equal("plugin-a", resolved[2].Name)
}

func (s *ManagerSuite) TestResolveDependencies_circular_detection() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server with circular dependency: A -> B -> A
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/plugin-a/versions",
			"/v1/plugins/bluelink/plugin-b/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-a/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:     "plugin-a_1.0.0.tar.gz",
					DownloadURL:  "/download/plugin-a_1.0.0.tar.gz",
					Shasum:       "abc123",
					Dependencies: map[string]string{"bluelink/plugin-b": "1.0.0"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-b/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:     "plugin-b_1.0.0.tar.gz",
					DownloadURL:  "/download/plugin-b_1.0.0.tar.gz",
					Shasum:       "def456",
					Dependencies: map[string]string{"bluelink/plugin-a": "1.0.0"}, // Circular!
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "plugin-a",
		Version:      "1.0.0",
	}

	_, err := manager.resolveDependencies(context.Background(), []*PluginID{pluginID})
	s.Error(err)
	s.Contains(err.Error(), "circular dependency")
}

func (s *ManagerSuite) TestResolveDependencies_skips_installed() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server first: A depends on B (which will be pre-installed)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/plugin-a/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-a/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:     "plugin-a_1.0.0.tar.gz",
					DownloadURL:  "/download/plugin-a_1.0.0.tar.gz",
					Shasum:       "abc123",
					Dependencies: map[string]string{"bluelink/plugin-b": "1.0.0"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	// Extract host from server URL for manifest key (remove http:// prefix)
	serverHost := server.URL

	// Pre-install plugin-b using the same registry host that will be inherited
	manager := &Manager{pluginsDir: pluginsDir}
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			serverHost + "/bluelink/plugin-b": {
				ID:           "bluelink/plugin-b@1.0.0",
				Version:      "1.0.0",
				RegistryHost: serverHost,
				Shasum:       "def456",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager = NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "plugin-a",
		Version:      "1.0.0",
	}

	resolved, err := manager.resolveDependencies(context.Background(), []*PluginID{pluginID})
	s.NoError(err)
	// Only plugin-a should be in the result (plugin-b is already installed)
	s.Len(resolved, 1)
	s.Equal("plugin-a", resolved[0].Name)
}

func (s *ManagerSuite) createTestArchive(archivePath string, files map[string]string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Sort keys for consistent order
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}

	for _, name := range keys {
		content := files[name]
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func (s *ManagerSuite) TestResolveVersion_exact_version() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.0.0",
	}

	// Exact version should return as-is without making API calls
	version, err := manager.ResolveVersion(context.Background(), pluginID)
	s.NoError(err)
	s.Equal("1.0.0", version)
}

func (s *ManagerSuite) TestResolveVersion_caret_constraint() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server that returns available versions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/aws/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{
					{Version: "2.0.0"},
					{Version: "1.2.0"},
					{Version: "1.1.0"},
					{Version: "1.0.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "^1.0.0", // Caret: >=1.0.0, <2.0.0
	}

	version, err := manager.ResolveVersion(context.Background(), pluginID)
	s.NoError(err)
	s.Equal("1.2.0", version) // Should pick highest 1.x version
}

func (s *ManagerSuite) TestResolveVersion_tilde_constraint() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/aws/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{
					{Version: "2.0.0"},
					{Version: "1.1.0"},
					{Version: "1.0.5"},
					{Version: "1.0.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "~1.0.0", // Tilde: >=1.0.0, <1.1.0
	}

	version, err := manager.ResolveVersion(context.Background(), pluginID)
	s.NoError(err)
	s.Equal("1.0.5", version) // Should pick highest 1.0.x version
}

func (s *ManagerSuite) TestResolveVersion_no_matching_version() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/aws/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{
					{Version: "1.0.0"},
					{Version: "1.1.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "^2.0.0", // No 2.x versions available
	}

	_, err := manager.ResolveVersion(context.Background(), pluginID)
	s.Error(err)
	s.Contains(err.Error(), "no version matching constraint")
}

func (s *ManagerSuite) TestResolveVersion_empty_version_gets_latest() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/aws/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{
					{Version: "2.0.0"},
					{Version: "1.1.0"},
					{Version: "1.0.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "aws",
		// Version is empty - should resolve to latest
	}

	version, err := manager.ResolveVersion(context.Background(), pluginID)
	s.NoError(err)
	s.Equal("2.0.0", version) // First version in list (latest)
}

func (s *ManagerSuite) TestResolveDependencies_with_constraint() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")
	authPath := filepath.Join(s.tempDir, "plugins.auth.json")
	tokenPath := filepath.Join(s.tempDir, "plugins.tokens.json")

	// Create mock server: A depends on B with ^1.0.0 constraint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/bluelink-services.json":
			doc := registries.ServiceDiscoveryDocument{
				ProviderV1: &registries.PluginServiceConfig{
					Endpoint: "/v1/plugins",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)

		case "/v1/plugins/bluelink/plugin-a/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{{Version: "1.0.0"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/v1/plugins/bluelink/plugin-b/versions":
			resp := registries.PluginVersionsResponse{
				Versions: []registries.PluginVersionInfo{
					{Version: "2.0.0"},
					{Version: "1.5.0"},
					{Version: "1.0.0"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-a/1.0.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:    "plugin-a_1.0.0.tar.gz",
					DownloadURL: "/download/plugin-a_1.0.0.tar.gz",
					Shasum:      "abc123",
					Dependencies: map[string]string{
						"bluelink/plugin-b": "^1.0.0", // Caret constraint
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else if matched, _ := filepath.Match("/v1/plugins/bluelink/plugin-b/1.5.0/package/*/*", r.URL.Path); matched {
				metadata := registries.PluginPackageMetadata{
					Filename:    "plugin-b_1.5.0.tar.gz",
					DownloadURL: "/download/plugin-b_1.5.0.tar.gz",
					Shasum:      "def456",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(metadata)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	authStore := registries.NewAuthConfigStoreWithPath(authPath)
	tokenStore := registries.NewTokenStoreWithPath(tokenPath)
	discoveryClient := registries.NewServiceDiscoveryClientWithHTTPClient(server.Client())
	registryClient := registries.NewRegistryClientWithHTTPClient(
		server.Client(), authStore, tokenStore, discoveryClient,
	)

	manager := NewManagerWithPluginsDir(registryClient, discoveryClient, pluginsDir)

	pluginID := &PluginID{
		RegistryHost: server.URL,
		Namespace:    "bluelink",
		Name:         "plugin-a",
		Version:      "1.0.0",
	}

	resolved, err := manager.resolveDependencies(context.Background(), []*PluginID{pluginID})
	s.NoError(err)
	s.Len(resolved, 2)
	// plugin-b should be resolved to 1.5.0 (highest matching ^1.0.0)
	s.Equal("plugin-b", resolved[0].Name)
	s.Equal("1.5.0", resolved[0].Version)
	s.Equal("plugin-a", resolved[1].Name)
}

func (s *ManagerSuite) TestUninstall_removes_plugin_from_manifest() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest with a plugin
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Create plugin directory
	pluginDir := filepath.Join(pluginsDir, "bin", "bluelink", "aws", "1.0.0")
	err = os.MkdirAll(pluginDir, 0755)
	s.Require().NoError(err)
	err = os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("binary"), 0755)
	s.Require().NoError(err)

	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}

	result := manager.Uninstall(pluginID)
	s.Equal(UninstallStatusRemoved, result.Status)
	s.NoError(result.Error)

	// Verify manifest no longer contains the plugin
	loadedManifest, err := manager.LoadManifest()
	s.NoError(err)
	s.Len(loadedManifest.Plugins, 0)
}

func (s *ManagerSuite) TestUninstall_removes_plugin_files() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Create plugin directory with files
	pluginDir := filepath.Join(pluginsDir, "bin", "bluelink", "aws", "1.0.0")
	err = os.MkdirAll(pluginDir, 0755)
	s.Require().NoError(err)
	err = os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("binary"), 0755)
	s.Require().NoError(err)

	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}

	result := manager.Uninstall(pluginID)
	s.Equal(UninstallStatusRemoved, result.Status)

	// Verify plugin directory was removed
	_, err = os.Stat(pluginDir)
	s.True(os.IsNotExist(err))
}

func (s *ManagerSuite) TestUninstall_returns_not_found_for_missing_plugin() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "nonexistent",
	}

	result := manager.Uninstall(pluginID)
	s.Equal(UninstallStatusNotFound, result.Status)
	s.NoError(result.Error)
}

func (s *ManagerSuite) TestUninstallAll_removes_multiple_plugins() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest with multiple plugins
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
			"registry.bluelink.dev/bluelink/gcp": {
				ID:           "bluelink/gcp@2.0.0",
				Version:      "2.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "def456",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Create plugin directories
	for _, name := range []string{"aws", "gcp"} {
		version := "1.0.0"
		if name == "gcp" {
			version = "2.0.0"
		}
		pluginDir := filepath.Join(pluginsDir, "bin", "bluelink", name, version)
		err = os.MkdirAll(pluginDir, 0755)
		s.Require().NoError(err)
		err = os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("binary"), 0755)
		s.Require().NoError(err)
	}

	pluginIDs := []*PluginID{
		{RegistryHost: DefaultRegistryHost, Namespace: "bluelink", Name: "aws"},
		{RegistryHost: DefaultRegistryHost, Namespace: "bluelink", Name: "gcp"},
	}

	results := manager.UninstallAll(pluginIDs)
	s.Len(results, 2)
	s.Equal(UninstallStatusRemoved, results[0].Status)
	s.Equal(UninstallStatusRemoved, results[1].Status)

	// Verify manifest is empty
	loadedManifest, err := manager.LoadManifest()
	s.NoError(err)
	s.Len(loadedManifest.Plugins, 0)
}

func (s *ManagerSuite) TestUninstall_cleans_up_empty_parent_directories() {
	pluginsDir := filepath.Join(s.tempDir, "plugins")

	manager := &Manager{pluginsDir: pluginsDir}

	// Pre-populate manifest
	manifest := &PluginManifest{
		Plugins: map[string]*InstalledPlugin{
			"registry.bluelink.dev/bluelink/aws": {
				ID:           "bluelink/aws@1.0.0",
				Version:      "1.0.0",
				RegistryHost: "registry.bluelink.dev",
				Shasum:       "abc123",
				InstalledAt:  time.Now(),
			},
		},
	}
	err := manager.SaveManifest(manifest)
	s.Require().NoError(err)

	// Create plugin directory
	pluginDir := filepath.Join(pluginsDir, "bin", "bluelink", "aws", "1.0.0")
	err = os.MkdirAll(pluginDir, 0755)
	s.Require().NoError(err)
	err = os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("binary"), 0755)
	s.Require().NoError(err)

	pluginID := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}

	result := manager.Uninstall(pluginID)
	s.Equal(UninstallStatusRemoved, result.Status)

	// Verify empty parent directories were cleaned up
	nameDir := filepath.Join(pluginsDir, "bin", "bluelink", "aws")
	_, err = os.Stat(nameDir)
	s.True(os.IsNotExist(err))

	namespaceDir := filepath.Join(pluginsDir, "bin", "bluelink")
	_, err = os.Stat(namespaceDir)
	s.True(os.IsNotExist(err))
}
