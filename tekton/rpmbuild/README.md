Tekton CLI RPM Build
====================

This is a Tekton task to run an rpm build on copr.

It only supports the latest release as released on GitHub. It queries the GitHub
api to get the latest release.

It uses the docker image from `quay.io/chmouel/rpmbuild`. The Dockerfile is located in
this directory.

The task uploads the release to
`https://copr.fedorainfracloud.org/coprs/chmouel/tektoncd-cli/`. The distros
supported are:

* Epel for CentOS 7
* Fedora 30/31
* RHEL8

You simply have to run this to get it installed:

```
dnf copr enable chmouel/tektoncd-cli
dnf install tektoncd-cli
```

USAGE
=====

Same as when you use the [release.pipeline.yaml](../release-pipeline.yml), you
need to have a PipelineResource for your git repository. See
[here](../release-pipeline-run.yml) for an example.

* You need to have your user added to the `https://copr.fedorainfracloud.org/coprs/chmouel/tektoncd-cli/` request it by going [here](https://copr.fedorainfracloud.org/coprs/chmouel/tektoncd-cli/permissions/) and asking for admin access.

* You need to get your API file from https://copr.fedorainfracloud.org/api/ and have it saved to `~/.config/copr`. You will need to change the 
`username` field to `chmouel` since the copr repo is currently `/chmouel/tektoncd-cli/`.

* Make sure you have the GitHub token set as documented in [RELEASE_PROCESS.md](../../RELEASE_PROCESS.md). You can also just add the `--namespace` option to all `kubectl` and `tkn` commands below and specify the namespace where you ran the release (by default, the namespace used in `release.sh` is `release`).

* You create the secret from that copr config file:

```
kubectl create secret generic copr-cli-config --from-file=copr=${HOME}/.config/copr
```

* Make sure you already have git-clone Task installed. To verify, run the command:

```
tkn task list | grep "git-clone"
```

If the above command doesn't return anything then you can install using:

```
tkn hub install task git-clone
```

* You should be able create the task with:

```
kubectl create -f rpmbuild.yml
```

And run it with:

```
kubectl create -f rpmbuild-run.yml
```

* Use `tkn pr desc rpmbuild-pipelinerun` to make sure the `PipelineRun` didn't fail on validation 

And use the following command to get the logs of the `PipelineRun`: 

```
tkn pr logs rpmbuild-pipelinerun -f
```
