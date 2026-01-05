package destroyui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type StatusHelpersSuite struct {
	suite.Suite
}

func TestStatusHelpersSuite(t *testing.T) {
	suite.Run(t, new(StatusHelpersSuite))
}

// --- IsRollingBackOrFailedStatus Tests ---

func (s *StatusHelpersSuite) Test_IsRollingBackOrFailedStatus_returns_true_for_rolling_back() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDestroyRollingBack))
}

func (s *StatusHelpersSuite) Test_IsRollingBackOrFailedStatus_returns_true_for_rollback_failed() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDestroyRollbackFailed))
}

func (s *StatusHelpersSuite) Test_IsRollingBackOrFailedStatus_returns_true_for_rollback_complete() {
	s.True(IsRollingBackOrFailedStatus(core.InstanceStatusDestroyRollbackComplete))
}

func (s *StatusHelpersSuite) Test_IsRollingBackOrFailedStatus_returns_false_for_destroying() {
	s.False(IsRollingBackOrFailedStatus(core.InstanceStatusDestroying))
}

func (s *StatusHelpersSuite) Test_IsRollingBackOrFailedStatus_returns_false_for_destroyed() {
	s.False(IsRollingBackOrFailedStatus(core.InstanceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_IsRollingBackOrFailedStatus_returns_false_for_preparing() {
	s.False(IsRollingBackOrFailedStatus(core.InstanceStatusPreparing))
}

// --- IsRollingBackStatus Tests ---

func (s *StatusHelpersSuite) Test_IsRollingBackStatus_returns_true_for_rolling_back() {
	s.True(IsRollingBackStatus(core.InstanceStatusDestroyRollingBack))
}

func (s *StatusHelpersSuite) Test_IsRollingBackStatus_returns_false_for_rollback_complete() {
	s.False(IsRollingBackStatus(core.InstanceStatusDestroyRollbackComplete))
}

// --- IsFailedStatus Tests ---

func (s *StatusHelpersSuite) Test_IsFailedStatus_returns_true_for_destroy_failed() {
	s.True(IsFailedStatus(core.InstanceStatusDestroyFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedStatus_returns_true_for_rollback_complete() {
	s.True(IsFailedStatus(core.InstanceStatusDestroyRollbackComplete))
}

func (s *StatusHelpersSuite) Test_IsFailedStatus_returns_true_for_rollback_failed() {
	s.True(IsFailedStatus(core.InstanceStatusDestroyRollbackFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedStatus_returns_false_for_destroyed() {
	s.False(IsFailedStatus(core.InstanceStatusDestroyed))
}

// --- IsInProgressResourceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsInProgressResourceStatus_returns_true_for_destroying() {
	s.True(IsInProgressResourceStatus(core.ResourceStatusDestroying))
}

func (s *StatusHelpersSuite) Test_IsInProgressResourceStatus_returns_true_for_rolling_back() {
	s.True(IsInProgressResourceStatus(core.ResourceStatusRollingBack))
}

func (s *StatusHelpersSuite) Test_IsInProgressResourceStatus_returns_false_for_destroyed() {
	s.False(IsInProgressResourceStatus(core.ResourceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_IsInProgressResourceStatus_returns_false_for_failed() {
	s.False(IsInProgressResourceStatus(core.ResourceStatusDestroyFailed))
}

// --- IsInProgressInstanceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsInProgressInstanceStatus_returns_true_for_destroying() {
	s.True(IsInProgressInstanceStatus(core.InstanceStatusDestroying))
}

func (s *StatusHelpersSuite) Test_IsInProgressInstanceStatus_returns_true_for_rolling_back() {
	s.True(IsInProgressInstanceStatus(core.InstanceStatusDestroyRollingBack))
}

func (s *StatusHelpersSuite) Test_IsInProgressInstanceStatus_returns_false_for_destroyed() {
	s.False(IsInProgressInstanceStatus(core.InstanceStatusDestroyed))
}

// --- IsInProgressLinkStatus Tests ---

func (s *StatusHelpersSuite) Test_IsInProgressLinkStatus_returns_true_for_destroying() {
	s.True(IsInProgressLinkStatus(core.LinkStatusDestroying))
}

func (s *StatusHelpersSuite) Test_IsInProgressLinkStatus_returns_true_for_rolling_back() {
	s.True(IsInProgressLinkStatus(core.LinkStatusDestroyRollingBack))
}

func (s *StatusHelpersSuite) Test_IsInProgressLinkStatus_returns_false_for_destroyed() {
	s.False(IsInProgressLinkStatus(core.LinkStatusDestroyed))
}

// --- IsFailedResourceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsFailedResourceStatus_returns_true_for_destroy_failed() {
	s.True(IsFailedResourceStatus(core.ResourceStatusDestroyFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedResourceStatus_returns_true_for_rollback_failed() {
	s.True(IsFailedResourceStatus(core.ResourceStatusRollbackFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedResourceStatus_returns_false_for_destroyed() {
	s.False(IsFailedResourceStatus(core.ResourceStatusDestroyed))
}

// --- IsFailedInstanceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsFailedInstanceStatus_returns_true_for_destroy_failed() {
	s.True(IsFailedInstanceStatus(core.InstanceStatusDestroyFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedInstanceStatus_returns_true_for_rollback_failed() {
	s.True(IsFailedInstanceStatus(core.InstanceStatusDestroyRollbackFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedInstanceStatus_returns_false_for_destroyed() {
	s.False(IsFailedInstanceStatus(core.InstanceStatusDestroyed))
}

// --- IsFailedLinkStatus Tests ---

func (s *StatusHelpersSuite) Test_IsFailedLinkStatus_returns_true_for_destroy_failed() {
	s.True(IsFailedLinkStatus(core.LinkStatusDestroyFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedLinkStatus_returns_true_for_rollback_failed() {
	s.True(IsFailedLinkStatus(core.LinkStatusDestroyRollbackFailed))
}

func (s *StatusHelpersSuite) Test_IsFailedLinkStatus_returns_false_for_destroyed() {
	s.False(IsFailedLinkStatus(core.LinkStatusDestroyed))
}

// --- IsInterruptedResourceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsInterruptedResourceStatus_returns_true_for_interrupted() {
	s.True(IsInterruptedResourceStatus(core.ResourceStatusDestroyInterrupted))
}

func (s *StatusHelpersSuite) Test_IsInterruptedResourceStatus_returns_false_for_destroyed() {
	s.False(IsInterruptedResourceStatus(core.ResourceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_IsInterruptedResourceStatus_returns_false_for_destroying() {
	s.False(IsInterruptedResourceStatus(core.ResourceStatusDestroying))
}

// --- IsInterruptedInstanceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsInterruptedInstanceStatus_returns_true_for_interrupted() {
	s.True(IsInterruptedInstanceStatus(core.InstanceStatusDestroyInterrupted))
}

func (s *StatusHelpersSuite) Test_IsInterruptedInstanceStatus_returns_false_for_destroyed() {
	s.False(IsInterruptedInstanceStatus(core.InstanceStatusDestroyed))
}

// --- IsInterruptedLinkStatus Tests ---

func (s *StatusHelpersSuite) Test_IsInterruptedLinkStatus_returns_true_for_interrupted() {
	s.True(IsInterruptedLinkStatus(core.LinkStatusDestroyInterrupted))
}

func (s *StatusHelpersSuite) Test_IsInterruptedLinkStatus_returns_false_for_destroyed() {
	s.False(IsInterruptedLinkStatus(core.LinkStatusDestroyed))
}

// --- IsSuccessResourceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsSuccessResourceStatus_returns_true_for_destroyed() {
	s.True(IsSuccessResourceStatus(core.ResourceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_IsSuccessResourceStatus_returns_true_for_rollback_complete() {
	s.True(IsSuccessResourceStatus(core.ResourceStatusRollbackComplete))
}

func (s *StatusHelpersSuite) Test_IsSuccessResourceStatus_returns_false_for_failed() {
	s.False(IsSuccessResourceStatus(core.ResourceStatusDestroyFailed))
}

func (s *StatusHelpersSuite) Test_IsSuccessResourceStatus_returns_false_for_destroying() {
	s.False(IsSuccessResourceStatus(core.ResourceStatusDestroying))
}

// --- IsSuccessInstanceStatus Tests ---

func (s *StatusHelpersSuite) Test_IsSuccessInstanceStatus_returns_true_for_destroyed() {
	s.True(IsSuccessInstanceStatus(core.InstanceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_IsSuccessInstanceStatus_returns_false_for_failed() {
	s.False(IsSuccessInstanceStatus(core.InstanceStatusDestroyFailed))
}

func (s *StatusHelpersSuite) Test_IsSuccessInstanceStatus_returns_false_for_destroying() {
	s.False(IsSuccessInstanceStatus(core.InstanceStatusDestroying))
}

// --- IsSuccessLinkStatus Tests ---

func (s *StatusHelpersSuite) Test_IsSuccessLinkStatus_returns_true_for_destroyed() {
	s.True(IsSuccessLinkStatus(core.LinkStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_IsSuccessLinkStatus_returns_false_for_failed() {
	s.False(IsSuccessLinkStatus(core.LinkStatusDestroyFailed))
}

// --- ResourceStatusToAction Tests ---

func (s *StatusHelpersSuite) Test_ResourceStatusToAction_returns_destroyed_for_destroyed() {
	s.Equal("destroyed", ResourceStatusToAction(core.ResourceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_ResourceStatusToAction_returns_rolled_back_for_rollback_complete() {
	s.Equal("rolled back", ResourceStatusToAction(core.ResourceStatusRollbackComplete))
}

func (s *StatusHelpersSuite) Test_ResourceStatusToAction_returns_string_for_unknown_status() {
	// For statuses not in the map, should return the String() value
	action := ResourceStatusToAction(core.ResourceStatusDestroying)
	s.NotEmpty(action)
}

// --- InstanceStatusToAction Tests ---

func (s *StatusHelpersSuite) Test_InstanceStatusToAction_returns_destroyed_for_destroyed() {
	s.Equal("destroyed", InstanceStatusToAction(core.InstanceStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_InstanceStatusToAction_returns_string_for_unknown_status() {
	action := InstanceStatusToAction(core.InstanceStatusDestroying)
	s.NotEmpty(action)
}

// --- LinkStatusToAction Tests ---

func (s *StatusHelpersSuite) Test_LinkStatusToAction_returns_destroyed_for_destroyed() {
	s.Equal("destroyed", LinkStatusToAction(core.LinkStatusDestroyed))
}

func (s *StatusHelpersSuite) Test_LinkStatusToAction_returns_string_for_unknown_status() {
	action := LinkStatusToAction(core.LinkStatusDestroying)
	s.NotEmpty(action)
}
