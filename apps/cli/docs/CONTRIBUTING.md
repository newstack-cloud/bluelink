# Contributing to the bluelink cli

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.22

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

## Releasing

TODO: Outline a more involved release process to ship binaries!

To release a new version of the library, you need to create a new tag and push it to the repository.

The format must be `apps/cli/vX.Y.Z` where `X.Y.Z` is the semantic version number.
The reason for this is that Go's mechanism for picking up modules from multi-repo packages is based on the sub-directory path being in the version tag.

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
git tag -a apps/blueprint/v0.2.0 -m "chore(cli): Release v0.2.0"
git push --tags
```

Be sure to add a release for the tag with notes following this template:

Title: `Blueprint Framework - v0.2.0`

```markdown
## Fixed:

- Corrects claims handling for JWT middleware.

## Added

- Adds dihandlers-compatible middleware for access control.
```

3. Prompt Go to update its index of modules with the new release:

```bash
GOPROXY=proxy.golang.org go list -m github.com/newstack-cloud/bluelink/apps/cli@v0.2.0
```

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(cli): correct cyclic dependency bug'
```
