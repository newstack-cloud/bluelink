package stageui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg routes keyboard input to the appropriate handler based on current state.
func (m StageModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.err != nil {
		return m.handleKeyMsgInErrorState(msg)
	}

	if m.showingOverview {
		return m.handleOverviewKeyMsg(msg)
	}

	if m.showingExportsView {
		return m.handleKeyMsgInExportsView(msg)
	}

	if m.driftReviewMode {
		return m.handleKeyMsgInDriftReview(msg)
	}

	if !m.finished {
		return m, nil
	}

	return m.handleKeyMsgInFinishedState(msg)
}

// handleKeyMsgInErrorState handles keyboard input when there's an error.
func (m StageModel) handleKeyMsgInErrorState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" || msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	return m, nil
}

// handleKeyMsgInExportsView handles keyboard input when viewing exports.
func (m StageModel) handleKeyMsgInExportsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "e", "esc", "q":
		m.showingExportsView = false
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.exportsModel, cmd = m.exportsModel.Update(msg)
		return m, cmd
	}
}

// handleKeyMsgInDriftReview handles keyboard input during drift review.
func (m StageModel) handleKeyMsgInDriftReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a", "A":
		return m, applyReconciliationCmd(m)
	case "q":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.driftSplitPane, cmd = m.driftSplitPane.Update(msg)
		return m, cmd
	}
}

// handleKeyMsgInFinishedState handles keyboard input after staging completes.
func (m StageModel) handleKeyMsgInFinishedState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o", "O":
		m.showingOverview = true
		m.overviewViewport.SetContent(m.renderOverviewContent())
		m.overviewViewport.GotoTop()
		return m, nil

	case "e", "E":
		return m.toggleExportsView()

	default:
		var cmd tea.Cmd
		m.splitPane, cmd = m.splitPane.Update(msg)
		return m, cmd
	}
}

// toggleExportsView toggles the exports view on or off.
func (m StageModel) toggleExportsView() (tea.Model, tea.Cmd) {
	if m.showingExportsView {
		m.showingExportsView = false
		return m, nil
	}

	if m.completeChanges == nil || !HasAnyExportChanges(m.completeChanges) {
		return m, nil
	}

	m.showingExportsView = true
	m.exportsModel = NewStageExportsModel(
		m.completeChanges,
		m.instanceName,
		m.width, m.height,
		m.styles,
	)

	var cmd tea.Cmd
	m.exportsModel, cmd = m.exportsModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return m, cmd
}
