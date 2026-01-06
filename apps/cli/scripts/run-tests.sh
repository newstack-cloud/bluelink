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

  cleanup() {
    echo "Stopping deploy-engine..."
    docker compose -f "$CLI_DIR/e2e/docker-compose.test.yaml" --project-directory "$CLI_DIR/e2e" down
  }
  trap cleanup EXIT

  # Create E2E coverage directory
  E2E_COVER_DIR="$CLI_DIR/coverage/e2e"
  mkdir -p "$E2E_COVER_DIR"

  # Run E2E tests with coverage
  GOCOVERDIR="$E2E_COVER_DIR" go test -tags=e2e -timeout 120000ms -v ./e2e/...

  # Convert E2E coverage data to text format
  echo "Processing E2E coverage..."
  go tool covdata textfmt -i="$E2E_COVER_DIR" -o=coverage/e2e.txt

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
