# Contributing to the blueprint state library

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.22
- [Docker](https://docs.docker.com/get-docker/) >=25.0.3
- [jq](https://stedolan.github.io/jq/download/) >=1.7 - used in test harness to populate seed data from JSON files
- [migrate cli](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) >=4.18.2 - used to run migrations for the postgres state container
- [psql](https://www.postgresql.org/download/) >=17.0 - PostgresSQL CLI is used to load seed data into the test postgres database

Dependencies are managed with Go modules (go.mod) and will be installed automatically when you first run tests.

If you want to install dependencies manually you can run:

```bash
go mod download
```

## Running tests

### Running full test suite

To run all tests in an isolated environment that is torn down after the tests are complete, run:

```bash
bash ./scripts/run-tests.sh
```

Updating test snapshots:

```bash
bash ./scripts/run-tests.sh --update-snapshots
```

### Running tests for debugging

To bring up dependencies and run tests in a local environment, run:

```bash
docker compose --env-file .env.test -f docker-compose.test-deps.yml up
```

Optionally, depending on the tests you are debugging, you can seed the test database with data by running:

```bash
bash scripts/populate-seed-data.sh
```

Then in another terminal session run the tests:

```bash
source .env.test
go test -timeout 30000ms -race ./...
```

or run individual tests through your editor/IDE or by specifying individual tests/test suites:

```bash
source .env.test
go test -timeout 30000ms -race ./... -run TestPostgresStateContainerInstancesTestSuite
```

**You will need to make sure that you clean up the test dependency volumes after running tests locally to avoid state inconsistencies:**

```bash
docker compose --env-file .env.test -f docker-compose.test-deps.yml rm -v -f
```

## Migrations

[Postgres migrations](./POSTGRES_MIGRATIONS.md) are used to manage the schema for the Postgres state container.

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this library (e.g., `feat(blueprint-state): ...` or `fix(blueprint-state): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `blueprint-state/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `libs/blueprint-state/v{version}` - The canonical Go module tag. Use this for dependencies and references.

### Go module indexing

When a library release tag is pushed, the `index-go-library.yml` workflow automatically indexes the new version with the Go module proxy (pkg.go.dev).

### Tag format

Tags follow Go module conventions: `libs/blueprint-state/vX.Y.Z`

Example: `libs/blueprint-state/v0.5.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(blueprint-state): correct schema for postgres state container'
```
