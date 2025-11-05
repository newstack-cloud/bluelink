package main

import (
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
	port := config.Port
	useUnixSocket := config.UseUnixSocket
	unixSocketPath := config.UnixSocketPath
	loopbackOnly := config.LoopbackOnly
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

	cleanupOnShutdown(cleanup, useUnixSocket, unixSocketPath)

	srv := &http.Server{
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           r,
	}

	if !useUnixSocket {
		serverAddr := determineServerAddr(loopbackOnly, port)
		srv.Addr = serverAddr
		log.Fatal(srv.ListenAndServe())
	} else {
		startUnixSocketServer(srv, unixSocketPath)
	}
}

func cleanupOnShutdown(
	cleanup func(),
	useUnixSocket bool,
	unixSocketPath string,
) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if useUnixSocket {
			os.Remove(unixSocketPath)
		}
		if cleanup != nil {
			cleanup()
		}
		os.Exit(1)
	}()
}

func startUnixSocketServer(srv *http.Server, unixSocketPath string) {
	listener, err := net.Listen("unix", unixSocketPath)
	if err != nil {
		log.Fatalf("error creating listener for unix socket: %s", err)
	}

	defer listener.Close()
	log.Fatal(srv.Serve(listener))
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
