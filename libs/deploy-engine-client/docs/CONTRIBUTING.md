# Contributing to the deploy engine client

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

To release a new version of the library, you need to create a new tag and push it to the repository.

The format must be `libs/deploy-engine-client/vX.Y.Z` where `X.Y.Z` is the semantic version number.
The reason for this is that Go's mechanism for picking up modules from multi-repo packages is based on the sub-directory path being in the version tag.

See [here](https://go.dev/wiki/Modules#publishing-a-release).

1. add a change log entry to the `CHANGELOG.md` file following the template below:

```markdown
## [0.2.0] - 2024-06-05

### Fixed

- Fixes bugs in the slice search utility function.

### Added

- Adds new utility functions for handling maps.
```

2. Create and push the new tag prefixed by sub-directory path:

```bash
git tag -a libs/deploy-engine-client/v0.2.0 -m "chore(deploy-engine-client): Release v0.2.0"
git push --tags
```

Be sure to add a release for the tag with notes following this template:

Title: `Deploy Engine Client - v0.2.0`

```markdown
### Fixed

- Fixes bugs in SSE streaming for deployment events.

### Added

- Adds new utility types for handling deployment events.
```

3. Prompt Go to update its index of modules with the new release:

```bash
GOPROXY=proxy.golang.org go list -m github.com/newstack-cloud/bluelink/libs/deploy-engine-client@v0.2.0
```

## Commit scope

**common**

Example commit:

```bash
git commit -m 'fix(deploy-engine-client): fix bugs in sse streaming for deployment events'
```
