package container

import (
	"context"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/mockclock"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type DestroyResourceRetainTestSuite struct {
	suite.Suite
}

func (s *DestroyResourceRetainTestSuite) Test_retain_emits_retained_status_and_skips_provider_destroy() {
	destroyer := NewDefaultResourceDestroyer(&mockclock.StaticClock{}, nil)

	channels := CreateDeployChannels()
	instanceID := "instance-abc"
	resourceID := "resource-xyz"
	resourceName := "ordersTable"

	deployCtx := &DeployContext{
		Channels: channels,
		State:    &defaultDeploymentState{},
		Logger:   core.NewNopLogger(),
		InstanceStateSnapshot: &state.InstanceState{
			InstanceID:   instanceID,
			InstanceName: "my-instance",
			ResourceIDs: map[string]string{
				resourceName: resourceID,
			},
			Resources: map[string]*state.ResourceState{
				resourceID: {
					ResourceID: resourceID,
					Name:       resourceName,
					Type:       "aws/dynamodb/table",
					InstanceID: instanceID,
				},
			},
		},
	}

	element := &ResourceIDInfo{
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Retained:     true,
	}

	go destroyer.Retain(context.Background(), element, instanceID, deployCtx)

	select {
	case msg := <-channels.ResourceUpdateChan:
		s.Equal(resourceID, msg.ResourceID)
		s.Equal(resourceName, msg.ResourceName)
		s.Equal(core.ResourceStatusRetained, msg.Status)
		s.Equal(core.PreciseResourceStatusRetained, msg.PreciseStatus)
	case err := <-channels.ErrChan:
		s.Failf("unexpected error from Retain", "err=%v", err)
	case <-time.After(time.Second):
		s.Fail("timed out waiting for Retained status update")
	}
}

func (s *DestroyResourceRetainTestSuite) Test_retain_reports_error_when_resource_missing_from_state() {
	destroyer := NewDefaultResourceDestroyer(&mockclock.StaticClock{}, nil)

	channels := CreateDeployChannels()
	deployCtx := &DeployContext{
		Channels: channels,
		State:    &defaultDeploymentState{},
		Logger:   core.NewNopLogger(),
		InstanceStateSnapshot: &state.InstanceState{
			InstanceID: "instance-abc",
			Resources:  map[string]*state.ResourceState{},
		},
	}

	element := &ResourceIDInfo{
		ResourceID:   "missing",
		ResourceName: "missing",
		Retained:     true,
	}

	go destroyer.Retain(context.Background(), element, "instance-abc", deployCtx)

	select {
	case <-channels.ResourceUpdateChan:
		s.Fail("did not expect a status update when resource is missing from state")
	case err := <-channels.ErrChan:
		s.Require().Error(err)
	case <-time.After(time.Second):
		s.Fail("timed out waiting for error from Retain")
	}
}

func TestDestroyResourceRetainTestSuite(t *testing.T) {
	suite.Run(t, new(DestroyResourceRetainTestSuite))
}
