package container

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ChangeStagingRemoveUtilsTestSuite struct {
	suite.Suite
}

func (s *ChangeStagingRemoveUtilsTestSuite) Test_splits_resources_by_removal_policy() {
	instance := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"deleteMe":  {Name: "deleteMe"},
			"retainMe":  {Name: "retainMe", RemovalPolicy: string(schema.RemovalPolicyRetain)},
			"deleteToo": {Name: "deleteToo", RemovalPolicy: string(schema.RemovalPolicyDelete)},
		},
	}

	bpChanges := getInstanceRemovalChanges(instance)

	s.ElementsMatch(bpChanges.RemovedResources, []string{"deleteMe", "deleteToo"})
	s.ElementsMatch(bpChanges.RetainedResources, []string{"retainMe"})
}

func (s *ChangeStagingRemoveUtilsTestSuite) Test_effective_removal_policy_defaults_to_delete() {
	s.Equal(string(schema.RemovalPolicyDelete), effectiveRemovalPolicy(""))
	s.Equal(string(schema.RemovalPolicyDelete), effectiveRemovalPolicy("delete"))
	s.Equal(string(schema.RemovalPolicyDelete), effectiveRemovalPolicy("unrecognised"))
	s.Equal(string(schema.RemovalPolicyRetain), effectiveRemovalPolicy("retain"))
}

func TestChangeStagingRemoveUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeStagingRemoveUtilsTestSuite))
}
