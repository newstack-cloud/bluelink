package objectstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// Drift describes a resource or link, so it goes when they do, as it does in postgres where
// the drift tables cascade on delete. Nothing else would ever remove the drift object. Each
// step runs in a fresh engine, so the drift is only known from the bucket.
type DriftRemovalSuite struct {
	objectstoreSuite
}

func (s *DriftRemovalSuite) SetupTest() {
	s.objectstoreSuite.SetupTest()
	ctx := context.Background()
	s.Require().NoError(s.newContainer().Instances().Save(ctx, recordsInstance()))

	timestamp := 1733145728
	s.Require().NoError(s.newContainer().Resources().SaveDrift(ctx, state.ResourceDriftState{
		ResourceID:   recordsResourceID,
		ResourceName: recordsResourceName,
		Timestamp:    &timestamp,
	}))
	s.Require().NoError(s.newContainer().Links().SaveDrift(ctx, state.LinkDriftState{
		LinkID:    recordsLinkID,
		LinkName:  recordsLinkName,
		Timestamp: &timestamp,
	}))
	s.Require().NotEmpty(s.objectKeys(testPrefix + "resource_drift/"))
	s.Require().NotEmpty(s.objectKeys(testPrefix + "link_drift/"))
}

func (s *DriftRemovalSuite) Test_removing_a_resource_deletes_its_drift() {
	_, err := s.newContainer().Resources().Remove(context.Background(), recordsResourceID)
	s.Require().NoError(err)

	s.Assert().Empty(s.objectKeys(testPrefix + "resource_drift/"))
}

func (s *DriftRemovalSuite) Test_removing_a_link_deletes_its_drift() {
	_, err := s.newContainer().Links().Remove(context.Background(), recordsLinkID)
	s.Require().NoError(err)

	s.Assert().Empty(s.objectKeys(testPrefix + "link_drift/"))
}

func (s *DriftRemovalSuite) Test_removing_an_instance_deletes_the_drift_of_its_resources_and_links() {
	_, err := s.newContainer().Instances().Remove(context.Background(), recordsInstanceID)
	s.Require().NoError(err)

	s.Assert().Empty(s.objectKeys(testPrefix + "resource_drift/"))
	s.Assert().Empty(s.objectKeys(testPrefix + "link_drift/"))
}

func (s *DriftRemovalSuite) Test_finishing_an_interrupted_instance_removal_deletes_the_drift() {
	ctx := context.Background()
	s.svc.SetHooks(mockservice.Hooks{
		BeforeDelete: func(_ context.Context, key string) error {
			if strings.HasSuffix(key, "instances/"+recordsInstanceID+".json") {
				return errInjected
			}
			return nil
		},
	})
	_, err := s.newContainer().Instances().Remove(ctx, recordsInstanceID)
	s.Require().ErrorIs(err, errInjected)
	s.svc.SetHooks(mockservice.Hooks{})
	s.Require().NotEmpty(s.objectKeys(testPrefix + "resource_drift/"))

	_, err = s.newContainer().Instances().Remove(ctx, recordsInstanceID)
	s.Require().NoError(err)

	s.Assert().Empty(s.objectKeys(testPrefix))
}

func TestDriftRemovalSuite(t *testing.T) {
	suite.Run(t, new(DriftRemovalSuite))
}
