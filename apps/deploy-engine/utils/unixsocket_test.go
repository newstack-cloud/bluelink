package utils

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UnixSocketSuite struct {
	suite.Suite
}

// A unix socket path is limited to around 104 bytes on darwin and 108 on linux, and
// t.TempDir() produces paths long enough to exceed that, so these tests need a short root
// rather than the usual per-test directory.
func (s *UnixSocketSuite) shortTempDir() string {
	dir, err := os.MkdirTemp("", "bl")
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

func (s *UnixSocketSuite) Test_prepare_dir_creates_a_directory_restricted_to_the_owner() {
	socketPath := filepath.Join(s.shortTempDir(), "engine", "api.sock")

	s.Require().NoError(PrepareUnixSocketDir(socketPath))

	info, err := os.Stat(filepath.Dir(socketPath))
	s.Require().NoError(err)
	s.Require().True(info.IsDir())

	if runtime.GOOS == "windows" {
		// Access control there is the directory's ACL, inherited from its parent, which the
		// default path places under the per-user app data directory. Permission bits are not
		// the mechanism, so there is nothing to assert on them.
		return
	}
	s.Assert().Equal(
		os.FileMode(0o700),
		info.Mode().Perm(),
		"the directory holding the socket is reachable by users other than the owner",
	)
}

// A directory left behind by an earlier run, or created by something else, must be tightened
// rather than accepted as found.
func (s *UnixSocketSuite) Test_prepare_dir_tightens_an_existing_permissive_directory() {
	if runtime.GOOS == "windows" {
		s.T().Skip("access control on windows is the directory ACL, not permission bits")
	}

	dir := filepath.Join(s.shortTempDir(), "engine")
	s.Require().NoError(os.MkdirAll(dir, 0o755))

	s.Require().NoError(PrepareUnixSocketDir(filepath.Join(dir, "api.sock")))

	info, err := os.Stat(dir)
	s.Require().NoError(err)
	s.Assert().Equal(os.FileMode(0o700), info.Mode().Perm())
}

// The whole point of the directory is that a socket can then be served from it.
func (s *UnixSocketSuite) Test_a_socket_in_the_prepared_dir_accepts_connections() {
	socketPath := filepath.Join(s.shortTempDir(), "engine", "api.sock")
	s.Require().NoError(PrepareUnixSocketDir(socketPath))

	listener, err := net.Listen("unix", socketPath)
	s.Require().NoError(err)
	defer listener.Close()

	s.Require().NoError(RestrictUnixSocket(socketPath))

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	conn, err := net.Dial("unix", socketPath)
	s.Require().NoError(err)
	s.Require().NoError(conn.Close())
}

func (s *UnixSocketSuite) Test_clearing_removes_a_socket_nothing_is_listening_on() {
	socketPath := filepath.Join(s.shortTempDir(), "api.sock")

	listener, err := net.Listen("unix", socketPath)
	s.Require().NoError(err)
	// Closed without removing the file, which is what an abrupt termination leaves behind.
	s.Require().NoError(listener.Close())
	s.Require().NoError(os.WriteFile(socketPath, nil, 0o600))
	_ = os.Remove(socketPath)

	listener, err = net.Listen("unix", socketPath)
	s.Require().NoError(err)
	s.Require().NoError(listener.Close())

	s.Require().NoError(ClearStaleUnixSocket(socketPath))

	_, err = os.Stat(socketPath)
	s.Assert().True(os.IsNotExist(err), "a stale socket file was left in place")
}

func (s *UnixSocketSuite) Test_clearing_refuses_a_path_that_is_not_a_socket() {
	path := filepath.Join(s.shortTempDir(), "not-a-socket")
	s.Require().NoError(os.WriteFile(path, []byte("important"), 0o600))

	err := ClearStaleUnixSocket(path)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "refusing to remove it")

	_, statErr := os.Stat(path)
	s.Assert().NoError(statErr, "the file was removed despite the refusal")
}

// A live engine must not have its socket taken from under it by a second one.
func (s *UnixSocketSuite) Test_clearing_refuses_a_socket_with_a_listener_on_it() {
	socketPath := filepath.Join(s.shortTempDir(), "api.sock")

	listener, err := net.Listen("unix", socketPath)
	s.Require().NoError(err)
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	err = ClearStaleUnixSocket(socketPath)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "appears to be listening")

	_, statErr := os.Stat(socketPath)
	s.Assert().NoError(statErr, "a live engine's socket was removed")
}

func (s *UnixSocketSuite) Test_clearing_a_path_that_does_not_exist_is_not_an_error() {
	s.Assert().NoError(
		ClearStaleUnixSocket(filepath.Join(s.shortTempDir(), "absent.sock")),
	)
}

func TestUnixSocketSuite(t *testing.T) {
	suite.Run(t, new(UnixSocketSuite))
}
