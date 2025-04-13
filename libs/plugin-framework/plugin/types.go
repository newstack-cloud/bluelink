package plugin

import (
	"net"

	"github.com/two-hundred/celerity/libs/plugin-framework/pluginservicev1"
)

// ServePluginConfiguration contains configuration for serving a plugin.
type ServePluginConfiguration struct {
	// The unique identifier for the plugin.
	// In addition to being unique, the ID should point to the location
	// where the plugin can be downloaded.
	// {hostname/}?{namespace}/{pluginName}
	//
	// For example:
	// registry.celerityframework.io/celerity/aws
	// celerity/aws
	//
	// For providers, the last portion of the ID is the unique name of the provider
	// that is expected to be used as the namespace for resources, data sources
	// and custom variable types used in blueprints.
	// For example, the namespace for AWS resources is "aws"
	// used in the resource type "aws/lambda/function".
	// For transformers, the last portion of the ID is the unique name of the transformer,
	// unlike providers, transformer elements are not namespaced so it is purely an ID
	// for the plugin.
	ID string

	// ProtocolVersion is the protocol version that should be
	// used for the plugin.
	// Currently, the only supported protocol version is "1.0".
	ProtocolVersion string

	// PluginMetadata is the metadata for the plugin.
	// This is used to provide information about the plugin
	// to the host service.
	PluginMetadata *pluginservicev1.PluginMetadata

	// Debug runs the plugin in a mode compatible with
	// debugging processes such as delve.
	Debug bool

	// UnixSocketPath is the path to the Unix socket that the
	// plugin should listen on.
	// If this is set, the TCPPort should be empty.
	UnixSocketPath string

	// TCPPort is the port that the plugin should listen on.
	// If this is set, the UnixSocketPath should be empty.
	// If this is not set and UnixSocketPath is not set, the
	// plugin will listen on the next available port.
	TCPPort int

	// Listener is the listener that the plugin server should use.
	// If this is provided, TCPPort and UnixSocketPath will be ignored.
	Listener net.Listener
}
