#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -h|--help)
    HELP=yes
    shift # past argument
    ;;
    --update-snapshots)
    UPDATE_SNAPSHOTS=yes
    shift # past argument
    ;;
    --e2e)
    E2E=yes
    shift # past argument
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

function help {
  cat << EOF
Test runner
Runs tests for the CLI:

Usage:
  bash scripts/run-tests.sh [options]

Options:
  -h, --help           Show this help message
  --update-snapshots   Update test snapshots
  --e2e                Include E2E tests (requires Docker)
                       This will start the deploy-engine in Docker,
                       run E2E tests with coverage, and merge coverage
                       with unit and integration test coverage.

Note: This script requires Docker to be running. It starts cloud storage
emulators (LocalStack, fake-gcs-server, Azurite) for stateio integration tests.

Examples:
  # Run unit and integration tests only
  bash scripts/run-tests.sh

  # Run all tests including E2E (requires Docker)
  bash scripts/run-tests.sh --e2e
EOF
}

if [ -n "$HELP" ]; then
  help
  exit 0
fi

set -e

cd "$CLI_DIR"

# Create coverage directory
mkdir -p coverage

# Copy postgres migrations from blueprint-state library
echo "Copying postgres migrations from blueprint-state library..."
mkdir -p "$CLI_DIR/internal/stateio/postgres/migrations"
cp -a "$CLI_DIR/../../libs/blueprint-state/postgres/migrations/"* "$CLI_DIR/internal/stateio/postgres/migrations/"

# Start stateio integration test dependencies (cloud storage emulators + postgres)
echo "Starting test dependencies (cloud storage emulators + postgres)..."
docker compose --env-file "$CLI_DIR/.env.test" -f "$CLI_DIR/internal/stateio/docker-compose.test-deps.yml" --project-directory "$CLI_DIR/internal/stateio" up -d

cleanup() {
  echo "Stopping test dependencies..."
  docker compose --env-file "$CLI_DIR/.env.test" -f "$CLI_DIR/internal/stateio/docker-compose.test-deps.yml" --project-directory "$CLI_DIR/internal/stateio" down
}
trap cleanup EXIT

get_docker_container_status() {
  docker inspect -f "{{ .State.Status }} {{ .State.ExitCode }}" $1
}

# Wait for postgres migrations to complete
echo "Waiting for postgres migrations to complete..."
status="$(get_docker_container_status stateio_test_postgres_migrate)"
while [ "$status" != "exited 0" ]; do
  if [ "$status" == "exited 1" ]; then
    echo "Postgres migration failed, see logs below:"
    docker logs stateio_test_postgres_migrate
    exit 1
  fi
  sleep 1
  status="$(get_docker_container_status stateio_test_postgres_migrate)"
done

echo "Waiting for LocalStack to be ready..."
start=$EPOCHSECONDS
completed="false"
while [ "$completed" != "true" ]; do
  sleep 5
  completed=$(curl -s localhost:4580/_localstack/init/ready | jq .completed 2>/dev/null || echo "false")
  if (( EPOCHSECONDS - start > 60 )); then
    echo "LocalStack readiness timed out"
    exit 1
  fi
done

echo "Creating S3 test bucket and uploading test files..."
aws --endpoint-url=http://localhost:4580 s3 mb s3://test-bucket --region eu-west-2 2>/dev/null || true
aws --endpoint-url=http://localhost:4580 s3api put-object --bucket test-bucket \
  --body "$CLI_DIR/internal/stateio/__testdata/s3/instances.json" --key instances.json --region eu-west-2

echo "Waiting for Azurite to be ready..."
sleep 3

# Export environment variables for integration tests
echo "Exporting environment variables for test suite..."
set -a
source "$CLI_DIR/.env.test"
set +a

# Make sure the Google Cloud SDK uses the fake GCS server emulator
export STORAGE_EMULATOR_HOST="http://localhost:8185"

