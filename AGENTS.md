# Bluelink

Bluelink defines and deploys infrastructure from blueprints, with permissions and network access
following from the links between resources. This Go monorepo holds the blueprint framework, the
plugin framework, state persistence, the deploy engine and the CLI. deploy-cli-sdk, the Celerity
CLI, the providers and the transformers all build on its libraries.

## Layout

Each module is released on its own by release-please; `go.work` ties them together.
[docs/INDEX.md](docs/INDEX.md) describes each, and
[ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md) how they fit together.

| Path | Holds |
|---|---|
| `apps/cli` | The Bluelink CLI |
| `apps/deploy-engine` | The deploy engine |
| `libs/blueprint` | Blueprint framework: spec, validation, changes, deployment, drift |
| `libs/blueprint-state` | State stores (postgres, objectstore, memfile) and migrations |
| `libs/blueprint-resolvers` | Child blueprint resolvers |
| `libs/plugin-framework` | gRPC plugin protocol, plugin host and Go plugin SDK |
| `libs/deploy-engine-client` | Go client for the engine API |
| `libs/common` | Shared packages, including crypto and the telemetry seam |
| `libs/telemetry-otel` | OpenTelemetry for the telemetry seam |
| `tools/blueprint-ls`, `tools/plugin-docgen`, `tools/bluelink-manager` | Language server, registry docs generator, installer |

## Checks

Run from the module's directory, for each module the change touches, including the modules
that use a changed `libs/*` API: CI only runs a module's checks when its own directory changes.

- **Tools:** `staticcheck` on `PATH` for lint, and docker for the suites below, which also need
  `psql` and `jq` (`blueprint-state`), `awslocal`, `jq` and `curl` (`blueprint-resolvers`) and
  `curl` (`deploy-engine`). A missing `psql` or `jq` fails `blueprint-state`'s seeding silently,
  and its tests then run without seed data.
- **Lint:** `scripts/lint.sh` runs `staticcheck` and `go vet` per package, after `go generate`
  in `deploy-engine` and `plugin-docgen`.
- **Tests:** `scripts/run-tests.sh`.
  - Starts a docker compose stack in `blueprint-state` (postgres with migrations, LocalStack,
    fake-gcs-server, Azurite), `blueprint-resolvers` (LocalStack, fake-gcs-server, Azurite)
    and `deploy-engine` (postgres with migrations, LocalStack). The other modules start
    nothing. No suite uses real cloud accounts.
  - Run one stack at a time. Container names and compose project names are fixed, so the same
    suite can't run in two worktrees at once, and `blueprint-state`'s teardown removes another
    worktree's stack. `blueprint-state` and `blueprint-resolvers` both bind Azurite's port, and
    `blueprint-state` binds the same postgres port as the Docker-mode engine.
  - `--update-snapshots` works only in `blueprint`, `blueprint-state` and `deploy-engine`;
    elsewhere set `UPDATE_SNAPSHOTS=true` instead. Scripts' `--help` text isn't reliable.
  - The CLI's `--e2e` runs the CLI against the published nightly engine image
    (`ghcr.io/newstack-cloud/deploy-engine:nightly`, port 18325), not the working tree.
- **Generated code:** `apps/deploy-engine/core/versions.go` is gitignored, so `go build`,
  `go vet` or `go test` run directly in the engine fails until `go generate ./...` has run.
  `tools/plugin-docgen/internal/env/versions.go` is committed, and CI fails if it's stale
  after a pin bump. `*.pb.go` files are generated from the module's `.proto` files, as
  `libs/blueprint` and `libs/plugin-framework` `docs/CONTRIBUTING.md` set out; don't edit them
  by hand.
- **Gates:**
  - The pre-commit hook, when enabled with `git config core.hooksPath .githooks`, runs
    `lint.sh` for each module with tracked changes (untracked files don't count). `commit-msg`
    runs commitlint through `npm exec`, which needs a root `node_modules`: Yarn 4 creates one
    only with a `.yarnrc.yml` setting `nodeLinker: node-modules`, and that file is gitignored.
    `core.hooksPath` holds an absolute path, so worktrees run the main checkout's hooks.
  - CI runs lint and tests per module, path-filtered. The CLI's e2e suite runs only on a daily
    schedule.
  - `go.work` is committed, so the hook and CI build each module against the others' working
    trees, not their pinned versions. The only module-mode build is the nightly engine image,
    which builds the engine module alone. See the newstack plugin's `cross-repo` skill.

## Running locally

`scripts/run-local.sh`, run from `apps/deploy-engine`, runs the engine on port 8325. It needs
`.env` (and for `--host`, `.env.host`) copied from the `.env.example*` files, and runs
`go generate ./...` first. The engine runs under `air` in both modes, with memfile state by
default:

- **Docker (default):** the container mounts only `apps/deploy-engine`, so it builds against
  the library versions pinned in its `go.mod`, and `libs/` changes never reach it. Starts an
  OIDC server and postgres (unused with memfile). State is lost each time the script runs.
- **`--host`:** builds on the host with `go.work`, so uncommitted changes anywhere in the
  monorepo ship with each rebuild. Starts only the OIDC server.

`air` watches only `apps/deploy-engine`: saving a Go file there rebuilds and restarts a running
engine. To know what a running
engine contains, compare the modification time of `apps/deploy-engine/tmp/main` with the
source files', not with what's committed. Plugin binaries under `apps/deploy-engine/.bluelink/`
match `pgrep` with no engine running; check the port instead.

## Compatibility

- **Deployment boundary:** the engine runs behind a TLS-terminating gateway or load balancer,
  with plugins on the same host only. Authorization, users and SSO belong to the layer built on
  top. The engine owns encryption at rest, redaction, authentication, audit attribution and
  plugin channel access control.
- **Engine API:** additive changes only, so released Bluelink and Celerity CLIs keep working. A
  breaking change needs a coordinated release of `libs/deploy-engine-client`, deploy-cli-sdk
  and both CLIs, and is called out in the PR because older CLIs break.
- **Plugin wire protocol (`.proto`):** additive only; keep field numbers. Plugins built against
  the previous `libs/plugin-framework` release must keep working.
- **Go library interfaces:** `libs/blueprint` and `libs/plugin-framework` interfaces may change,
  weighing the cost to the other modules here, deploy-cli-sdk, the Celerity CLI and the
  plugins, which then rebuild against the new release.
- **Persisted state:** no deployed engines exist yet (as of 2026-10-21). Store layout changes
  need no migration, backfill or fallback for older layouts; say in the PR what changed.
  Postgres migrations must still apply from scratch, and each `down` must reverse its `up`.

## Conventions

- Commits: [COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md), scoped by module
  (`feat(blueprint-state): ...`). Don't edit `CHANGELOG.md` files or versions by hand.
- Migrations live in `libs/blueprint-state/postgres/migrations`; the engine's scripts copy them
  into `apps/deploy-engine/postgres/` (gitignored). Edit the originals.

## Safety

- Never restart the deploy engine or rebuild plugins or the CLI. The engine may be serving
  end-to-end runs against real cloud accounts, and a restart pulls the working tree into them.
  Say when a change needs a restart or rebuild, and wait.
- Don't run anything against real cloud accounts unless asked.
- Don't create tags, releases or release-please changes unless asked.

## Known mistakes

<!-- One line per mistake an agent has made here, phrased as the rule that prevents it. -->
