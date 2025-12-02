# Contributing to the blueprint framework

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.23

Dependencies are managed with Go modules (go.mod) and will be installed automatically when you first
run tests.

If you want to install dependencies manually you can run:

```bash
go mod download
```

## Running tests

```bash
bash ./scripts/run-tests.sh

# to re-generate snapshots (For spec/schema tests)
bash scripts/run-tests.sh --update-snapshots
```

## Generating protobuf code

The blueprint framework uses protobuf to store and transmit an expanded version of a blueprint. Expanded blueprints include AST-like expansions of substitutions that can be cached with an implementation of the `cache.BlueprintCache` interface.

1. Follow the instructions [here](https://grpc.io/docs/protoc-installation/#install-using-a-package-manager) to install the `protoc` compiler.

2. Install the Go protoc plugin:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

3. Run the following command from the `libs/blueprint` directory to generate the protobuf code:

```bash
protoc --go_out=./schemapb --go_opt=paths=source_relative ./schema.proto
```

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this library (e.g., `feat(blueprint): ...` or `fix(blueprint): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Git tag is created in format `libs/blueprint/v{version}` (e.g., `libs/blueprint/v0.37.0`)

### Go module indexing

When a library release tag is pushed, the `index-go-library.yml` workflow automatically indexes the new version with the Go module proxy (pkg.go.dev).

### Tag format

Tags follow Go module conventions: `libs/blueprint/vX.Y.Z`

Example: `libs/blueprint/v0.37.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(blueprint): correct cyclic dependency bug'
```
