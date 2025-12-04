# Bluelink Manager

A cross-platform CLI tool for installing, updating, and managing Bluelink components.

## Overview

Bluelink Manager handles the installation and lifecycle management of all Bluelink tools:

- **Bluelink CLI** - Command-line interface for infrastructure management
- **Deploy Engine** - Background service for deployments
- **Blueprint Language Server** - IDE support for `.blueprint.(yaml|yml|jsonc)` files

## Installation

### macOS / Linux

```bash
curl -fsSL https://get.bluelink.dev | sh
```

### Windows

Download and run the MSI installer from the [releases page](https://github.com/newstack-cloud/bluelink/releases).

## Usage

### Install Bluelink

Downloads and installs all Bluelink components:

```bash
bluelink-manager install
```

Options:
- `--cli-version` - Specific CLI version (default: latest)
- `--engine-version` - Specific Deploy Engine version (default: latest)
- `--ls-version` - Specific Blueprint LS version (default: latest)
- `--no-modify-path` - Skip PATH modification
- `--no-service` - Skip service installation
- `--no-plugins` - Skip core plugin installation
- `--force` - Force reinstall/regenerate config

### Update Components

Updates all components to their latest versions:

```bash
bluelink-manager update
```

### Check Status

Shows installation status and service state:

```bash
bluelink-manager status
```

### Manage Deploy Engine Service

```bash
bluelink-manager start    # Start the service
bluelink-manager stop     # Stop the service
bluelink-manager restart  # Restart the service
```

### Uninstall

Remove Bluelink binaries (preserves config):

```bash
bluelink-manager uninstall
```

Remove everything including config and data:

```bash
bluelink-manager uninstall --all
```

### Self-Update

Update the manager itself:

```bash
bluelink-manager self-update
```

## Installation Directories

| Platform | Location |
|----------|----------|
| macOS/Linux | `~/.bluelink/` |
| Windows | `%LOCALAPPDATA%\NewStack\Bluelink\` |

Directory structure:
```
.bluelink/
├── bin/           # Binaries (bluelink, deploy-engine, blueprint-ls)
├── config/        # CLI configuration
└── engine/        # Deploy Engine data
    ├── plugins/   # Installed plugins
    └── state/     # Deployment state
```

## Service Management

The Deploy Engine runs as a background service:

| Platform | Service Manager |
|----------|----------------|
| macOS | launchd (`~/Library/LaunchAgents/dev.bluelink.deploy-engine.plist`) |
| Linux | systemd user service (`~/.config/systemd/user/bluelink-deploy-engine.service`) |
| Windows | Windows Service (`BluelinkDeployEngine`) |

## Development

### Building

```bash
cd tools/bluelink-manager
go build -o bluelink-manager ./cmd
```

### Cross-compiling

```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o bluelink-manager-darwin-amd64 ./cmd
GOOS=darwin GOARCH=arm64 go build -o bluelink-manager-darwin-arm64 ./cmd

# Linux
GOOS=linux GOARCH=amd64 go build -o bluelink-manager-linux-amd64 ./cmd
GOOS=linux GOARCH=arm64 go build -o bluelink-manager-linux-arm64 ./cmd

# Windows
GOOS=windows GOARCH=amd64 go build -o bluelink-manager-windows-amd64.exe ./cmd
```

### Running Tests

```bash
go test ./...
```

## Architecture

```
tools/bluelink-manager/
├── cmd/
│   ├── main.go              # Entry point
│   └── commands/            # Cobra commands
│       ├── root.go
│       ├── install.go
│       ├── update.go
│       ├── uninstall.go
│       ├── status.go
│       ├── service.go       # start/stop/restart
│       ├── self_update.go
│       └── version.go
├── internal/
│   ├── config/              # Auth configuration
│   ├── github/              # GitHub API client for downloads
│   ├── paths/               # Platform-specific paths
│   ├── plugins/             # Plugin installation
│   ├── service/             # Service management (launchd/systemd/Windows)
│   │   ├── launchd.go       # macOS (//go:build darwin)
│   │   ├── systemd.go       # Linux (//go:build linux)
│   │   └── windows.go       # Windows (//go:build windows)
│   ├── shell/               # PATH modification
│   │   ├── profile_unix.go  # bash/zsh/fish
│   │   └── profile_windows.go # Registry
│   └── ui/                  # Terminal output formatting
└── install.sh               # Bootstrap script for Unix
```
