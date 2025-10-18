package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FileSourceTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *FileSourceTestSuite) SetupTest() {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "file-source-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *FileSourceTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func (s *FileSourceTestSuite) Test_local_file_source_reads_file() {
	// Create test file
	testFile := filepath.Join(s.tempDir, "test.txt")
	testContent := []byte("test content")
	err := os.WriteFile(testFile, testContent, 0644)
	s.Require().NoError(err)

	source := &LocalFileSource{}

	// Test reading file
	data, err := source.ReadFile(context.Background(), testFile)
	s.Assert().NoError(err)
	s.Assert().Equal(testContent, data)
}

func (s *FileSourceTestSuite) Test_local_file_source_strips_file_prefix() {
	// Create test file
	testFile := filepath.Join(s.tempDir, "test.txt")
	testContent := []byte("test content")
	err := os.WriteFile(testFile, testContent, 0644)
	s.Require().NoError(err)

	source := &LocalFileSource{}

	// Test reading file with file:// prefix
	data, err := source.ReadFile(context.Background(), "file://"+testFile)
	s.Assert().NoError(err)
	s.Assert().Equal(testContent, data)
}

func (s *FileSourceTestSuite) Test_local_file_source_returns_error_for_nonexistent_file() {
	source := &LocalFileSource{}

	_, err := source.ReadFile(context.Background(), "/nonexistent/file.txt")
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "unable to read local file")
}

func (s *FileSourceTestSuite) Test_registry_reads_local_file_by_default() {
	registry := NewFileSourceRegistry()

	// Create test file
	testFile := filepath.Join(s.tempDir, "test.txt")
	testContent := []byte("registry test content")
	err := os.WriteFile(testFile, testContent, 0644)
	s.Require().NoError(err)

	// Test reading through registry
	data, err := registry.ReadFile(context.Background(), testFile)
	s.Assert().NoError(err)
	s.Assert().Equal(testContent, data)
}

func (s *FileSourceTestSuite) Test_registry_supports_custom_file_sources() {
	registry := NewFileSourceRegistry()

	// Register a custom file source for "mock" scheme
	mockSource := &MockFileSource{
		files: map[string][]byte{
			"mock://bucket/file.txt": []byte("mock content"),
		},
	}
	err := registry.Register(mockSource)
	s.Require().NoError(err)

	// Test reading from mock source
	data, err := registry.ReadFile(context.Background(), "mock://bucket/file.txt")
	s.Assert().NoError(err)
	s.Assert().Equal([]byte("mock content"), data)
}

func (s *FileSourceTestSuite) Test_registry_returns_error_for_duplicate_scheme() {
	registry := NewFileSourceRegistry()

	mockSource := &MockFileSource{
		files: map[string][]byte{},
	}

	err := registry.Register(mockSource)
	s.Require().NoError(err)

	// Try to register another source with the same scheme
	err = registry.Register(&MockFileSource{files: map[string][]byte{}})
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "already registered")
}

func (s *FileSourceTestSuite) Test_registry_returns_error_for_unknown_scheme() {
	registry := NewFileSourceRegistry()

	// Try to read from unknown scheme
	_, err := registry.ReadFile(context.Background(), "unknown://bucket/file.txt")
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "no file source handler registered for scheme")
}

func (s *FileSourceTestSuite) Test_extract_scheme_from_various_paths() {
	tests := []struct {
		path           string
		expectedScheme string
	}{
		{"/local/path/file.txt", ""},
		{"relative/path/file.txt", ""},
		{"file:///local/path/file.txt", "file"},
		{"s3://bucket/key", "s3"},
		{"gs://bucket/object", "gs"},
		{"https://example.com/file", "https"},
		{"S3://bucket/key", "s3"}, // Should be normalized to lowercase
	}

	for _, tt := range tests {
		s.Run(tt.path, func() {
			scheme := extractScheme(tt.path)
			s.Assert().Equal(tt.expectedScheme, scheme)
		})
	}
}

func TestFileSourceTestSuite(t *testing.T) {
	suite.Run(t, new(FileSourceTestSuite))
}

// MockFileSource is a test implementation of FileSource
type MockFileSource struct {
	files map[string][]byte
}

func (m *MockFileSource) Scheme() string {
	return "mock"
}

func (m *MockFileSource) ReadFile(ctx context.Context, path string) ([]byte, error) {
	data, exists := m.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return data, nil
}
