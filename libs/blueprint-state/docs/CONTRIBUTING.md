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

To release a new version of the library, you need to create a new tag and push it to the repository.

The format must be `libs/blueprint-state/vX.Y.Z` where `X.Y.Z` is the semantic version number.
The reason for this is that Go's mechanism for picking up modules from multi-repo packages is based on the sub-directory path being in the version tag.

See [here](https://go.dev/wiki/Modules#publishing-a-release).

1. add a change log entry to the `CHANGELOG.md` file following the template below:

```markdown
## [0.2.0] - 2024-06-05

### Fixed:

- Corrects postgres state container schema.

### Added

- Improves performance for querying resources in postgres-backed state container.
```

2. Create and push the new tag prefixed by sub-directory path:

```bash
git tag -a libs/blueprint-state/v0.2.0 -m "chore(blueprint-state): Release v0.2.0"
git push --tags
```

Be sure to add a release for the tag with notes following this template:

Title: `Blueprint State - v0.2.0`

```markdown
## Fixed:

- Corrects postgres state container schema.

## Added

- Improves performance for querying resources in postgres-backed state container.
```

3. Prompt Go to update its index of modules with the new release:

```bash
GOPROXY=proxy.golang.org go list -m github.com/newstack-cloud/bluelink/libs/blueprint-state@v0.2.0
```

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(blueprint-state): correct schema for postgres state container'
```
