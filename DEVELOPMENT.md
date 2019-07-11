# Developing

## Getting started

1. Create [a GitHub account](https://github.com/join)
1. Setup
   [GitHub access via SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1. [Create and checkout a repo fork](#checkout-your-fork)
1. Set up your [shell environment](#environment-setup)
1. Install [requirements](#requirements)
1. [Set up a Kubernetes cluster](#kubernetes-cluster)

Then you can [iterate](#iterating).

### Checkout your fork

The Go tools require that you clone the repository to the
`src/github.com/tektoncd/cli` directory in your
[`GOPATH`](https://github.com/golang/go/wiki/SettingGOPATH).

To check out this repository:

1. Create your own
   [fork of this repo](https://help.github.com/articles/fork-a-repo/)
1. Clone it to your machine:

```shell
mkdir -p ${GOPATH}/src/github.com/tektoncd
cd ${GOPATH}/src/github.com/tektoncd
git clone git@github.com:${YOUR_GITHUB_USERNAME}/cli.git
cd cli
git remote add upstream git@github.com:tektoncd/cli.git
git remote set-url --push upstream no_push
```

_Adding the `upstream` remote sets you up nicely for regularly
[syncing your fork](https://help.github.com/articles/syncing-a-fork/)._

### Requirements

You must install these tools:

1. [`go`](https://golang.org/doc/install): The language Tekton
   Pipelines CLI is built in
1. [`git`](https://help.github.com/articles/set-up-git/): For source control
1. [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
   (optional): For interacting with your kube cluster

## Kubernetes cluster

Docker for Desktop using an edge version has been proven to work for both
developing and running Pipelines. Your Kubernetes version must be 1.11 or later.

To setup a cluster with GKE:

1. [Install required tools and setup GCP project](https://github.com/knative/docs/blob/master/docs/install/Knative-with-GKE.md#before-you-begin)
   (You may find it useful to save the ID of the project in an environment
   variable (e.g. `PROJECT_ID`).
1. [Create a GKE cluster](https://github.com/knative/docs/blob/master/docs/install/Knative-with-GKE.md#creating-a-kubernetes-cluster)

Note that
[the `--scopes` argument to `gcloud container cluster create`](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create#--scopes)
controls what GCP resources the cluster's default service account has access to;
for example to give the default service account full access to your GCR
registry, you can add `storage-full` to your `--scopes` arg.

## Environment Setup

To build the Tekton pipelines cli, you'll need to set `GO111MODULE=on`
environment variable to force `go` to use [go
modules](https://github.com/golang/go/wiki/Modules#quick-start).

## Iterating

While iterating on the project, you may need to:

1. [Install Tekton pipeline](#install-pipeline)
2. [Building Tekton client](#building-tekton-client)
1. [Add and run tests](./test/README.md#tests)

## Building Tekton client

**Dependencies:**

[go mod](https://github.com/golang/go/wiki/Modules#quick-start) is used and required for dependencies.

**Building In Local Directory:**

The following command builds the `tkn` binary in your current directory:

```sh
go build ./cmd/tkn
```

After it finishes, you can run the following to execute the binary to
verify it works:

```sh
./tkn
```

**Building and Adding to $PATH:**

If you want this `tkn` binary on your `$PATH`, a couple options are:


1. Use `go install ./cmd/tkn` to install the binary into your `$GOBIN`. Rerun
`go install ./cmd/tkn` when you want to update the binary.

2. Add a soft link to the binary into your `$PATH`. Rerun `go build` when
   you want to update the binary.

   ```bash
   go build ./cmd/tkn
   ln -s $PWD/tkn $GOPATH/bin/tkn
   ```
3. After it finishes, you can run the following to execute the binary to verify it works:

   ```sh
   ./tkn
   ```

Whether you have added the `tkn` binary to your current directory or added it to
your `$PATH`, you are should now be ready to make changes to `tkn`.

**Notes:**

- For building, Go `1.12` is required
- If you are building in your `$GOPATH` folder, you need to specify `GO111MODULE` for building it

```sh
# if you are building in your $GOPATH
GO111MODULE=on go build ./cmd/tkn
```

You can now try updating code for client and test out the changes by building the `tkn` binary.


## Install Pipeline

You can stand up a version of this controller on-cluster (to your
`kubectl config current-context`):

- using a `release.yaml` file from the Tekton pipelines
  [releases](https://github.com/tektoncd/cli/releases)
- using [tektoncd/cli](https://github.com/tektoncd/cli)
  sources, following
  [DEVELOPMENT.md](https://github.com/tektoncd/cli/blob/master/DEVELOPMENT.md).
