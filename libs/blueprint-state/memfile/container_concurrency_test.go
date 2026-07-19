package memfile

import (
	"context"
	"fmt"
	"path"
	"sync"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

// Regression suite for the data race between link saves and state reads
// observed during parallel link deployments (concurrent map access on the
// shared instance state). Run with -race to exercise the guarantees.
type MemFileStateContainerConcurrencyTestSuite struct {
	container state.Container
	suite.Suite
}

func (s *MemFileStateContainerConcurrencyTestSuite) SetupTest() {
	stateDir := path.Join("__testdata", "initial-state")
	memoryFS := afero.NewMemMapFs()
	loadMemoryFS(stateDir, memoryFS, &s.Suite)
	container, err := LoadStateContainer(stateDir, memoryFS, core.NewNopLogger())
	s.Require().NoError(err)
	s.container = container
}

func (s *MemFileStateContainerConcurrencyTestSuite) Test_concurrent_link_saves_and_state_reads() {
	ctx := context.Background()
	links := s.container.Links()
	instances := s.container.Instances()

	const iterations = 50
	errs := make(chan error, iterations*2)
	var wg sync.WaitGroup
	for i := range iterations {
		wg.Add(3)
		go func() {
			defer wg.Done()
			errs <- links.Save(ctx, state.LinkState{
				LinkID:     fmt.Sprintf("concurrent-link-%d", i),
				Name:       fmt.Sprintf("resourceA::resourceB%d", i),
				InstanceID: existingLinkInstanceID,
			})
		}()
		go func() {
			defer wg.Done()
			_, err := links.GetByName(ctx, existingLinkInstanceID, existingLinkName)
			errs <- err
		}()
		go func() {
			defer wg.Done()
			// Instances().Get copies the full instance including the links
			// map that the concurrent saves mutate.
			_, _ = instances.Get(ctx, existingLinkInstanceID)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		s.Require().NoError(err)
	}
}

func TestMemFileStateContainerConcurrencyTestSuite(t *testing.T) {
	suite.Run(t, new(MemFileStateContainerConcurrencyTestSuite))
}
