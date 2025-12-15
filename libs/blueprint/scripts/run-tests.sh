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
Runs tests for the core blueprint framework:
bash scripts/run-tests.sh

Run tests and re-generate snapshots (For spec/schema tests):
bash scripts/run-tests.sh --update-snapshots
EOF
}

if [[ -n "$HELP" ]]; then
  help
  exit 0
fi

set -e
echo "" > coverage.txt

if [[ -n "$UPDATE_SNAPSHOTS" ]]; then
  # Exclude generated protobuf code from coverage.
  UPDATE_SNAPSHOTS=true go test -timeout 120000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic `go list ./... | egrep -v '(/(schemapb|testutils))$'`

else
  # Exclude generated protobuf code from coverage.
  go test -timeout 120000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic `go list ./... | egrep -v '(/(schemapb|testutils))$'`
fi

if [[ -z "$GITHUB_ACTION" ]]; then
  # We are on a dev machine so produce html output of coverage
  # to get a visual to better reveal uncovered lines.
  go tool cover -html=coverage.txt -o coverage.html
fi

if [[ -n "$GITHUB_ACTION" ]]; then
  # We are in a CI environment so run tests again to generate JSON report.
  go test -timeout 120000ms -json -tags "$TEST_TYPES" `go list ./... | egrep -v '(/(schemapb|testutils))$'` > report.json
fi
