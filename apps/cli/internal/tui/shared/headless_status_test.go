package shared

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type HeadlessStatusTestSuite struct {
	suite.Suite
}

func TestHeadlessStatusTestSuite(t *testing.T) {
	suite.Run(t, new(HeadlessStatusTestSuite))
}

// Resource Status Icon Tests

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_creating() {
	s.Equal("...", ResourceStatusHeadlessIcon(core.ResourceStatusCreating))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_updating() {
	s.Equal("...", ResourceStatusHeadlessIcon(core.ResourceStatusUpdating))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_destroying() {
	s.Equal("...", ResourceStatusHeadlessIcon(core.ResourceStatusDestroying))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_created() {
	s.Equal("OK", ResourceStatusHeadlessIcon(core.ResourceStatusCreated))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_updated() {
	s.Equal("OK", ResourceStatusHeadlessIcon(core.ResourceStatusUpdated))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_destroyed() {
	s.Equal("OK", ResourceStatusHeadlessIcon(core.ResourceStatusDestroyed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_create_failed() {
	s.Equal("ERR", ResourceStatusHeadlessIcon(core.ResourceStatusCreateFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_update_failed() {
	s.Equal("ERR", ResourceStatusHeadlessIcon(core.ResourceStatusUpdateFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_destroy_failed() {
	s.Equal("ERR", ResourceStatusHeadlessIcon(core.ResourceStatusDestroyFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_rolling_back() {
	s.Equal("<-", ResourceStatusHeadlessIcon(core.ResourceStatusRollingBack))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_rollback_failed() {
	s.Equal("!!", ResourceStatusHeadlessIcon(core.ResourceStatusRollbackFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_rollback_complete() {
	s.Equal("RB", ResourceStatusHeadlessIcon(core.ResourceStatusRollbackComplete))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_create_interrupted() {
	s.Equal("INT", ResourceStatusHeadlessIcon(core.ResourceStatusCreateInterrupted))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_update_interrupted() {
	s.Equal("INT", ResourceStatusHeadlessIcon(core.ResourceStatusUpdateInterrupted))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_destroy_interrupted() {
	s.Equal("INT", ResourceStatusHeadlessIcon(core.ResourceStatusDestroyInterrupted))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessIcon_unknown_default() {
	// When status is not in map, returns default "  "
	s.Equal("  ", ResourceStatusHeadlessIcon(core.ResourceStatusUnknown))
}

// Resource Status Text Tests

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_creating() {
	s.Equal("creating", ResourceStatusHeadlessText(core.ResourceStatusCreating))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_created() {
	s.Equal("created", ResourceStatusHeadlessText(core.ResourceStatusCreated))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_create_failed() {
	s.Equal("create failed", ResourceStatusHeadlessText(core.ResourceStatusCreateFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_updating() {
	s.Equal("updating", ResourceStatusHeadlessText(core.ResourceStatusUpdating))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_updated() {
	s.Equal("updated", ResourceStatusHeadlessText(core.ResourceStatusUpdated))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_update_failed() {
	s.Equal("update failed", ResourceStatusHeadlessText(core.ResourceStatusUpdateFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_destroying() {
	s.Equal("destroying", ResourceStatusHeadlessText(core.ResourceStatusDestroying))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_destroyed() {
	s.Equal("destroyed", ResourceStatusHeadlessText(core.ResourceStatusDestroyed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_destroy_failed() {
	s.Equal("destroy failed", ResourceStatusHeadlessText(core.ResourceStatusDestroyFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_rolling_back() {
	s.Equal("rolling back", ResourceStatusHeadlessText(core.ResourceStatusRollingBack))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_rollback_failed() {
	s.Equal("rollback failed", ResourceStatusHeadlessText(core.ResourceStatusRollbackFailed))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_rollback_complete() {
	s.Equal("rolled back", ResourceStatusHeadlessText(core.ResourceStatusRollbackComplete))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_create_interrupted() {
	s.Equal("create interrupted", ResourceStatusHeadlessText(core.ResourceStatusCreateInterrupted))
}

func (s *HeadlessStatusTestSuite) Test_ResourceStatusHeadlessText_unknown_default() {
	// When status is not in map, returns default "pending"
	s.Equal("pending", ResourceStatusHeadlessText(core.ResourceStatusUnknown))
}

// Instance Status Icon Tests

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_preparing() {
	s.Equal("  ", InstanceStatusHeadlessIcon(core.InstanceStatusPreparing))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deploying() {
	s.Equal("...", InstanceStatusHeadlessIcon(core.InstanceStatusDeploying))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deployed() {
	s.Equal("OK", InstanceStatusHeadlessIcon(core.InstanceStatusDeployed))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deploy_failed() {
	s.Equal("ERR", InstanceStatusHeadlessIcon(core.InstanceStatusDeployFailed))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deploy_rolling_back() {
	s.Equal("<-", InstanceStatusHeadlessIcon(core.InstanceStatusDeployRollingBack))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deploy_rollback_failed() {
	s.Equal("!!", InstanceStatusHeadlessIcon(core.InstanceStatusDeployRollbackFailed))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deploy_rollback_complete() {
	s.Equal("RB", InstanceStatusHeadlessIcon(core.InstanceStatusDeployRollbackComplete))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_deploy_interrupted() {
	s.Equal("INT", InstanceStatusHeadlessIcon(core.InstanceStatusDeployInterrupted))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessIcon_unmapped_default() {
	// When status is not in map, returns default "  "
	s.Equal("  ", InstanceStatusHeadlessIcon(core.InstanceStatus(-1)))
}

// Instance Status Text Tests

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_preparing() {
	s.Equal("preparing", InstanceStatusHeadlessText(core.InstanceStatusPreparing))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_deploying() {
	s.Equal("deploying", InstanceStatusHeadlessText(core.InstanceStatusDeploying))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_deployed() {
	s.Equal("deployed", InstanceStatusHeadlessText(core.InstanceStatusDeployed))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_deploy_failed() {
	s.Equal("deploy failed", InstanceStatusHeadlessText(core.InstanceStatusDeployFailed))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_deploy_rolling_back() {
	s.Equal("rolling back deploy", InstanceStatusHeadlessText(core.InstanceStatusDeployRollingBack))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_deploy_rollback_complete() {
	s.Equal("deploy rolled back", InstanceStatusHeadlessText(core.InstanceStatusDeployRollbackComplete))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_not_deployed() {
	s.Equal("not deployed", InstanceStatusHeadlessText(core.InstanceStatusNotDeployed))
}

func (s *HeadlessStatusTestSuite) Test_InstanceStatusHeadlessText_unmapped_default() {
	// When status is not in map, returns default "unknown"
	s.Equal("unknown", InstanceStatusHeadlessText(core.InstanceStatus(-1)))
}

// Link Status Icon Tests

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_creating() {
	s.Equal("...", LinkStatusHeadlessIcon(core.LinkStatusCreating))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_created() {
	s.Equal("OK", LinkStatusHeadlessIcon(core.LinkStatusCreated))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_create_failed() {
	s.Equal("ERR", LinkStatusHeadlessIcon(core.LinkStatusCreateFailed))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_create_rolling_back() {
	s.Equal("<-", LinkStatusHeadlessIcon(core.LinkStatusCreateRollingBack))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_create_rollback_failed() {
	s.Equal("!!", LinkStatusHeadlessIcon(core.LinkStatusCreateRollbackFailed))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_create_rollback_complete() {
	s.Equal("RB", LinkStatusHeadlessIcon(core.LinkStatusCreateRollbackComplete))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_create_interrupted() {
	s.Equal("INT", LinkStatusHeadlessIcon(core.LinkStatusCreateInterrupted))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessIcon_unmapped_default() {
	// When status is not in map, returns default "  "
	s.Equal("  ", LinkStatusHeadlessIcon(core.LinkStatus(-1)))
}

// Link Status Text Tests

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessText_creating() {
	s.Equal("creating", LinkStatusHeadlessText(core.LinkStatusCreating))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessText_created() {
	s.Equal("created", LinkStatusHeadlessText(core.LinkStatusCreated))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessText_create_failed() {
	s.Equal("create failed", LinkStatusHeadlessText(core.LinkStatusCreateFailed))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessText_create_rolling_back() {
	s.Equal("rolling back create", LinkStatusHeadlessText(core.LinkStatusCreateRollingBack))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessText_create_rollback_complete() {
	s.Equal("create rolled back", LinkStatusHeadlessText(core.LinkStatusCreateRollbackComplete))
}

func (s *HeadlessStatusTestSuite) Test_LinkStatusHeadlessText_unmapped_default() {
	// When status is not in map, returns default "pending"
	s.Equal("pending", LinkStatusHeadlessText(core.LinkStatus(-1)))
}
