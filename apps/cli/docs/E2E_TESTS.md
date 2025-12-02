# E2E Tests

E2E tests verify the CLI works correctly as a complete binary, including integration with the deploy engine. They use the [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) framework which provides a shell-like scripting environment for testing command-line tools.

## Prerequisites

- Docker and Docker Compose (for running the deploy engine and MinIO)

## How E2E Tests Work

1. **Binary Build**: `TestMain` in `e2e/e2e_test.go` builds a coverage-instrumented CLI binary
2. **Docker Stack**: The test infrastructure runs deploy-engine and MinIO via `e2e/docker-compose.test.yaml`
3. **Test Scripts**: Individual tests are `.txtar` files in `e2e/testdata/scripts/`
4. **Coverage**: Binary coverage is collected via `GOCOVERDIR` and merged with unit test coverage

## Test Structure

Tests are organized by command in `e2e/testdata/scripts/`:

```
e2e/
├── docker-compose.test.yaml    # Test infrastructure (deploy-engine, MinIO)
├── e2e_test.go                 # Test runner and setup
└── testdata/
    ├── fixtures/               # Test data files (e.g., blueprints for MinIO)
    └── scripts/
        ├── init/               # Tests for `bluelink init`
        └── validate/           # Tests for `bluelink validate`
```

## Writing E2E Tests

E2E tests use the `.txtar` format. Each file contains:
- Comment header explaining the test
- Shell-like commands to execute
- Embedded files needed for the test

### Basic Example

```txtar
# Test description here

exec bluelink version
stdout 'bluelink version'

-- bluelink.config.toml --
# Empty config file
```

### Key Commands

- `exec <cmd>`: Run a command, expect success (exit 0)
- `! exec <cmd>`: Run a command, expect failure (non-zero exit)
- `stdout '<pattern>'`: Assert stdout contains pattern
- `stderr '<pattern>'`: Assert stderr contains pattern
- `wait_engine`: Custom command to wait for deploy-engine to be ready

### Test with Deploy Engine

For tests that need the deploy engine:

```txtar
# Test blueprint validation with deploy engine

wait_engine

! exec bluelink validate --blueprint-file=s3://test-blueprints/valid.blueprint.yaml --connect-protocol=tcp --engine-endpoint=$BLUELINK_ENGINE_ENDPOINT --engine-auth-config-file=engine.auth.json
stdout 'Validation complete'
stdout 'diagnostic'

-- bluelink.config.toml --
# Empty config file

-- engine.auth.json --
{
  "method": "apiKey",
  "apiKey": "test-api-key"
}
```

## Environment Variables

The test runner sets these environment variables for validate tests:
- `BLUELINK_ENGINE_ENDPOINT`: Deploy engine URL (default: `http://localhost:18325`)
- `BLUELINK_CONNECT_PROTOCOL`: Connection protocol (`tcp`)
- `GOCOVERDIR`: Coverage output directory (when running with coverage)

## Adding New Tests

1. **Create test file**: Add a `.txtar` file in the appropriate `e2e/testdata/scripts/<command>/` directory
2. **Include required files**: Embed any config files needed (e.g., `bluelink.config.toml`, `engine.auth.json`)
3. **Use fixtures for remote files**: For S3-based tests, add fixtures to `e2e/testdata/fixtures/` and update `docker-compose.test.yaml` to upload them to MinIO

## Running E2E Tests Manually

```bash
# Start the test infrastructure
docker compose -f e2e/docker-compose.test.yaml up -d --wait

# Run all E2E tests
go test -tags=e2e -v ./e2e/...

# Run a specific test
go test -tags=e2e -v ./e2e/... -run TestScriptsValidate/validate_happy

# Stop the infrastructure
docker compose -f e2e/docker-compose.test.yaml down
```

## Test Fixtures

Test fixtures in `e2e/testdata/fixtures/` are uploaded to MinIO at startup. To add a new fixture:

1. Add the file to `e2e/testdata/fixtures/`
2. Update the `minio-setup` service in `docker-compose.test.yaml` to upload it
