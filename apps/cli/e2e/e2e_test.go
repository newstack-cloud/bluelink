//go:build e2e

package e2e

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

var binaryPath string

// coverDir holds the absolute path to the coverage directory.
// This needs to be stored because testscript runs in its own working directory.
var coverDir string

func TestMain(m *testing.M) {
	// Build the CLI binary with coverage instrumentation
	tmpDir, err := os.MkdirTemp("", "bluelink-e2e-*")
	if err != nil {
		panic(err)
	}

	binaryPath = filepath.Join(tmpDir, "bluelink")

	// Store absolute path to coverage directory if set
	if dir := os.Getenv("GOCOVERDIR"); dir != "" {
		absDir, err := filepath.Abs(dir)
		if err == nil {
			coverDir = absDir
		}
	}

	// Build with coverage instrumentation (Go 1.20+)
	// Use atomic mode to match unit test coverage for merging
	// Limit coverage to CLI packages only (not dependencies)
	cmd := exec.Command("go", "build", "-cover", "-covermode=atomic", "-coverpkg=github.com/newstack-cloud/bluelink/apps/cli/...", "-o", binaryPath, "../cmd")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	exitCode := m.Run()

	// Clean up
	os.RemoveAll(tmpDir)

	os.Exit(exitCode)
}

// TestScriptsInit runs init command test scripts.
// These tests don't require the deploy engine.
func TestScriptsInit(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/init",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))

			// Use the absolute path stored during TestMain
			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
	})
}

// TestScriptsValidate runs validate command test scripts.
// These tests require the deploy engine to be running.
func TestScriptsValidate(t *testing.T) {
	engineEndpoint := os.Getenv("DEPLOY_ENGINE_ENDPOINT")
	if engineEndpoint == "" {
		// Use port 18325 to avoid conflicts with locally running deploy-engine on 8325
		engineEndpoint = "http://localhost:18325"
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/validate",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))
			env.Setenv("BLUELINK_ENGINE_ENDPOINT", engineEndpoint)
			env.Setenv("BLUELINK_CONNECT_PROTOCOL", "tcp")

			// Use the absolute path stored during TestMain
			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"wait_engine": waitForEngine,
		},
	})
}

// TestScriptsStage runs stage command test scripts.
// These tests require the deploy engine to be running.
func TestScriptsStage(t *testing.T) {
	engineEndpoint := os.Getenv("DEPLOY_ENGINE_ENDPOINT")
	if engineEndpoint == "" {
		// Use port 18325 to avoid conflicts with locally running deploy-engine on 8325
		engineEndpoint = "http://localhost:18325"
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/stage",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))
			env.Setenv("BLUELINK_ENGINE_ENDPOINT", engineEndpoint)
			env.Setenv("BLUELINK_CONNECT_PROTOCOL", "tcp")

			// Use the absolute path stored during TestMain
			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"wait_engine": waitForEngine,
		},
	})
}

// TestScriptsDeploy runs deploy command test scripts.
// These tests require the deploy engine to be running.
func TestScriptsDeploy(t *testing.T) {
	engineEndpoint := os.Getenv("DEPLOY_ENGINE_ENDPOINT")
	if engineEndpoint == "" {
		engineEndpoint = "http://localhost:18325"
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/deploy",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))
			env.Setenv("BLUELINK_ENGINE_ENDPOINT", engineEndpoint)
			env.Setenv("BLUELINK_CONNECT_PROTOCOL", "tcp")

			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"wait_engine": waitForEngine,
		},
	})
}

// TestScriptsDestroy runs destroy command test scripts.
// These tests require the deploy engine to be running.
func TestScriptsDestroy(t *testing.T) {
	engineEndpoint := os.Getenv("DEPLOY_ENGINE_ENDPOINT")
	if engineEndpoint == "" {
		engineEndpoint = "http://localhost:18325"
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/destroy",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))
			env.Setenv("BLUELINK_ENGINE_ENDPOINT", engineEndpoint)
			env.Setenv("BLUELINK_CONNECT_PROTOCOL", "tcp")

			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"wait_engine": waitForEngine,
		},
	})
}

// TestScriptsInstances runs instances inspect and list command test scripts.
// These tests require the deploy engine to be running.
func TestScriptsInstances(t *testing.T) {
	engineEndpoint := os.Getenv("DEPLOY_ENGINE_ENDPOINT")
	if engineEndpoint == "" {
		engineEndpoint = "http://localhost:18325"
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/instances",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))
			env.Setenv("BLUELINK_ENGINE_ENDPOINT", engineEndpoint)
			env.Setenv("BLUELINK_CONNECT_PROTOCOL", "tcp")

			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"wait_engine": waitForEngine,
		},
	})
}

// TestScriptsState runs state command test scripts.
// These tests don't require the deploy engine as they work directly with state backends.
func TestScriptsState(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/state",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))

			// Use the absolute path stored during TestMain
			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
	})
}

// TestScriptsCleanup runs cleanup command test scripts.
// These tests require the deploy engine to be running.
func TestScriptsCleanup(t *testing.T) {
	engineEndpoint := os.Getenv("DEPLOY_ENGINE_ENDPOINT")
	if engineEndpoint == "" {
		engineEndpoint = "http://localhost:18325"
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts/cleanup",
		Setup: func(env *testscript.Env) error {
			env.Setenv("PATH", filepath.Dir(binaryPath)+":"+env.Getenv("PATH"))
			env.Setenv("BLUELINK_ENGINE_ENDPOINT", engineEndpoint)
			env.Setenv("BLUELINK_CONNECT_PROTOCOL", "tcp")

			if coverDir != "" {
				env.Setenv("GOCOVERDIR", coverDir)
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"wait_engine": waitForEngine,
		},
	})
}

// waitForEngine waits for the deploy-engine to be ready by polling its health endpoint.
// Usage in .txtar: wait_engine [timeout_seconds]
// Default timeout is 30 seconds.
func waitForEngine(ts *testscript.TestScript, neg bool, args []string) {
	timeout := 30 * time.Second
	if len(args) > 0 {
		if d, err := time.ParseDuration(args[0] + "s"); err == nil {
			timeout = d
		}
	}

	endpoint := ts.Getenv("BLUELINK_ENGINE_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:18325"
	}
	healthURL := endpoint + "/v1/health"

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if neg {
					ts.Fatalf("expected engine to be unavailable, but it responded")
				}
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !neg {
		ts.Fatalf("deploy-engine did not become ready at %s within %v", healthURL, timeout)
	}
}
