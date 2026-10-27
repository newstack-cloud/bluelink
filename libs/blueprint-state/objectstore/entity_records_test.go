package objectstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

const (
	recordsInstanceID   = "records-instance"
	recordsInstanceName = "RecordsInstance"
	recordsResourceID   = "records-resource"
	recordsResourceName = "recordsTable"
	recordsLinkID       = "records-link"
	recordsLinkName     = "recordsFunction::recordsTable"
)

var errInjected = errors.New("injected failure")

// Resources and links are found by ID through records naming the instance that holds them.
// These tests check what reaches the bucket, including when a write sequence is cut short.
type EntityRecordsSuite struct {
	objectstoreSuite
}

func (s *EntityRecordsSuite) Test_saving_an_instance_writes_a_record_for_each_resource_and_link() {
	s.Require().NoError(s.newContainer().Instances().Save(context.Background(), recordsInstance()))

	s.Assert().Equal(recordsInstanceID, s.recordedInstanceID("resources_by_id/"+recordsResourceID))
	s.Assert().Equal(recordsInstanceID, s.recordedInstanceID("links_by_id/"+recordsLinkID))

	fresh := s.newContainer()
	_, err := fresh.Resources().Get(context.Background(), recordsResourceID)
	s.Assert().NoError(err)
	_, err = fresh.Links().Get(context.Background(), recordsLinkID)
	s.Assert().NoError(err)
}

func (s *EntityRecordsSuite) Test_removing_a_resource_or_link_deletes_its_record() {
	s.Require().NoError(s.newContainer().Instances().Save(context.Background(), recordsInstance()))

	s.Require().Equal(
		[]string{testPrefix + "resources_by_id/" + recordsResourceID + ".json"},
		s.objectKeys(testPrefix+"resources_by_id/"),
	)
	s.Require().Equal(
		[]string{testPrefix + "links_by_id/" + recordsLinkID + ".json"},
		s.objectKeys(testPrefix+"links_by_id/"),
	)
	s.Require().Equal(recordsInstanceID, s.recordedInstanceID("resources_by_id/"+recordsResourceID))
	s.Require().Equal(recordsInstanceID, s.recordedInstanceID("links_by_id/"+recordsLinkID))

	_, err := s.newContainer().Resources().Remove(context.Background(), recordsResourceID)
	s.Require().NoError(err)
	_, err = s.newContainer().Links().Remove(context.Background(), recordsLinkID)
	s.Require().NoError(err)

	s.Assert().Empty(s.objectKeys(testPrefix + "resources_by_id/"))
	s.Assert().Empty(s.objectKeys(testPrefix + "links_by_id/"))
}

// Whichever deletion fails, the instance object is still there, so no record is left
// pointing at a missing instance; the instance reads as not found; and removing it again
// finishes the removal.
func (s *EntityRecordsSuite) Test_an_instance_removal_cut_short_at_any_step_resumes_cleanly() {
	for _, failingKey := range []string{
		"resources_by_id/" + recordsResourceID + ".json",
		"links_by_id/" + recordsLinkID + ".json",
		"instances_by_name/" + recordsInstanceName + ".json",
		"instances/" + recordsInstanceID + ".json",
	} {
		s.Run(failingKey, func() {
			ctx := context.Background()
			s.Require().NoError(s.newContainer().Instances().Save(ctx, recordsInstance()))

			s.svc.SetHooks(mockservice.Hooks{
				BeforeDelete: func(_ context.Context, key string) error {
					if strings.HasSuffix(key, failingKey) {
						return errInjected
					}
					return nil
				},
			})
			_, err := s.newContainer().Instances().Remove(ctx, recordsInstanceID)
			s.Require().ErrorIs(err, errInjected)
			s.svc.SetHooks(mockservice.Hooks{})

			s.assertNoRecordOutlivesItsInstance()
			_, err = s.newContainer().Instances().Get(ctx, recordsInstanceID)
			s.Assert().True(state.IsInstanceNotFound(err), "got %v", err)

			_, err = s.newContainer().Instances().Remove(ctx, recordsInstanceID)
			s.Require().NoError(err)
			s.Assert().Empty(s.objectKeys(testPrefix))
		})
	}
}

func (s *EntityRecordsSuite) Test_initialising_an_instance_with_resources_writes_their_records() {
	_, err := s.newContainer().Instances().InitialiseAndClaim(
		context.Background(), recordsInstance(), core.InstanceStatusPreparing,
	)
	s.Require().NoError(err)

	s.Assert().Equal(recordsInstanceID, s.recordedInstanceID("resources_by_id/"+recordsResourceID))
	_, err = s.newContainer().Resources().Get(context.Background(), recordsResourceID)
	s.Assert().NoError(err)
}

// The name is reserved before the instance object is created, so a clash on the instance
// object must give the reservation back rather than leave a second name for the instance.
func (s *EntityRecordsSuite) Test_initialising_an_existing_instance_under_another_name_reserves_nothing() {
	ctx := context.Background()
	s.Require().NoError(s.newContainer().Instances().Save(ctx, recordsInstance()))

	other := recordsInstance()
	other.InstanceName = "OtherRecordsInstance"
	_, err := s.newContainer().Instances().InitialiseAndClaim(ctx, other, core.InstanceStatusPreparing)
	s.Require().ErrorIs(err, state.ErrInstanceAlreadyExists)

	s.Assert().Equal(
		[]string{testPrefix + "instances_by_name/" + recordsInstanceName + ".json"},
		s.objectKeys(testPrefix+"instances_by_name/"),
	)
}

// Each subtest starts from an empty bucket.
func (s *EntityRecordsSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *EntityRecordsSuite) recordedInstanceID(keyWithoutExtension string) string {
	data, _, err := s.svc.Get(context.Background(), testPrefix+keyWithoutExtension+".json")
	s.Require().NoError(err)
	record := struct {
		InstanceID string `json:"instanceId"`
	}{}
	s.Require().NoError(json.Unmarshal(data, &record))
	return record.InstanceID
}

func (s *EntityRecordsSuite) assertNoRecordOutlivesItsInstance() {
	for _, prefix := range []string{"resources_by_id/", "links_by_id/"} {
		for _, key := range s.objectKeys(testPrefix + prefix) {
			recordName := strings.TrimSuffix(strings.TrimPrefix(key, testPrefix), ".json")
			instanceID := s.recordedInstanceID(recordName)
			_, err := s.svc.Head(context.Background(), testPrefix+"instances/"+instanceID+".json")
			s.Assert().NoError(err, "record %s names missing instance %s", key, instanceID)
		}
	}
}

func recordsInstance() state.InstanceState {
	instance := eqInstance(
		recordsInstanceID, recordsInstanceName,
		[]*state.ResourceState{eqResource(recordsInstanceID, recordsResourceID, recordsResourceName, "aws/dynamodb/table")},
		map[string]*state.InstanceState{},
	)
	instance.Links[recordsLinkName] = &state.LinkState{
		LinkID:     recordsLinkID,
		Name:       recordsLinkName,
		InstanceID: recordsInstanceID,
	}
	return *instance
}

func TestEntityRecordsSuite(t *testing.T) {
	suite.Run(t, new(EntityRecordsSuite))
}
