package deployui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/ui/splitpane"
)

// ExportsModel is a Bubble Tea model for viewing blueprint instance exports.
// It uses a split pane with instance hierarchy on the left and export details on the right.
type ExportsModel struct {
	splitPane     splitpane.Model
	instanceState *state.InstanceState
	instanceName  string
	styles        *styles.Styles
	width, height int
}

// NewExportsModel creates a new exports view model.
func NewExportsModel(
	instanceState *state.InstanceState,
	instanceName string,
	width, height int,
	s *styles.Styles,
) ExportsModel {
	// Build the instance hierarchy for the left pane
	items := BuildInstanceHierarchy(instanceState, instanceName)

	// Create the split pane configuration
	config := splitpane.Config{
		Styles:          s,
		DetailsRenderer: &ExportsDetailsRenderer{},
		HeaderRenderer:  &ExportsHeaderRenderer{InstanceName: instanceName},
		FooterRenderer:  &ExportsFooterRenderer{},
		Title:           "Exports",
		LeftPaneRatio:   0.35, // Slightly narrower left pane for exports
		MaxExpandDepth:  0,    // No expansion in exports view
	}

	sp := splitpane.New(config)
	sp.SetItems(items)

	return ExportsModel{
		splitPane:     sp,
		instanceState: instanceState,
		instanceName:  instanceName,
		styles:        s,
		width:         width,
		height:        height,
	}
}

// Init implements tea.Model.
func (m ExportsModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ExportsModel) Update(msg tea.Msg) (ExportsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.splitPane, cmd = m.splitPane.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Handle quit from split pane
		switch msg.String() {
		case "q", "ctrl+c":
			// Let the parent handle quit
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
func (m ExportsModel) View() string {
	return m.splitPane.View()
}

// UpdateInstanceState updates the exports view with new instance state.
// This allows the exports view to dynamically update during deployment.
func (m *ExportsModel) UpdateInstanceState(instanceState *state.InstanceState) {
	if instanceState == nil {
		return
	}
	m.instanceState = instanceState
	items := BuildInstanceHierarchy(instanceState, m.instanceName)
	m.splitPane.UpdateItems(items)
}

// HasExports returns true if any instance in the hierarchy has exports.
func (m ExportsModel) HasExports() bool {
	return hasExportsInHierarchy(m.instanceState)
}

// hasExportsInHierarchy recursively checks if any instance has exports.
func hasExportsInHierarchy(instanceState *state.InstanceState) bool {
	if instanceState == nil {
		return false
	}
	if len(instanceState.Exports) > 0 {
		return true
	}
	for _, child := range instanceState.ChildBlueprints {
		if hasExportsInHierarchy(child) {
			return true
		}
	}
	return false
}

// InstanceStateHasExports checks if an instance state has any exports
// (either directly or in child blueprints).
func InstanceStateHasExports(instanceState *state.InstanceState) bool {
	return hasExportsInHierarchy(instanceState)
}
