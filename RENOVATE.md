# Dependency automation: Renovate + Dependabot (security-only)

All `newstack-cloud` repos use **Renovate for version updates** and keep
**Dependabot for security updates only**. Renovate also drives the Bluelink
release **cascade**: when a core library releases, Renovate opens the `deps(...)`
bump PRs in every consumer, which feed each repo's release-please (or svu) flow.

Config is layered so product concerns don't leak across product families:

```
newstack-cloud/renovate-config//default              ← org-wide policy; ALL repos
        ▲
        │ extends
newstack-cloud/bluelink//renovate/bluelink-consumers  ← Bluelink cascade only
        ▲
        │ extends
bluelink · celerity-provider-aws · deploy-cli-sdk
        (celerity-node-sdk extends ONLY the org preset — no Bluelink dependency)
```

- **Org layer** ([`newstack-cloud/renovate-config`](https://github.com/newstack-cloud/renovate-config))
  — universal, product-agnostic policy: schedule, the `deps` / `ci(deps)` commit
  convention, auto-merge tiers, and the Dependabot-security-only stance.
- **Bluelink layer** ([`renovate/bluelink-consumers.json`](renovate/bluelink-consumers.json))
  — the cascade rule: auto-merge internal `github.com/newstack-cloud/bluelink/*`
  bumps and let them bypass the weekly schedule. Only repos in the Bluelink graph
  extend it.

## Why

- release-please has no `go-workspace` plugin, so intra-monorepo Go bumps
  (`blueprint` → `plugin-framework` → `plugin-docgen` → …) can't self-cascade.
  They were previously hand-written `deps(scope): bump ...` commits.
- Dependabot's `go-deps` group (`patterns: ["*"]`) already bumped the internal
  `bluelink/*` modules, so running Renovate alongside it would produce competing
  PRs. One tool is cleaner.
- Renovate adds cross-repo cascade, tiered auto-merge, regex managers (e.g. the
  pinned `plugin-docgen` version in the AWS provider), and a dependency dashboard.

## How the cascade works

- **Bump commits keep the exact prefixes release-please expects**
  (`deps(<component>): …`, `ci(deps): …`, `deps(docker): …`). Only the tool that
  opens the PR changes and release behaviour does not.
- **Auto-merge tiers** (all CI-gated by `platformAutomerge`):
  - internal Bluelink module bumps → auto-merge, bypassing the weekly schedule so
    the cascade reacts promptly;
  - third-party minor/patch/digest → auto-merge;
  - majors → held for manual review (`major-update` label).
- **The human gate is the release PR.** Renovate lands the `deps(...)` bump on
  `main`; release-please/svu then opens a release PR for a person to review and merges.
  Merging publishes the release, which Renovate picks up in the next consumer.

```
blueprint release published
  → deps(plugin-framework): bump blueprint  (+ resolvers, state, ...)  [auto-merged]
    → release-please "release plugin-framework" PR  [human merges]
      → plugin-framework published
        → deps(plugin-docgen): bump plugin-framework  [auto-merged] → release PR → ...
        → cross-repo: celerity-provider-aws & deploy-cli-sdk get their bump PRs
          → deploy-cli-sdk release → deps(cli): bump deploy-cli-sdk in bluelink
```

## One-time setup

1. **Create the org preset repo.** Publish the staged
   `newstack-cloud/renovate-config` repo (its `default.json` + README) to GitHub on
   its default branch. Nothing resolves until this exists.
2. **`RENOVATE_TOKEN` secret** in each repo, a **GitHub App installation token or
   PAT**, *not* `GITHUB_TOKEN`. PRs opened by `GITHUB_TOKEN` don't trigger other
   workflows, so CI wouldn't run on Renovate PRs and auto-merge could never
   complete. The token needs **read access to `newstack-cloud/renovate-config`**
   (org preset) and, for Bluelink-graph repos, **`newstack-cloud/bluelink`**
   (cascade preset). One org-level GitHub App reused across repos is the tidiest approach.
3. **Dependabot → security-only.** The `.github/dependabot.yml` version-update
   files are removed. In each repo's **Settings → Advanced Security**, ensure
   **Dependabot alerts** and **Dependabot security updates** are enabled — security
   updates are a repo setting and need no config file.
4. The `renovate.yml` workflow runs hourly and on demand. Trigger a **dry run**
   first via *Actions → Renovate → Run workflow* with `dryRun: full`.

## Validate before updates

```bash
npx --yes --package renovate renovate-config-validator \
  renovate.json renovate/bluelink-consumers.json
```

(Run against `default.json` in the `renovate-config` repo as well when changes are made.)
