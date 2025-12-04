# Index of Projects

This document provides an overview of all projects in the Bluelink monorepo.

## Applications

| Project | Path | Description |
|---------|------|-------------|
| [CLI](../apps/cli) | `apps/cli` | Command-line interface for managing blueprints, deploying infrastructure, and managing Bluelink plugins |
| [Deploy Engine](../apps/deploy-engine) | `apps/deploy-engine` | The engine that validates and deploys blueprints, bundling the plugin framework and state persistence |

## Libraries

| Project | Path | Description |
|---------|------|-------------|
| [Blueprint](../libs/blueprint) | `libs/blueprint` | Core framework for blueprint deployments, implementing the [Blueprint Specification](https://www.bluelink.dev/docs/bluelink/blueprints/specification/) |
| [Blueprint Resolvers](../libs/blueprint-resolvers) | `libs/blueprint-resolvers` | Collection of `ChildResolver` implementations for sourcing child blueprints (File system, S3, GCS, Azure Blob, HTTPS) |
| [Blueprint State](../libs/blueprint-state) | `libs/blueprint-state` | State container implementations for blueprint entity management (Postgres, in-memory with file persistence) |
| [Common](../libs/common) | `libs/common` | Shared utility packages used across Bluelink projects |
| [Deploy Engine Client](../libs/deploy-engine-client) | `libs/deploy-engine-client` | Go client library for the Deploy Engine API |
| [Plugin Framework](../libs/plugin-framework) | `libs/plugin-framework` | gRPC-based plugin system for provider and transformer plugins, including a Go SDK |

## Tools

| Project | Path | Description |
|---------|------|-------------|
| [Bluelink Manager](../tools/bluelink-manager) | `tools/bluelink-manager` | Cross-platform CLI for installing, updating, and managing Bluelink components |
| [Blueprint Language Server](../tools/blueprint-ls) | `tools/blueprint-ls` | LSP-compatible language server for `.blueprint.(yaml\|yml\|jsonc)` files |
| [Plugin Docgen](../tools/plugin-docgen) | `tools/plugin-docgen` | Generates JSON documentation from plugins for the Bluelink Registry |
| [Windows Installer](../tools/windows-installer) | `tools/windows-installer` | WiX-based MSI installer for Windows |
