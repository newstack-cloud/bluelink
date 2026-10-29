package memfile

import (
	"context"
	"encoding/json"
	"path"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

// Drift describes a resource or link, so it goes when they do, as it does in postgres where
// the drift tables cascade on delete. Nothing else would ever remove it.
type DriftRemovalTestSuite struct {
	container state.Container
	stateDir  string
	fs        afero.Fs
	suite.Suite
}

func (s *DriftRemovalTestSuite) SetupTest() {
	s.stateDir = path.Join("__testdata", "initial-state")
	s.fs = afero.NewMemMapFs()
	loadMemoryFS(s.stateDir, s.fs, &s.Suite)
	container, err := LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *DriftRemovalTestSuite) Test_removing_a_resource_deletes_its_drift() {
	s.Require().Contains(s.persistedDriftIDs("resource_drift_index.json"), existingResourceID)

	_, err := s.container.Resources().Remove(context.Background(), existingResourceID)
	s.Require().NoError(err)

	s.Assert().NotContains(s.persistedDriftIDs("resource_drift_index.json"), existingResourceID)
}

func (s *DriftRemovalTestSuite) Test_removing_a_link_deletes_its_drift() {
	s.Require().Contains(s.persistedDriftIDs("link_drift_index.json"), existingLinkID)

	_, err := s.container.Links().Remove(context.Background(), existingLinkID)
	s.Require().NoError(err)

	s.Assert().NotContains(s.persistedDriftIDs("link_drift_index.json"), existingLinkID)
}

func (s *DriftRemovalTestSuite) Test_removing_an_instance_deletes_the_drift_of_its_resources_and_links() {
	_, err := s.container.Instances().Remove(context.Background(), "blueprint-instance-1")
	s.Require().NoError(err)

	s.Assert().NotContains(s.persistedDriftIDs("resource_drift_index.json"), existingResourceID)
	s.Assert().NotContains(s.persistedDriftIDs("link_drift_index.json"), existingLinkID)
}

// The chunk index names every persisted drift entry, so an ID missing from it is gone from
// the chunks as well.
func (s *DriftRemovalTestSuite) persistedDriftIDs(indexFile string) []string {
	data, err := afero.ReadFile(s.fs, path.Join(s.stateDir, indexFile))
	s.Require().NoError(err)

	index := map[string]json.RawMessage{}
	s.Require().NoError(json.Unmarshal(data, &index))

	ids := make([]string, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}
	return ids
}

func TestDriftRemovalTestSuite(t *testing.T) {
	suite.Run(t, new(DriftRemovalTestSuite))
}
