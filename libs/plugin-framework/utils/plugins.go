package utils

import "strings"

// ExtractPluginNamespace extracts the plugin namespace to be used with
// the blueprint framework from the given plugin ID.
// For example, the plugin namespace for the plugin ID "registry.customhost.com/bluelink/azure"
// would be "azure".
func ExtractPluginNamespace(pluginID string) string {
	parts := strings.Split(pluginID, "/")
	return parts[len(parts)-1]
}
