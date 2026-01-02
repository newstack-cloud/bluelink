package stageui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/driftui"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// handleWindowSizeMsg handles terminal resize events.
func (m StageModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (StageModel, []tea.Cmd) {
	var cmds []tea.Cmd

	m.width = msg.Width
	m.height = msg.Height

	var cmd tea.Cmd
	m.splitPane, cmd = m.splitPane.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if m.showingExportsView {
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.overviewViewport.Width = msg.Width
	m.overviewViewport.Height = msg.Height - stageOverviewFooterHeight()

	return m, cmds
}

// handleSelectBlueprintMsg handles blueprint selection to start staging.
func (m StageModel) handleSelectBlueprintMsg(msg sharedui.SelectBlueprintMsg) (StageModel, []tea.Cmd) {
	m.blueprintFile = msg.BlueprintFile
	m.blueprintSource = msg.Source

	if m.streaming {
		return m, nil
	}

	m.streaming = true
	return m, []tea.Cmd{startStagingCmd(m), waitForNextEventCmd(m), checkForErrCmd(m)}
}

// handleStageStartedMsg handles the staging started message.
func (m StageModel) handleStageStartedMsg(msg StageStartedMsg) StageModel {
	if m.err != nil {
		return m
	}

	m.changesetID = msg.ChangesetID
	m.footerRenderer.ChangesetID = msg.ChangesetID
	if m.headlessMode && !m.jsonMode {
		m.printHeadlessHeader()
	}
	return m
}

// handleStageStartedWithStateMsg handles staging started with instance state.
func (m StageModel) handleStageStartedWithStateMsg(msg StageStartedWithStateMsg) StageModel {
	if m.err != nil {
		return m
	}

	m.changesetID = msg.ChangesetID
	m.instanceState = msg.InstanceState
	m.footerRenderer.ChangesetID = msg.ChangesetID
	if m.headlessMode && !m.jsonMode {
		m.printHeadlessHeader()
	}
	return m
}

// handleStageEventMsg handles staging events from the stream.
func (m StageModel) handleStageEventMsg(msg StageEventMsg) (StageModel, []tea.Cmd) {
	if m.err != nil {
		return m, nil
	}

	var cmds []tea.Cmd
	event := types.ChangeStagingEvent(msg)
	m.processEvent(&event)
	cmds = append(cmds, checkForErrCmd(m))

	if eventData, ok := event.AsCompleteChanges(); ok {
		return m.handleCompleteChangesEvent(eventData, cmds)
	}

	if _, ok := event.AsDriftDetected(); ok {
		return m.handleDriftDetectedEvent(cmds)
	}

	cmds = append(cmds, waitForNextEventCmd(m))
	return m, cmds
}

// handleCompleteChangesEvent handles the completion of staging.
func (m StageModel) handleCompleteChangesEvent(
	eventData *types.CompleteChangesEventData,
	cmds []tea.Cmd,
) (StageModel, []tea.Cmd) {
	m.finished = true
	m.completeChanges = eventData.Changes

	if len(m.items) == 0 && m.completeChanges != nil {
		m.populateItemsFromCompleteChanges(m.completeChanges, m.instanceState)
	}

	m.splitPane.SetItems(ToSplitPaneItems(m.items))
	m.updateFooterCounts()

	changesetID := m.changesetID
	completeChanges := m.completeChanges
	items := m.items
	instanceState := m.instanceState
	cmds = append(cmds, func() tea.Msg {
		return StageCompleteMsg{
			ChangesetID:   changesetID,
			Changes:       completeChanges,
			Items:         items,
			InstanceState: instanceState,
		}
	})

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSON()
		} else {
			m.printHeadlessSummary()
		}
		cmds = append(cmds, tea.Quit)
	}

	return m, cmds
}

// handleDriftDetectedEvent handles drift detection during staging.
func (m StageModel) handleDriftDetectedEvent(cmds []tea.Cmd) (StageModel, []tea.Cmd) {
	m.driftReviewMode = true
	m.streaming = false

	if m.driftResult != nil {
		driftItems := BuildDriftItems(m.driftResult, m.instanceState)
		m.driftSplitPane.SetItems(driftItems)
	}

	driftResult := m.driftResult
	driftMessage := m.driftMessage
	instanceID := m.instanceID
	instanceState := m.instanceState
	cmds = append(cmds, func() tea.Msg {
		return driftui.DriftDetectedMsg{
			ReconciliationResult: driftResult,
			Message:              driftMessage,
			InstanceID:           instanceID,
			InstanceState:        instanceState,
		}
	})

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONDrift()
		} else {
			m.printHeadlessDriftDetected()
		}
		cmds = append(cmds, tea.Quit)
	}

	return m, cmds
}

// handleStageErrorMsg handles errors during staging.
func (m StageModel) handleStageErrorMsg(msg StageErrorMsg) (StageModel, tea.Cmd) {
	if msg.Err == nil {
		return m, nil
	}

	m.err = msg.Err
	m.streaming = false

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(msg.Err)
		} else {
			m.printHeadlessError(msg.Err)
		}
		return m, tea.Quit
	}

	return m, nil
}

// handleStageStreamClosedMsg handles unexpected stream closure.
func (m StageModel) handleStageStreamClosedMsg() (StageModel, tea.Cmd) {
	if m.finished {
		return m, nil
	}

	m.finished = true
	m.streaming = false
	m.err = fmt.Errorf("staging event stream closed unexpectedly (connection timeout or dropped)")

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(m.err)
		} else {
			m.printHeadlessError(m.err)
		}
		return m, tea.Quit
	}

	return m, nil
}

// handleReconciliationCompleteMsg handles successful reconciliation.
func (m StageModel) handleReconciliationCompleteMsg() (StageModel, []tea.Cmd) {
	m.driftReviewMode = false
	m.driftResult = nil
	m.driftMessage = ""
	m.streaming = false
	m.items = []StageItem{}
	m.resourceChanges = make(map[string]*ResourceChangeState)
	m.childChanges = make(map[string]*ChildChangeState)
	m.linkChanges = make(map[string]*LinkChangeState)

	m.streaming = true
	return m, []tea.Cmd{startStagingCmd(m), waitForNextEventCmd(m), checkForErrCmd(m)}
}

// handleReconciliationErrorMsg handles reconciliation failure.
func (m StageModel) handleReconciliationErrorMsg(msg driftui.ReconciliationErrorMsg) (StageModel, tea.Cmd) {
	if msg.Err == nil {
		return m, nil
	}

	m.err = msg.Err
	m.driftReviewMode = false

	if m.headlessMode {
		if m.jsonMode {
			m.outputJSONError(msg.Err)
		} else {
			m.printHeadlessError(msg.Err)
		}
		return m, tea.Quit
	}

	return m, nil
}

// handleMouseMsg handles mouse events.
func (m StageModel) handleMouseMsg(msg tea.MouseMsg) (StageModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.driftReviewMode {
		m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
		return m, cmd
	}

	if m.finished && m.err == nil {
		m.splitPane, cmd = m.splitPane.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleSplitpaneMsg handles splitpane messages (quit, back, expanded).
func (m StageModel) handleSplitpaneMsg(msg tea.Msg) (StageModel, tea.Cmd) {
	switch msg.(type) {
	case splitpane.QuitMsg:
		return m, tea.Quit
	case splitpane.BackMsg:
		if m.driftReviewMode {
			return m, tea.Quit
		}
	case splitpane.ItemExpandedMsg:
		// Expansion state handled internally by splitpane
	}
	return m, nil
}
