# Contributing to the blueprint resolvers library

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.22
- [Docker](https://docs.docker.com/get-docker/) >=25.0.3
- [jq](https://stedolan.github.io/jq/download/) >=1.7 - used in test runner script to parse JSON
- [AWS CLI](https://aws.amazon.com/cli/) >=2.7.21 - used in test runner script to interact with S3

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

This will spin up a docker compose stack with cloud object storage emulators, once they are up and running the script will then run the tests.

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this library (e.g., `feat(blueprint-resolvers): ...` or `fix(blueprint-resolvers): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `blueprint-resolvers/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `libs/blueprint-resolvers/v{version}` - The canonical Go module tag. Use this for dependencies and references.

### Go module indexing

When a library release tag is pushed, the `index-go-library.yml` workflow automatically indexes the new version with the Go module proxy (pkg.go.dev).

### Tag format

Tags follow Go module conventions: `libs/blueprint-resolvers/vX.Y.Z`

Example: `libs/blueprint-resolvers/v0.1.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(blueprint-resolvers): correct file system resolver to handle nested directories'
```
