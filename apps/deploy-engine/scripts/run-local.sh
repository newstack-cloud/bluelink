#!/bin/bash

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -h|--help)
    HELP=yes
    shift # past argument
    ;;
    --host)
    HOST=yes
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
Local deploy engine runner

To run locally as a dockerised application:
bash scripts/run-local.sh

To run locally as an application on the host machine:
bash scripts/run-local.sh --host

Dependencies will run in docker compose stacks for both modes.
EOF
}

if [ -n "$HELP" ]; then
  help
  exit 0
fi

# Generate dynamic code such as the version constants so there are no missing
# files when building the app.
go generate ./...

if [ -n "$HOST" ]; then
  echo "Exporting environment variables for local application on the host machine ..."
  set -a
  source .env.host
  set +a

  mkdir -p ./.bluelink/deploy-engine/plugins/bin
  mkdir -p ./.bluelink/deploy-engine/plugins/logs

  echo "Bringing up docker compose dependency stack for local application on the host machine ..."
  docker compose --env-file .env -f docker-compose.local.host.yml up --build --force-recreate -d

  air
else
  # This script pulls whatever db migrations are in the blueprint state library
  # for the current branch or tag that is checked out for the bluelink monorepo,
  # this may not be the same as a released version of the library that is reported
  # to have issues.
  # For debugging purposes, it will often require manually copying the migration
  # files from a specific version of the state library.
  echo "Copying postgres migrations from the blueprint state library ..."

  mkdir -p ./postgres/migrations
  cp -r ../../libs/blueprint-state/postgres/migrations/ ./postgres/migrations/

  docker compose --env-file .env -f docker-compose.local.yml up --build --force-recreate
fi
