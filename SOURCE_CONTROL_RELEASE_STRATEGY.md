# Source Control & Release Strategy

## Source control & development workflow

- Development work by core contributes should be carried out on the main branch for most contributions, with the exception being longer projects (weeks or months worth of work) or experimental new versions of a package or application. For the exceptions, feature/hotfix branches should be used.
- All development work by non-core contributes should be carried out on feature/hotfix branches on your fork, pull requests should be utilised for code reviews and merged (**rebase!**) back into the main branch of the primary repo.
- All commits should follow the [commit guidelines](./COMMIT_GUIDELINES.md).
- Work should be commited in small, specific commits where it makes sense to do so.

## Release strategy

To allow for a high degree of flexibility, every key component has its own version. Tags used for releases need to be in the following format:

```
{app_or_package}-MAJOR.MINOR.PATCH

e.g. blueprint-0.1.0
```

Each key component will specify the release tag format in the README.

**_A key component consists of one or more related applications and libraries. (e.g. the nodejs runtime which consists of an application and multiple NPM packages)_**

You will find each key component listed in the commit scopes section of the [commit guidelines](./COMMIT_GUIDELINES.md#commit-scopes).

The automated tooling bundled with Bluelink will handle ensuring the correct release tags are produced and the corresponding artifacts are published, given you follow the release workflow outlined in the following section.

## Release workflow

This current iteration of the release workflow needs to be carried out for each key component individually.
The reason releases should be carried out individually is to ensure the neccessary level of care is taken when making new releases. Automating the workflow to automatically publish all the changed packages and applications across Bluelink would make it hard to track and review the changes made to each individually versioned key component, therefore leading to an increased likelihood of error-prone releases.

1. Ensure all relevant changes have been merged (rebased) into the trunk (main).
2. Create a new release branch for `release/{app_or_package}-MAJOR.MINOR.PATCH` (e.g. `release/blueprint-0.1.1`) with the approximate next version. (This branch is short-lived so it is not crucial to get the version 100% correct)
3. Push the release branch and this will trigger a GitHub actions workflow that will determine the actual version from commits and update the change log for the target application or library.
4. The automated workflow from step 3 will create a PR that generates a preliminary set of release notes. Review and edit the release notes accordingly and then rebase the PR into main. (These release notes will be used in a further automated release publishing step)
5. Rebasing the PR into main will trigger the process of creating the tag and release in GitHub along with building and publishing artifacts for the libraries and applications of the target key component.
