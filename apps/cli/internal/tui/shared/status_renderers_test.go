package shared

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type StatusRenderersTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestStatusRenderersTestSuite(t *testing.T) {
	suite.Run(t, new(StatusRenderersTestSuite))
}

func (s *StatusRenderersTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_creating() {
	result := RenderResourceStatus(core.ResourceStatusCreating, s.testStyles)
	s.Contains(result, "Creating")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_created() {
	result := RenderResourceStatus(core.ResourceStatusCreated, s.testStyles)
	s.Contains(result, "Created")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_create_failed() {
	result := RenderResourceStatus(core.ResourceStatusCreateFailed, s.testStyles)
	s.Contains(result, "Create Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_updating() {
	result := RenderResourceStatus(core.ResourceStatusUpdating, s.testStyles)
	s.Contains(result, "Updating")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_updated() {
	result := RenderResourceStatus(core.ResourceStatusUpdated, s.testStyles)
	s.Contains(result, "Updated")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_update_failed() {
	result := RenderResourceStatus(core.ResourceStatusUpdateFailed, s.testStyles)
	s.Contains(result, "Update Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_destroying() {
	result := RenderResourceStatus(core.ResourceStatusDestroying, s.testStyles)
	s.Contains(result, "Destroying")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_destroyed() {
	result := RenderResourceStatus(core.ResourceStatusDestroyed, s.testStyles)
	s.Contains(result, "Destroyed")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_destroy_failed() {
	result := RenderResourceStatus(core.ResourceStatusDestroyFailed, s.testStyles)
	s.Contains(result, "Destroy Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_rolling_back() {
	result := RenderResourceStatus(core.ResourceStatusRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_rollback_failed() {
	result := RenderResourceStatus(core.ResourceStatusRollbackFailed, s.testStyles)
	s.Contains(result, "Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_rollback_complete() {
	result := RenderResourceStatus(core.ResourceStatusRollbackComplete, s.testStyles)
	s.Contains(result, "Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_create_interrupted() {
	result := RenderResourceStatus(core.ResourceStatusCreateInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_update_interrupted() {
	result := RenderResourceStatus(core.ResourceStatusUpdateInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_destroy_interrupted() {
	result := RenderResourceStatus(core.ResourceStatusDestroyInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderResourceStatus_unknown() {
	result := RenderResourceStatus(core.ResourceStatus(9999), s.testStyles)
	s.Contains(result, "Unknown")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_preparing() {
	result := RenderInstanceStatus(core.InstanceStatusPreparing, s.testStyles)
	s.Contains(result, "Preparing")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deploying() {
	result := RenderInstanceStatus(core.InstanceStatusDeploying, s.testStyles)
	s.Contains(result, "Deploying")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deployed() {
	result := RenderInstanceStatus(core.InstanceStatusDeployed, s.testStyles)
	s.Contains(result, "Deployed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deploy_failed() {
	result := RenderInstanceStatus(core.InstanceStatusDeployFailed, s.testStyles)
	s.Contains(result, "Deploy Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_updating() {
	result := RenderInstanceStatus(core.InstanceStatusUpdating, s.testStyles)
	s.Contains(result, "Updating")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_updated() {
	result := RenderInstanceStatus(core.InstanceStatusUpdated, s.testStyles)
	s.Contains(result, "Updated")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_update_failed() {
	result := RenderInstanceStatus(core.InstanceStatusUpdateFailed, s.testStyles)
	s.Contains(result, "Update Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroying() {
	result := RenderInstanceStatus(core.InstanceStatusDestroying, s.testStyles)
	s.Contains(result, "Destroying")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroyed() {
	result := RenderInstanceStatus(core.InstanceStatusDestroyed, s.testStyles)
	s.Contains(result, "Destroyed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroy_failed() {
	result := RenderInstanceStatus(core.InstanceStatusDestroyFailed, s.testStyles)
	s.Contains(result, "Destroy Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deploy_rolling_back() {
	result := RenderInstanceStatus(core.InstanceStatusDeployRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back Deploy")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deploy_rollback_failed() {
	result := RenderInstanceStatus(core.InstanceStatusDeployRollbackFailed, s.testStyles)
	s.Contains(result, "Deploy Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deploy_rollback_complete() {
	result := RenderInstanceStatus(core.InstanceStatusDeployRollbackComplete, s.testStyles)
	s.Contains(result, "Deploy Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_update_rolling_back() {
	result := RenderInstanceStatus(core.InstanceStatusUpdateRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back Update")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_update_rollback_failed() {
	result := RenderInstanceStatus(core.InstanceStatusUpdateRollbackFailed, s.testStyles)
	s.Contains(result, "Update Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_update_rollback_complete() {
	result := RenderInstanceStatus(core.InstanceStatusUpdateRollbackComplete, s.testStyles)
	s.Contains(result, "Update Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroy_rolling_back() {
	result := RenderInstanceStatus(core.InstanceStatusDestroyRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back Destroy")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroy_rollback_failed() {
	result := RenderInstanceStatus(core.InstanceStatusDestroyRollbackFailed, s.testStyles)
	s.Contains(result, "Destroy Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroy_rollback_complete() {
	result := RenderInstanceStatus(core.InstanceStatusDestroyRollbackComplete, s.testStyles)
	s.Contains(result, "Destroy Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_not_deployed() {
	result := RenderInstanceStatus(core.InstanceStatusNotDeployed, s.testStyles)
	s.Contains(result, "Not Deployed")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_deploy_interrupted() {
	result := RenderInstanceStatus(core.InstanceStatusDeployInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_update_interrupted() {
	result := RenderInstanceStatus(core.InstanceStatusUpdateInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_destroy_interrupted() {
	result := RenderInstanceStatus(core.InstanceStatusDestroyInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderInstanceStatus_unknown() {
	result := RenderInstanceStatus(core.InstanceStatus(9999), s.testStyles)
	s.Contains(result, "Unknown")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_creating() {
	result := RenderLinkStatus(core.LinkStatusCreating, s.testStyles)
	s.Contains(result, "Creating")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_created() {
	result := RenderLinkStatus(core.LinkStatusCreated, s.testStyles)
	s.Contains(result, "Created")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_create_failed() {
	result := RenderLinkStatus(core.LinkStatusCreateFailed, s.testStyles)
	s.Contains(result, "Create Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_updating() {
	result := RenderLinkStatus(core.LinkStatusUpdating, s.testStyles)
	s.Contains(result, "Updating")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_updated() {
	result := RenderLinkStatus(core.LinkStatusUpdated, s.testStyles)
	s.Contains(result, "Updated")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_update_failed() {
	result := RenderLinkStatus(core.LinkStatusUpdateFailed, s.testStyles)
	s.Contains(result, "Update Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroying() {
	result := RenderLinkStatus(core.LinkStatusDestroying, s.testStyles)
	s.Contains(result, "Destroying")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroyed() {
	result := RenderLinkStatus(core.LinkStatusDestroyed, s.testStyles)
	s.Contains(result, "Destroyed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroy_failed() {
	result := RenderLinkStatus(core.LinkStatusDestroyFailed, s.testStyles)
	s.Contains(result, "Destroy Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_create_rolling_back() {
	result := RenderLinkStatus(core.LinkStatusCreateRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back Create")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_create_rollback_failed() {
	result := RenderLinkStatus(core.LinkStatusCreateRollbackFailed, s.testStyles)
	s.Contains(result, "Create Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_create_rollback_complete() {
	result := RenderLinkStatus(core.LinkStatusCreateRollbackComplete, s.testStyles)
	s.Contains(result, "Create Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_update_rolling_back() {
	result := RenderLinkStatus(core.LinkStatusUpdateRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back Update")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_update_rollback_failed() {
	result := RenderLinkStatus(core.LinkStatusUpdateRollbackFailed, s.testStyles)
	s.Contains(result, "Update Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_update_rollback_complete() {
	result := RenderLinkStatus(core.LinkStatusUpdateRollbackComplete, s.testStyles)
	s.Contains(result, "Update Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroy_rolling_back() {
	result := RenderLinkStatus(core.LinkStatusDestroyRollingBack, s.testStyles)
	s.Contains(result, "Rolling Back Destroy")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroy_rollback_failed() {
	result := RenderLinkStatus(core.LinkStatusDestroyRollbackFailed, s.testStyles)
	s.Contains(result, "Destroy Rollback Failed")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroy_rollback_complete() {
	result := RenderLinkStatus(core.LinkStatusDestroyRollbackComplete, s.testStyles)
	s.Contains(result, "Destroy Rolled Back")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_create_interrupted() {
	result := RenderLinkStatus(core.LinkStatusCreateInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_update_interrupted() {
	result := RenderLinkStatus(core.LinkStatusUpdateInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_destroy_interrupted() {
	result := RenderLinkStatus(core.LinkStatusDestroyInterrupted, s.testStyles)
	s.Contains(result, "Interrupted")
}

func (s *StatusRenderersTestSuite) Test_RenderLinkStatus_unknown() {
	result := RenderLinkStatus(core.LinkStatus(9999), s.testStyles)
	s.Contains(result, "Unknown")
}
