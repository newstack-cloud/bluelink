//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableWindowsANSI enables virtual terminal processing on Windows.
// This allows ANSI escape codes to work in Windows 10+ terminals.
// Returns true if ANSI is supported, false otherwise.
func enableWindowsANSI() bool {
	// Get the stdout handle
	handle := windows.Handle(os.Stdout.Fd())

	// Get the current console mode
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}

	// Enable virtual terminal processing
	// ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	const ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	if err := windows.SetConsoleMode(handle, mode|ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return false
	}

	return true
}
