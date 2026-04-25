#!/usr/bin/env bash

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
Runs tests for the library:
bash scripts/run-tests.sh
EOF
}

if [ -n "$HELP" ]; then
  help
  exit 0
fi

finish() {
  echo "Taking down test dependencies docker compose stack ..."
  docker compose --env-file .env.test -f docker-compose.test-deps.yml down
}

trap finish EXIT

setup_deps() {
  # Guarantee a clean slate for every invocation — a previous run killed
  # mid-flight or direct `go test` sessions can leave containers (and
  # their DB state) up, which breaks the "each script run is isolated"
  # contract the suite relies on.
  echo "Tearing down any existing test dependencies docker compose stack ..."
  docker compose --env-file .env.test -f docker-compose.test-deps.yml down -v --remove-orphans

  echo "Bringing up docker compose stack for test dependencies ..."

  docker compose --env-file .env.test -f docker-compose.test-deps.yml up -d

  echo "Waiting a few seconds to allow db migrations to complete ..."
  sleep 5

  echo "Populating test databases with seed data ..."
  ./scripts/populate-seed-data.sh
}

teardown_deps() {
  echo "Taking down test dependencies docker compose stack ..."
  docker compose --env-file .env.test -f docker-compose.test-deps.yml down
  docker compose --env-file .env.test -f docker-compose.test-deps.yml rm -v -f
}

setup_deps

echo "Exporting environment variables for test suite ..."
set -a
source .env.test
set +a

set -e
echo "" > coverage.txt

if [ -n "$UPDATE_SNAPSHOTS" ]; then

  UPDATE_SNAPSHOTS=true go test -count=1 -timeout 60000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic `go list ./... | egrep -v '(/(testutils))$'`
else

  go test -count=1 -timeout 60000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic `go list ./... | egrep -v '(/(testutils))$'`

fi

if [ -z "$GITHUB_ACTION" ]; then
  # We are on a dev machine so produce html output of coverage
  # to get a visual to better reveal uncovered lines.
  go tool cover -html=coverage.txt -o coverage.html
fi

if [ -n "$GITHUB_ACTION" ]; then
  echo ""
  echo "Re-running tests to generate JSON report ..."
  echo ""
  teardown_deps
  setup_deps
  # We are in a CI environment so run tests again to generate JSON report.
  go test -count=1 -timeout 60000ms -json `go list ./... | egrep -v '(/(testutils))$'` > report.json
fi
