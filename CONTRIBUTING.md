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
- [Bluelink API](./apps/api)

### Libraries

- [Blueprint](./libs/blueprint)
- [Deploy Engine](./libs/deploy-engine)
- [Common](./libs/common)

## Further documentation

- [Commit Guidelines](./COMMIT_GUIDELINES.md)
- [Source Control and Release Strategy](./SOURCE_CONTROL_RELEASE_STRATEGY.md)