package memfile

import (
	"os"
	"path"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

func loadMemoryFS(stateDir string, fs afero.Fs, s *suite.Suite) {
	dirEntries, err := os.ReadDir(stateDir)
	s.Require().NoError(err)

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			dirName := dirEntry.Name()
			fs.Mkdir(dirName, 0755)
			loadMemoryFS(path.Join(stateDir, dirName), fs, s)
		} else {
			fileName := dirEntry.Name()
			fileBytes, err := os.ReadFile(path.Join(stateDir, fileName))
			s.Require().NoError(err)

			err = afero.WriteFile(fs, path.Join(stateDir, fileName), fileBytes, 0644)
			s.Require().NoError(err)
		}
	}
}

func loadMalformedStateContainer(s *suite.Suite) (state.Container, error) {
	stateDir := path.Join("__testdata", "malformed-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, s)
	return LoadStateContainer(stateDir, memoryFS, core.NewNopLogger())
}
