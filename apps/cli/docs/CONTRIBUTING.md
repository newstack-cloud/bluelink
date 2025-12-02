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

### Prepare engine auth file

For running the CLI locally, you need an an engine auth config file
to authenticate with the deploy engine.

You can copy `engine.auth.example.json` to `engine.auth.json` and update the file with the appropriate credentials, default credentials should work out of the box for local development.

## Running tests


### Unit tests only

```bash
bash ./scripts/run-tests.sh
```

### Unit tests and E2E tests

```bash
bash ./scripts/run-tests.sh --e2e
```

For detailed information about E2E tests, including how to write and run them, see [E2E_TESTS.md](E2E_TESTS.md).

## Testing the CLI locally

### Interactive mode (default in a terminal)

Testing in an interactive mode can be done by running:

```bash
go run ./cmd/main.go <command> <options>
```

For example:

```bash
go run ./cmd/main.go validate --blueprint-file app.blueprint.yml
```

### Non-interactive mode

Testing in a non-interactive mode can be done by running:

```bash
go run ./cmd/main.go <command> <options>  2>&1 | tee debug.log
```

For example:

```bash
go run ./cmd/main.go validate --blueprint-file app.blueprint.yml 2>&1 | tee debug.log
```

This will output the output to the console and also to the `debug.log` file.

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this app (e.g., `feat(cli): ...` or `fix(cli): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `cli/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `apps/cli/v{version}` - The canonical tag. Use this for workflows and references.

### Build artifacts

When a release tag is pushed, separate workflows will build and publish artifacts (binaries). These workflows are triggered by tags matching `apps/cli/v*`.

### Tag format

Tags follow the pattern: `apps/cli/vX.Y.Z`

Example: `apps/cli/v1.0.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(cli): correct cyclic dependency bug'
```
