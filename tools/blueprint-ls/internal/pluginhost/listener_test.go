package pluginhost

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ListenerSuite struct {
	suite.Suite
}

func (s *ListenerSuite) TestStartPluginServiceListener_binds_ephemeral_loopback_port() {
	listener, err := StartPluginServiceListener()
	s.Require().NoError(err)
	defer listener.Close()

	tcpAddr, isTCPAddr := listener.Addr().(*net.TCPAddr)
	s.Require().True(isTCPAddr)
	s.Assert().True(tcpAddr.IP.IsLoopback())
	s.Assert().NotZero(tcpAddr.Port)
}

func (s *ListenerSuite) TestStartPluginServiceListener_avoids_collisions_between_hosts() {
	first, err := StartPluginServiceListener()
	s.Require().NoError(err)
	defer first.Close()

	second, err := StartPluginServiceListener()
	s.Require().NoError(err)
	defer second.Close()

	s.Assert().NotEqual(first.Addr().String(), second.Addr().String())
}

func (s *ListenerSuite) TestPluginExecutorEnvVars_carries_the_bound_port() {
	listener, err := StartPluginServiceListener()
	s.Require().NoError(err)
	defer listener.Close()

	envVars := PluginExecutorEnvVars(listener)

	tcpAddr, isTCPAddr := listener.Addr().(*net.TCPAddr)
	s.Require().True(isTCPAddr)
	s.Assert().Equal(
		strconv.Itoa(tcpAddr.Port),
		envVars["BLUELINK_BUILD_ENGINE_PLUGIN_SERVICE_PORT"],
	)
}

func (s *ListenerSuite) TestPluginExecutorEnvVars_handles_nil_listener() {
	s.Assert().Empty(PluginExecutorEnvVars(nil))
}

func TestListenerSuite(t *testing.T) {
	suite.Run(t, new(ListenerSuite))
}
