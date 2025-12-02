# Contributing to the plugin docgen tool

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

## Building the plugin docgen binary

```bash
go generate ./...
go build -o plugin-docgen ./cmd/main.go
```

For releases, you will need to make sure that the generated code includes the new version number in the generated versions.go file.
You can do this by running the following command:

```bash
export PLUGIN_DOCGEN_APPLICATION_VERSION="v0.2.0"
go generate ./...
go build -o plugin-docgen ./cmd/main.go
```

Replace `v0.2.0` with the version you are releasing, keeping the `v` prefix.

The `versions.go` file is committed to version control so that the `go install` command can be used to install the binary directly from source.
The trunk should contain the `versions.go` file generated for the latest release.

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this tool (e.g., `feat(plugin-docgen): ...` or `fix(plugin-docgen): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `plugin-docgen/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `tools/plugin-docgen/v{version}` - The canonical tag. Use this for workflows and references.

### Build artifacts

When a release tag is pushed, separate workflows will build and publish artifacts (binaries). These workflows are triggered by tags matching `tools/plugin-docgen/v*`.

### Tag format

Tags follow the pattern: `tools/plugin-docgen/vX.Y.Z`

Example: `tools/plugin-docgen/v1.0.0`

## Commit scope

**plugin-docgen**

Example commit:

```bash
git commit -m 'fix(plugin-docgen): add correction to config field generation'
```
