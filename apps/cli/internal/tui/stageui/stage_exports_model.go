package stageui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// StageExportsModel is a Bubble Tea model for viewing blueprint export changes.
// It uses a split pane with instance hierarchy on the left and export change details on the right.
type StageExportsModel struct {
	splitPane    splitpane.Model
	changes      *changes.BlueprintChanges
	instanceName string
	styles       *styles.Styles
	width        int
	height       int
}

// NewStageExportsModel creates a new exports change view model.
func NewStageExportsModel(
	blueprintChanges *changes.BlueprintChanges,
	instanceName string,
	width, height int,
	s *styles.Styles,
) StageExportsModel {
	// Build the instance hierarchy for the left pane
	items := BuildExportChangeHierarchy(blueprintChanges, instanceName)

	// Create the split pane configuration
	config := splitpane.Config{
		Styles:          s,
		DetailsRenderer: &StageExportsDetailsRenderer{},
		HeaderRenderer:  &StageExportsHeaderRenderer{InstanceName: instanceName},
		FooterRenderer:  &StageExportsFooterRenderer{},
		Title:           "Export Changes",
		LeftPaneRatio:   0.35, // Slightly narrower left pane for exports
		MaxExpandDepth:  0,    // No expansion in exports view
	}

	sp := splitpane.New(config)
	sp.SetItems(items)

	return StageExportsModel{
		splitPane:    sp,
		changes:      blueprintChanges,
		instanceName: instanceName,
		styles:       s,
		width:        width,
		height:       height,
	}
}

// Init implements tea.Model.
func (m StageExportsModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m StageExportsModel) Update(msg tea.Msg) (StageExportsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.splitPane, cmd = m.splitPane.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Handle quit/close from exports view
		switch msg.String() {
		case "q", "ctrl+c", "e", "esc":
			// Let the parent handle closing the exports view
			return m, nil
		}

		// Forward other keys to split pane
		m.splitPane, cmd = m.splitPane.Update(msg)
		return m, cmd
	}

	// Forward other messages to split pane
	m.splitPane, cmd = m.splitPane.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m StageExportsModel) View() string {
	return m.splitPane.View()
}

// HasExportChanges returns true if there are any export changes.
func (m StageExportsModel) HasExportChanges() bool {
	return HasAnyExportChanges(m.changes)
}
