package ui

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/term"
)

// ANSI color codes
var (
	red    = "\033[0;31m"
	green  = "\033[0;32m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	bold   = "\033[1m"
	reset  = "\033[0m"
)

func init() {
	// Disable colors if not a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		disableColors()
		return
	}

	// On Windows, enable virtual terminal processing for ANSI colors
	// Modern Windows 10+ supports ANSI, but we need to enable it
	if runtime.GOOS == "windows" {
		if !enableWindowsANSI() {
			disableColors()
		}
	}
}

func disableColors() {
	red = ""
	green = ""
	yellow = ""
	blue = ""
	bold = ""
	reset = ""
}

// Info prints an info message.
func Info(format string, args ...any) {
	fmt.Printf("%sinfo%s: %s\n", blue, reset, fmt.Sprintf(format, args...))
}

// Success prints a success message.
func Success(format string, args ...any) {
	fmt.Printf("%ssuccess%s: %s\n", green, reset, fmt.Sprintf(format, args...))
}

// Warn prints a warning message.
func Warn(format string, args ...any) {
	fmt.Printf("%swarning%s: %s\n", yellow, reset, fmt.Sprintf(format, args...))
}

// Error prints an error message to stderr.
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%serror%s: %s\n", red, reset, fmt.Sprintf(format, args...))
}

// Println prints a line with optional formatting.
func Println(format ...any) {
	if len(format) == 0 {
		fmt.Println()
		return
	}
	if len(format) == 1 {
		fmt.Println(format[0])
		return
	}
	fmt.Printf(format[0].(string)+"\n", format[1:]...)
}

// Bold prints bold text.
func Bold(format string, args ...any) {
	fmt.Printf("%s%s%s\n", bold, fmt.Sprintf(format, args...), reset)
}

// PrintRed prints red text.
func PrintRed(format string, args ...any) {
	fmt.Printf("%s%s%s\n", red, fmt.Sprintf(format, args...), reset)
}

// PrintGreen prints green text.
func PrintGreen(format string, args ...any) {
	fmt.Printf("%s%s%s\n", green, fmt.Sprintf(format, args...), reset)
}

// PrintYellow prints yellow text.
func PrintYellow(format string, args ...any) {
	fmt.Printf("%s%s%s\n", yellow, fmt.Sprintf(format, args...), reset)
}

// PrintHeader prints a styled header.
func PrintHeader(title string) {
	fmt.Println()
	Bold("%s", title)
	fmt.Println("==================")
	fmt.Println()
}

// PrintNextSteps prints the post-installation next steps.
func PrintNextSteps() {
	Bold("Next steps:")
	if runtime.GOOS == "windows" {
		fmt.Println("  1. Open a new terminal (or sign out and back in) for PATH changes")
	} else {
		fmt.Println("  1. Open a new terminal or run: source ~/.bashrc (or ~/.zshrc)")
	}
	fmt.Println("  2. Browse more plugins at: https://registry.bluelink.dev")
	fmt.Println("  3. Install plugins:   bluelink plugins install newstack-cloud/<plugin>")
	fmt.Println("  4. Get started:       bluelink --help")
	fmt.Println()
}
