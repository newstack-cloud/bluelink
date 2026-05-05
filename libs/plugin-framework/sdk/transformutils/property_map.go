package transformutils

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// PropertyMap is a declarative description of how an abstract
// resource's substitution-referenceable properties map to a target's
// concrete equivalents. The majority of cases will be 1:1 renames and
// value-ref redirects; a small number of cases will involve structural
// reshapes or literal index injections.
type PropertyMap struct {
	// Renames contains simple 1:1 mappings of abstract property name to concrete property name.
	// The key is a dotted abstract path and value consists of concrete path
	// segments. Array indices in the abstract path are auto-preserved at their
	// original positions (RewriteFields semantics).
	Renames map[string][]string

	// ValueRefs contains mappings of abstract refs that should redirect to
	// a transformer-derived value. For example, a composite abstract value
	// derived from multiple concrete resource properties.
	// The key is a dotted abstract path and values describe the value reference.
	ValueRefs map[string]*ValueRefSpec

	// Custom contains custom rules that don't fit into the rename
	// or value-ref categories.
	Custom []*PropertyRule
}

// Rewriter materialises the PropertyMap as a ResourcePropertyRewriter
// closure parameterised with the abstract ↔ concrete name pair from one
// resolved primary.
//
// Renames / ValueRefs key splits and Custom MatchPaths patterns are
// pre-compiled once per closure so per-ref dispatch is O(rules) without
// repeated string parsing.
func (pm *PropertyMap) Rewriter(abstractName, concreteName string) ResourcePropertyRewriter {
	if pm == nil {
		return func(*substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
			return nil
		}
	}

	renameSegments := make(map[string][]string, len(pm.Renames))
	for key := range pm.Renames {
		renameSegments[key] = strings.Split(key, ".")
	}

	valueRefSegments := make(map[string][]string, len(pm.ValueRefs))
	for key := range pm.ValueRefs {
		valueRefSegments[key] = strings.Split(key, ".")
	}

	rules := compileCustomRules(pm.Custom)

	rewriteCtx := RewriteContext{
		AbstractName: abstractName,
		ConcreteName: concreteName,
	}

	return func(ref *substitutions.SubstitutionResourceProperty) *substitutions.Substitution {
		if ref == nil || ref.ResourceName != abstractName {
			return nil
		}

		for key, newFields := range pm.Renames {
			if PathExact(ref, renameSegments[key]...) {
				return RewriteFields(ref, concreteName, newFields...)
			}
		}

		for key, spec := range pm.ValueRefs {
			if spec == nil {
				continue
			}
			if PathExact(ref, valueRefSegments[key]...) {
				return ValueRef(concreteName+spec.Suffix, spec.Path...)
			}
		}

		for _, rule := range rules {
			if !ruleMatchesPath(ref, rule.patterns) {
				continue
			}
			if rule.predicate != nil && !rule.predicate(ref) {
				continue
			}
			if rule.rewrite == nil {
				return nil
			}
			return rule.rewrite(ref, rewriteCtx)
		}

		return nil
	}
}

// compiledCustomRule is a Custom PropertyRule with its MatchPaths pre-
// parsed into patternItem sequences. Built once per Rewriter call so
// each ref dispatch is a slice walk, not a re-parse.
type compiledCustomRule struct {
	patterns  [][]patternItem
	predicate func(*substitutions.SubstitutionResourceProperty) bool
	rewrite   func(*substitutions.SubstitutionResourceProperty, RewriteContext) *substitutions.Substitution
}

func compileCustomRules(rules []*PropertyRule) []compiledCustomRule {
	compiled := make([]compiledCustomRule, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		entry := compiledCustomRule{
			predicate: rule.Predicate,
			rewrite:   rule.Rewrite,
		}
		for _, pattern := range rule.MatchPaths {
			entry.patterns = append(entry.patterns, parsePathPattern(pattern))
		}
		compiled = append(compiled, entry)
	}
	return compiled
}

func ruleMatchesPath(
	ref *substitutions.SubstitutionResourceProperty,
	patterns [][]patternItem,
) bool {
	for _, pattern := range patterns {
		if matchPathPattern(ref, pattern) {
			return true
		}
	}
	return false
}

// patternItem is one parsed element of a Custom rule MatchPaths pattern.
// Either fieldName is non-empty (a literal field segment) or arrayWild is
// true (a "[*]" segment that requires an array-index path item).
type patternItem struct {
	fieldName string
	arrayWild bool
}