# Run unit and integration tests
echo "Running unit and integration tests..."
echo "" > coverage/unit.txt
go test -timeout 30000ms -race -coverprofile=coverage/unit.txt -coverpkg=./... -covermode=atomic $(go list ./... | grep -v '/e2e$' | grep -v '/testutils$')

if [ -n "$E2E" ]; then
  echo ""
  echo "Running E2E tests..."

  # Start deploy-engine with test provider
  # Run from e2e directory so relative volume paths in docker-compose.test.yaml resolve correctly
  echo "Starting deploy-engine with test provider..."
  docker compose -f "$CLI_DIR/e2e/docker-compose.test.yaml" --project-directory "$CLI_DIR/e2e" up -d --wait

  # Update cleanup to also stop deploy-engine
  cleanup() {
    echo "Stopping deploy-engine..."
    docker compose -f "$CLI_DIR/e2e/docker-compose.test.yaml" --project-directory "$CLI_DIR/e2e" down
    echo "Stopping test dependencies..."
    docker compose --env-file "$CLI_DIR/.env.test" -f "$CLI_DIR/internal/stateio/docker-compose.test-deps.yml" --project-directory "$CLI_DIR/internal/stateio" down
  }
  trap cleanup EXIT

  # Create E2E coverage directory
  E2E_COVER_DIR="$CLI_DIR/coverage/e2e"
  mkdir -p "$E2E_COVER_DIR"

  # Run E2E tests with coverage
  GOCOVERDIR="$E2E_COVER_DIR" go test -tags=e2e -timeout 120000ms -v ./e2e/...

  # Convert E2E coverage data to text format
  echo "Processing E2E coverage..."
  # Check if any coverage files were generated before processing
  if ls "$E2E_COVER_DIR"/cov* 1> /dev/null 2>&1; then
    go tool covdata textfmt -i="$E2E_COVER_DIR" -o=coverage/e2e.txt
  else
    echo "Note: No E2E coverage data found, creating empty coverage file"
    echo "mode: atomic" > coverage/e2e.txt
  fi

  # Merge coverage files
  echo "Merging coverage..."
  # Merge by taking the maximum count for each coverage block
  # Coverage format: file:startLine.startCol,endLine.endCol numStatements count
  {
    echo "mode: atomic"
    # Combine both files, skip mode lines, sort by file:line, and take max count for duplicates
    cat coverage/unit.txt coverage/e2e.txt | \
      grep -v "^mode:" | \
      grep -v "^$" | \
      sort | \
      awk '{
        # Format: file:line.col,line.col numStatements count
        key = $1 " " $2
        count = $3
        if (key in counts) {
          if (count + 0 > counts[key] + 0) counts[key] = count
        } else {
          counts[key] = count
          order[++n] = key
        }
      }
      END {
        for (i = 1; i <= n; i++) {
          print order[i] " " counts[order[i]]
        }
      }'
  } > coverage/total.txt

  COVERAGE_FILE="coverage/total.txt"
else
  COVERAGE_FILE="coverage/unit.txt"
fi

# Copy to legacy location for backwards compatibility
cp "$COVERAGE_FILE" coverage.txt

if [ -z "$GITHUB_ACTION" ]; then
  # We are on a dev machine so produce html output of coverage
  # to get a visual to better reveal uncovered lines.
  go tool cover -html="$COVERAGE_FILE" -o coverage.html
  echo ""
  echo "Coverage report: coverage.html"
fi

if [ -n "$GITHUB_ACTION" ]; then
  # We are in a CI environment so run tests again to generate JSON report.
  TEST_TAGS=""
  if [ -n "$E2E" ]; then
    TEST_TAGS="e2e"
  fi
  go test -timeout 30000ms -json -tags "$TEST_TAGS" $(go list ./... | grep -v '/testutils$') > report.json
fi

echo ""
echo "Tests complete!"
