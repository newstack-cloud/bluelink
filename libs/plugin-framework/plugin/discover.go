package plugin

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/common/core"
	"github.com/spf13/afero"
)

const (
	// The name of the executable file for a plugin.
	pluginFileName = "plugin"
	// The maximum depth to search for plugins in.
	// Plugin paths can of the following forms:
	// - {pluginRootDir}/{plural(pluginType)}/{namespace}/{pluginName}/{version}/plugin
	// - {pluginRootDir}/{plural(pluginType)}/{hostname}/{namespace}/{pluginName}/{version}/plugin
	// "/" is a placeholder for the path separator in the host OS.
	maxPluginDirDepth = 5
)

var (
	// The expected depths for executable plugin files in the plugin directories.
	pluginFileExpectedDepths = []int{maxPluginDirDepth - 1, maxPluginDirDepth}
)

// PluginPathInfo contains important metadata extracted from a plugin path.
type PluginPathInfo struct {
	// The absolute path to the plugin executable.
	AbsolutePath string
	// The plugin type extracted from the path.
	PluginType string
	// The ID of the plugin extracted from the path.
	// This is essential to track the plugins that have
	// registered with the host service.
	ID string
	// The version of the plugin extracted from the path.
	Version string
}

// DiscoverPlugins handles the discovery of plugins
// in the current host environment.
// The provided plugin path is expected to be a colon-separated
// list of root directories to search for plugins in.
// This returns a list of discovered plugin paths with important
// plugin metadata extracted from the file paths.
func DiscoverPlugins(pluginPath string, fs afero.Fs, logger bpcore.Logger) ([]*PluginPathInfo, error) {
	pluginRootDirs := strings.Split(pluginPath, ":")
	discoveredPlugins := []*PluginPathInfo{}

	for _, pluginRootDir := range pluginRootDirs {
		err := discoverPluginsInDir(
			pluginRootDir,
			pluginRootDir,
			fs,
			logger,
			0,
			&discoveredPlugins,
		)
		if err != nil {
			return nil, err
		}
	}

	return discoveredPlugins, nil
}

func discoverPluginsInDir(
	currentDirPath string,
	pluginRootDirPath string,
	fs afero.Fs,
	logger bpcore.Logger,
	depth int,
	collected *[]*PluginPathInfo,
) error {
	if depth > maxPluginDirDepth {
		return nil
	}

	dirContents, err := afero.ReadDir(fs, currentDirPath)
	if err != nil {
		return err
	}

	for _, dirContent := range dirContents {
		if dirContent.IsDir() {
			fullDirPath := filepath.Join(currentDirPath, dirContent.Name())
			logger.Debug(
				fmt.Sprintf(
					"searching for plugins in directory: %s",
					fullDirPath,
				),
			)
			err := discoverPluginsInDir(
				fullDirPath,
				pluginRootDirPath,
				fs,
				logger,
				depth+1,
				collected,
			)
			if err != nil {
				return err
			}
		} else if dirContent.Name() == pluginFileName &&
			slices.Contains(pluginFileExpectedDepths, depth) {
			fullPluginPath := filepath.Join(currentDirPath, pluginFileName)
			relativePluginPath := strings.TrimPrefix(fullPluginPath, pluginRootDirPath)
			pluginPathInfo, isValidPath := extractPluginPathInfo(fullPluginPath, relativePluginPath)
			if isValidPath {
				logger.Debug(
					fmt.Sprintf("found valid plugin at path %s", fullPluginPath),
				)
				*collected = append(*collected, pluginPathInfo)
			}
		}
	}

	return nil
}

func extractPluginPathInfo(fullPluginPath string, relativePluginPath string) (*PluginPathInfo, bool) {
	pluginDir := filepath.Dir(relativePluginPath)
	pluginDirParts := core.Filter(
		strings.Split(pluginDir, string(filepath.Separator)),
		func(part string, _ int) bool {
			return strings.TrimSpace(part) != ""
		},
	)
	if len(pluginDirParts) < maxPluginDirDepth-1 || len(pluginDirParts) > maxPluginDirDepth {
		return nil, false
	}

	pluginTypeDir := pluginDirParts[0]
	pluginType := pluginTypeFromDir(pluginTypeDir)
	pluginID := extractPluginID(pluginDirParts)
	pluginVersion := pluginDirParts[len(pluginDirParts)-1]

	return &PluginPathInfo{
		AbsolutePath: fullPluginPath,
		PluginType:   pluginType,
		ID:           pluginID,
		Version:      pluginVersion,
	}, true
}

func extractPluginID(relativePathDirParts []string) string {
	// The plugin ID will be of the form {hostname}/{namespace}/{pluginName}
	// or {namespace}/{pluginName}.
	return strings.Join(relativePathDirParts[1:len(relativePathDirParts)-1], "/")
}

func pluginTypeFromDir(pluginTypeDir string) string {
	// The plugin type is expected to be the singular form of the directory name.
	// The directory name should be in lowercase.
	return strings.TrimSuffix(pluginTypeDir, "s")
}
