package helpinfo

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// FormatLinkCardinality renders a provider.LinkCardinality as a short
// human-readable range (e.g. "exactly 1", "at least 2", "at most 5",
// "1..3", "unconstrained").
func FormatLinkCardinality(c provider.LinkCardinality) string {
	switch {
	case c.Min == 0 && c.Max == 0:
		return "unconstrained"
	case c.Min == c.Max:
		return fmt.Sprintf("exactly %d", c.Min)
	case c.Max == 0:
		return fmt.Sprintf("at least %d", c.Min)
	case c.Min == 0:
		return fmt.Sprintf("at most %d", c.Max)
	default:
		return fmt.Sprintf("%d..%d", c.Min, c.Max)
	}
}

// LinkCardinalityInfo carries the cardinality of a link in both directions
// along with the resource types on each side. It is passed to renderers so
// cardinality can be surfaced alongside annotation definitions.
type LinkCardinalityInfo struct {
	TypeA        string
	TypeB        string
	CardinalityA provider.LinkCardinality
	CardinalityB provider.LinkCardinality
}

// LinkSelectorTargetInfo describes a single resource that a linkSelector will
// link to, with the outgoing-link cardinality from the selecting resource.
// Cardinality may be nil when no link is registered between the pair of
// resource types.
type LinkSelectorTargetInfo struct {
	Name         string
	ResourceType string
	Cardinality  *provider.LinkCardinality
}
