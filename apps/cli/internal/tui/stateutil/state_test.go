package stateutil

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type StateUtilTestSuite struct {
	suite.Suite
}

func TestStateUtilTestSuite(t *testing.T) {
	suite.Run(t, new(StateUtilTestSuite))
}

func (s *StateUtilTestSuite) Test_FindResourceState_returns_nil_for_nil_instance() {
	result := FindResourceState(nil, "resource1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindResourceState_returns_nil_for_nil_resource_ids() {
	instanceState := &state.InstanceState{
		Resources: map[string]*state.ResourceState{
			"res-123": {Name: "resource1"},
		},
	}
	result := FindResourceState(instanceState, "resource1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindResourceState_returns_nil_for_nil_resources() {
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"resource1": "res-123",
		},
	}
	result := FindResourceState(instanceState, "resource1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindResourceState_returns_nil_for_unknown_name() {
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"resource1": "res-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-123": {Name: "resource1"},
		},
	}
	result := FindResourceState(instanceState, "unknown")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindResourceState_returns_resource_by_name() {
	expectedResource := &state.ResourceState{
		Name: "resource1",
		Type: "aws/s3/bucket",
	}
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"resource1": "res-123",
		},
		Resources: map[string]*state.ResourceState{
			"res-123": expectedResource,
		},
	}
	result := FindResourceState(instanceState, "resource1")
	s.Equal(expectedResource, result)
}

func (s *StateUtilTestSuite) Test_FindLinkState_returns_nil_for_nil_instance() {
	result := FindLinkState(nil, "link1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindLinkState_returns_nil_for_nil_links() {
	instanceState := &state.InstanceState{}
	result := FindLinkState(instanceState, "link1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindLinkState_returns_nil_for_unknown_name() {
	instanceState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"link1": {Name: "link1"},
		},
	}
	result := FindLinkState(instanceState, "unknown")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindLinkState_returns_link_by_name() {
	expectedLink := &state.LinkState{
		Name: "link1",
	}
	instanceState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"link1": expectedLink,
		},
	}
	result := FindLinkState(instanceState, "link1")
	s.Equal(expectedLink, result)
}

func (s *StateUtilTestSuite) Test_FindChildInstanceState_returns_nil_for_nil_instance() {
	result := FindChildInstanceState(nil, "child1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindChildInstanceState_returns_nil_for_nil_children() {
	instanceState := &state.InstanceState{}
	result := FindChildInstanceState(instanceState, "child1")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindChildInstanceState_returns_nil_for_unknown_name() {
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"child1": {InstanceName: "child1"},
		},
	}
	result := FindChildInstanceState(instanceState, "unknown")
	s.Nil(result)
}

func (s *StateUtilTestSuite) Test_FindChildInstanceState_returns_child_by_name() {
	expectedChild := &state.InstanceState{
		InstanceName: "child1",
	}
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"child1": expectedChild,
		},
	}
	result := FindChildInstanceState(instanceState, "child1")
	s.Equal(expectedChild, result)
}

func (s *StateUtilTestSuite) Test_FindResourceState_handles_multiple_resources() {
	resource1 := &state.ResourceState{Name: "resource1"}
	resource2 := &state.ResourceState{Name: "resource2"}
	resource3 := &state.ResourceState{Name: "resource3"}

	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{
			"resource1": "res-1",
			"resource2": "res-2",
			"resource3": "res-3",
		},
		Resources: map[string]*state.ResourceState{
			"res-1": resource1,
			"res-2": resource2,
			"res-3": resource3,
		},
	}

	s.Equal(resource1, FindResourceState(instanceState, "resource1"))
	s.Equal(resource2, FindResourceState(instanceState, "resource2"))
	s.Equal(resource3, FindResourceState(instanceState, "resource3"))
}

func (s *StateUtilTestSuite) Test_FindChildInstanceState_nested_children() {
	grandchild := &state.InstanceState{
		InstanceName: "grandchild",
	}
	child := &state.InstanceState{
		InstanceName: "child",
		ChildBlueprints: map[string]*state.InstanceState{
			"grandchild": grandchild,
		},
	}
	parent := &state.InstanceState{
		InstanceName: "parent",
		ChildBlueprints: map[string]*state.InstanceState{
			"child": child,
		},
	}

	foundChild := FindChildInstanceState(parent, "child")
	s.Equal(child, foundChild)

	foundGrandchild := FindChildInstanceState(foundChild, "grandchild")
	s.Equal(grandchild, foundGrandchild)
}
