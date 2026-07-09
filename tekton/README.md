# Tekton CLI Repo CI/CD

We dogfood our project by using Tekton Pipelines to build, test, and
release the Tekton Pipelines cli!

This directory contains the
[`Tasks`](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md) and
[`Pipelines`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md)
that we use for releasing the cli.

TODO(tektoncd/pipeline#538): In tektoncd/pipeline#538 or tektoncd/pipeline#537 we will update
[Prow](https://github.com/tektoncd/pipeline/blob/master/CONTRIBUTING.md#pull-request-process)
to invoke these `Pipelines` automatically, but, for now, we invoke them manually.

## Release Pipeline

You can use the script [`./release.sh`](release.sh) that will do everything
that needs to be done for the release. 

The first argument to `release.sh` is the release version. It
needs to be in a format like `v1.2.3`, which is basically SEMVER.

Release branches use the `release-vX.Y.x` naming convention (e.g.,
`release-v0.43.x`). The script automatically determines the branch name from
the release version.

- **New minor release** (e.g., `v0.46.0`): The script creates a new
  `release-v0.46.x` branch from `main`.
- **Patch release** (e.g., `v0.43.3`): The script checks out the existing
  `release-v0.43.x` branch. All patch fixes should already be merged into
  the `.x` branch via pull requests before running the release.

The release script will ask you to provide a GitHub token with appropriate priviledges as 
documented under the [release process prerequisites](../RELEASE_PROCESS.md#Prerequisites).

It will then use your Kubernetes cluster with Tekton and apply what needs to be done for
running the release.

Finally it will launch the `tkn` cli that you have installed locally to show the logs.

## Debugging

You can define the env variable `PUSH_REMOTE` to push to your own remote (i.e
which point to your username on github).

You need to be **careful** if you have write access to the `Homebrew` repository since it
will do a release there. Until we can find a proper fix, `release.sh` generates a commit that 
removes the brews data and cherry-picks it in the script.
