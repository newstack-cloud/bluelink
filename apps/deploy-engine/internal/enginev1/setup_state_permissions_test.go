package enginev1

import (
	"net"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/stretchr/testify/suite"
)

// Exercised against the real filesystem rather than an in-memory one, since what is being
// asserted is the mode that lands on disk, umask included.
type SetupStatePermissionsTestSuite struct {
	suite.Suite
}

func (s *SetupStatePermissionsTestSuite) SetupTest() {
	if runtime.GOOS == "windows" {
		s.T().Skip("mode bits are not the access control on windows")
	}
}

// Runs Setup against the given state directory, which is as close as the engine gets to a
// seam for the directory preparation that happens at startup.
func (s *SetupStatePermissionsTestSuite) setup(stateDir string) {
	config, err := core.LoadConfig()
	s.Require().NoError(err)

	config.State.StorageEngine = memfileStorageEngine
	config.State.MemFileStateDir = stateDir

	// Port 0 so the plugin service listener does not collide with another test or a running
	// engine.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer listener.Close()

	_, cleanup, setupErr := Setup(mux.NewRouter(), &config, listener)
	if cleanup != nil {
		s.T().Cleanup(cleanup)
	}
	s.Require().NoError(setupErr)
}

func (s *SetupStatePermissionsTestSuite) Test_creates_the_state_directory_private_to_its_owner() {
	stateDir := path.Join(s.T().TempDir(), "state")

	s.setup(stateDir)

	info, err := os.Stat(stateDir)
	s.Require().NoError(err)
	s.Assert().Equal(os.FileMode(0o700), info.Mode().Perm())
}

// A deployment upgrading from a version that created the directory world-readable has to end
// up protected too, and MkdirAll leaves an existing directory's mode alone.
func (s *SetupStatePermissionsTestSuite) Test_tightens_a_state_directory_left_open_by_an_earlier_version() {
	stateDir := path.Join(s.T().TempDir(), "state")
	s.Require().NoError(os.MkdirAll(stateDir, 0o755))
	// MkdirAll applies the umask, so the mode is set explicitly to get a permissive one.
	s.Require().NoError(os.Chmod(stateDir, 0o755))

	s.setup(stateDir)

	info, err := os.Stat(stateDir)
	s.Require().NoError(err)
	s.Assert().Equal(os.FileMode(0o700), info.Mode().Perm())
}

func TestSetupStatePermissionsTestSuite(t *testing.T) {
	suite.Run(t, new(SetupStatePermissionsTestSuite))
}
