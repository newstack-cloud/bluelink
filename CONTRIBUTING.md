# Contributing to Bluelink

## Setup

Ensure git uses the custom directory for git hooks so the pre-commit and commit-msg linting hooks
kick in.

```bash
git config core.hooksPath .githooks
```

### NPM dependencies

There are npm dependencies that provide tools that are used in git hooks and scripting that span
multiple applications and libraries.

Install dependencies from the root directory by simply running:
```bash
yarn
```

## Build-time

Tools and libraries that make up the Bluelink "build-time engine" that handles everything involved in validating source code/configuration, parsing, packaging and deploying Bluelink backend services.

### Binary applications

- [Bluelink CLI](./apps/cli)
- [Bluelink Deploy Engine](./apps/deploy-engine)

### Libraries

- [Blueprint](./libs/blueprint)
- [Common](./libs/common)
- [Plugin Framework](./libs/plugin-framework)
- [Blueprint Resolvers](./libs/blueprint-resolvers)
- [Blueprint State](./libs/blueprint-state)
- [Deploy Engine Client](./libs/deploy-engine-client)

## Further documentation

- [Commit Guidelines](./COMMIT_GUIDELINES.md)
- [Source Control and Release Strategy](./SOURCE_CONTROL_RELEASE_STRATEGY.md)