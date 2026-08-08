package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/enginev1"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/types"
)

func main() {
	config, err := core.LoadConfig()
	if err != nil {
		log.Fatalf("error loading config: %s", err)
	}

	apiVersion := config.APIVersion
	useUnixSocket := config.UseUnixSocket
	unixSocketPath := config.UnixSocketPath
	setup, setupExists := apiVersions[apiVersion]
	if !setupExists {
		log.Fatalf("version \"%s\" does not exist", apiVersion)
	}

	r := mux.NewRouter().PathPrefix(fmt.Sprintf("/%s", apiVersion)).Subrouter()

	_, cleanup, err := setup(
		r,
		&config,
		// Use the default TCP port for the plugin service.
		/* pluginServiceListener */
		nil,
	)
	if err != nil {
		log.Fatalf("error setting up Deploy Engine API: %s", err)
	}

	srv := &http.Server{
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           r,
	}

	shutdownTimeout := config.GetShutdownDrainTimeout()
	serverErr := runServer(srv, &config)
	// Shutdown signal has been received, or the server has terminated with an error.
	runShutdown(srv, cleanup, useUnixSocket, unixSocketPath, shutdownTimeout)
	if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		log.Fatalf("server error: %s", serverErr)
	}
}

// Starts the HTTP server in a goroutine so the main goroutine can
// wait on a termination signal. Returns the server's terminal error (usually
// http.ErrServerClosed after a graceful shutdown) once the server has stopped.
func runServer(srv *http.Server, config *core.Config) error {
	serverErrCh := make(chan error, 1)
	go func() {
		if config.UseUnixSocket {
			serverErrCh <- serveUnixSocket(srv, config.UnixSocketPath)
			return
		}
		srv.Addr = determineServerAddr(config.LoopbackOnly, config.Port)
		serverErrCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrCh:
		return err
	case <-sigCh:
		return nil
	}
}

// Drains in-flight HTTP handlers, runs the engine cleanup chain
// (which includes cancelling in-flight deploy/destroy/rollback goroutines and
// waiting for the library to persist their terminal status), then removes the
// unix socket file if one was in use. The drainTimeout bounds the entire
// shutdown sequence so a wedged goroutine cannot block process exit
// indefinitely.
func runShutdown(
	srv *http.Server,
	cleanup func(),
	useUnixSocket bool,
	unixSocketPath string,
	drainTimeout time.Duration,
) {
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("http server shutdown error: %s", err)
	}

	if cleanup != nil {
		cleanup()
	}

	if useUnixSocket {
		if err := os.Remove(unixSocketPath); err != nil && !os.IsNotExist(err) {
			log.Printf("failed to remove unix socket %q: %s", unixSocketPath, err)
		}
	}
}

const unixSocketDialTimeout = 200 * time.Millisecond

func serveUnixSocket(srv *http.Server, unixSocketPath string) error {
	if err := prepareUnixSocketPath(unixSocketPath); err != nil {
		return err
	}

	listener, err := net.Listen("unix", unixSocketPath)
	if err != nil {
		return fmt.Errorf("error creating listener for unix socket: %w", err)
	}
	defer listener.Close()

	// Restrict the socket to the owning user. net.Listen creates it with
	// umask-dependent permissions (often group/world accessible), which would
	// undermine the reason for preferring a socket over a loopback TCP port.
	if err := os.Chmod(unixSocketPath, 0o600); err != nil {
		return fmt.Errorf("error restricting unix socket permissions: %w", err)
	}

	return srv.Serve(listener)
}

// Clears a stale socket file left behind by a previous
// run so net.Listen does not fail with "address already in use". If a live
// engine is still listening on the socket it is left untouched and an error is
// returned rather than hijacking the path.
func prepareUnixSocketPath(unixSocketPath string) error {
	info, err := os.Stat(unixSocketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error inspecting unix socket path: %w", err)
	}

	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf(
			"path %q exists and is not a socket, refusing to remove it",
			unixSocketPath,
		)
	}

	if conn, err := net.DialTimeout("unix", unixSocketPath, unixSocketDialTimeout); err == nil {
		conn.Close()
		return fmt.Errorf(
			"another deploy engine appears to be listening on %q",
			unixSocketPath,
		)
	}

	if err := os.Remove(unixSocketPath); err != nil {
		return fmt.Errorf("error removing stale unix socket: %w", err)
	}

	return nil
}

func determineServerAddr(loopbackOnly bool, port int) string {
	if loopbackOnly {
		return fmt.Sprintf("127.0.0.1:%d", port)
	}

	return fmt.Sprintf(":%d", port)
}

var apiVersions = map[string]types.SetupFunc{
	"v1": enginev1.Setup,
}
