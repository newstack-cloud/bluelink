package pluginbase

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/pluginservicev1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/pluginutils"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServerOption is a function that configures a server.
type ServerOption[ServerType any] func(*Server[ServerType])

// WithDebug is a server option that enables debug mode.
func WithDebug[ServerType any]() ServerOption[ServerType] {
	return func(s *Server[ServerType]) {
		s.debug = true
	}
}

// WithUnixSocket is a server option that sets the Unix socket path.
func WithUnixSocket[ServerType any](path string) ServerOption[ServerType] {
	return func(s *Server[ServerType]) {
		s.unixSocket = path
	}
}

// WithTCPPort is a server option that sets the TCP port.
func WithTCPPort[ServerType any](port int) ServerOption[ServerType] {
	return func(s *Server[ServerType]) {
		s.tcpPort = port
	}
}

// WithListener is a server option that sets the listener
// that the server should use.
func WithListener[ServerType any](listener net.Listener) ServerOption[ServerType] {
	return func(s *Server[ServerType]) {
		s.listener = listener
	}
}

// Server is a plugin server.
type Server[ServerType any] struct {
	corePluginConfig   *CorePluginConfig[ServerType]
	pluginMetadata     *pluginservicev1.PluginMetadata
	registerPluginFunc func(s grpc.ServiceRegistrar, srv ServerType)
	debug              bool
	unixSocket         string
	tcpPort            int
	pluginService      pluginservicev1.ServiceClient
	hostInfoContainer  pluginutils.HostInfoContainer
	listener           net.Listener
}

// CorePluginConfig is a struct that contains the
// core configuration for a plugin server.
type CorePluginConfig[ServerType any] struct {
	PluginID        string
	PluginType      pluginservicev1.PluginType
	ProtocolVersion string
	PluginServer    ServerType
}

// NewServer creates a new plugin server that is used
// as the base for all plugin type servers.
func NewServer[ServerType any](
	corePluginConfig *CorePluginConfig[ServerType],
	registerPluginFunc func(s grpc.ServiceRegistrar, srv ServerType),
	pluginMetadata *pluginservicev1.PluginMetadata,
	pluginServiceClient pluginservicev1.ServiceClient,
	hostInfoContainer pluginutils.HostInfoContainer,
	opts ...ServerOption[ServerType],
) *Server[ServerType] {
	server := &Server[ServerType]{
		corePluginConfig:   corePluginConfig,
		pluginMetadata:     pluginMetadata,
		registerPluginFunc: registerPluginFunc,
		pluginService:      pluginServiceClient,
		hostInfoContainer:  hostInfoContainer,
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

func (s *Server[ServerType]) Serve() (func(), error) {
	listener, err := s.createListener()
	if err != nil {
		return nil, err
	}

	// If the TCP port is not set and a unix socket is not provided,
	// get the dynamically assigned port if a custom listener
	// has not been provided.
	if s.tcpPort == 0 && s.unixSocket == "" && s.listener == nil {
		s.tcpPort = listener.Addr().(*net.TCPAddr).Port
	}

	opts := []grpc.ServerOption{
		grpc.Creds(insecure.NewCredentials()),
	}

	grpcServer := grpc.NewServer(opts...)
	s.registerPluginFunc(grpcServer, s.corePluginConfig.PluginServer)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf(
				"failed to serve %s plugin server: %s",
				getPluginTypeLabel(s.corePluginConfig.PluginType),
				err,
			)
		}
	}()

	closer := s.createCloser(grpcServer)

	resp, err := s.pluginService.Register(
		context.TODO(),
		&pluginservicev1.PluginRegistrationRequest{
			PluginId:   s.corePluginConfig.PluginID,
			PluginType: s.corePluginConfig.PluginType,
			// Process IDs are sufficient for plugin instance IDs,
			// in the future we may want to allow for plugins that run in
			// containers.
			InstanceId:       strconv.Itoa(os.Getpid()),
			ProtocolVersions: []string{s.corePluginConfig.ProtocolVersion},
			Port:             int32(s.tcpPort),
			Metadata:         s.pluginMetadata,
			UnixSocket:       s.unixSocket,
		},
	)
	if err != nil {
		return closer, err
	}

	if !resp.Success {
		return closer, fmt.Errorf("failed to register plugin with host service: %s", resp.Message)
	}

	s.hostInfoContainer.SetID(resp.HostId)

	return closer, nil
}

func (s *Server[ServerType]) createCloser(baseServer *grpc.Server) func() {
	return func() {
		if s.listener != nil {
			err := s.listener.Close()
			if err != nil {
				log.Printf(
					"failed to close listener for %s plugin server: %s",
					getPluginTypeLabel(s.corePluginConfig.PluginType),
					err,
				)
			}
			s.deregisterPlugin()
			baseServer.Stop()
		}
	}
}

func (s *Server[ServerType]) deregisterPlugin() {
	resp, err := s.pluginService.Deregister(
		context.TODO(),
		&pluginservicev1.PluginDeregistrationRequest{
			PluginType: s.corePluginConfig.PluginType,
			InstanceId: strconv.Itoa(os.Getpid()),
			HostId:     s.hostInfoContainer.GetID(),
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to deregister plugin with host service: %s\n", err)
	} else if !resp.Success {
		fmt.Fprintf(os.Stderr, "failed to deregister plugin with host service: %s\n", resp.Message)
	}
}

func (s *Server[ServerType]) createListener() (net.Listener, error) {
	return utils.CreateListener(&utils.ListenerConfig{
		Listener:   s.listener,
		UnixSocket: s.unixSocket,
		TCPPort:    s.tcpPort,
	})
}

func getPluginTypeLabel(pluginType pluginservicev1.PluginType) string {
	switch pluginType {
	case pluginservicev1.PluginType_PLUGIN_TYPE_PROVIDER:
		return "provider"
	case pluginservicev1.PluginType_PLUGIN_TYPE_TRANSFORMER:
		return "transformer"
	default:
		return "unknown"
	}
}
