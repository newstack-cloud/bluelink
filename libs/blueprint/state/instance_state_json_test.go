package state_test

import (
	"encoding/json"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type InstanceStateJSONTestSuite struct {
	suite.Suite
}

func (s *InstanceStateJSONTestSuite) Test_absent_maps_serialise_as_empty_objects() {
	serialised, err := json.Marshal(&state.InstanceState{
		InstanceID: "instance-1",
		ChildBlueprints: map[string]*state.InstanceState{
			"child": {InstanceID: "child-1"},
		},
	})
	s.Require().NoError(err)

	s.Assert().JSONEq(
		`{
			"id": "instance-1",
			"name": "",
			"status": 0,
			"lastDeployedTimestamp": 0,
			"lastDeployAttemptTimestamp": 0,
			"resourceIds": {},
			"resources": {},
			"links": {},
			"metadata": {},
			"exports": {},
			"childBlueprints": {
				"child": {
					"id": "child-1",
					"name": "",
					"status": 0,
					"lastDeployedTimestamp": 0,
					"lastDeployAttemptTimestamp": 0,
					"resourceIds": {},
					"resources": {},
					"links": {},
					"metadata": {},
					"exports": {},
					"childBlueprints": {},
					"version": 0
				}
			},
			"version": 0
		}`,
		string(serialised),
	)
}

func (s *InstanceStateJSONTestSuite) Test_round_trips_through_json() {
	instance := state.InstanceState{
		InstanceID:  "instance-1",
		ResourceIDs: map[string]string{"resourceA": "resource-1"},
		Version:     3,
	}
	serialised, err := json.Marshal(instance)
	s.Require().NoError(err)

	decoded := state.InstanceState{}
	s.Require().NoError(json.Unmarshal(serialised, &decoded))
	s.Assert().Equal(instance.ResourceIDs, decoded.ResourceIDs)
	s.Assert().Equal(int64(3), decoded.Version)
}

func TestInstanceStateJSONTestSuite(t *testing.T) {
	suite.Run(t, new(InstanceStateJSONTestSuite))
}
