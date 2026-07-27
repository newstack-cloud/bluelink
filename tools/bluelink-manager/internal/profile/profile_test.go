package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_Bluelink(t *testing.T) {
	p, err := Get("bluelink")
	require.NoError(t, err)
	assert.Equal(t, "bluelink", p.Name)
	assert.Equal(t, "Bluelink", p.DisplayName)
	assert.Equal(t, "bluelink", p.CLIBinary)
}

func TestGet_Celerity(t *testing.T) {
	p, err := Get("celerity")
	require.NoError(t, err)
	assert.Equal(t, "celerity", p.Name)
	assert.Equal(t, "Celerity", p.DisplayName)
	assert.Equal(t, "celerity", p.CLIBinary)
}

func TestGet_CaseInsensitive(t *testing.T) {
	p, err := Get("BLUELINK")
	require.NoError(t, err)
	assert.Equal(t, "bluelink", p.Name)
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
}

func TestResolve_Default(t *testing.T) {
	profiles, err := Resolve(nil)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "bluelink", profiles[0].Name)
}

func TestResolve_Multiple(t *testing.T) {
	profiles, err := Resolve([]string{"bluelink", "celerity"})
	require.NoError(t, err)
	require.Len(t, profiles, 2)
	assert.Equal(t, "bluelink", profiles[0].Name)
	assert.Equal(t, "celerity", profiles[1].Name)
}

func TestResolve_Deduplicates(t *testing.T) {
	profiles, err := Resolve([]string{"bluelink", "bluelink"})
	require.NoError(t, err)
	require.Len(t, profiles, 1)
}

func TestMergeComponents_SingleProfile(t *testing.T) {
	merged := MergeComponents(&Bluelink)
	assert.Len(t, merged, 3)

	names := make([]string, len(merged))
	for i, c := range merged {
		names[i] = c.BinaryName
	}
	assert.Contains(t, names, "bluelink")
	assert.Contains(t, names, "deploy-engine")
	assert.Contains(t, names, "blueprint-ls")
}

func TestMergeComponents_TwoProfiles_DeduplicatesShared(t *testing.T) {
	merged := MergeComponents(&Bluelink, &Celerity)

	// Should have 4 components: bluelink, celerity, deploy-engine, blueprint-ls
	assert.Len(t, merged, 4)

	names := make(map[string]bool)
	for _, c := range merged {
		names[c.BinaryName] = true
	}
	assert.True(t, names["bluelink"])
	assert.True(t, names["celerity"])
	assert.True(t, names["deploy-engine"])
	assert.True(t, names["blueprint-ls"])
}

func TestMergePlugins_SingleProfile(t *testing.T) {
	plugins := MergePlugins(&Bluelink)
	assert.Equal(t, []string{"newstack-cloud/aws"}, plugins)
}

func TestMergePlugins_TwoProfiles_Union(t *testing.T) {
	plugins := MergePlugins(&Bluelink, &Celerity)
	assert.Equal(t, []string{"newstack-cloud/aws", "newstack-cloud/celerity"}, plugins)
}

func TestMergePlugins_NoDuplicates(t *testing.T) {
	plugins := MergePlugins(&Celerity, &Bluelink)
	// newstack-cloud/aws appears in both, should only appear once
	count := 0
	for _, p := range plugins {
		if p == "newstack-cloud/aws" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestDisplayNames_Single(t *testing.T) {
	assert.Equal(t, "Bluelink", DisplayNames([]*Profile{&Bluelink}))
}

func TestDisplayNames_Multiple(t *testing.T) {
	assert.Equal(t, "Bluelink + Celerity", DisplayNames([]*Profile{&Bluelink, &Celerity}))
}

func TestCelerityProfile_UsesCorrectRepo(t *testing.T) {
	cli := Celerity.Components[0]
	assert.Equal(t, "newstack-cloud", cli.RepoOwner)
	assert.Equal(t, "celerity", cli.RepoName)
	assert.Equal(t, "apps/cli", cli.TagPrefix)
	assert.Equal(t, "celerity", cli.BinaryName)
	assert.False(t, cli.Shared)
}

func TestBluelinkProfile_UsesCorrectRepo(t *testing.T) {
	cli := Bluelink.Components[0]
	assert.Equal(t, "newstack-cloud", cli.RepoOwner)
	assert.Equal(t, "bluelink", cli.RepoName)
	assert.Equal(t, "apps/cli", cli.TagPrefix)
	assert.Equal(t, "bluelink", cli.BinaryName)
	assert.False(t, cli.Shared)
}

func TestSharedComponents_AreFlaggedShared(t *testing.T) {
	for _, p := range All() {
		for _, c := range p.Components {
			if c.BinaryName == "deploy-engine" || c.BinaryName == "blueprint-ls" {
				assert.True(t, c.Shared, "%s should be shared in %s profile", c.BinaryName, p.Name)
			}
		}
	}
}
