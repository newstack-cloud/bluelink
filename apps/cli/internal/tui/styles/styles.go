package styles

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// BluelinkStyles holds the styles to be used across command TUI components.
type BluelinkStyles struct {
	Selected          lipgloss.Style
	SelectedListItem  lipgloss.Style
	Selectable        lipgloss.Style
	Title             lipgloss.Style
	ListItem          lipgloss.Style
	Pagination        lipgloss.Style
	Help              lipgloss.Style
	Error             lipgloss.Style
	Warning           lipgloss.Style
	Info              lipgloss.Style
	Muted             lipgloss.Style
	Spinner           lipgloss.Style
	Category          lipgloss.Style
	Location          lipgloss.Style
	DiagnosticMessage lipgloss.Style
	DiagnosticAction  lipgloss.Style
}

// NewBluelinkStyles creates a new instance of the styles used in the TUI.
func NewBluelinkStyles(r *lipgloss.Renderer) *BluelinkStyles {
	// SelectedListItem uses PaddingLeft(2) to align with the "> " prefix added in rendering
	selectedListItem := r.NewStyle().
		PaddingLeft(2).
		Foreground(BluelinkPrimary)

	return &BluelinkStyles{
		Selected:          r.NewStyle().Foreground(BluelinkPrimary).Bold(true),
		SelectedListItem:  selectedListItem,
		Selectable:        r.NewStyle().Foreground(BluelinkSecondary),
		Title:             r.NewStyle().Foreground(BluelinkPrimary).Bold(true),
		ListItem:          r.NewStyle().PaddingLeft(4),
		Pagination:        list.DefaultStyles().PaginationStyle.PaddingLeft(4),
		Help:              list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1),
		Error:             r.NewStyle().Foreground(BluelinkError),
		Warning:           r.NewStyle().Foreground(BluelinkWarning),
		Info:              r.NewStyle().Foreground(BluelinkInfo),
		Muted:             r.NewStyle().Foreground(BluelinkMuted),
		Spinner:           r.NewStyle().Foreground(BluelinkPrimary),
		Category:          r.NewStyle().Foreground(BluelinkPrimary),
		Location:          r.NewStyle().MarginLeft(2).Foreground(BluelinkPrimary),
		DiagnosticMessage: r.NewStyle().MarginLeft(2),
		DiagnosticAction:  r.NewStyle().MarginTop(2),
	}
}

// NewDefaultBluelinkStyles creates a new instance of the styles used in the TUI
// with the default renderer.
func NewDefaultBluelinkStyles() *BluelinkStyles {
	return NewBluelinkStyles(lipgloss.DefaultRenderer())
}

var (
	// DefaultBluelinkStyles is the default instance of the styles used in the TUI.
	DefaultBluelinkStyles = NewDefaultBluelinkStyles()
)

// Bluelink color constants - exported for use in custom rendering
var (
	BluelinkPrimary   = lipgloss.AdaptiveColor{Light: "#072f8c", Dark: "#5882e2"}
	BluelinkSecondary = lipgloss.Color("#2b63e3")
	BluelinkError     = lipgloss.Color("#dc2626")
	BluelinkWarning   = lipgloss.Color("#f97316")
	BluelinkInfo      = lipgloss.Color("#2563eb")
	BluelinkMuted     = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
)

// NewBluelinkHuhTheme creates a huh form theme using the Bluelink color scheme.
func NewBluelinkHuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused field styles
	t.Focused.Title = lipgloss.NewStyle().Foreground(BluelinkPrimary).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{
			Light: BluelinkMuted.Light,
			Dark:  BluelinkMuted.Dark,
		},
	)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(BluelinkError)
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(BluelinkError)

	// Select styles
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(BluelinkPrimary).SetString("> ")
	t.Focused.Option = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{
			Light: "#333333",
			Dark:  "#cccccc",
		},
	)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(BluelinkPrimary).Bold(true)

	// Text input styles
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(BluelinkPrimary)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(BluelinkPrimary)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{
			Light: "#333333",
			Dark:  "#ffffff",
		},
	)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{
			Light: "#999999",
			Dark:  "#666666",
		},
	)

	// Confirm button styles
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(BluelinkSecondary).
		Padding(0, 1).
		Bold(true)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(
			lipgloss.AdaptiveColor{
				Light: "#333333",
				Dark:  "#cccccc",
			},
		).
		Padding(0, 1)

	// Blurred field styles (less prominent)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(BluelinkMuted)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{
			Light: "#999999",
			Dark:  "#666666",
		},
	)
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(BluelinkMuted)
	t.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.Option = lipgloss.NewStyle().Foreground(BluelinkMuted)
	t.Blurred.SelectedOption = lipgloss.NewStyle().Foreground(BluelinkMuted)

	t.Blurred.FocusedButton = lipgloss.NewStyle().
		Foreground(
			lipgloss.AdaptiveColor{
				Light: "#333333",
				Dark:  "#cccccc",
			},
		).
		Padding(0, 1)
	t.Blurred.BlurredButton = lipgloss.NewStyle().
		Foreground(BluelinkMuted).
		Padding(0, 1)

	return t
}
