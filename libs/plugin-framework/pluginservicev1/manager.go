package pluginservicev1

import (
	"fmt"
	"strings"
	sync "sync"
)

// Manager is an interface that defines the methods for
// registering and deregistering plugins along with
// retrieving plugin instances to interact with.
// The caller is responsible for type assertions to derive
// the actual client interface based on the plugin type.
type Manager interface {
	// RegisterPlugin registers a plugin with the host system
	// that the manager represents.
	RegisterPlugin(info *PluginInstanceInfo) error
	// DeregisterPlugin deregisters a plugin from the host system
	// that the manager represents.
	DeregisterPlugin(pluginType PluginType, id string) error
	// GetPlugin retrieves a plugin instance based on the plugin type
	// and the plugin ID.
	GetPlugin(pluginType PluginType, id string) *PluginInstance
	// GetPluginMetadata retrieves the metadata of a plugin instance
	// based on the plugin type and the plugin ID.
	// This is useful for retrieving core metadata for plugin
	// documentation.
	GetPluginMetadata(pluginType PluginType, id string) *PluginExtendedMetadata
	// GetPlugins retrieves all plugin instances for a given plugin type.
	GetPlugins(pluginType PluginType) []*PluginInstance
}

type managerImpl struct {
	pluginTypeProtocolVersions map[PluginType]string
	pluginInstances            map[PluginType]map[string]*PluginInstance
	pluginMetadata             map[PluginType]map[string]*PluginExtendedMetadata
	pluginFactory              PluginFactory
	hostID                     string
	mu                         sync.RWMutex
}

// PluginExtendedMetadata holds the metadata of a plugin instance
// that a plugin will provide during registration along with
// the protocol versions that the plugin supports.
type PluginExtendedMetadata struct {
	// A semver version for the plugin that can be used
	// for documentation and debugging purposes.
	PluginVersion string
	// A friendly name for the plugin to be displayed
	// in documentation and user interfaces.
	DisplayName string
	// A plain text description of the plugin.
	PlainTextDescription string
	// A formatted description of the plugin
	// that can be formatted with markdown.
	FormattedDescription string
	// The URL of the git repository that the plugin
	// is hosted in.
	RepositoryUrl string
	// The company or individual that authored the plugin.
	Author string
	// The protocol versions that the plugin supports.
	ProtocolVersions []string
}

// NewManager creates a new instance of a plugin manager
// that enables the deploy engine to interact with plugins
// and backs the "pluginservice" gRPC service that plugins
// can register with.
func NewManager(
	protocolVersions map[PluginType]string,
	pluginFactory PluginFactory,
	hostID string,
) Manager {
	return &managerImpl{
		pluginTypeProtocolVersions: protocolVersions,
		pluginInstances:            make(map[PluginType]map[string]*PluginInstance),
		pluginMetadata:             make(map[PluginType]map[string]*PluginExtendedMetadata),
		pluginFactory:              pluginFactory,
		hostID:                     hostID,
	}
}

