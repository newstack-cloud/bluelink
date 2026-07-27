package profile

import (
	"fmt"
	"strings"
)

// Component defines a downloadable component within a profile.
type Component struct {
	Name        string // Display name (e.g., "Bluelink CLI")
	RepoOwner   string // GitHub org (e.g., "newstack-cloud")
	RepoName    string // GitHub repo (e.g., "bluelink")
	TagPrefix   string // Release tag prefix (e.g., "apps/cli")
	ArchiveName string // Archive file prefix (e.g., "bluelink")
	BinaryName  string // Installed binary name (e.g., "bluelink")
	Shared      bool   // true for components shared across profiles
}

// Profile defines a product installation profile.
type Profile struct {
	Name        string      // Identifier (e.g., "bluelink", "celerity")
	DisplayName string      // User-facing name (e.g., "Bluelink", "Celerity")
	Components  []Component // Components to install
	CorePlugins []string    // Plugin identifiers to install by default
	CLIBinary   string      // CLI binary name for plugin commands
	RegistryURL string      // Plugin registry URL
}

// Get returns a profile by name. Returns an error if the profile is not found.
func Get(name string) (*Profile, error) {
	name = strings.ToLower(name)
	for _, p := range All() {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unknown profile %q (available: bluelink, celerity)", name)
}

// All returns all built-in profiles.
func All() []*Profile {
	return []*Profile{&Bluelink, &Celerity}
}

// Resolve resolves a list of profile names to profiles.
// An empty list defaults to ["bluelink"].
func Resolve(names []string) ([]*Profile, error) {
	if len(names) == 0 {
		names = []string{"bluelink"}
	}

	seen := make(map[string]bool)
	var profiles []*Profile
	for _, name := range names {
		name = strings.ToLower(name)
		if seen[name] {
			continue
		}
		seen[name] = true

		p, err := Get(name)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// MergeComponents returns a deduplicated list of components across profiles.
// Shared components (same BinaryName) appear only once.
func MergeComponents(profiles ...*Profile) []Component {
	seen := make(map[string]bool)
	var merged []Component
	for _, p := range profiles {
		for _, c := range p.Components {
			if !seen[c.BinaryName] {
				seen[c.BinaryName] = true
				merged = append(merged, c)
			}
		}
	}
	return merged
}

// MergePlugins returns a deduplicated union of plugin lists across profiles.
func MergePlugins(profiles ...*Profile) []string {
	seen := make(map[string]bool)
	var merged []string
	for _, p := range profiles {
		for _, plugin := range p.CorePlugins {
			if !seen[plugin] {
				seen[plugin] = true
				merged = append(merged, plugin)
			}
		}
	}
	return merged
}

// DisplayNames returns a comma-separated list of profile display names.
func DisplayNames(profiles []*Profile) string {
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.DisplayName
	}
	return strings.Join(names, " + ")
}
