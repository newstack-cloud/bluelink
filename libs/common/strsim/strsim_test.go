package strsim

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type LevenshteinDistanceSuite struct {
	suite.Suite
}

func (s *LevenshteinDistanceSuite) TestIdenticalStrings() {
	s.Equal(0, LevenshteinDistance("hello", "hello"))
	s.Equal(0, LevenshteinDistance("", ""))
	s.Equal(0, LevenshteinDistance("a", "a"))
	s.Equal(0, LevenshteinDistance("test string", "test string"))
}

func (s *LevenshteinDistanceSuite) TestEmptyStrings() {
	// Distance from empty to non-empty is the length of non-empty (all insertions)
	s.Equal(5, LevenshteinDistance("", "hello"))
	s.Equal(5, LevenshteinDistance("hello", ""))
	s.Equal(1, LevenshteinDistance("", "a"))
	s.Equal(1, LevenshteinDistance("a", ""))
}

func (s *LevenshteinDistanceSuite) TestSingleCharacterDifference() {
	// Insertion
	s.Equal(1, LevenshteinDistance("cat", "cart"))  // insert 'r'
	s.Equal(1, LevenshteinDistance("cat", "cats"))  // insert 's'
	s.Equal(1, LevenshteinDistance("at", "cat"))    // insert 'c'

	// Deletion
	s.Equal(1, LevenshteinDistance("cart", "cat"))  // delete 'r'
	s.Equal(1, LevenshteinDistance("cats", "cat"))  // delete 's'
	s.Equal(1, LevenshteinDistance("cat", "at"))    // delete 'c'

	// Substitution
	s.Equal(1, LevenshteinDistance("cat", "car"))   // substitute 't' -> 'r'
	s.Equal(1, LevenshteinDistance("cat", "bat"))   // substitute 'c' -> 'b'
	s.Equal(1, LevenshteinDistance("cat", "cot"))   // substitute 'a' -> 'o'
}

func (s *LevenshteinDistanceSuite) TestMultipleEdits() {
	// Two edits
	s.Equal(2, LevenshteinDistance("cat", "cars"))      // substitute t->r, insert s
	s.Equal(1, LevenshteinDistance("kitten", "sitten")) // k->s (just one substitution)
	s.Equal(3, LevenshteinDistance("kitten", "sitting")) // k->s, e->i, insert g

	// Complete replacement
	s.Equal(3, LevenshteinDistance("abc", "xyz"))    // all substitutions
	s.Equal(4, LevenshteinDistance("abcd", "wxyz"))  // all substitutions
}

func (s *LevenshteinDistanceSuite) TestCaseSensitivity() {
	// Levenshtein is case-sensitive - each different character is 1 edit
	s.Equal(1, LevenshteinDistance("Hello", "hello"))   // H->h
	s.Equal(5, LevenshteinDistance("HELLO", "hello"))   // all 5 chars different
	s.Equal(3, LevenshteinDistance("HeLLo", "hello"))   // H->h, L->l, L->l
}

func (s *LevenshteinDistanceSuite) TestRealWorldTypos() {
	// Common typos
	s.Equal(2, LevenshteinDistance("teh", "the"))           // transposition needs 2 edits
	s.Equal(2, LevenshteinDistance("recieve", "receive"))   // r-e-c-i-e-v-e vs r-e-c-e-i-v-e (2 edits)
	s.Equal(1, LevenshteinDistance("seperate", "separate")) // s-e-p-e-r-a-t-e vs s-e-p-a-r-a-t-e (just 1: e->a at pos 4)
	s.Equal(1, LevenshteinDistance("occured", "occurred"))  // insert 'r'

	// Field name typos (relevant for our use case)
	s.Equal(1, LevenshteinDistance("handleName", "handlerName"))  // insert 'r'
	s.Equal(8, LevenshteinDistance("codeLocation", "code"))       // delete "Location"
	s.Equal(1, LevenshteinDistance("timout", "timeout"))          // insert 'e'
	s.Equal(1, LevenshteinDistance("memmory", "memory"))          // delete extra 'm'
}

func (s *LevenshteinDistanceSuite) TestSymmetry() {
	// Distance should be symmetric: d(a,b) == d(b,a)
	s.Equal(LevenshteinDistance("abc", "xyz"), LevenshteinDistance("xyz", "abc"))
	s.Equal(LevenshteinDistance("hello", "hallo"), LevenshteinDistance("hallo", "hello"))
	s.Equal(LevenshteinDistance("", "test"), LevenshteinDistance("test", ""))
}

func (s *LevenshteinDistanceSuite) TestTriangleInequality() {
	// d(a,c) <= d(a,b) + d(b,c) (triangle inequality)
	a, b, c := "hello", "hallo", "hillo"
	dAB := LevenshteinDistance(a, b)
	dBC := LevenshteinDistance(b, c)
	dAC := LevenshteinDistance(a, c)
	s.LessOrEqual(dAC, dAB+dBC)
}

func TestLevenshteinDistanceSuite(t *testing.T) {
	suite.Run(t, new(LevenshteinDistanceSuite))
}

type FindSimilarSuite struct {
	suite.Suite
}

func (s *FindSimilarSuite) TestEmptyCandidates() {
	result := FindSimilar("target", []string{}, 3, 0)
	s.Nil(result)
}

