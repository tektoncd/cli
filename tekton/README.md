# Tekton CLI Repo CI/CD

We dogfood our project by using Tekton Pipelines to build, test, and
release the Tekton Pipelines cli!

This directory contains the
[`Tasks`](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md) and
[`Pipelines`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md)
that we use for releases and pull requests.

TODO(tektoncd/pipeline#538): In tektoncd/pipeline#538 or tektoncd/pipeline#537 we will update
[Prow](https://github.com/tektoncd/pipeline/blob/master/CONTRIBUTING.md#pull-request-process)
to invoke these `Pipelines` automatically, but, for now, we invoke them manually.

## Release Pipeline

You can use the script [`./release.sh`](release.sh) that will do everything
that needs to be done for the release. In order to run `release.sh`, you will 
need [these tools](https://github.com/tektoncd/cli/blob/master/tekton/release.sh#L13y) installed locally.

The first argument to `release.sh` is the release version. It
needs to be in a format like `v1.2.3`, which is basically SEMVER.

If it detects that you are doing a minor release, it will ask you for some
commits to be cherry-picked in this release. Alternatively, you can provide the
commits separated by a space to the second argument of the script. You do need to
make sure to provide them in order from the oldest to the newest.

If you give the `*` argument for the commits to pick up, it would apply all the new
commits.

It will them use your Kubernetes cluster with Tekton and apply what needs to be done for
running the release.

Finally it will launch the `tkn` cli that you have installed locally to show the logs.

Make sure we have a nice ChangeLog before doing the release, listing `features`
and `bugs` and be thankful to the contributors by listing them. An example is shown [here](https://github.com/tektoncd/cli/releases/tag/v0.6.0).

## Debugging

You can define the env variable `PUSH_REMOTE` to push to your own remote (i.e
which point to your username on github).

You need to be **careful** if you have write access to the `Homebrew` repository since it
will do a release there. Until we can find a proper fix, `release.sh` generates a commit that 
removes the brews data and cherry-picks it in the script.
