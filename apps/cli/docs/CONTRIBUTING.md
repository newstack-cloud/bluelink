# Contributing to the bluelink cli

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >= 1.26.5, which the repo's `go.work` requires

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
go run ./cmd/main.go <command> <options> 2>&1 | tee debug.log
```

For example:

```bash
go run ./cmd/main.go validate --blueprint-file app.blueprint.yml 2>&1 | tee debug.log
```

This will write the output to the console and also to the `debug.log` file.

### Testing plugin commands

You'll find a detailed guide for testing plugin commands against a local registry server in [TESTING_PLUGIN_REGISTRY.md](TESTING_PLUGIN_REGISTRY.md).

## Adding a command

Commands that the Celerity CLI also needs (stage, deploy, destroy, instances, state, cleanup)
belong in deploy-cli-sdk, not here. For a Bluelink-only command, `cmd/commands/init.go` is the
complete example: its own TUI, headless validation and headless output.

1. Add `cmd/commands/<name>.go` with a `setup<Name>Command(rootCmd, confProvider)` function
   (`version` needs only `rootCmd`) and call it from `NewRootCmd` in `cmd/commands/root.go`.
   Subcommands get their own `setup<Parent><Sub>Command(parentCmd, ...)`, as in `plugins.go`.
2. For each flag, bind it to a config key and an environment variable, both prefixed with the
   command path:

   ```go
   nameCmd.PersistentFlags().String("blueprint-file", project.DetectBlueprintFile("."), "...")
   confProvider.BindPFlag("<name>BlueprintFile", nameCmd.PersistentFlags().Lookup("blueprint-file"))
   confProvider.BindEnvVar("<name>BlueprintFile", "BLUELINK_CLI_<NAME>_BLUEPRINT_FILE")
   ```

3. In `RunE`, call `headless.Validate` first for any flags required without a terminal; it does
   nothing in a terminal. Read flags through `confProvider`, set up `utils.SetupLogger()` if the
   command logs, and set `cmd.SilenceUsage = true` once errors are no longer about usage.
4. Call `tea.LogToFile("bluelink-output.log", "simple")` before starting the program. Detect a
   terminal with `term.IsTerminal(int(os.Stdout.Fd()))`: in one, run with `tea.WithAltScreen()`
   and `tea.WithMouseCellMotion()`; without one, with `tea.WithInput(nil)` and
   `tea.WithoutRenderer()`, passing headless mode and `os.Stdout` to the model so it writes
   plain output there (`headless.Printer` helps, as in `pluginlistui`).
5. Put the TUI in `internal/tui/<command><subcommand>ui/`, reusing deploy-cli-sdk's `styles`,
   `ui` and `headless` packages.
6. Test flags, help and headless validation through `NewRootCmd()` and `SetArgs` in a
   `testify/suite` suite that changes into a temporary directory, as
   `cmd/commands/init_test.go` does. Test the TUI beside it with `teatest`, as
   `internal/tui/initui/tui_test.go` does.

Document the command in bluelink-docs (`content/docs/cli/reference.mdx`).

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

**cli**

Example commit:

```bash
git commit -m 'fix(cli): correct cyclic dependency bug'
```
