package pluginhost

import (
	"fmt"
	"net"
)

// The environment variable that plugins read
// (via `pluginservicev1.NewEnvServiceClient`) to find the plugin service to
// register with.
const pluginServicePortEnvVar = "BLUELINK_BUILD_ENGINE_PLUGIN_SERVICE_PORT"

// StartPluginServiceListener creates a loopback listener on an ephemeral port
// for the language server's plugin service.
//
// A dynamic port is used because the deploy engine can bind to the well-known
// default port (43044) on the same machine. The server also communicates over
// stdio, so each client connection gets its own process, and clients that spawn
// a server per window or workspace (VS Code, for one) can run several at once.
// Callers must pass the returned listener to both WithPluginServiceListener and
// PluginExecutorEnvVars so that plugins spawned by this host register with this
// host rather than whichever process owns the default port.
func StartPluginServiceListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// PluginExecutorEnvVars derives the environment variables that plugins spawned
// by this host need in order to locate its plugin service. The result is passed
// to `plugin.NewOSCmdExecutor` so every launched plugin inherits them.
//
// An empty map is returned for non-TCP listeners, in which case plugins fall
// back to the framework default port.
func PluginExecutorEnvVars(listener net.Listener) map[string]string {
	if listener == nil {
		return map[string]string{}
	}

	tcpAddr, isTCPAddr := listener.Addr().(*net.TCPAddr)
	if !isTCPAddr {
		return map[string]string{}
	}

	return map[string]string{
		pluginServicePortEnvVar: fmt.Sprintf("%d", tcpAddr.Port),
	}
}
