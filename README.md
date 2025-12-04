![Bluelink](/resources/logo.svg)

Infrastructure management simplified

- [Contributing](./CONTRIBUTING.md)
- [Architecture Overview](./ARCHITECTURE_OVERVIEW.md)
- [Docs Site](https://bluelink.dev)

# Components of Bluelink

## Blueprints

Blueprints are a way to define and manage the lifecycle of resources defined in a blueprint. Blueprints adhere to a [specification](https://www.bluelink.dev/docs/bluelink/blueprints/specification/) that is the foundation of the Bluelink resource model used at build time and runtime.
The specification is broad and can be used to define resources in any environment (e.g. Cloud provider resources from the likes of AWS, Azure and Google Cloud).

### Blueprint Framework

The blueprint framework provides a set of interfaces and tools to deploy and manage the lifecycle of resources that can be represented as a blueprint. The framework is designed to be minimal at its core, yet extensible and can be used to deploy resources in any environment.

The blueprint framework is an implementation of the [Bluelink Blueprint Specification](https://www.bluelink.dev/docs/bluelink/blueprints/specification/).

[Blueprint Framework](./libs/blueprint)

### Blueprint Language Server (blueprint-ls)

`blueprint-ls` is a language server that provides LSP support for the Blueprint Specification. The language server provides features such as code completion, go to definitions and diagnostics.

The language server can be used with any language server protocol compatible editor such as Visual Studio Code, NeoVim,  Atom etc.

The language server supports `yaml` and `jsonc` formats. `jsonc` refers to the [JSON with Commas and Comments](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html) extension to provide a more intuitive experience for writing blueprints with a JSON-based syntax. This can also be referred to as "Human JSON" and editor-specific extensions that are clients of the language server will usually support the `.jsonc` and `.hujson` file extensions.

[Blueprint Language Server](./tools/blueprint-ls)

### Deploy Engine

The deploy engine is the application that is responsible for deploying blueprints to target environments. It brings together the blueprint framework, the plugin framework and persistence layers to provide a complete and extensible solution for deploying infrastructure defined in blueprints.

The deploy engine exposes a HTTP API that can be used to validate blueprints, stage changes and deploy changes to target environments. As a part of the HTTP API, events can be streamed over Server-Sent Events (SSE) to provide real-time updates on the status of deployments, change staging and validation. 

[Deploy Engine](./apps/deploy-engine)

## Plugins

Plugins are foundational to Bluelink. Developers can create plugins to extend the capabilities of the deploy engine to deploy resources, source data and manage the lifecycle of resources in upstream providers (such as AWS, Azure, GCP).
There are two types of plugins, `Providers` and `Transformers`.
- **Providers** are plugins that are responsible for deploying resources to a target environment. Providers can be used to deploy resources to any environment, including cloud providers, on-premises environments and local development environments. In addition to resources, providers can also implement data sources, custom variable types and links between resource types.
- **Transformers** are plugins that are responsible for transforming blueprints. These are powerful plugins that enable abstract resources that can be defined by users and then transformed into concrete resources that can be deployed to a concrete target environment. For example, the [Celerity](https://celerityframework.io) application primitives are abstract resources that are transformed into concrete resources at deploy time that can be deployed to a target environment such as AWS, Azure or Google Cloud.

### Plugin Framework

The plugin framework provides the foundations for a plugin system that uses gRPC over a local network that includes a Go SDK that provides a smooth plugin development experience where knowledge of the ins and outs of the underlying communication protocol is not required.

[Plugin Framework](./libs/plugin-framework)

## CLI

The Bluelink CLI brings all the components of Bluelink together. It is a command line tool that can be used to create, build, deploy and manage blueprints.
It also provides commands for installing and managing plugins, using the [Registry protocol](https://www.bluelink.dev/plugin-framework/docs/registry-protocols-formats/registry-protocol) to source, verify and install plugins from the official and custom plugin registries.

Under the hood, the CLI uses the deploy engine to validate blueprints, stage changes and deploy applications (or standalone blueprints) to target environments.
The CLI can use local or remote instances of the deploy engine, this can be configured using command line options, environment variables or configuration files.

[CLI](./apps/cli)

# Additional Documentation

- [Index of Projects](./docs/INDEX.md) - A full index of all the projects in the core Bluelink monorepo.
