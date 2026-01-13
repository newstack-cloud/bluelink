package plugins

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PluginIDSuite struct {
	suite.Suite
}

func TestPluginIDSuite(t *testing.T) {
	suite.Run(t, new(PluginIDSuite))
}

func (s *PluginIDSuite) TestParsePluginID_default_registry_without_version() {
	id, err := ParsePluginID("bluelink/aws")
	s.NoError(err)
	s.Equal(DefaultRegistryHost, id.RegistryHost)
	s.Equal("bluelink", id.Namespace)
	s.Equal("aws", id.Name)
	s.Empty(id.Version)
}

func (s *PluginIDSuite) TestParsePluginID_default_registry_with_version() {
	id, err := ParsePluginID("bluelink/aws@1.0.0")
	s.NoError(err)
	s.Equal(DefaultRegistryHost, id.RegistryHost)
	s.Equal("bluelink", id.Namespace)
	s.Equal("aws", id.Name)
	s.Equal("1.0.0", id.Version)
}

func (s *PluginIDSuite) TestParsePluginID_custom_registry_without_version() {
	id, err := ParsePluginID("registry.example.com/my-org/custom-plugin")
	s.NoError(err)
	s.Equal("registry.example.com", id.RegistryHost)
	s.Equal("my-org", id.Namespace)
	s.Equal("custom-plugin", id.Name)
	s.Empty(id.Version)
}

func (s *PluginIDSuite) TestParsePluginID_custom_registry_with_version() {
	id, err := ParsePluginID("registry.example.com/my-org/custom-plugin@2.3.4")
	s.NoError(err)
	s.Equal("registry.example.com", id.RegistryHost)
	s.Equal("my-org", id.Namespace)
	s.Equal("custom-plugin", id.Name)
	s.Equal("2.3.4", id.Version)
}

func (s *PluginIDSuite) TestParsePluginID_localhost_with_port() {
	id, err := ParsePluginID("localhost:8080/bluelink/test-provider@1.0.0")
	s.NoError(err)
	s.Equal("localhost:8080", id.RegistryHost)
	s.Equal("bluelink", id.Namespace)
	s.Equal("test-provider", id.Name)
	s.Equal("1.0.0", id.Version)
}

func (s *PluginIDSuite) TestParsePluginID_semver_with_prerelease() {
	id, err := ParsePluginID("bluelink/aws@1.0.0-beta.1")
	s.NoError(err)
	s.Equal("1.0.0-beta.1", id.Version)
}

func (s *PluginIDSuite) TestParsePluginID_error_empty_input() {
	_, err := ParsePluginID("")
	s.Error(err)
	s.Contains(err.Error(), "cannot be empty")
}

func (s *PluginIDSuite) TestParsePluginID_error_single_part() {
	_, err := ParsePluginID("aws")
	s.Error(err)
	s.Contains(err.Error(), "invalid plugin ID format")
}

func (s *PluginIDSuite) TestParsePluginID_error_too_many_parts() {
	_, err := ParsePluginID("a/b/c/d")
	s.Error(err)
	s.Contains(err.Error(), "invalid plugin ID format")
}

func (s *PluginIDSuite) TestParsePluginID_error_empty_version() {
	_, err := ParsePluginID("bluelink/aws@")
	s.Error(err)
	s.Contains(err.Error(), "version cannot be empty")
}

func (s *PluginIDSuite) TestParsePluginID_error_empty_namespace() {
	_, err := ParsePluginID("/aws")
	s.Error(err)
	s.Contains(err.Error(), "namespace cannot be empty")
}

func (s *PluginIDSuite) TestParsePluginID_error_empty_name() {
	_, err := ParsePluginID("bluelink/")
	s.Error(err)
	s.Contains(err.Error(), "plugin name cannot be empty")
}

func (s *PluginIDSuite) TestParsePluginID_error_empty_host() {
	_, err := ParsePluginID("/bluelink/aws")
	s.Error(err)
	s.Contains(err.Error(), "registry host cannot be empty")
}

func (s *PluginIDSuite) TestString_default_registry_without_version() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}
	s.Equal("bluelink/aws", id.String())
}

func (s *PluginIDSuite) TestString_default_registry_with_version() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.0.0",
	}
	s.Equal("bluelink/aws@1.0.0", id.String())
}

func (s *PluginIDSuite) TestString_custom_registry_without_version() {
	id := &PluginID{
		RegistryHost: "registry.example.com",
		Namespace:    "my-org",
		Name:         "plugin",
	}
	s.Equal("registry.example.com/my-org/plugin", id.String())
}

func (s *PluginIDSuite) TestString_custom_registry_with_version() {
	id := &PluginID{
		RegistryHost: "registry.example.com",
		Namespace:    "my-org",
		Name:         "plugin",
		Version:      "2.0.0",
	}
	s.Equal("registry.example.com/my-org/plugin@2.0.0", id.String())
}

