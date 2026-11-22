# Bluelink CLI

The `bluelink` command. The repo root's `AGENTS.md` covers the checks, gates and safety rules;
this file adds what's specific to the CLI.

## Where commands live

- `cmd/commands/root.go` registers every command. `init`, `plugins`, `templates`, `validate`
  and `version` are this module's own, each set up by a `setup<Name>Command` function in its
  own file, with TUIs per command or subcommand in `internal/tui/` (`initui`, `pluginlistui`,
  `templatelistui`, and `preflightui` for the plugin check). `validate` uses deploy-cli-sdk's
  `validateui`.
- `stage`, `deploy`, `destroy`, `instances`, `state` and `cleanup` come from deploy-cli-sdk
  (`sdkcommands.Setup*Command`), as do their TUIs, `headless` and `jsonout`, so change them in
  deploy-cli-sdk. The Celerity CLI is meant to share them but so far uses only its `config`,
  `styles`, `engine` and `tui/precommand` packages.
- deploy-cli-sdk is a separate repo and isn't in `go.work`: the CLI builds against the version
  pinned in `go.mod`, so read the SDK at that tag (`git show v<version>:<path>`), not its
  checkout. A change there reaches the CLI only after a deploy-cli-sdk release and a bump
  here; see the newstack plugin's `cross-repo` skill.

Adding a command: [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md#adding-a-command).

## Running the CLI

`go run ./cmd/main.go <command>` from a checkout builds a temporary binary and doesn't replace
an installed one. What it reaches is the concern:

- By default it connects to the local engine over a unix socket (`--connect-protocol`), the
  engine that may be serving runs against real cloud accounts. Run `stage`, `deploy`,
  `destroy`, `cleanup` or `state import` only against the e2e test engine, or when asked.
  `scripts/run-tests.sh --e2e` stops that engine when it exits, so start it on its own:

  ```bash
  docker compose -f e2e/docker-compose.test.yaml up -d --wait
  go run ./cmd/main.go deploy --connect-protocol tcp \
    --engine-endpoint http://localhost:18325 --engine-auth-config-file <file> ...
  docker compose -f e2e/docker-compose.test.yaml down
  ```

  `<file>` holds `{"method": "apiKey", "apiKey": "test-api-key"}`, the key the test engine
  accepts; write it outside this directory rather than editing `engine.auth.json`.
- Before `validate`, `stage`, `deploy` and `destroy`, a preflight check installs missing
  plugins from `registry.bluelink.dev` into the engine's plugin directory
  (`$HOME/.bluelink/engine/plugins`). Pass `--skip-plugin-check`, and don't run
  `plugins install`, `plugins uninstall` or `plugins login` (which writes credentials under
  `$HOME/.bluelink/clients/`) unless asked.

## Conventions

- Flags are kebab-case (`--blueprint-file`). Config keys are camelCase and environment
  variables `BLUELINK_CLI_<...>`, both prefixed with the command path (`pluginsListType`,
  `BLUELINK_CLI_PLUGINS_LIST_TYPE`); global flags have no prefix
  (`BLUELINK_CLI_ENGINE_ENDPOINT`).
- The TUI owns stdout. Logs go to `bluelink.log` through `utils.SetupLogger`, and Bubble Tea's
  own log to `bluelink-output.log` through `tea.LogToFile`; never log to stdout.
- Without a terminal, a command runs headless: no input, no renderer, and the model writes
  plain output to stdout (`pluginlistui` and `templatelistui` use deploy-cli-sdk's
  `headless.Printer`). Only deploy-cli-sdk's commands
  have `--json`.
- Command tests (`cmd/commands/*_test.go`) cover flags, help and headless validation through
  `NewRootCmd()`; TUI behaviour is tested beside each TUI with `teatest`.
- The CLI's user docs are in bluelink-docs (`content/docs/cli/reference.mdx` and the pages
  beside it). A change to a command, flag or output says what those pages need.

## Local files

A maintainer's local test setup lives in this directory: `bluelink.config.toml`,
`bluelink.deploy.jsonc`, `*.blueprint.yaml`, `engine.auth.json`, and the logs and coverage
reports the CLI and scripts write. All are gitignored; don't delete or edit them. Other names
the CLI reads by default (`bluelink.deploy.json`, and `project.blueprint` with `.yml`, `.json` or
`.jsonc`, or `project.bp`) aren't ignored, so don't commit one.

## Known mistakes

<!-- One line per mistake an agent has made here, phrased as the rule that prevents it. -->
