package lang

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
)

// ParseString parses a blueprint language source string and returns
// a standardised blueprint model.
// This returns a structured error with source positions
// on any parse failures.
func ParseString(src string) (*schema.Blueprint, error) {
	return newParser(src).parse()
}

// ParseFile reads a .blueprint / .bp file and returns
// a standardised blueprint model.
// This returns a structured error with source positions
// on any parse failures.
func ParseFile(path string) (*schema.Blueprint, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseString(string(contents))
}
