// Package strsim provides string similarity functions for fuzzy matching.
package strsim

import (
	"cmp"
	"slices"
	"strings"
)

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to transform string a into string b.
//
// This is a dynamic programming algorithm that builds a matrix where each cell [i][j]
// represents the edit distance between the first i characters of a and the first j characters of b.
//
// Time complexity: O(len(a) * len(b))
// Space complexity: O(len(a) * len(b))
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create a matrix with dimensions (len(a)+1) x (len(b)+1)
	// The +1 accounts for the empty string prefix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		// First column: cost of deleting all characters from a[0:i]
		matrix[i][0] = i
	}
	// First row: cost of inserting all characters to form b[0:j]
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill the matrix using dynamic programming
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			// Cost is 0 if characters match, 1 otherwise
			substitutionCost := 1
			if a[i-1] == b[j-1] {
				substitutionCost = 0
			}

			// Take the minimum of three operations:
			// 1. Delete from a: matrix[i-1][j] + 1
			// 2. Insert into a: matrix[i][j-1] + 1
			// 3. Substitute (or match): matrix[i-1][j-1] + substitutionCost
			matrix[i][j] = min(
				matrix[i-1][j]+1,                  // deletion
				matrix[i][j-1]+1,                  // insertion
				matrix[i-1][j-1]+substitutionCost, // substitution or match
			)
		}
	}

	// The bottom-right cell contains the final edit distance
	return matrix[len(a)][len(b)]
}

// FindSimilar returns strings from candidates that are similar to the target string,
// using Levenshtein distance for fuzzy matching. Results are sorted by similarity
// (most similar first) and limited to maxResults.
//
// The maxDistance parameter controls how different a candidate can be from the target.
// If maxDistance <= 0, a default threshold of max(len(target)/2, 2) is used.
//
// Comparison is case-insensitive.
func FindSimilar(target string, candidates []string, maxResults int, maxDistance int) []string {
	if len(candidates) == 0 || maxResults <= 0 {
		return nil
	}

	// Use default threshold if not specified
	if maxDistance <= 0 {
		maxDistance = max(len(target)/2, 2)
	}

	type match struct {
		value    string
		distance int
	}

	targetLower := strings.ToLower(target)
	var matches []match

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)
		dist := LevenshteinDistance(targetLower, candidateLower)
		if dist <= maxDistance {
			matches = append(matches, match{value: candidate, distance: dist})
		}
	}

	// Sort by distance (ascending), then alphabetically for stable ordering
	slices.SortFunc(matches, func(a, b match) int {
		if a.distance != b.distance {
			return cmp.Compare(a.distance, b.distance)
		}
		return cmp.Compare(a.value, b.value)
	})

	// Limit results
	resultCount := min(maxResults, len(matches))
	result := make([]string, resultCount)
	for i := range resultCount {
		result[i] = matches[i].value
	}

	return result
}
