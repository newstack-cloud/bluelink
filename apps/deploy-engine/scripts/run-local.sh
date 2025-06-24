#!/bin/bash

# This script pulls whatever db migrations are in the blueprint state library
# for the current branch or tag that is checked out for the bluelink monorepo,
# this may not be the same as a released version of the library that is reported
# to have issues.
# For debugging purposes, it will often require manually copying the migration
# files from a specific version of the state library.
echo "Copying postgres migrations from the blueprint state library ..."

mkdir -p ./postgres/migrations
cp -r ../../libs/blueprint-state/postgres/migrations/ ./postgres/migrations/

# Generate dynamic code such as the version constants so there are no missing
# files when building the app.
go generate ./...

docker compose -f docker-compose.local.yml up --build --force-recreate
