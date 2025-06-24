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

To release a new version of the plugin framework, you need to create a new tag and push it to the repository.

The format must be `libs/plugin-framework/vX.Y.Z` where `X.Y.Z` is the semantic version number.
The reason for this is that Go's mechanism for picking up modules from multi-repo packages is based on the sub-directory path being in the version tag.

See [here](https://go.dev/wiki/Modules#publishing-a-release).

1. add a change log entry to the `CHANGELOG.md` file following the template below:

```markdown
## [0.2.0] - 2024-06-05

### Fixed:

- Corrects error reporting for transformer plugins.

### Added

- Adds retry behaviour to resource provider wrapper.
```

2. Create and push the new tag prefixed by sub-directory path:

```bash
git tag -a libs/plugin-framework/v0.2.0 -m "chore(plugin-framework): Release v0.2.0"
git push --tags
```

Be sure to add a release for the tag with notes following this template:

Title: `Plugin Framework - v0.2.0`

```markdown
## Fixed:

- Corrects error reporting for transformer plugins.

## Added

- Adds retry behaviour to resource provider wrapper.
```

3. Prompt Go to update its index of modules with the new release:

```bash
GOPROXY=proxy.golang.org go list -m github.com/newstack-cloud/bluelink/libs/plugin-framework@v0.2.0
```

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(plugin-framework): correct cyclic dependency bug'
```
