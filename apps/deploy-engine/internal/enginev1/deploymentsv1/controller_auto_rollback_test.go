package deploymentsv1

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/assert"
)

func Test_shouldTriggerAutoRollback(t *testing.T) {
	tests := []struct {
		name               string
		status             core.InstanceStatus
		expectedTrigger    bool
		expectedType       AutoRollbackType
	}{
		{
			name:            "triggers destroy rollback on DeployFailed",
			status:          core.InstanceStatusDeployFailed,
			expectedTrigger: true,
			expectedType:    AutoRollbackTypeDestroy,
		},
		{
			name:            "triggers revert rollback on UpdateFailed",
			status:          core.InstanceStatusUpdateFailed,
			expectedTrigger: true,
			expectedType:    AutoRollbackTypeRevert,
		},
		{
			name:            "triggers revert rollback on DestroyFailed",
			status:          core.InstanceStatusDestroyFailed,
			expectedTrigger: true,
			expectedType:    AutoRollbackTypeRevert,
		},
		{
			name:            "does not trigger on Deployed",
			status:          core.InstanceStatusDeployed,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on Updated",
			status:          core.InstanceStatusUpdated,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on DeployRollingBack",
			status:          core.InstanceStatusDeployRollingBack,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on DeployRollbackFailed",
			status:          core.InstanceStatusDeployRollbackFailed,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on DeployRollbackComplete",
			status:          core.InstanceStatusDeployRollbackComplete,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on UpdateRollingBack",
			status:          core.InstanceStatusUpdateRollingBack,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on UpdateRollbackFailed",
			status:          core.InstanceStatusUpdateRollbackFailed,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on UpdateRollbackComplete",
			status:          core.InstanceStatusUpdateRollbackComplete,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on Destroyed",
			status:          core.InstanceStatusDestroyed,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on Preparing",
			status:          core.InstanceStatusPreparing,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
		{
			name:            "does not trigger on Deploying",
			status:          core.InstanceStatusDeploying,
			expectedTrigger: false,
			expectedType:    AutoRollbackTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, rollbackType := shouldTriggerAutoRollback(tt.status)
			assert.Equal(t, tt.expectedTrigger, trigger)
			assert.Equal(t, tt.expectedType, rollbackType)
		})
	}
}
