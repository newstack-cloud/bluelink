# Source Control & Release Strategy

## Source control & development workflow

- Development work by core contributes should be carried out on the main branch for most contributions, with the exception being longer projects (weeks or months worth of work) or experimental new versions of a package or application. For the exceptions, feature/hotfix branches should be used.
- All development work by non-core contributes should be carried out on feature/hotfix branches on your fork, pull requests should be utilised for code reviews and merged (**rebase!**) back into the main branch of the primary repo.
- All commits should follow the [commit guidelines](./COMMIT_GUIDELINES.md).
- Work should be commited in small, specific commits where it makes sense to do so.

## Release strategy

Every component (library, app, or tool) has its own version. Tags use Go module conventions with the full path prefix:

```
{path}/vMAJOR.MINOR.PATCH

e.g. libs/blueprint/v0.37.0
     apps/deploy-engine/v1.0.0
     tools/blueprint-ls/v1.0.0
```

Each component specifies its release tag format in its CONTRIBUTING.md.

You will find each component listed in the commit scopes section of the [commit guidelines](./COMMIT_GUIDELINES.md#commit-scopes).

## Release workflow

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching a component (e.g., `feat(blueprint): ...` or `fix(cli): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - **Short tag** (e.g., `blueprint/v0.37.0`) - Used internally by release-please for tracking. Do not use this tag for anything else.
     - **Full path tag** (e.g., `libs/blueprint/v0.37.0`) - The canonical Go module tag. Use this for dependencies, workflows, and references.

4. **Post-release automation**:
   - **Libraries**: The `index-go-library.yml` workflow automatically indexes new versions with pkg.go.dev
   - **Apps/Tools**: Separate workflows build and publish artifacts (Docker images, binaries) triggered by tag patterns
