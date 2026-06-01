package project

import (
	"os"
	"path/filepath"
)

// DefaultBlueprintFile is the blueprint file name used when no existing
// project blueprint file can be detected in the working directory.
const DefaultBlueprintFile = "project.blueprint.yaml"

// blueprintFileCandidates lists the project blueprint file names to probe for,
// in priority order. The serialised formats take precedence over the blueprint
// language so existing YAML/JSONC projects keep their current behaviour.
var blueprintFileCandidates = []string{
	"project.blueprint.yaml",
	"project.blueprint.yml",
	"project.blueprint.jsonc",
	"project.blueprint.json",
	"project.bp",
	"project.blueprint",
}

// DetectBlueprintFile returns the name of the first existing project blueprint
// file in the given directory, probing the supported formats in priority order.
// It falls back to DefaultBlueprintFile when none are present.
func DetectBlueprintFile(directory string) string {
	for _, candidate := range blueprintFileCandidates {
		if _, err := os.Stat(filepath.Join(directory, candidate)); err == nil {
			return candidate
		}
	}

	return DefaultBlueprintFile
}
