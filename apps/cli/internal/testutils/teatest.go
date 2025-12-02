package testutils

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// WaitFor is a helper function to wait for a condition to be true with a
// pre-defined check interval (20ms) and timeout duration (10 seconds).
func WaitFor(t *testing.T, output io.Reader, condition func(output []byte) bool) {
	teatest.WaitFor(
		t, output, condition,
		teatest.WithCheckInterval(20*time.Millisecond),
		teatest.WithDuration(10*time.Second),
	)
}

// WaitForContains is a helper function to wait for a string to be present in the output.
func WaitForContains(t *testing.T, output io.Reader, text string) {
	WaitFor(t, output, func(bts []byte) bool {
		return strings.Contains(string(bts), text)
	})
}

// WaitForContainsAll is a helper function to wait for all strings to be present in the output.
func WaitForContainsAll(t *testing.T, output io.Reader, texts ...string) {
	WaitFor(t, output, func(bts []byte) bool {
		for _, text := range texts {
			if !strings.Contains(string(bts), text) {
				return false
			}
		}
		return true
	})
}

// KeyEnter is a helper function to send a key enter message to the model.
func KeyEnter(model *teatest.TestModel) {
	model.Send(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
	)
}

// KeyDown is a helper function to send a key down message to the model.
func KeyDown(model *teatest.TestModel) {
	model.Send(
		tea.KeyMsg{
			Type: tea.KeyDown,
		},
	)
}

// KeyUp is a helper function to send a key up message to the model.
func KeyUp(model *teatest.TestModel) {
	model.Send(
		tea.KeyMsg{
			Type: tea.KeyUp,
		},
	)
}

// KeyLeft is a helper function to send a key left message to the model.
func KeyLeft(model *teatest.TestModel) {
	model.Send(
		tea.KeyMsg{
			Type: tea.KeyLeft,
		},
	)
}

// KeyRight is a helper function to send a key right message to the model.
func KeyRight(model *teatest.TestModel) {
	model.Send(
		tea.KeyMsg{
			Type: tea.KeyRight,
		},
	)
}

// KeyQ is a helper function to send a key q message to the model.
func KeyQ(model *teatest.TestModel) {
	model.Send(
		tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("q"),
		},
	)
}
