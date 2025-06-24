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

To release a new version of the library, you need to create a new tag and push it to the repository.

The format must be `libs/blueprint-resolvers/vX.Y.Z` where `X.Y.Z` is the semantic version number.
The reason for this is that Go's mechanism for picking up modules from multi-repo packages is based on the sub-directory path being in the version tag.

See [here](https://go.dev/wiki/Modules#publishing-a-release).

1. add a change log entry to the `CHANGELOG.md` file following the template below:

```markdown
## [0.2.0] - 2024-06-05

### Fixed:

- Corrects file system resolver to handle nested directories.

### Added

- Adds new resolver to source child blueprints from S3.
```

2. Create and push the new tag prefixed by sub-directory path:

```bash
git tag -a libs/blueprint-resolvers/v0.2.0 -m "chore(blueprint-resolvers): Release v0.2.0"
git push --tags
```

Be sure to add a release for the tag with notes following this template:

Title: `Blueprint Resolvers - v0.2.0`

```markdown
## Fixed:

- Corrects file system resolver to handle nested directories.

## Added

- Adds new resolver to source child blueprints from S3.
```

3. Prompt Go to update its index of modules with the new release:

```bash
GOPROXY=proxy.golang.org go list -m github.com/newstack-cloud/bluelink/libs/blueprint-resolvers@v0.2.0
```

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(blueprint-resolvers): correct file system resolver to handle nested directories'
```
