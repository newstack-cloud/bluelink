package pluginservicev1

import (
	"log"
	"net"

	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// DefaultPort is the default TCP port for the plugin service
	// gRPC server.
	DefaultPort = 43044
)

// ServerOption is a function that configures a server.
type ServerOption func(*Server)

// WithDebug is a server option that enables debug mode.
func WithDebug() ServerOption {
	return func(s *Server) {
		s.debug = true
	}
}

// WithUnixSocket is a server option that sets the Unix socket path.
func WithUnixSocket(path string) ServerOption {
	return func(s *Server) {
		s.unixSocket = path
	}
}

// WithTCPPort is a server option that sets the TCP port.
func WithTCPPort(port int) ServerOption {
	return func(s *Server) {
		s.tcpPort = port
	}
}

// WithListener is a server option that sets the listener
// that the server should use.
func WithListener(listener net.Listener) ServerOption {
	return func(s *Server) {
		s.listener = listener
	}
}

// Server for the plugin service that manages registration and deregistration of plugins
// and allows plugins to make calls to a subset of functionality provided by other plugins.
type Server struct {
	debug         bool
	unixSocket    string
	tcpPort       int
	pluginService ServiceServer
	listener      net.Listener
}

func NewServer(
	pluginService ServiceServer,
	opts ...ServerOption,
) *Server {
	server := &Server{
		pluginService: pluginService,
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

func (s *Server) Serve() (func(), error) {
	// If the TCP port is not set and a unix socket is not provided,
	// use the default port for the plugin service.
	if s.tcpPort == 0 && s.unixSocket == "" {
		s.tcpPort = DefaultPort
	}

	listener, err := s.createListener()
	if err != nil {
		return nil, err
	}

	opts := []grpc.ServerOption{
		grpc.Creds(insecure.NewCredentials()),
	}

	grpcServer := grpc.NewServer(opts...)
	RegisterServiceServer(grpcServer, s.pluginService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("failed to serve plugin service server: %s", err)
		}
	}()

	closer := s.createCloser(grpcServer)

	return closer, nil
}

func (s *Server) createCloser(baseServer *grpc.Server) func() {
	return func() {
		if s.listener != nil {
			err := s.listener.Close()
			if err != nil {
				log.Printf("failed to close listener for provider plugin server: %s", err)
			}
			baseServer.Stop()
		}
	}
}

func (s *Server) createListener() (net.Listener, error) {
	return utils.CreateListener(&utils.ListenerConfig{
		Listener:   s.listener,
		UnixSocket: s.unixSocket,
		TCPPort:    s.tcpPort,
	})
}
