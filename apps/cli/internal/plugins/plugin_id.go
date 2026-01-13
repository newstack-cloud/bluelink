package plugins

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins/version"
)

// DefaultRegistryHost is the default registry for plugins without an explicit host.
const DefaultRegistryHost = "registry.bluelink.dev"

// PluginID represents a parsed plugin identifier with registry, namespace, name, and version.
type PluginID struct {
	RegistryHost string // e.g., "registry.bluelink.dev" or "registry.example.com"
	Namespace    string // e.g., "bluelink", "my-org"
	Name         string // e.g., "aws", "gcp"
	Version      string // e.g., "1.0.0" or "" for latest
}

// ParsePluginID parses a plugin identifier string into a PluginID struct.
//
// Supported formats:
//   - "namespace/name" -> uses DefaultRegistryHost
//   - "namespace/name@version" -> uses DefaultRegistryHost with version
//   - "host/namespace/name" -> custom registry host
//   - "host/namespace/name@version" -> custom registry host with version
func ParsePluginID(input string) (*PluginID, error) {
	if input == "" {
		return nil, fmt.Errorf("plugin ID cannot be empty")
	}

	// Split version if present
	var version string
	if idx := strings.LastIndex(input, "@"); idx != -1 {
		version = input[idx+1:]
		input = input[:idx]
		if version == "" {
			return nil, fmt.Errorf("version cannot be empty when @ is specified")
		}
	}

	parts := strings.Split(input, "/")

	switch len(parts) {
	case 2:
		// Format: namespace/name (uses default registry)
		namespace, name := parts[0], parts[1]
		if err := validateNamespaceName(namespace, name); err != nil {
			return nil, err
		}
		return &PluginID{
			RegistryHost: DefaultRegistryHost,
			Namespace:    namespace,
			Name:         name,
			Version:      version,
		}, nil

	case 3:
		// Format: host/namespace/name (custom registry)
		host, namespace, name := parts[0], parts[1], parts[2]
		if host == "" {
			return nil, fmt.Errorf("registry host cannot be empty")
		}
		if err := validateNamespaceName(namespace, name); err != nil {
			return nil, err
		}
		return &PluginID{
			RegistryHost: host,
			Namespace:    namespace,
			Name:         name,
			Version:      version,
		}, nil

	default:
		return nil, fmt.Errorf(
			"invalid plugin ID format: expected 'namespace/name' or 'host/namespace/name', got %q",
			input,
		)
	}
}

func validateNamespaceName(namespace, name string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	return nil
}

// String returns the short form of the plugin ID.
// For default registry: "namespace/name" or "namespace/name@version"
// For custom registry: "host/namespace/name" or "host/namespace/name@version"
func (p *PluginID) String() string {
	var base string
	if p.RegistryHost == DefaultRegistryHost {
		base = fmt.Sprintf("%s/%s", p.Namespace, p.Name)
	} else {
		base = fmt.Sprintf("%s/%s/%s", p.RegistryHost, p.Namespace, p.Name)
	}

	if p.Version != "" {
		return base + "@" + p.Version
	}
	return base
}

// FullyQualified returns the fully qualified plugin ID including the registry host.
// Format: "host/namespace/name" or "host/namespace/name@version"
func (p *PluginID) FullyQualified() string {
	base := fmt.Sprintf("%s/%s/%s", p.RegistryHost, p.Namespace, p.Name)
	if p.Version != "" {
		return base + "@" + p.Version
	}
	return base
}

// ManifestKey returns the key used to identify this plugin in the manifest file.
// This excludes the version since the manifest tracks installed versions separately.
// Format: "host/namespace/name"
func (p *PluginID) ManifestKey() string {
	return fmt.Sprintf("%s/%s/%s", p.RegistryHost, p.Namespace, p.Name)
}

// WithVersion returns a copy of the PluginID with the specified version.
func (p *PluginID) WithVersion(version string) *PluginID {
	return &PluginID{
		RegistryHost: p.RegistryHost,
		Namespace:    p.Namespace,
		Name:         p.Name,
		Version:      version,
	}
}

// IsDefaultRegistry returns true if the plugin uses the default Bluelink registry.
func (p *PluginID) IsDefaultRegistry() bool {
	return p.RegistryHost == DefaultRegistryHost
}

// IsVersionConstraint returns true if the version contains constraint prefixes (^ or ~).
func (p *PluginID) IsVersionConstraint() bool {
	if p.Version == "" {
		return false
	}
	return strings.HasPrefix(p.Version, "^") || strings.HasPrefix(p.Version, "~")
}

// ParseVersionConstraint parses the version as a constraint.
// Returns an exact constraint if no prefix is present.
// Returns an error if no version is specified.
func (p *PluginID) ParseVersionConstraint() (*version.Constraint, error) {
	if p.Version == "" {
		return nil, fmt.Errorf("no version specified")
	}
	return version.ParseConstraint(p.Version)
}
