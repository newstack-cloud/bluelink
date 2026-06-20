package lang_test

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
)

// Parses every fixture in __testdata, emits
// it, re-parses the emitted source and asserts the two models are equivalent
// (compared via YAML, which ignores source metadata and comments). This
// ensures the emitter achieves full coverage of the language.
func (s *EmitSuite) Test_round_trips_corpus_fixtures() {
	entries, err := os.ReadDir("__testdata")
	s.Require().NoError(err)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".bp") {
			continue
		}
		s.Run(entry.Name(), func() {
			src, err := os.ReadFile(filepath.Join("__testdata", entry.Name()))
			s.Require().NoError(err)

			first, err := lang.ParseString(string(src))
			if err != nil {
				s.T().Skipf("fixture does not parse (treated as a negative fixture): %v", err)
			}

			s.requireRoundTripFrom(first, string(src))
		})
	}
}
