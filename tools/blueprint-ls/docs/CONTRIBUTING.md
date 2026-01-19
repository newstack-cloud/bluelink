# Contributing to the blueprint language server

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.22
- A C compiler (required for tree-sitter parsing via CGO)

#### Installing a C Compiler

The language server uses [tree-sitter](https://tree-sitter.github.io/) for robust YAML and JSON parsing, which requires CGO. You'll need a C compiler installed:

**macOS:**
```bash
xcode-select --install
```

**Linux (Debian/Ubuntu):**
```bash
sudo apt-get update && sudo apt-get install -y gcc
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf install gcc
```

**Windows:**

Install [MinGW-w64](https://www.mingw-w64.org/) or use [MSYS2](https://www.msys2.org/):
```bash
pacman -S mingw-w64-x86_64-gcc
```

### Installing Dependencies

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

CGO must be enabled for tree-sitter support:

```bash
CGO_ENABLED=1 go build -o blueprint-language-server ./cmd/main.go
```

On most systems CGO is enabled by default when building for the native platform.

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this tool (e.g., `feat(blueprint-ls): ...` or `fix(blueprint-ls): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `blueprint-ls/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `tools/blueprint-ls/v{version}` - The canonical tag. Use this for workflows and references.

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
