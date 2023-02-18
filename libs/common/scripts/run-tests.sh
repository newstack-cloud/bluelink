#!/usr/bin/env bash


POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -t|--tags)
    TEST_TYPES="$2"
    shift # past argument
    shift # past value
    ;;
    -h|--help)
    HELP=yes
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
Runs tests for the library.
To run unit tests:
bash ./scripts/run-tests.sh --tags unit
EOF
}

if [ -n "$HELP" ]; then
  help
  exit 0
fi

set -e
echo "" > coverage.txt

go test -timeout 30000ms -tags "$TEST_TYPES" -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic ./...

if [ -z "$GITHUB_ACTION" ]; then
  # We are on a dev machine so produce html output of coverage
  # to get a visual to better reveal uncovered lines.
  go tool cover -html=coverage.txt
fi

if [ -n "$GITHUB_ACTION" ]; then
  # We are in a CI environment so run tests again to generate JSON report.
  go test -timeout 30000ms -json -tags "$TEST_TYPES" ./... > report.json
fi
