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

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this library (e.g., `feat(deploy-engine-client): ...` or `fix(deploy-engine-client): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Git tag is created in format `libs/deploy-engine-client/v{version}` (e.g., `libs/deploy-engine-client/v0.1.0`)

### Go module indexing

When a library release tag is pushed, the `index-go-library.yml` workflow automatically indexes the new version with the Go module proxy (pkg.go.dev).

### Tag format

Tags follow Go module conventions: `libs/deploy-engine-client/vX.Y.Z`

Example: `libs/deploy-engine-client/v0.1.0`

## Commit scope

**common**

Example commit:

```bash
git commit -m 'fix(deploy-engine-client): fix bugs in sse streaming for deployment events'
```
