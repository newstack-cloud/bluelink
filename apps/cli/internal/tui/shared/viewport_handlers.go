package shared

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewportKeyResult contains the result of handling a viewport key event.
type ViewportKeyResult struct {
	ShouldClose bool
	ShouldQuit  bool
	Viewport    viewport.Model
	Cmd         tea.Cmd
}

// HandleViewportKeyMsg handles key messages for viewport overlay views.
// toggleKeys are the keys that close the overlay (e.g., "o", "O" for overview).
// Always handles "esc" for closing and "q"/"ctrl+c" for quitting.
func HandleViewportKeyMsg(msg tea.KeyMsg, vp viewport.Model, toggleKeys ...string) ViewportKeyResult {
	key := msg.String()

	// Check for quit keys
	if key == "q" || key == "ctrl+c" {
		return ViewportKeyResult{ShouldQuit: true, Viewport: vp}
	}

	// Check for close keys (esc + custom toggle keys)
	if key == "esc" {
		return ViewportKeyResult{ShouldClose: true, Viewport: vp}
	}
	for _, tk := range toggleKeys {
		if key == tk {
			return ViewportKeyResult{ShouldClose: true, Viewport: vp}
		}
	}

	// Default: delegate to viewport
	var cmd tea.Cmd
	vp, cmd = vp.Update(msg)
	return ViewportKeyResult{Viewport: vp, Cmd: cmd}
}

// ExportsKeyAction indicates what action to take after handling an exports view key.
type ExportsKeyAction int

const (
	// ExportsKeyActionDelegate means the key should be delegated to the exports model.
	ExportsKeyActionDelegate ExportsKeyAction = iota
	// ExportsKeyActionClose means the exports view should be closed.
	ExportsKeyActionClose
	// ExportsKeyActionQuit means the application should quit.
	ExportsKeyActionQuit
)

// CheckExportsKeyMsg checks if a key message should close/quit the exports view.
// Returns the action to take. If ExportsKeyActionDelegate is returned, the caller
// should delegate the key to the exports model.
func CheckExportsKeyMsg(msg tea.KeyMsg) ExportsKeyAction {
	key := msg.String()

	if key == "q" || key == "ctrl+c" {
		return ExportsKeyActionQuit
	}

	if key == "esc" || key == "e" || key == "E" {
		return ExportsKeyActionClose
	}

	return ExportsKeyActionDelegate
}
