package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("BLUELINK_INSTALL_DIR", tmpDir)
}

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	setupTestDir(t)

	m, err := Load()
	require.NoError(t, err)
	assert.Empty(t, m.Profiles)
	assert.Empty(t, m.Components)
	assert.True(t, m.IsEmpty())
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	setupTestDir(t)

	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}
	m.AddProfile("bluelink")
	m.SetComponent("bluelink", "1.0.0", "bluelink")
	m.SetComponent("deploy-engine", "2.0.0", "bluelink")

	err := m.Save()
	require.NoError(t, err)

	loaded, err := Load()
	require.NoError(t, err)
	assert.True(t, loaded.HasProfile("bluelink"))
	assert.Equal(t, "1.0.0", loaded.Components["bluelink"].Version)
	assert.Equal(t, "2.0.0", loaded.Components["deploy-engine"].Version)
}

func TestAddProfile_UpdatesTimestamp(t *testing.T) {
	setupTestDir(t)

	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	m.AddProfile("bluelink")
	first := m.Profiles["bluelink"]

	m.AddProfile("bluelink")
	second := m.Profiles["bluelink"]

	assert.Equal(t, first.InstalledAt, second.InstalledAt)
	assert.True(t, !second.UpdatedAt.Before(first.UpdatedAt))
}

func TestRemoveProfile_RemovesExclusiveComponents(t *testing.T) {
	setupTestDir(t)

	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	m.AddProfile("bluelink")
	m.SetComponent("bluelink", "1.0.0", "bluelink")
	m.SetComponent("deploy-engine", "2.0.0", "bluelink")

	removable := m.RemoveProfile("bluelink")

	assert.Contains(t, removable, "bluelink")
	assert.Contains(t, removable, "deploy-engine")
	assert.Empty(t, m.Components)
	assert.True(t, m.IsEmpty())
}

func TestRemoveProfile_KeepsSharedComponents(t *testing.T) {
	setupTestDir(t)

	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	// Both profiles share deploy-engine
	m.AddProfile("bluelink")
	m.AddProfile("celerity")
	m.SetComponent("bluelink", "1.0.0", "bluelink")
	m.SetComponent("celerity", "1.0.0", "celerity")
	m.SetComponent("deploy-engine", "2.0.0", "bluelink")
	m.SetComponent("deploy-engine", "2.0.0", "celerity")

	removable := m.RemoveProfile("celerity")

	// Only celerity binary should be removable, deploy-engine is still used by bluelink
	assert.Contains(t, removable, "celerity")
	assert.NotContains(t, removable, "deploy-engine")
	assert.NotContains(t, removable, "bluelink")

	// deploy-engine should still be tracked
	de, ok := m.Components["deploy-engine"]
	assert.True(t, ok)
	assert.Equal(t, []string{"bluelink"}, de.Profiles)

	// bluelink still installed
	assert.True(t, m.HasProfile("bluelink"))
	assert.False(t, m.HasProfile("celerity"))
}

func TestSetComponent_AddsProfileAssociation(t *testing.T) {
	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	m.SetComponent("deploy-engine", "1.0.0", "bluelink")
	m.SetComponent("deploy-engine", "1.0.0", "celerity")

	comp := m.Components["deploy-engine"]
	assert.Equal(t, []string{"bluelink", "celerity"}, comp.Profiles)
}

func TestSetComponent_DoesNotDuplicateProfile(t *testing.T) {
	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	m.SetComponent("deploy-engine", "1.0.0", "bluelink")
	m.SetComponent("deploy-engine", "1.1.0", "bluelink")

	comp := m.Components["deploy-engine"]
	assert.Equal(t, "1.1.0", comp.Version)
	assert.Equal(t, []string{"bluelink"}, comp.Profiles)
}

func TestInstalledProfileNames(t *testing.T) {
	m := &Manifest{
		Profiles:   make(map[string]InstalledProfile),
		Components: make(map[string]InstalledComponent),
	}

	m.AddProfile("bluelink")
	m.AddProfile("celerity")

	names := m.InstalledProfileNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "bluelink")
	assert.Contains(t, names, "celerity")
}

func TestManifestPath(t *testing.T) {
	setupTestDir(t)
	p := manifestPath()
	assert.Equal(t, filepath.Join(os.Getenv("BLUELINK_INSTALL_DIR"), "manifest.json"), p)
}
