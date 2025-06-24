package types

import (
	"io"
	"net"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
)

// SetupFunc is a function that is used to set up
// a specific version of the API.
type SetupFunc func(router *mux.Router, config *core.Config, pluginServiceListener net.Listener) (io.WriteCloser, func(), error)
