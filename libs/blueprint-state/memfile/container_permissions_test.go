package memfile

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

// Exercised against the real filesystem rather than an in-memory one, since what is being
// asserted is the mode that lands on disk, umask included.
type MemFileStateContainerPermissionsTestSuite struct {
	suite.Suite
	stateDir  string
	container *StateContainer
}

func (s *MemFileStateContainerPermissionsTestSuite) SetupTest() {
	if runtime.GOOS == "windows" {
		s.T().Skip("mode bits are not the access control on windows")
	}

	s.stateDir = path.Join(s.T().TempDir(), "state")
	s.Require().NoError(os.MkdirAll(s.stateDir, 0o700))

	container, err := LoadStateContainer(s.stateDir, afero.NewOsFs(), core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

// State holds resolved secrets whether or not encryption is enabled, so it must not be
// readable by other users of the host. With encryption off, which is the default, these modes
// are the only protection the state has.
func (s *MemFileStateContainerPermissionsTestSuite) Test_state_files_are_readable_only_by_their_owner() {
	s.Require().NoError(s.container.Instances().Save(context.Background(), state.InstanceState{
		InstanceID:   "0c1a1b2c-3d4e-5f60-7182-93a4b5c6d7e8",
		InstanceName: "orders-api",
		Status:       core.InstanceStatusDeployed,
		Metadata: map[string]*core.MappingNode{
			"connectionString": core.MappingNodeFromString("postgres://user:s3cr3t@host/db"),
		},
	}))

	files := s.writtenFiles()
	s.Require().NotEmpty(files, "the container wrote nothing to inspect")

	for path, info := range files {
		s.Assert().Equal(
			os.FileMode(0o600),
			info.Mode().Perm(),
			"%s is accessible to users other than its owner",
			path,
		)
	}
}

// Per-entity categories write into a directory of their own, which the storage creates. A
// directory other users can traverse exposes whatever is in it.
func (s *MemFileStateContainerPermissionsTestSuite) Test_directories_the_storage_creates_are_private_to_their_owner() {
	s.Require().NoError(s.container.CleanupOperations().Save(
		context.Background(),
		&manage.CleanupOperation{
			ID:          "1b2c3d4e-5f60-7182-93a4-b5c6d7e8f901",
			CleanupType: manage.CleanupTypeReconciliationResults,
			StartedAt:   1791853915,
		},
	))

	created := path.Join(s.stateDir, "cleanup_operations")
	info, err := os.Stat(created)
	s.Require().NoError(err, "the storage did not create a directory to inspect")
	s.Assert().Equal(os.FileMode(0o700), info.Mode().Perm())
}

func (s *MemFileStateContainerPermissionsTestSuite) writtenFiles() map[string]fs.FileInfo {
	files := map[string]fs.FileInfo{}
	err := filepath.Walk(s.stateDir, func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			files[filePath] = info
		}

		return nil
	})
	s.Require().NoError(err)

	return files
}

func TestMemFileStateContainerPermissionsTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerPermissionsTestSuite))
}
