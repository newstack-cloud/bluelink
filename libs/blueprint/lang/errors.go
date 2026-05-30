package lang

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
)

const (
	childErrorsFormatStr = "\n\t- %s"
)

// LexError represents a single error encountered during lexing,
// with a message and source position.
type LexError struct {
	Message    string
	SourceMeta *source.Meta
}

func (e *LexError) Error() string {
	if e.SourceMeta == nil {
		return fmt.Sprintf("lex error: %s", e.Message)
	}
	return fmt.Sprintf(
		"lex error: %s at %d:%d",
		e.Message,
		e.SourceMeta.Position.Line,
		e.SourceMeta.Position.Column,
	)
}

// LexErrors represents multiple errors encountered during lexing,
// with a message and a list of child errors.
type LexErrors struct {
	Message     string
	ChildErrors []error
}

func (e *LexErrors) Error() string {
	var errStr strings.Builder
	fmt.Fprintf(&errStr, "lex errors: %s", e.Message)
	for _, err := range e.ChildErrors {
		fmt.Fprintf(&errStr, childErrorsFormatStr, err.Error())
	}
	return errStr.String()
}

// ParseError represents a single error encountered during parsing,
// with a message and source position.
type ParseError struct {
	Message    string
	SourceMeta *source.Meta
}

func (e *ParseError) Error() string {
	if e.SourceMeta == nil {
		return fmt.Sprintf("parse error: %s", e.Message)
	}
	return fmt.Sprintf(
		"parse error: %s at %d:%d",
		e.Message,
		e.SourceMeta.Position.Line,
		e.SourceMeta.Position.Column,
	)
}

// Errors aggregates the diagnostics collected across lexing and parsing
// into a single returnable error. When Source is non-empty, each child
// error is rendered with a source snippet showing the offending line and a
// caret column-aligned under the column the error points at.
type Errors struct {
	ChildErrors []error
	// Source is the original blueprint-language source. Optional; when set,
	// Error() renders a source-snippet under each child diagnostic.
	Source string
}

func (e *Errors) Error() string {
	var errStr strings.Builder
	errStr.WriteString("blueprint language errors:")
	for _, err := range e.ChildErrors {
		fmt.Fprintf(&errStr, childErrorsFormatStr, err.Error())
		if e.Source != "" {
			errStr.WriteString(snippetForError(e.Source, err))
		}
	}
	return errStr.String()
}

type diagnostics struct {
	errs []error
	// cap to avoid runaway on pathological input (Go's scanner uses 10)
	max int
	// elided counts diagnostics dropped after the cap is hit, so the final
	// Errors envelope can tell the user how many were suppressed.
	elided int
	// source is the original blueprint-language source; passed through to
	// Errors so its Error() output can render source snippets.
	source string
}

func (d *diagnostics) add(err error) {
	if len(d.errs) < d.max {
		d.errs = append(d.errs, err)
		return
	}
	d.elided++
}

func (d *diagnostics) asError() error {
	if len(d.errs) == 0 {
		return nil
	}
	errs := d.errs
	if d.elided > 0 {
		errs = append(errs, &ParseError{
			Message: fmt.Sprintf(
				"%d more error(s) elided (raise the diagnostic cap to see them)",
				d.elided,
			),
		})
	}
	return &Errors{ChildErrors: errs, Source: d.source}
}

// snippetForError extracts the source position from a child diagnostic and
// returns the source-snippet block to append under its message. Returns an
// empty string when the error carries no position or the position is out of
// range.
func snippetForError(src string, err error) string {
	pos, ok := errorPosition(err)
	if !ok {
		return ""
	}
	return renderSourceSnippet(src, pos)
}

func errorPosition(err error) (source.Position, bool) {
	switch e := err.(type) {
	case *ParseError:
		if e.SourceMeta == nil {
			return source.Position{}, false
		}
		return e.SourceMeta.Position, true
	case *LexError:
		if e.SourceMeta == nil {
			return source.Position{}, false
		}
		return e.SourceMeta.Position, true
	}
	return source.Position{}, false
}

// renders a two-line snippet showing the source line that
// the error references and a caret column-aligned under the offending column,
// rustc-style. The leading "\n\t" matches the bullet indentation used by
// Errors.Error() so the snippet sits cleanly under its diagnostic.
func renderSourceSnippet(src string, pos source.Position) string {
	lines := strings.Split(src, "\n")
	if pos.Line < 1 || pos.Line > len(lines) {
		return ""
	}
	line := lines[pos.Line-1]

	lineNumStr := fmt.Sprintf("%d", pos.Line)
	gutterPad := strings.Repeat(" ", len(lineNumStr))

	caretCol := max(pos.Column, 1)
	caretPad := strings.Repeat(" ", caretCol-1)

	return fmt.Sprintf(
		"\n\t  %s | %s\n\t  %s | %s^",
		lineNumStr, line,
		gutterPad, caretPad,
	)
}
