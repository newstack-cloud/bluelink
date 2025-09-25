# Contributing to the bluelink deploy engine

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.22
- [Air](https://github.com/air-verse/air) >=1.63.0 - For hot reloading when running locally on the host machine

Dependencies are managed with Go modules (go.mod) and will be installed automatically when you first
run tests.

If you want to install dependencies manually you can run:

```bash
go mod download
```

## Running tests

```bash
bash ./scripts/run-tests.sh
```

To update the snapshot output you use the `--update-snapshots` flag as follows:

```bash
bash ./scripts/run-tests.sh --update-snapshots
```

## Running the deploy engine locally

To run the deploy engine locally for development purposes, you can bring up the local docker compose stack including the deploy engine and various dependencies.
It is best to use the `run-local.sh` script to prepare the environment and run the docker compose command.

### Dockerised application

1. Copy .env.example to .env and adjust the values as needed.

2. Run the script to prepare the environment and bring up the local docker compose stack.

```bash
# Run locally as a dockerised application
bash ./scripts/run-local.sh
```

### Application on the host machine

1. Copy .env.example.host to .env.host and adjust the values as needed.
2. Run the script to prepare the environment and bring up the local application and dependency stack.

```bash
# You'll need to either set the storage engine to `memfile` as a postgres
# database is not brought up when running locally as an application on the host machine.
bash ./scripts/run-local.sh --host
```

## Releasing

To release a new version of the deploy engine, you need to create a new tag and push it to the repository.

The format must be `apps/deploy-engine/vX.Y.Z` where `X.Y.Z` is the semantic version number.
The reason for this is that Go's mechanism for picking up modules from multi-repo packages is based on the sub-directory path being in the version tag.
Even though the deploy engine is an executable, we use the same mechanism as libraries in this repo to be consistent.

See [here](https://go.dev/wiki/Modules#publishing-a-release).

1. add a change log entry to the `CHANGELOG.md` file following the template below:

```markdown
## [0.2.0] - 2024-06-05

### Fixed:

- Corrects error reporting for change staging.

### Added

- Adds retry behaviour to resource providers.
```

2. Create and push the new tag prefixed by sub-directory path:

```bash
git tag -a apps/deploy-engine/v0.2.0 -m "chore(deploy-engine): Release v0.2.0"
git push --tags
```

Be sure to add a release for the tag with notes following this template:

Title: `Deploy Engine - v0.2.0`

```markdown
## Fixed:

- Corrects claims handling for JWT middleware.

## Added

- Adds dihandlers-compatible middleware for access control.
```

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(deploy-engine): correct cyclic dependency bug'
```
