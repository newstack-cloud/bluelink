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

PACKAGES=$(go list ./... | egrep -v '(/(schemapb|testutils))$')
TEST_FLAGS="-timeout 120000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic"

if [[ -n "$UPDATE_SNAPSHOTS" ]]; then
  export UPDATE_SNAPSHOTS=true
fi

if [[ -n "$GITHUB_ACTION" ]]; then
  # In CI, include -json output in a single test run
  # to avoid running the full suite twice.
  go test ${TEST_FLAGS} -json ${PACKAGES} 2>&1 | tee report.json
  test ${PIPESTATUS[0]} -eq 0
else
  go test ${TEST_FLAGS} ${PACKAGES}
  # On a dev machine, produce html output of coverage
  # to get a visual to better reveal uncovered lines.
  go tool cover -html=coverage.txt -o coverage.html
fi
