package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// Parse parses a version string (e.g., "1.2.3" or "1.2.3-beta.1").
func Parse(v string) (*Version, error) {
	if v == "" {
		return nil, fmt.Errorf("version cannot be empty")
	}

	// Split off prerelease suffix
	var prerelease string
	if idx := strings.Index(v, "-"); idx != -1 {
		prerelease = v[idx+1:]
		v = v[:idx]
		if prerelease == "" {
			return nil, fmt.Errorf("prerelease suffix cannot be empty")
		}
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("version must have exactly 3 parts (major.minor.patch), got %d", len(parts))
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

// String returns the string representation of the version.
func (v *Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		return base + "-" + v.Prerelease
	}
	return base
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other.
// Prerelease versions are considered less than release versions with the same major.minor.patch.
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareInt(v.Patch, other.Patch)
	}

	// Handle prerelease comparison
	// A version without prerelease is greater than one with prerelease
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	// Both have prerelease or both don't - compare lexically
	return strings.Compare(v.Prerelease, other.Prerelease)
}

// LessThan returns true if v < other.
func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// GreaterThanOrEqual returns true if v >= other.
func (v *Version) GreaterThanOrEqual(other *Version) bool {
	return v.Compare(other) >= 0
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
