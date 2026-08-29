# Contributing to the bluelink deploy engine

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.25
- [Air](https://github.com/air-verse/air) >=1.63.0 - For hot reloading when running locally on the host machine

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

To update the snapshot output you use the `--update-snapshots` flag as follows:

```bash
bash ./scripts/run-tests.sh --update-snapshots
```

## Running the deploy engine locally

To run the deploy engine locally for development purposes, you can bring up the local docker compose stack including the deploy engine and various dependencies.
It is best to use the `run-local.sh` script to prepare the environment and run the docker compose command.

### Dockerised application

1. Copy .env.example to .env and adjust the values as needed.

2. Run the script to prepare the environment and bring up the local docker compose stack.

```bash
# Run locally as a dockerised application
bash ./scripts/run-local.sh
```

### Application on the host machine

1. Copy .env.example.host to .env.host and adjust the values as needed.
2. Run the script to prepare the environment and bring up the local application and dependency stack.

```bash
# You'll either need to bring up a postgres database first or
# set the storage engine to `memfile` as a postgres
# database will not be brought up automatically when running locally as an application on the host machine.
bash ./scripts/run-local.sh --host
```

## Building plugins for local testing

The deploy engine discovers plugins under a plugin directory tree rooted at
`.bluelink/deploy-engine/plugins/bin` when running locally, with each plugin
binary installed at:

```
<providers|transformers>/<namespace>/<name>/<version>/plugin
```

`scripts/build-local-plugins.sh` builds plugins from their source repositories
and installs them into that tree. It has no knowledge of where any plugin lives,
the locations are given as specs of the form:

```
<namespace>/<name>=<source directory>[@<version>]
```

where the source directory is the directory holding the plugin's main package
and a leading `~/` is expanded. When no version is given it is derived from the
latest git tag in the source directory's repository, falling back to `0.0.0-dev`
when there are no tags. The version is used both for the install directory and
for the version compiled into the plugin binary.

```bash
bash ./scripts/build-local-plugins.sh \
  --provider newstack-cloud/aws=~/repos/bluelink-provider-aws \
  --transformer newstack-cloud/celerity=~/repos/bluelink-transformer-celerity
```

Both options can be repeated to build any number of plugins, and a version can
be pinned instead of derived from git tags:

```bash
bash ./scripts/build-local-plugins.sh --provider newstack-cloud/aws=~/repos/bluelink-provider-aws@0.4.2
```

### Config file

So the locations on your machine do not have to be retyped, plugin specs can be
kept in a config file. Copy `local-plugins.example.conf` to `local-plugins.conf`
and adjust the directories, `local-plugins.conf` is not tracked in version
control. When no `--provider` or `--transformer` option is given the specs are
read from it:

```bash
bash ./scripts/build-local-plugins.sh
```

Use `--config <file>` or the `BLUELINK_LOCAL_PLUGINS_CONFIG` environment
variable for a config file in another location, `--only <namespace>/<name>` to
build a subset of the plugins in it, and `--list` to see what would be built
without building it.

The deploy engine loads plugins at startup, so restart it after building to pick
up new or rebuilt plugins.

### Test plugins

The repository also carries plugins under `testplugins/` for exercising the
deploy engine locally without talking to any upstream system. These are built
with `scripts/build-test-plugins.sh`, see
[testplugins/README.md](../testplugins/README.md) for what they provide and how
to use them.

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this app (e.g., `feat(deploy-engine): ...` or `fix(deploy-engine): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `deploy-engine/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `apps/deploy-engine/v{version}` - The canonical tag. Use this for workflows and references.

### Build artifacts

When a release tag is pushed, separate workflows will build and publish artifacts (Docker images). These workflows are triggered by tags matching `apps/deploy-engine/v*`.

### Tag format

Tags follow the pattern: `apps/deploy-engine/vX.Y.Z`

Example: `apps/deploy-engine/v1.0.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(deploy-engine): correct cyclic dependency bug'
```
