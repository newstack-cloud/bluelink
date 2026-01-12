package cleanupui

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
)

var errCleanupOperationNil = errors.New("cleanup operation returned nil")

// CleanupStartedMsg is sent when a cleanup operation has been started.
type CleanupStartedMsg struct {
	Operation *manage.CleanupOperation
}

// CleanupCompletedMsg is sent when a cleanup operation has completed.
type CleanupCompletedMsg struct {
	Operation *manage.CleanupOperation
}

// CleanupErrorMsg is sent when a cleanup operation encounters an error.
type CleanupErrorMsg struct {
	Err error
}

// AllCleanupsDoneMsg is sent when all cleanup operations have completed.
type AllCleanupsDoneMsg struct{}

func startCleanupCmd(eng engine.DeployEngine, cleanupType manage.CleanupType) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var op *manage.CleanupOperation
		var err error

		switch cleanupType {
		case manage.CleanupTypeValidations:
			op, err = eng.CleanupBlueprintValidations(ctx)
		case manage.CleanupTypeChangesets:
			op, err = eng.CleanupChangesets(ctx)
		case manage.CleanupTypeReconciliationResults:
			op, err = eng.CleanupReconciliationResults(ctx)
		case manage.CleanupTypeEvents:
			op, err = eng.CleanupEvents(ctx)
		}

		if err != nil {
			return CleanupErrorMsg{Err: err}
		}

		if op == nil {
			return CleanupErrorMsg{Err: errCleanupOperationNil}
		}

		return CleanupStartedMsg{Operation: op}
	}
}

func waitForCleanupCompletionCmd(eng engine.DeployEngine, op *manage.CleanupOperation) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		finalOp, err := eng.WaitForCleanupCompletion(
			ctx,
			op.CleanupType,
			op.ID,
			500*time.Millisecond,
		)
		if err != nil {
			return CleanupErrorMsg{Err: err}
		}
		return CleanupCompletedMsg{Operation: finalOp}
	}
}
