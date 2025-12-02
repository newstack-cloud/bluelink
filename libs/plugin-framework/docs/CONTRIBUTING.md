# Contributing to the bluelink plugin framework

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

## Generating gRPC protobuf code

The plugin framework gRPC for the plugin system that includes providers, transformers and the service hub/manager that plugins register with and use as a gateway to call functions provided by other plugins.

1. Follow the instructions [here](https://grpc.io/docs/protoc-installation/#install-using-a-package-manager) to install the `protoc` compiler.

2. Install the Go protoc plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

3. Run the following command from the `libs/plugin-framework` directory to generate gRPC protobuf code for shared protobuf messages used by the plugin system:

```bash
protoc --proto_path=.. --go_out=.. --go_opt=paths=source_relative \
  --go-grpc_out=.. --go-grpc_opt=paths=source_relative \
  plugin-framework/sharedtypesv1/types.proto
```

4. Run the following command from the `libs/plugin-framework` directory to generate the gRPC protobuf code for the plugin service that plugins register with that also allows them to call functions:

```bash
protoc --proto_path=.. --go_out=.. --go_opt=paths=source_relative \
  --go-grpc_out=.. --go-grpc_opt=paths=source_relative \
  plugin-framework/pluginservicev1/service.proto
```

5. Run the following command from the `libs/plugin-framework` directory to generate the gRPC protobuf code for provider plugins:

```bash
protoc --proto_path=.. --go_out=.. --go_opt=paths=source_relative \
  --go-grpc_out=.. --go-grpc_opt=paths=source_relative \
  plugin-framework/providerserverv1/provider.proto
```

6. Run the following command from the `libs/plugin-framework` directory to generate the gRPC protobuf code for transform plugins:

```bash
protoc --proto_path=.. --go_out=.. --go_opt=paths=source_relative \
  --go-grpc_out=.. --go-grpc_opt=paths=source_relative \
  plugin-framework/transformerserverv1/transformer.proto
```

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this library (e.g., `feat(plugin-framework): ...` or `fix(plugin-framework): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `plugin-framework/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `libs/plugin-framework/v{version}` - The canonical Go module tag. Use this for dependencies and references.

### Go module indexing

When a library release tag is pushed, the `index-go-library.yml` workflow automatically indexes the new version with the Go module proxy (pkg.go.dev).

### Tag format

Tags follow Go module conventions: `libs/plugin-framework/vX.Y.Z`

Example: `libs/plugin-framework/v0.1.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(plugin-framework): correct cyclic dependency bug'
```