func (s *PluginIDSuite) TestFullyQualified_default_registry() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.0.0",
	}
	s.Equal("registry.bluelink.dev/bluelink/aws@1.0.0", id.FullyQualified())
}

func (s *PluginIDSuite) TestFullyQualified_custom_registry() {
	id := &PluginID{
		RegistryHost: "registry.example.com",
		Namespace:    "my-org",
		Name:         "plugin",
	}
	s.Equal("registry.example.com/my-org/plugin", id.FullyQualified())
}

func (s *PluginIDSuite) TestManifestKey() {
	id := &PluginID{
		RegistryHost: "registry.example.com",
		Namespace:    "my-org",
		Name:         "plugin",
		Version:      "1.0.0",
	}
	// ManifestKey should not include version
	s.Equal("registry.example.com/my-org/plugin", id.ManifestKey())
}

func (s *PluginIDSuite) TestWithVersion() {
	original := &PluginID{
		RegistryHost: "registry.example.com",
		Namespace:    "my-org",
		Name:         "plugin",
	}

	withVersion := original.WithVersion("2.0.0")

	// Original should be unchanged
	s.Empty(original.Version)

	// New instance should have version
	s.Equal("2.0.0", withVersion.Version)
	s.Equal(original.RegistryHost, withVersion.RegistryHost)
	s.Equal(original.Namespace, withVersion.Namespace)
	s.Equal(original.Name, withVersion.Name)
}

func (s *PluginIDSuite) TestIsDefaultRegistry_true() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}
	s.True(id.IsDefaultRegistry())
}

func (s *PluginIDSuite) TestIsDefaultRegistry_false() {
	id := &PluginID{
		RegistryHost: "registry.example.com",
		Namespace:    "my-org",
		Name:         "plugin",
	}
	s.False(id.IsDefaultRegistry())
}

func (s *PluginIDSuite) TestParsePluginID_roundtrip_default_registry() {
	original := "bluelink/aws@1.0.0"
	id, err := ParsePluginID(original)
	s.NoError(err)
	s.Equal(original, id.String())
}

func (s *PluginIDSuite) TestParsePluginID_roundtrip_custom_registry() {
	original := "registry.example.com/my-org/plugin@1.0.0"
	id, err := ParsePluginID(original)
	s.NoError(err)
	s.Equal(original, id.String())
}

func (s *PluginIDSuite) TestIsVersionConstraint_exact_version() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.0.0",
	}
	s.False(id.IsVersionConstraint())
}

func (s *PluginIDSuite) TestIsVersionConstraint_caret() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "^1.0.0",
	}
	s.True(id.IsVersionConstraint())
}

func (s *PluginIDSuite) TestIsVersionConstraint_tilde() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "~1.0.0",
	}
	s.True(id.IsVersionConstraint())
}

func (s *PluginIDSuite) TestIsVersionConstraint_empty() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}
	s.False(id.IsVersionConstraint())
}

func (s *PluginIDSuite) TestParseVersionConstraint_exact() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "1.2.3",
	}
	c, err := id.ParseVersionConstraint()
	s.NoError(err)
	s.True(c.IsExact())
	s.Equal("1.2.3", c.String())
}

func (s *PluginIDSuite) TestParseVersionConstraint_caret() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "^1.2.3",
	}
	c, err := id.ParseVersionConstraint()
	s.NoError(err)
	s.False(c.IsExact())
	s.Equal("^1.2.3", c.String())
}

func (s *PluginIDSuite) TestParseVersionConstraint_tilde() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
		Version:      "~1.2.3",
	}
	c, err := id.ParseVersionConstraint()
	s.NoError(err)
	s.False(c.IsExact())
	s.Equal("~1.2.3", c.String())
}

func (s *PluginIDSuite) TestParseVersionConstraint_empty_version() {
	id := &PluginID{
		RegistryHost: DefaultRegistryHost,
		Namespace:    "bluelink",
		Name:         "aws",
	}
	_, err := id.ParseVersionConstraint()
	s.Error(err)
	s.Contains(err.Error(), "no version specified")
}

func (s *PluginIDSuite) TestParsePluginID_with_caret_constraint() {
	id, err := ParsePluginID("bluelink/aws@^1.0.0")
	s.NoError(err)
	s.Equal("^1.0.0", id.Version)
	s.True(id.IsVersionConstraint())
}

func (s *PluginIDSuite) TestParsePluginID_with_tilde_constraint() {
	id, err := ParsePluginID("bluelink/aws@~2.1.0")
	s.NoError(err)
	s.Equal("~2.1.0", id.Version)
	s.True(id.IsVersionConstraint())
}