// Splits a dotted pattern like "spec.foo[*].bar" into a
// sequence of patternItems. Each "." separates segments; trailing "[*]"
// (one or more, e.g. "matrix[*][*]") on a segment produces array-wildcard
// items after the segment's field. An empty pattern yields zero items
// (matches only an empty ref.Path).
func parsePathPattern(pattern string) []patternItem {
	items := []patternItem{}
	if pattern == "" {
		return items
	}
	for part := range strings.SplitSeq(pattern, ".") {
		for {
			idx := strings.Index(part, "[*]")
			if idx == -1 {
				break
			}
			if idx > 0 {
				items = append(items, patternItem{fieldName: part[:idx]})
			}
			items = append(items, patternItem{arrayWild: true})
			part = part[idx+len("[*]"):]
		}
		if part != "" {
			items = append(items, patternItem{fieldName: part})
		}
	}
	return items
}

// matchPathPattern returns true iff ref.Path matches items position-for-
// position: every field item must match exactly, and every "[*]" item
// must correspond to an array-index path item. Lengths must match —
// pattern matching is exact, unlike PathExact which ignores arrays.
func matchPathPattern(
	ref *substitutions.SubstitutionResourceProperty,
	items []patternItem,
) bool {
	if ref == nil {
		return len(items) == 0
	}
	if len(ref.Path) != len(items) {
		return false
	}
	for i, item := range items {
		pathItem := ref.Path[i]
		if pathItem == nil {
			return false
		}
		if item.arrayWild {
			if pathItem.ArrayIndex == nil {
				return false
			}
			continue
		}
		if pathItem.FieldName != item.fieldName {
			return false
		}
	}
	return true
}

// ValueRefSpec describes how to construct a value reference for a property
// that doesn't have a simple 1:1 mapping to a concrete resource property.
type ValueRefSpec struct {
	// Suffix appended to the concrete resource name to form the value name.
	// e.g. "_arn" -> ${values.<concreteName>_arn}
	Suffix string
	// Path is for complex derived values (e.g. {url, authType} objects).
	// The path descends into the value. Empty path is a flat ref.
	Path []*substitutions.SubstitutionPathItem
}

// PropertyRule describes a custom rule for rewriting a resource property reference
// that doesn't fit into the rename or value-ref categories. The Match function
// determines whether the rule applies to a given reference, and the Rewrite
// function produces the new substitution if it does.
type PropertyRule struct {
	// MatchPaths is the path-pattern set this rule handles.
	// Patterns may contain "[*]" to match an array index (e.g. "spec.vpc.securityGroups[*].id").
	// MatchPaths is the single source of truth for both runtime matching and capabilitiy extraction.
	MatchPaths []string

	// Predicate is an optional further filter beyond the path match.
	// Most rules will only need match paths and leave this as nil.
	// When defined, the rule applies only if the substitution's path
	// matches one of MatchPaths AND Predicate returns true.
	// Predicate-conditional matches are intentionally outside the capabilities
	// scope (capabilities are purely path-based); document conditional
	// behaviour via abstract resource or link definitions for users and your
	// own developer guidance for future maintainers.
	Predicate func(*substitutions.SubstitutionResourceProperty) bool

	// Rewrite produces the new substitution if the rule matches.
	Rewrite func(
		ref *substitutions.SubstitutionResourceProperty,
		rewriteCtx RewriteContext,
	) *substitutions.Substitution
}

// RewriteContext provides contextual information for custom property rewrite rules.
type RewriteContext struct {
	AbstractName string
	ConcreteName string
}

type Capabilities struct {
	// SupportedAbstractPaths is the set of dot notation abstract
	// property paths this (target, resource-type) pair handles.
	// This is generated from a PropertyMap.
	SupportedAbstractPaths []string
}

// CapabilitiesFromPropertyMap derives a Capabilities struct from a property map's
// rename and value reference paths.
func CapabilitiesFromPropertyMap(pm *PropertyMap) *Capabilities {
	supportedPaths := make([]string, 0)
	for path := range pm.Renames {
		supportedPaths = append(supportedPaths, path)
	}

	for path := range pm.ValueRefs {
		supportedPaths = append(supportedPaths, path)
	}

	for _, rule := range pm.Custom {
		supportedPaths = append(supportedPaths, rule.MatchPaths...)
	}

	return &Capabilities{
		SupportedAbstractPaths: supportedPaths,
	}
}