func (s *FindSimilarSuite) TestZeroMaxResults() {
	result := FindSimilar("target", []string{"target"}, 0, 0)
	s.Nil(result)
}

func (s *FindSimilarSuite) TestExactMatch() {
	candidates := []string{"code", "codeLocation", "handler", "timeout"}
	result := FindSimilar("code", candidates, 3, 0)

	s.Require().NotEmpty(result)
	s.Equal("code", result[0]) // Exact match should be first (distance 0)
}

func (s *FindSimilarSuite) TestSimilarFieldNames() {
	candidates := []string{"handlerName", "timeout", "memory", "runtime", "handler"}

	// "handleName" is 1 edit away from "handlerName" (missing 'r')
	result := FindSimilar("handleName", candidates, 3, 0)

	s.Require().NotEmpty(result)
	s.Contains(result, "handlerName")
}

func (s *FindSimilarSuite) TestCaseInsensitivity() {
	candidates := []string{"HandlerName", "TIMEOUT", "Memory"}

	result := FindSimilar("handlername", candidates, 3, 0)

	s.Require().NotEmpty(result)
	// Should find "HandlerName" despite case difference
	s.Equal("HandlerName", result[0])
}

func (s *FindSimilarSuite) TestNoMatchesWithinThreshold() {
	candidates := []string{"completely", "different", "words"}

	// "xyz" is very different from all candidates
	result := FindSimilar("xyz", candidates, 3, 2)

	s.Empty(result)
}

func (s *FindSimilarSuite) TestResultsLimitedToMaxResults() {
	candidates := []string{"aa", "ab", "ac", "ad", "ae"}

	// All are 1 edit from "a", but we only want 2 results
	result := FindSimilar("a", candidates, 2, 0)

	s.Len(result, 2)
}

func (s *FindSimilarSuite) TestSortedByDistance() {
	candidates := []string{"codebase", "code", "coded", "coder"}

	result := FindSimilar("code", candidates, 4, 5)

	s.Require().Len(result, 4)
	// "code" (distance 0) should be first
	s.Equal("code", result[0])
	// "coded" and "coder" (distance 1) should come next (alphabetically sorted)
	s.Contains(result[1:3], "coded")
	s.Contains(result[1:3], "coder")
}

func (s *FindSimilarSuite) TestStableSortingForEqualDistances() {
	candidates := []string{"zebra", "apple", "mango"}

	// All are equidistant from "xxxxx" (5 substitutions each)
	result := FindSimilar("xxxxx", candidates, 3, 5)

	s.Require().Len(result, 3)
	// Should be alphabetically sorted when distances are equal
	s.Equal("apple", result[0])
	s.Equal("mango", result[1])
	s.Equal("zebra", result[2])
}

func (s *FindSimilarSuite) TestDefaultThreshold() {
	candidates := []string{"handlerName", "h", "completely_different_string"}

	// For "handleName" (10 chars), default threshold is max(10/2, 2) = 5
	// "handlerName" is 1 edit away - should match
	// "h" is 9 edits away - should not match
	// "completely_different_string" is way more - should not match
	result := FindSimilar("handleName", candidates, 3, 0)

	s.Require().Len(result, 1)
	s.Equal("handlerName", result[0])
}

func (s *FindSimilarSuite) TestCustomThreshold() {
	candidates := []string{"abc", "abcd", "abcde", "abcdef"}

	// With threshold 1, only "abc" (0) and "abcd" (1) match for "abc"
	result := FindSimilar("abc", candidates, 10, 1)
	s.Len(result, 2)

	// With threshold 3, more matches
	result = FindSimilar("abc", candidates, 10, 3)
	s.Len(result, 4)
}

func (s *FindSimilarSuite) TestRealWorldScenario_FieldNameTypos() {
	// Simulating a resource spec with common field names
	availableFields := []string{
		"handlerName",
		"runtime",
		"memory",
		"timeout",
		"environment",
		"code",
		"codeUri",
		"description",
	}

	testCases := []struct {
		typo     string
		expected []string
	}{
		{
			typo:     "handleName",      // missing 'r' (distance 1)
			expected: []string{"handlerName"},
		},
		{
			typo:     "timout",          // missing 'e' (distance 1)
			expected: []string{"timeout"},
		},
		{
			typo:     "runtim",          // missing 'e' (distance 1)
			expected: []string{"runtime"},
		},
		{
			typo:     "enviroment",      // missing 'n' (distance 1)
			expected: []string{"environment"},
		},
		{
			typo:     "memry",           // missing 'o' (distance 1)
			expected: []string{"memory"},
		},
	}

	for _, tc := range testCases {
		result := FindSimilar(tc.typo, availableFields, 3, 0)
		s.Require().NotEmpty(result, "Expected suggestions for typo: %s", tc.typo)
		for _, exp := range tc.expected {
			s.Contains(result, exp, "Expected %s to be suggested for typo: %s", exp, tc.typo)
		}
	}
}

func (s *FindSimilarSuite) TestPreservesOriginalCase() {
	candidates := []string{"HandlerName", "TIMEOUT", "memory"}

	// Search is case-insensitive, but results preserve original case
	result := FindSimilar("HANDLERNAME", candidates, 3, 0)

	s.Require().NotEmpty(result)
	s.Equal("HandlerName", result[0]) // Original case preserved
}

func TestFindSimilarSuite(t *testing.T) {
	suite.Run(t, new(FindSimilarSuite))
}
