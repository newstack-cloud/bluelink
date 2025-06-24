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
Runs tests for the Deploy Engine:
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
  docker compose --env-file .env.test -f docker-compose.test-deps.yml rm -v -f
}

get_docker_container_status() {
  docker inspect -f "{{ .State.Status }} {{ .State.ExitCode }}" $1
}

trap finish EXIT

# This script pulls whatever db migrations are in the blueprint state library
# for the current branch or tag that is checked out for the bluelink monorepo,
# this may not be the same as a released version of the library that is reported
# to have issues.
# For debugging purposes, it will often require manually copying the migration
# files from a specific version of the state library.
echo "Copying postgres migrations from the blueprint state library ..."
echo ""

mkdir -p ./postgres/migrations
cp -a ../../libs/blueprint-state/postgres/migrations/* ./postgres/migrations/

echo "Preparing empty plugin directory (required by integrated tests) ..."

mkdir -p ./internal/integrated_test_suites/__testdata/tmp/plugins/bin

echo "Bringing up docker compose stack for test dependencies ..."

docker compose --env-file .env.test -f docker-compose.test-deps.yml up -d

# Wait for postgres to be ready with the migrations in place.
echo ""
echo "Waiting for postgres to be ready ..."
echo ""

status="$(get_docker_container_status deploy_engine_test_postgres_migrate)"
while [ "$status" != "exited 0" ]; do
  if [ "$status" == "exited 1" ]; then
    echo "Postgres migration failed, see logs below:"
    docker logs deploy_engine_test_postgres_migrate
    exit 1
  fi
  sleep 1
  status="$(get_docker_container_status deploy_engine_test_postgres_migrate)"
done

echo "Exporting environment variables for test suite ..."
set -a
source .env.test
set +a

set -e
echo "" > coverage.txt

# Generate dynamic code such as the version constants so there are no missing
# files when running the tests.
go generate ./...

go test -timeout 30000ms -race -coverprofile=coverage.txt -coverpkg=./... -covermode=atomic `go list ./... | egrep -v '(/(testutils))$'`

if [ -z "$GITHUB_ACTION" ]; then
  # We are on a dev machine so produce html output of coverage
  # to get a visual to better reveal uncovered lines.
  go tool cover -html=coverage.txt -o coverage.html
fi

if [ -n "$GITHUB_ACTION" ]; then
  # We are in a CI environment so run tests again to generate JSON report.
  go test -timeout 30000ms -json -tags "$TEST_TYPES" `go list ./... | egrep -v '(/(testutils))$'` > report.json
fi
