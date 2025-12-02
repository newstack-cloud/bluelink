# Contributing to the blueprint language server

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

## Building the language server binary

```bash
go build -o blueprint-language-server ./cmd/main.go
```

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this tool (e.g., `feat(blueprint-ls): ...` or `fix(blueprint-ls): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Git tag is created in format `tools/blueprint-ls/v{version}` (e.g., `tools/blueprint-ls/v1.0.0`)

### Build artifacts

When a release tag is pushed, separate workflows will build and publish artifacts (binaries). These workflows are triggered by tags matching `tools/blueprint-ls/v*`.

### Tag format

Tags follow the pattern: `tools/blueprint-ls/vX.Y.Z`

Example: `tools/blueprint-ls/v1.0.0`

## Commit scope

**blueprint-ls**

Example commit:

```bash
git commit -m 'fix(blueprint-ls): correct syntax highlighting bug'
```
