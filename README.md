# Tekton Pipelines cli

[![Go Report Card](https://goreportcard.com/badge/tektoncd/cli)](https://goreportcard.com/report/tektoncd/cli)

The Tekton Pipelines cli project provides a CLI for interacting with Tekton!

## Getting Started

### Installing `tkn`

Download the latest binary executable for your operating system:

* Mac OS X

  - `tektoncd-cli` can be installed as a [brew tap](https://brew.sh):

  ```shell
  brew tap tektoncd/tools
  brew install tektoncd/tools/tektoncd-cli
  ```

  - Or by the [released tarball](https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Darwin_x86_64.tar.gz):

  ```shell
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Darwin_x86_64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.5.1_Darwin_x86_64.tar.gz -C /usr/local/bin tkn
  ```

* Windows

  - Uncompress the [zip file](https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Windows_x86_64.zip)
  - Add the location of where the executable is to your `Path` by opening `Control Panel>System and Security>System>Advanced System Settings`
  - Click on `Environment Variables`, select the `Path` variable, and click `Edit`
  - Click `New` and add the location of the uncompressed zip to the `Path`
  - Finish by clicking `Ok`

* [Linux AMD 64](https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Linux_x86_64.tar.gz)

  ```shell
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Linux_x86_64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.5.1_Linux_x86_64.tar.gz -C /usr/local/bin/ tkn
  ```

* [Linux ARM 64](https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Linux_arm64.tar.gz)

  ```shell
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.5.1/tkn_0.5.1_Linux_arm64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.5.1_Linux_arm64.tar.gz -C /usr/local/bin/ tkn
  ```

  If you are running on any of the following rpm based distros:

  * Fedora30
  * Fedora31
  * Centos7
  * Centos8
  * EPEL
  * RHEL8

  you would be able to use [@chmouel](https://github.com/chmouel)'s unofficial copr package
  repository by running the following commands:

  ```shell
  dnf copr enable chmouel/tektoncd-cli
  dnf install tektoncd-cli
  ```

* Source install

  If you have [go](https://golang.org/) installed, and you want to compile the CLI from source you can checkout the [Git repository](https://github.com/tektoncd/cli) and run the following commands:

  ```shell
  export GO111MODULE=on
  make bin/tkn
  ```
  This will output the `tkn` binary in `bin/tkn`

### `tkn` as a `kubectl` plugin

`kubectl` will find any binary named `kubectl-*` on your PATH and consider it as a plugin.
After installing tkn, create a link as kubectl-tkn
  ```shell
$ ln -s /usr/local/bin/tkn /usr/local/bin/kubectl-tkn
  ```

`kubectl` will find any binary named `kubectl-*` on your `PATH` and consider it as a plugin

  ```shell
$ ./kubectl plugin list
/usr/local/bin/kubectl-tkn
  ```

### Useful Commands

The following commands help you understand and effectively use the Tekton CLI:

 * `tkn help:` Displays a list of the commands with helpful information.
 * [`tkn completion:`](docs/cmd/tkn_completion.md) Outputs a BASH or ZSH completion script for `tkn` to allow command completion with Tab.
 * [`tkn version:`](docs/cmd/tkn_version.md) Outputs the cli version.
 * [`tkn pipeline:`](docs/cmd/tkn_pipeline.md) Parent command of the Pipeline command group.
 * [`tkn pipelinerun:`](docs/cmd/tkn_pipelinerun.md) Parent command of the Pipelinerun command group.
 * [`tkn task:`](docs/cmd/tkn_task.md) Parent command of the Task command group.
 * [`tkn taskrun:`](docs/cmd/tkn_taskrun.md) Parent command of the Taskrun command group.
 * [`tkn clustertask:`](docs/cmd/tkn_clustertask.md) Parent command of the ClusterTask command group.
 * [`tkn resource:`](docs/cmd/tkn_resource.md) Parent command of the Resource command group.
 * [`tkn condition:`](docs/cmd/tkn_condition.md) Parent command of the Condition command group.

For every `tkn` command, you can use `-h` or `--help` flags to display specific help for that command.

## Want to contribute

We are so excited to have you!

- See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
- See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
- Look at our
  [good first issues](https://github.com/tektoncd/cli/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our
  [help wanted issues](https://github.com/tektoncd/cli/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
