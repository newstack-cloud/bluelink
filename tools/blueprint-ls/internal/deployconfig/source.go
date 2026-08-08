package deployconfig

import (
	"path/filepath"
	"strings"
)

// Source reads a deploy configuration file of a particular format.
//
// A source is only ever handed a concrete path that already exists; it never
// decides where to look. Which paths are searched is the resolver's concern, so
// that a project can name its configuration whatever it likes without every
// source needing to know about it. Sources exist purely so format-specific and
// tool-specific knowledge stays isolated.
type Source interface {
	// Recognises reports whether this source can read the file at path, judged
	// from its name. This is how a configured path, whose name the resolver has
	// never seen before, is matched to a format.
	Recognises(path string) bool
	// Load reads the file and converts it to the canonical configuration.
	Load(path string) (*Config, error)
}

// DefaultCandidatePaths returns the conventional relative paths searched for in
// each directory, in preference order.
//
// The canonical Bluelink file comes first as it is the documented convention for a
// Bluelink project, needs no conversion, and its presence is a deliberate choice
// rather than a build artifact.
//
// Celerity's authoring file is preferred over the file Celerity generates from
// it, because the authoring file is the source of truth and is present whenever
// the project is configured at all, whereas the generated file only appears once
// a CLI command has been run. Preferring the generated file would make
// diagnostics depend on whether the user happened to run the CLI recently.
//
// Per-environment files (bluelink.deploy.dev.jsonc and so on) are deliberately
// absent as nothing in the filesystem says which environment is intended, so they
// are named explicitly instead. Template files are absent for the same reason a
// template is not configuration, they hold placeholders rather than values.
func DefaultCandidatePaths() []string {
	return []string{
		BluelinkDeployConfigFile,
		"bluelink.deploy.json",
		CelerityAppConfigFile,
		"app.deploy.json",
		filepath.Join(".celerity", "deploy-config.json"),
	}
}

// DefaultSources returns the sources used to match a path to a format.
//
// Order matters only for a path that more than one source would recognise, which
// the conventional names avoid.
func DefaultSources() []Source {
	return []Source{
		&CelerityAppSource{},
		&CelerityGeneratedSource{},
		&BluelinkSource{},
	}
}

// Picks the source that recognises a path, falling back to the
// canonical format.
//
// The fallback matters for a configured path with a name none of the
// conventions cover, the canonical format is what the deploy engine itself
// accepts, so it is the right assumption for a file the user named directly.
func sourceForPath(path string, sources []Source) Source {
	for _, source := range sources {
		if source.Recognises(path) {
			return source
		}
	}

	return &BluelinkSource{}
}

func hasBaseName(path string, name string) bool {
	return filepath.Base(path) == name
}

func hasBaseNamePrefix(path string, prefix string) bool {
	return strings.HasPrefix(filepath.Base(path), prefix)
}
