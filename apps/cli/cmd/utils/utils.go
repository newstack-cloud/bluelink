package utils

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

// Version info set by the commands package at init time.
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// WrappedFlagUsages wraps long descriptions for flags,
// this uses the users terminal size or
// width of 80 if cannot determine users width.
func WrappedFlagUsages(cmd *pflag.FlagSet) string {
	fd := int(os.Stdout.Fd())
	width := 80

	// Get the terminal width and dynamically set
	termWidth, _, err := term.GetSize(fd)
	if err == nil {
		width = termWidth
	}

	return cmd.FlagUsagesWrapped(width - 1)
}

// Logo is the ASCII art logo displayed in help output.
const Logo = `
  _     _            _ _       _
 | |   | |          | (_)     | |
 | |__ | |_   _  ___| |_ _ __ | | __
 | '_ \| | | | |/ _ \ | | '_ \| |/ /
 | |_) | | |_| |  __/ | | | | |   <
 |_.__/|_|\__,_|\___|_|_|_| |_|_|\_\
`

// VersionInfo returns a formatted string with version information.
func VersionInfo() string {
	return fmt.Sprintf("\nBluelink CLI %s (%s/%s, built %s)", Version, runtime.GOOS, runtime.GOARCH, BuildTime)
}

// UsageTemplate is identical to the default cobra usage template,
// but utilises WrappedFlagUsages to ensure flag usages don't wrap around.
// The logo and version info are prepended to the usage output.
var UsageTemplate = Logo + `{{versionInfo}}

Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{wrappedFlagUsages .LocalFlags | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{wrappedFlagUsages .InheritedFlags | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var HelpTemplate = `
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// SetupLogger creates a zap logger instance that writes to a file.
// Due to the CLI heavily using bubbletea to provide interactive experiences,
// we log to a file by default.
func SetupLogger() (*zap.Logger, *os.File, error) {
	logFileHandle, err := os.OpenFile("bluelink.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	writerSync := zapcore.NewMultiWriteSyncer(
		// stdout and stdin are used for communication with the client
		// and should not be logged to.
		// zapcore.AddSync(os.Stderr),
		zapcore.AddSync(logFileHandle),
	)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		writerSync,
		zap.DebugLevel,
	)
	logger := zap.New(core)
	return logger, logFileHandle, nil
}
