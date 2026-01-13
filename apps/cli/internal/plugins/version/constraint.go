package version

import (
	"fmt"
	"strings"
)

// ConstraintType identifies the type of version constraint.
type ConstraintType int

const (
	// ConstraintExact matches only the exact version (e.g., "1.0.0").
	ConstraintExact ConstraintType = iota
	// ConstraintCaret matches minor-compatible versions (e.g., "^1.0.0" matches >=1.0.0, <2.0.0).
	ConstraintCaret
	// ConstraintTilde matches patch-compatible versions (e.g., "~1.0.0" matches >=1.0.0, <1.1.0).
	ConstraintTilde
)

// Constraint represents a parsed version constraint.
type Constraint struct {
	Type    ConstraintType
	Version *Version
}

// ParseConstraint parses a version constraint string.
// Supported formats: "1.0.0", "^1.0.0", "~1.0.0"
func ParseConstraint(s string) (*Constraint, error) {
	if s == "" {
		return nil, fmt.Errorf("constraint cannot be empty")
	}

	constraintType := ConstraintExact
	versionStr := s

	if strings.HasPrefix(s, "^") {
		constraintType = ConstraintCaret
		versionStr = s[1:]
	} else if strings.HasPrefix(s, "~") {
		constraintType = ConstraintTilde
		versionStr = s[1:]
	}

	if versionStr == "" {
		return nil, fmt.Errorf("version cannot be empty after constraint prefix")
	}

	version, err := Parse(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version in constraint %q: %w", s, err)
	}

	return &Constraint{
		Type:    constraintType,
		Version: version,
	}, nil
}

// String returns the string representation of the constraint.
func (c *Constraint) String() string {
	switch c.Type {
	case ConstraintCaret:
		return "^" + c.Version.String()
	case ConstraintTilde:
		return "~" + c.Version.String()
	default:
		return c.Version.String()
	}
}

// IsExact returns true if this is an exact version constraint.
func (c *Constraint) IsExact() bool {
	return c.Type == ConstraintExact
}

// Matches returns true if the given version satisfies this constraint.
func (c *Constraint) Matches(v *Version) bool {
	switch c.Type {
	case ConstraintExact:
		return v.Compare(c.Version) == 0
	case ConstraintCaret:
		return c.matchesCaret(v)
	case ConstraintTilde:
		return c.matchesTilde(v)
	}
	return false
}

// matchesCaret checks if v satisfies ^constraint.Version (minor-compatible).
// ^1.2.3 matches >=1.2.3 and <2.0.0
func (c *Constraint) matchesCaret(v *Version) bool {
	if v.Major != c.Version.Major {
		return false
	}
	return v.GreaterThanOrEqual(c.Version)
}

// matchesTilde checks if v satisfies ~constraint.Version (patch-compatible).
// ~1.2.3 matches >=1.2.3 and <1.3.0
func (c *Constraint) matchesTilde(v *Version) bool {
	if v.Major != c.Version.Major || v.Minor != c.Version.Minor {
		return false
	}
	return v.GreaterThanOrEqual(c.Version)
}

// FindBestMatch finds the best (highest) matching version from a list.
// Returns nil if no version matches the constraint.
func (c *Constraint) FindBestMatch(versions []*Version) *Version {
	var best *Version
	for _, v := range versions {
		if !c.Matches(v) {
			continue
		}
		if best == nil || v.Compare(best) > 0 {
			best = v
		}
	}
	return best
}
