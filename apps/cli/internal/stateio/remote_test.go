package stateio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RemoteFileTestSuite struct {
	suite.Suite
}

func (s *RemoteFileTestSuite) Test_IsRemoteFile_returns_true_for_s3_urls() {
	s.True(IsRemoteFile("s3://bucket/path/to/file.tar.gz"))
	s.True(IsRemoteFile("s3://my-bucket/state.tar.gz"))
}

func (s *RemoteFileTestSuite) Test_IsRemoteFile_returns_true_for_gcs_urls() {
	s.True(IsRemoteFile("gcs://bucket/path/to/file.tar.gz"))
	s.True(IsRemoteFile("gcs://my-bucket/state.tar.gz"))
}

func (s *RemoteFileTestSuite) Test_IsRemoteFile_returns_true_for_azure_blob_urls() {
	s.True(IsRemoteFile("azureblob://container/path/to/file.tar.gz"))
	s.True(IsRemoteFile("azureblob://my-container/state.tar.gz"))
}

func (s *RemoteFileTestSuite) Test_IsRemoteFile_returns_false_for_local_paths() {
	s.False(IsRemoteFile("/path/to/file.tar.gz"))
	s.False(IsRemoteFile("./relative/path.json"))
	s.False(IsRemoteFile("file.json"))
	s.False(IsRemoteFile(""))
}

func (s *RemoteFileTestSuite) Test_IsRemoteFile_returns_false_for_http_urls() {
	s.False(IsRemoteFile("http://example.com/file.tar.gz"))
	s.False(IsRemoteFile("https://example.com/file.tar.gz"))
}

func (s *RemoteFileTestSuite) Test_DownloadRemoteFile_returns_error_for_unsupported_scheme() {
	_, err := DownloadRemoteFile(context.TODO(), "/local/path/file.tar.gz", nil)
	s.Require().Error(err)

	importErr, ok := err.(*ImportError)
	s.True(ok)
	s.Equal(ErrCodeFileNotFound, importErr.Code)
	s.Contains(importErr.Message, "unsupported remote source type")
}

func TestRemoteFileTestSuite(t *testing.T) {
	suite.Run(t, new(RemoteFileTestSuite))
}
