package headless

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// isHeadlessOverride allows tests to override headless detection
var isHeadlessOverride *bool

// IsHeadless returns true if not running in an interactive terminal
func IsHeadless() bool {
	if isHeadlessOverride != nil {
		return *isHeadlessOverride
	}
	return !term.IsTerminal(int(os.Stdout.Fd()))
}

// SetHeadlessForTesting overrides IsHeadless for testing purposes.
// Returns a cleanup function that restores the original behavior.
func SetHeadlessForTesting(headless bool) func() {
	isHeadlessOverride = &headless
	return func() { isHeadlessOverride = nil }
}

// Flag represents a flag value with its default status
type Flag struct {
	Name      string // Flag name for error messages (e.g., "blueprint-file")
	Value     string // Current value
	IsDefault bool   // True if using default value (not explicitly set)
}

// Requirement defines a validation rule
type Requirement interface {
	Validate() error
}

// Required creates a rule that the flag must be explicitly provided
func Required(f Flag) Requirement {
	return &requiredRule{flag: f}
}

type requiredRule struct {
	flag Flag
}

func (r *requiredRule) Validate() error {
	if r.flag.IsDefault || r.flag.Value == "" {
		return fmt.Errorf("required flag --%s must be provided in non-interactive mode", r.flag.Name)
	}
	return nil
}

// OneOf creates a rule that at least one of the flags must be provided
func OneOf(flags ...Flag) Requirement {
	return &oneOfRule{flags: flags}
}

type oneOfRule struct {
	flags []Flag
}

func (r *oneOfRule) Validate() error {
	for _, f := range r.flags {
		if !f.IsDefault && f.Value != "" {
			return nil // At least one is set
		}
	}
	names := make([]string, len(r.flags))
	for i, f := range r.flags {
		names[i] = "--" + f.Name
	}
	return fmt.Errorf("one of %s must be provided in non-interactive mode", strings.Join(names, " or "))
}

// Validate checks all requirements, but only if in headless mode.
// In interactive mode, returns nil (TUI will handle missing values).
func Validate(requirements ...Requirement) error {
	if !IsHeadless() {
		return nil
	}

	var errs []string
	for _, req := range requirements {
		if err := req.Validate(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return fmt.Errorf("%s", errs[0])
	}
	return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
}