func (m *managerImpl) RegisterPlugin(info *PluginInstanceInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pluginInstances[info.PluginType] == nil {
		m.pluginInstances[info.PluginType] = make(map[string]*PluginInstance)
	}

	hostProtocolVersion, ok := m.pluginTypeProtocolVersions[info.PluginType]
	if !ok {
		return fmt.Errorf("plugin type %d is not supported", info.PluginType)
	}

	pluginCompatibleWithHost, err := hostSupportsProtocolVersions(
		hostProtocolVersion,
		info.ProtocolVersions,
	)
	if err != nil {
		return fmt.Errorf("failed to check protocol versions: %w", err)
	}

	if !pluginCompatibleWithHost {
		protocolVersionsString := strings.Join(info.ProtocolVersions, ", ")
		return fmt.Errorf(
			"plugin protocol versions %q are not supported, expected <=%s",
			protocolVersionsString,
			hostProtocolVersion,
		)
	}

	_, hasPlugin := m.pluginInstances[info.PluginType][info.ID]
	if hasPlugin {
		return fmt.Errorf("plugin %s is already registered", info.ID)
	}

	client, closeConn, err := m.pluginFactory(info, m.hostID)
	if err != nil {
		return err
	}

	m.pluginInstances[info.PluginType][info.ID] = &PluginInstance{
		Info:      info,
		Client:    client,
		CloseConn: closeConn,
	}

	if m.pluginMetadata[info.PluginType] == nil {
		m.pluginMetadata[info.PluginType] = make(map[string]*PluginExtendedMetadata)
	}
	if info.Metadata != nil {
		m.pluginMetadata[info.PluginType][info.ID] = &PluginExtendedMetadata{
			PluginVersion:        info.Metadata.PluginVersion,
			DisplayName:          info.Metadata.DisplayName,
			PlainTextDescription: info.Metadata.PlainTextDescription,
			FormattedDescription: info.Metadata.FormattedDescription,
			RepositoryUrl:        info.Metadata.RepositoryUrl,
			Author:               info.Metadata.Author,
			ProtocolVersions:     info.ProtocolVersions,
		}
	} else {
		m.pluginMetadata[info.PluginType][info.ID] = &PluginExtendedMetadata{
			ProtocolVersions: info.ProtocolVersions,
		}
	}

	return nil
}

func (m *managerImpl) DeregisterPlugin(pluginType PluginType, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instancesForType, hasPluginType := m.pluginInstances[pluginType]
	if !hasPluginType {
		return fmt.Errorf("plugin type %d is not supported", pluginType)
	}

	metadataForType, hasMetadataType := m.pluginMetadata[pluginType]
	if hasMetadataType {
		delete(metadataForType, id)
	}

	delete(instancesForType, id)
	return nil
}

func (m *managerImpl) GetPlugin(pluginType PluginType, id string) *PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instancesForType, hasPluginType := m.pluginInstances[pluginType]
	if !hasPluginType {
		return nil
	}

	return instancesForType[id]
}

func (m *managerImpl) GetPluginMetadata(pluginType PluginType, id string) *PluginExtendedMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metadataForType, hasMetadataType := m.pluginMetadata[pluginType]
	if !hasMetadataType {
		return nil
	}

	metadata, hasMetadata := metadataForType[id]
	if !hasMetadata {
		return nil
	}

	return metadata
}

func (m *managerImpl) GetPlugins(pluginType PluginType) []*PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instancesForType, hasPluginType := m.pluginInstances[pluginType]
	if !hasPluginType {
		return []*PluginInstance{}
	}

	instances := make([]*PluginInstance, 0, len(instancesForType))
	for _, instance := range instancesForType {
		instances = append(instances, instance)
	}

	return instances
}

type PluginFactory func(*PluginInstanceInfo, string) (any, func(), error)

// PluginInstance represents an instance of a plugin
// that has been registered with the host system.
type PluginInstance struct {
	Info *PluginInstanceInfo
	// type assertions should be carried out by callers at runtime
	// to derive the actual client interface based on the plugin type.
	Client    any
	CloseConn func()
}

// PluginInstanceInfo represents the information about a plugin instance
// that is registered with the host system.
type PluginInstanceInfo struct {
	PluginType PluginType
	// ProtocolVersions contains the protocol versions that
	// the plugin supports.
	// Currently, the only supported protocol version is "1.0".
	ProtocolVersions []string
	// The unique identifier for the provider plugin.
	// In addition to being unique, the ID should point to the location
	// where the provider plugin can be downloaded.
	// {hostname/}?{namespace}/{provider}
	//
	// For example:
	// registry.bluelink.dev/bluelink/aws
	// bluelink/aws
	//
	// The last portion of the ID is the unique name of the provider
	// that is expected to be used as the namespace for resources, data sources
	// and custom variable types used in blueprints.
	// For example, the namespace for AWS resources is "aws"
	// used in the resource type "aws/lambda/function".
	ID string
	// Metadata that can be used for plugin documentation.
	Metadata *PluginMetadata
	// The ID of an instance of a plugin that has been loaded
	// by the host system.
	InstanceID     string
	TCPPort        int
	UnixSocketPath string
}
