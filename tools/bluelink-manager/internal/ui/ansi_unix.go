//go:build !windows

package ui

// enableWindowsANSI is a no-op on Unix systems.
// ANSI colors work out of the box on Unix terminals.
func enableWindowsANSI() bool {
	return true
}
