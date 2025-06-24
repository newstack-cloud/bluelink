package styles

import "github.com/charmbracelet/lipgloss"

// BluelinkStyles holds the styles to be used across command TUI components.
type BluelinkStyles struct {
	Selected   lipgloss.Style
	Selectable lipgloss.Style
}

// NewBluelinkStyles creates a new instance of the styles used in the TUI.
func NewBluelinkStyles(r *lipgloss.Renderer) *BluelinkStyles {
	return &BluelinkStyles{
		Selected:   r.NewStyle().Foreground(lipgloss.Color("#5882e2")).Bold(true),
		Selectable: r.NewStyle().Foreground(lipgloss.Color("#2b63e3")),
	}
}

// NewDefaultBluelinkStyles creates a new instance of the styles used in the TUI
// with the default renderer.
func NewDefaultBluelinkStyles() *BluelinkStyles {
	return NewBluelinkStyles(lipgloss.DefaultRenderer())
}
