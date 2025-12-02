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
# You'll either need to bring up a postgres database first or
# set the storage engine to `memfile` as a postgres
# database will not be brought up automatically when running locally as an application on the host machine.
bash ./scripts/run-local.sh --host
```

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this app (e.g., `feat(deploy-engine): ...` or `fix(deploy-engine): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `deploy-engine/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `apps/deploy-engine/v{version}` - The canonical tag. Use this for workflows and references.

### Build artifacts

When a release tag is pushed, separate workflows will build and publish artifacts (Docker images). These workflows are triggered by tags matching `apps/deploy-engine/v*`.

### Tag format

Tags follow the pattern: `apps/deploy-engine/vX.Y.Z`

Example: `apps/deploy-engine/v1.0.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(deploy-engine): correct cyclic dependency bug'
```
