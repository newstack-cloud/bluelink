package validation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/common/strsim"
)

const (
	// SuggestionsMetadataKey is the key in the metadata of an error context that
	// holds the values similar to the one that could not be resolved, ordered from
	// the most similar to the least similar.
	// Tools that surface diagnostics use this to offer corrections for the value
	// (e.g. quick fixes in the language server).
	SuggestionsMetadataKey = "suggestions"
	// AvailableValuesMetadataKey is the key in the metadata of an error context that
	// holds all the values that could have been used in the place of the one that
	// could not be resolved.
	AvailableValuesMetadataKey = "availableValues"

	// The number of suggestions to offer for a value that could
	// not be resolved, enough to cover a value that is close to more than one of the
	// available values without overwhelming the message shown to a user.
	maxSuggestions = 3
	// Caps the values reported for a lookup that failed.
	// A plugin can provide a large number of item types and listing all of them is
	// of little use to a user, the suggestions are what usually resolves the issue.
	maxAvailableValues = 50
	// The greatest number of edits that a value can be away
	// from the one that could not be resolved and still be offered as a suggestion.
	// Values such as resource types are long, the default threshold scales with the
	// length of the value and would offer values that are unrelated to the one that
	// was provided. (e.g. "eu-west-1" for "eu-central-4")
	maxSuggestionDistance = 3
	// The minimum suggestion distance keeps short values from being compared so strictly that
	// a transposition of two characters is missed. (e.g. "naem" for "name")
	minSuggestionDistance = 2
	// The maximum options in summary caps the options listed in the message of an error, a type
	// with a large number of options would otherwise produce a message that is
	// difficult to read.
	maxOptionsInSummary = 10
)

func suggestionDistanceFor(value string) int {
	return min(maxSuggestionDistance, max(minSuggestionDistance, len(value)/4))
}

// OptionsSummary produces a summary of the values that can be used in the place of
// a value that could not be resolved, for errors where the set of values is small
// and known up front (e.g. the options for a custom variable type).
// Listing the values in the message of an error saves a user from having to look
// them up elsewhere.
func OptionsSummary(options []string) string {
	if len(options) == 0 {
		return "the custom variable type does not provide any options"
	}

	// Options are collected from a map, sort them so that the message produced for
	// a blueprint is the same each time it is validated.
	sorted := slices.Sorted(slices.Values(options))

	if len(sorted) <= maxOptionsInSummary {
		return fmt.Sprintf(
			"the options for the type are: %s",
			strings.Join(sorted, ", "),
		)
	}

	return fmt.Sprintf(
		"the options for the type include: %s and %d more",
		strings.Join(sorted[:maxOptionsInSummary], ", "),
		len(sorted)-maxOptionsInSummary,
	)
}

// AddSuggestionsToMetadata adds the values similar to the given value, along with
// the values that were available, to the metadata of an error context.
// The metadata is returned unchanged when there are no available values to
// compare against, which is the case when the source of the available values could
// not be reached.
func AddSuggestionsToMetadata(
	metadata map[string]any,
	value string,
	availableValues []string,
) map[string]any {
	withSuggestions := AddSuggestionsOnlyToMetadata(metadata, value, availableValues)

	if len(availableValues) > 0 && len(availableValues) <= maxAvailableValues {
		withSuggestions[AvailableValuesMetadataKey] = slices.Sorted(
			slices.Values(availableValues),
		)
	}

	return withSuggestions
}

// AddSuggestionsOnlyToMetadata adds the values similar to the given value to the
// metadata of an error context without the values that were available.
// This is for errors that already list the values that could have been used in
// their message (see OptionsSummary), where recording them a second time only
// leads to tools reporting the same list twice.
func AddSuggestionsOnlyToMetadata(
	metadata map[string]any,
	value string,
	availableValues []string,
) map[string]any {
	if len(availableValues) == 0 {
		return metadata
	}

	suggestions := strsim.FindSimilar(
		value,
		availableValues,
		maxSuggestions,
		suggestionDistanceFor(value),
	)
	if len(suggestions) > 0 {
		metadata[SuggestionsMetadataKey] = suggestions
	}

	return metadata
}
