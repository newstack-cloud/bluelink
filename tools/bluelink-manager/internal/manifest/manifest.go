package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
)

// Manifest tracks installed profiles and component versions.
type Manifest struct {
	Profiles   map[string]InstalledProfile   `json:"profiles"`
	Components map[string]InstalledComponent `json:"components"`
}

// InstalledProfile records when a profile was installed and last updated.
type InstalledProfile struct {
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// InstalledComponent records a component's version and which profiles use it.
type InstalledComponent struct {
	Version  string   `json:"version"`
	Profiles []string `json:"profiles"`
}

// New returns an empty, initialized manifest.
func New() *Manifest {
	return &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}
}

func manifestPath() string {
	return filepath.Join(paths.InstallDir(), "manifest.json")
}

// Load reads the manifest from disk. Returns an empty manifest if the file
// does not exist (backward compatibility with pre-profile installs).
func Load() (*Manifest, error) {
	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	data, err := os.ReadFile(manifestPath())
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}

	// Ensure maps are initialized even if JSON had nulls
	if m.Profiles == nil {
		m.Profiles = make(map[string]InstalledProfile)
	}
	if m.Components == nil {
		m.Components = make(map[string]InstalledComponent)
	}

	return m, nil
}

// Save writes the manifest to disk.
func (m *Manifest) Save() error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath(), data, 0644)
}

// AddProfile records a profile as installed with the current timestamp.
func (m *Manifest) AddProfile(name string) {
	now := time.Now()
	if existing, ok := m.Profiles[name]; ok {
		existing.UpdatedAt = now
		m.Profiles[name] = existing
	} else {
		m.Profiles[name] = InstalledProfile{
			InstalledAt: now,
			UpdatedAt:   now,
		}
	}
}

// RemoveProfile removes a profile and cleans up component associations.
// Returns the list of binary names that are no longer needed.
func (m *Manifest) RemoveProfile(name string) []string {
	delete(m.Profiles, name)

	var removable []string
	for binaryName, comp := range m.Components {
		comp.Profiles = slices.DeleteFunc(comp.Profiles, func(p string) bool {
			return p == name
		})
		if len(comp.Profiles) == 0 {
			removable = append(removable, binaryName)
			delete(m.Components, binaryName)
		} else {
			m.Components[binaryName] = comp
		}
	}
	return removable
}

// SetComponent records a component's version and associates it with a profile.
func (m *Manifest) SetComponent(binaryName, version, profileName string) {
	comp, ok := m.Components[binaryName]
	if !ok {
		comp = InstalledComponent{
			Version:  version,
			Profiles: []string{profileName},
		}
	} else {
		comp.Version = version
		if !slices.Contains(comp.Profiles, profileName) {
			comp.Profiles = append(comp.Profiles, profileName)
		}
	}
	m.Components[binaryName] = comp
}

// InstalledProfileNames returns the names of all installed profiles.
func (m *Manifest) InstalledProfileNames() []string {
	names := make([]string, 0, len(m.Profiles))
	for name := range m.Profiles {
		names = append(names, name)
	}
	return names
}

// HasProfile returns true if the given profile is installed.
func (m *Manifest) HasProfile(name string) bool {
	_, ok := m.Profiles[name]
	return ok
}

// IsEmpty returns true if no profiles are installed.
func (m *Manifest) IsEmpty() bool {
	return len(m.Profiles) == 0
}
