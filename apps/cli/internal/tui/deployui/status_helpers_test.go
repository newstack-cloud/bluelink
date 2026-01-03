package deployui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type StatusHelpersTestSuite struct {
	suite.Suite
}

func TestStatusHelpersTestSuite(t *testing.T) {
	suite.Run(t, new(StatusHelpersTestSuite))
}

// IsFailedStatus Tests - Critical for exit codes

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_deploy_failed() {
	s.True(IsFailedStatus(core.InstanceStatusDeployFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_update_failed() {
	s.True(IsFailedStatus(core.InstanceStatusUpdateFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_destroy_failed() {
	s.True(IsFailedStatus(core.InstanceStatusDestroyFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_deploy_rollback_complete() {
	// Rollback complete means original operation failed - should be treated as failure
	s.True(IsFailedStatus(core.InstanceStatusDeployRollbackComplete))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_update_rollback_complete() {
	s.True(IsFailedStatus(core.InstanceStatusUpdateRollbackComplete))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_destroy_rollback_complete() {
	s.True(IsFailedStatus(core.InstanceStatusDestroyRollbackComplete))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_deploy_rollback_failed() {
	s.True(IsFailedStatus(core.InstanceStatusDeployRollbackFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_update_rollback_failed() {
	s.True(IsFailedStatus(core.InstanceStatusUpdateRollbackFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_destroy_rollback_failed() {
	s.True(IsFailedStatus(core.InstanceStatusDestroyRollbackFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_deployed_is_not_failed() {
	s.False(IsFailedStatus(core.InstanceStatusDeployed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_updated_is_not_failed() {
	s.False(IsFailedStatus(core.InstanceStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_IsFailedStatus_deploying_is_not_failed() {
	s.False(IsFailedStatus(core.InstanceStatusDeploying))
}

// IsRollingBackOrFailedStatus Tests

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_deploy_rolling_back() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDeployRollingBack))
}

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_update_rolling_back() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusUpdateRollingBack))
}

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_destroy_rolling_back() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDestroyRollingBack))
}

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_rollback_complete() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDeployRollbackComplete))
}

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_rollback_failed() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDeployRollbackFailed))
}

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_deployed_is_false() {
	s.False(IsRollingBackOrFailedStatus(core.InstanceStatusDeployed))
}

func (s *StatusHelpersTestSuite) Test_IsRollingBackOrFailedStatus_deploying_is_false() {
	s.False(IsRollingBackOrFailedStatus(core.InstanceStatusDeploying))
}

// IsInProgressResourceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsInProgressResourceStatus_creating() {
	s.True(IsInProgressResourceStatus(core.ResourceStatusCreating))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressResourceStatus_updating() {
	s.True(IsInProgressResourceStatus(core.ResourceStatusUpdating))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressResourceStatus_destroying() {
	s.True(IsInProgressResourceStatus(core.ResourceStatusDestroying))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressResourceStatus_rolling_back() {
	s.True(IsInProgressResourceStatus(core.ResourceStatusRollingBack))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressResourceStatus_created_is_false() {
	s.False(IsInProgressResourceStatus(core.ResourceStatusCreated))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressResourceStatus_failed_is_false() {
	s.False(IsInProgressResourceStatus(core.ResourceStatusCreateFailed))
}

// IsInProgressInstanceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsInProgressInstanceStatus_deploying() {
	s.True(IsInProgressInstanceStatus(core.InstanceStatusDeploying))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressInstanceStatus_updating() {
	s.True(IsInProgressInstanceStatus(core.InstanceStatusUpdating))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressInstanceStatus_destroying() {
	s.True(IsInProgressInstanceStatus(core.InstanceStatusDestroying))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressInstanceStatus_deploy_rolling_back() {
	s.True(IsInProgressInstanceStatus(core.InstanceStatusDeployRollingBack))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressInstanceStatus_deployed_is_false() {
	s.False(IsInProgressInstanceStatus(core.InstanceStatusDeployed))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressInstanceStatus_failed_is_false() {
	s.False(IsInProgressInstanceStatus(core.InstanceStatusDeployFailed))
}

// IsInProgressLinkStatus Tests

func (s *StatusHelpersTestSuite) Test_IsInProgressLinkStatus_creating() {
	s.True(IsInProgressLinkStatus(core.LinkStatusCreating))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressLinkStatus_updating() {
	s.True(IsInProgressLinkStatus(core.LinkStatusUpdating))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressLinkStatus_destroying() {
	s.True(IsInProgressLinkStatus(core.LinkStatusDestroying))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressLinkStatus_create_rolling_back() {
	s.True(IsInProgressLinkStatus(core.LinkStatusCreateRollingBack))
}

func (s *StatusHelpersTestSuite) Test_IsInProgressLinkStatus_created_is_false() {
	s.False(IsInProgressLinkStatus(core.LinkStatusCreated))
}

// IsFailedResourceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsFailedResourceStatus_create_failed() {
	s.True(IsFailedResourceStatus(core.ResourceStatusCreateFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedResourceStatus_update_failed() {
	s.True(IsFailedResourceStatus(core.ResourceStatusUpdateFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedResourceStatus_destroy_failed() {
	s.True(IsFailedResourceStatus(core.ResourceStatusDestroyFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedResourceStatus_rollback_failed() {
	s.True(IsFailedResourceStatus(core.ResourceStatusRollbackFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedResourceStatus_created_is_false() {
	s.False(IsFailedResourceStatus(core.ResourceStatusCreated))
}

// IsFailedInstanceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsFailedInstanceStatus_deploy_failed() {
	s.True(IsFailedInstanceStatus(core.InstanceStatusDeployFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedInstanceStatus_update_failed() {
	s.True(IsFailedInstanceStatus(core.InstanceStatusUpdateFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedInstanceStatus_destroy_failed() {
	s.True(IsFailedInstanceStatus(core.InstanceStatusDestroyFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedInstanceStatus_deploy_rollback_failed() {
	s.True(IsFailedInstanceStatus(core.InstanceStatusDeployRollbackFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedInstanceStatus_deployed_is_false() {
	s.False(IsFailedInstanceStatus(core.InstanceStatusDeployed))
}

// IsFailedLinkStatus Tests

func (s *StatusHelpersTestSuite) Test_IsFailedLinkStatus_create_failed() {
	s.True(IsFailedLinkStatus(core.LinkStatusCreateFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedLinkStatus_update_failed() {
	s.True(IsFailedLinkStatus(core.LinkStatusUpdateFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedLinkStatus_destroy_failed() {
	s.True(IsFailedLinkStatus(core.LinkStatusDestroyFailed))
}

func (s *StatusHelpersTestSuite) Test_IsFailedLinkStatus_created_is_false() {
	s.False(IsFailedLinkStatus(core.LinkStatusCreated))
}

// IsInterruptedResourceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsInterruptedResourceStatus_create_interrupted() {
	s.True(IsInterruptedResourceStatus(core.ResourceStatusCreateInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedResourceStatus_update_interrupted() {
	s.True(IsInterruptedResourceStatus(core.ResourceStatusUpdateInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedResourceStatus_destroy_interrupted() {
	s.True(IsInterruptedResourceStatus(core.ResourceStatusDestroyInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedResourceStatus_created_is_false() {
	s.False(IsInterruptedResourceStatus(core.ResourceStatusCreated))
}

// IsInterruptedInstanceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsInterruptedInstanceStatus_deploy_interrupted() {
	s.True(IsInterruptedInstanceStatus(core.InstanceStatusDeployInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedInstanceStatus_update_interrupted() {
	s.True(IsInterruptedInstanceStatus(core.InstanceStatusUpdateInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedInstanceStatus_destroy_interrupted() {
	s.True(IsInterruptedInstanceStatus(core.InstanceStatusDestroyInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedInstanceStatus_deployed_is_false() {
	s.False(IsInterruptedInstanceStatus(core.InstanceStatusDeployed))
}

// IsInterruptedLinkStatus Tests

func (s *StatusHelpersTestSuite) Test_IsInterruptedLinkStatus_create_interrupted() {
	s.True(IsInterruptedLinkStatus(core.LinkStatusCreateInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedLinkStatus_update_interrupted() {
	s.True(IsInterruptedLinkStatus(core.LinkStatusUpdateInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedLinkStatus_destroy_interrupted() {
	s.True(IsInterruptedLinkStatus(core.LinkStatusDestroyInterrupted))
}

func (s *StatusHelpersTestSuite) Test_IsInterruptedLinkStatus_created_is_false() {
	s.False(IsInterruptedLinkStatus(core.LinkStatusCreated))
}

// IsSuccessResourceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsSuccessResourceStatus_created() {
	s.True(IsSuccessResourceStatus(core.ResourceStatusCreated))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessResourceStatus_updated() {
	s.True(IsSuccessResourceStatus(core.ResourceStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessResourceStatus_destroyed() {
	s.True(IsSuccessResourceStatus(core.ResourceStatusDestroyed))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessResourceStatus_rollback_complete() {
	s.True(IsSuccessResourceStatus(core.ResourceStatusRollbackComplete))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessResourceStatus_creating_is_false() {
	s.False(IsSuccessResourceStatus(core.ResourceStatusCreating))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessResourceStatus_failed_is_false() {
	s.False(IsSuccessResourceStatus(core.ResourceStatusCreateFailed))
}

// IsSuccessInstanceStatus Tests

func (s *StatusHelpersTestSuite) Test_IsSuccessInstanceStatus_deployed() {
	s.True(IsSuccessInstanceStatus(core.InstanceStatusDeployed))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessInstanceStatus_updated() {
	s.True(IsSuccessInstanceStatus(core.InstanceStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessInstanceStatus_destroyed() {
	s.True(IsSuccessInstanceStatus(core.InstanceStatusDestroyed))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessInstanceStatus_deploying_is_false() {
	s.False(IsSuccessInstanceStatus(core.InstanceStatusDeploying))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessInstanceStatus_failed_is_false() {
	s.False(IsSuccessInstanceStatus(core.InstanceStatusDeployFailed))
}

// IsSuccessLinkStatus Tests

func (s *StatusHelpersTestSuite) Test_IsSuccessLinkStatus_created() {
	s.True(IsSuccessLinkStatus(core.LinkStatusCreated))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessLinkStatus_updated() {
	s.True(IsSuccessLinkStatus(core.LinkStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessLinkStatus_destroyed() {
	s.True(IsSuccessLinkStatus(core.LinkStatusDestroyed))
}

func (s *StatusHelpersTestSuite) Test_IsSuccessLinkStatus_creating_is_false() {
	s.False(IsSuccessLinkStatus(core.LinkStatusCreating))
}

// ResourceStatusToAction Tests

func (s *StatusHelpersTestSuite) Test_ResourceStatusToAction_created() {
	s.Equal("created", ResourceStatusToAction(core.ResourceStatusCreated))
}

func (s *StatusHelpersTestSuite) Test_ResourceStatusToAction_updated() {
	s.Equal("updated", ResourceStatusToAction(core.ResourceStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_ResourceStatusToAction_destroyed() {
	s.Equal("destroyed", ResourceStatusToAction(core.ResourceStatusDestroyed))
}

func (s *StatusHelpersTestSuite) Test_ResourceStatusToAction_rollback_complete() {
	s.Equal("rolled back", ResourceStatusToAction(core.ResourceStatusRollbackComplete))
}

func (s *StatusHelpersTestSuite) Test_ResourceStatusToAction_unmapped_returns_string() {
	// For unmapped statuses, falls back to String()
	result := ResourceStatusToAction(core.ResourceStatusCreating)
	s.NotEmpty(result)
}

// InstanceStatusToAction Tests

func (s *StatusHelpersTestSuite) Test_InstanceStatusToAction_deployed() {
	s.Equal("deployed", InstanceStatusToAction(core.InstanceStatusDeployed))
}

func (s *StatusHelpersTestSuite) Test_InstanceStatusToAction_updated() {
	s.Equal("updated", InstanceStatusToAction(core.InstanceStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_InstanceStatusToAction_destroyed() {
	s.Equal("destroyed", InstanceStatusToAction(core.InstanceStatusDestroyed))
}

func (s *StatusHelpersTestSuite) Test_InstanceStatusToAction_unmapped_returns_string() {
	result := InstanceStatusToAction(core.InstanceStatusDeploying)
	s.NotEmpty(result)
}

// LinkStatusToAction Tests

func (s *StatusHelpersTestSuite) Test_LinkStatusToAction_created() {
	s.Equal("created", LinkStatusToAction(core.LinkStatusCreated))
}

func (s *StatusHelpersTestSuite) Test_LinkStatusToAction_updated() {
	s.Equal("updated", LinkStatusToAction(core.LinkStatusUpdated))
}

func (s *StatusHelpersTestSuite) Test_LinkStatusToAction_destroyed() {
	s.Equal("destroyed", LinkStatusToAction(core.LinkStatusDestroyed))
}

func (s *StatusHelpersTestSuite) Test_LinkStatusToAction_unmapped_returns_string() {
	result := LinkStatusToAction(core.LinkStatusCreating)
	s.NotEmpty(result)
}
