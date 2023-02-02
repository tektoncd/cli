# Tekton Pipelines CLI (`tkn`)

[![Go Report Card](https://goreportcard.com/badge/tektoncd/cli)](https://goreportcard.com/report/tektoncd/cli)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/6510/badge)](https://bestpractices.coreinfrastructure.org/projects/6510)

<p align="center">
<img width="250" height="175" src="https://github.com/cdfoundation/artwork/blob/main/tekton/additional-artwork/tekton-cli/color/tektoncli_color.svg" alt="Tekton logo"></img>
</p>

The _Tekton Pipelines CLI_ project provides a command-line interface (CLI) for interacting with [Tekton](https://tekton.dev/), an open-source framework for Continuous Integration and Delivery (CI/CD) systems.

## Installing `tkn`

Download the latest binary executable for your operating system.

### Mac OS X

- Use [Homebrew](https://brew.sh)

```shell
  brew install tektoncd-cli
```

- Use [released tarball](https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Darwin_all.tar.gz)

  ```shell
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Darwin_all.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.29.1_Darwin_all.tar.gz -C /usr/local/bin tkn
  ```

### Windows

- Use [Chocolatey](https://chocolatey.org/packages/tektoncd-cli)

```shell
choco install tektoncd-cli --confirm
```

- Use [Scoop](https://scoop.sh)
```powershell
scoop install tektoncd-cli
```

- Use [Powershell](https://docs.microsoft.com/en-us/powershell) [released zip](https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Windows_x86_64.zip)

```powershell
#Create directory
New-Item -Path "$HOME/tektoncd/cli" -Type Directory
# Download file
Start-BitsTransfer -Source https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Windows_x86_64.zip -Destination "$HOME/tektoncd/cli/."
# Uncompress zip file
Expand-Archive $HOME/tektoncd/cli/*.zip -DestinationPath C:\Users\Developer\tektoncd\cli\.
#Add to Windows `Environment Variables`
[Environment]::SetEnvironmentVariable("Path",$($env:Path + ";$Home\tektoncd\cli"),'User')
```

### Linux tarballs

* [Linux AMD 64](https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_x86_64.tar.gz)

  ```shell
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_x86_64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.29.1_Linux_x86_64.tar.gz -C /usr/local/bin/ tkn
  ```

* [Linux AARCH 64](https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_aarch64.tar.gz)

  ```shell
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_aarch64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.29.1_Linux_aarch64.tar.gz -C /usr/local/bin/ tkn
  ```

* [Linux IBM Z](https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_s390x.tar.gz)

  ```shell
  # Get the tar.gz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_s390x.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.29.1_Linux_s390x.tar.gz -C /usr/local/bin/ tkn
  ```

* [Linux IBM P](https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_ppc64le.tar.gz)

  ```shell
  # Get the tar.gz
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.1/tkn_0.29.1_Linux_ppc64le.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  sudo tar xvzf tkn_0.29.1_Linux_ppc64le.tar.gz -C /usr/local/bin/ tkn
  ```

### Linux RPMs

  If you are running on any of the following rpm based distros:

  * Latest Fedora and the two versions behind.
  * Centos Stream
  * EPEL
  * Latest RHEL

  you would be able to use [@chmouel](https://github.com/chmouel)'s unofficial copr package
  repository by running the following commands:

  ```shell
  dnf copr enable chmouel/tektoncd-cli
  dnf install tektoncd-cli
  ```

  * [Binary RPM package](https://github.com/tektoncd/cli/releases/download/v0.29.1/tektoncd-cli-0.29.1_Linux-64bit.rpm)

  On any other RPM based distros, you can install the rpm directly:

   ```shell
    rpm -Uvh https://github.com/tektoncd/cli/releases/download/v0.29.1/tektoncd-cli-0.29.1_Linux-64bit.rpm
   ```

### Linux Debs

  * [Ubuntu PPA](https://launchpad.net/~tektoncd/+archive/ubuntu/cli/+packages)

  If you are running on the latest rolling Ubuntu or Debian, you can use the TektonCD CLI PPA:

  ```shell
  sudo apt update;sudo apt install -y gnupg
  sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 3EFE0E0A2F2F60AA
  echo "deb http://ppa.launchpad.net/tektoncd/cli/ubuntu jammy main"|sudo tee /etc/apt/sources.list.d/tektoncd-ubuntu-cli.list
  sudo apt update && sudo apt install -y tektoncd-cli
  ```

  The PPA may work with older releases, but that hasn't been tested.

  * [Binary DEB package](https://github.com/tektoncd/cli/releases/download/v0.29.1/tektoncd-cli-0.29.1_Linux-64bit.deb)

  On any other Debian or Ubuntu based distro, you can simply install the binary package directly with `dpkg`:

  ```shell
  curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.1/tektoncd-cli-0.29.1_Linux-64bit.deb
  dpkg -i tektoncd-cli-0.29.1_Linux-64bit.deb
  ```

### NixOS/Nix

You can install `tektoncd-cli` from [nixpkgs](https://github.com/NixOS/nixpkgs) on any system that supports the `nix` package manager.

```shell
nix-env --install tektoncd-cli
```
### Arch / Manjaro

You can install [`tektoncd-cli`](https://archlinux.org/packages/community/x86_64/tekton-cli/) from the official arch package repository :

```shell
pacman -S tektoncd-cli
```

### Homebrew on Linux

You can install the latest tektoncd-cli if you are using [Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux) as for the osx version you need to simply do :

```shell
brew install tektoncd-cli
```

### Source install

  If you have [go](https://golang.org/) installed and you want to compile the CLI from source, you can checkout the [Git repository](https://github.com/tektoncd/cli) and run the following commands:

  ```shell
  make bin/tkn
  ```

  This will output the `tkn` binary in `bin/tkn`

### `tkn` as a `kubectl` plugin

`kubectl` will find any binary named `kubectl-*` on your PATH and consider it as a plugin.
After installing tkn, create a link as kubectl-tkn
  ```shell
ln -s /usr/local/bin/tkn /usr/local/bin/kubectl-tkn
  ```
Run the following to confirm tkn is available as a plugin:
  ```shell
kubectl plugin list
  ```
You should see the following after running kubectl plugin list if tkn is available as a plugin:
  ```shell
/usr/local/bin/kubectl-tkn
```
If the output above is shown, run kubectl-tkn to see the list of available tkn commands to run.

## Useful Commands

The following commands help you understand and effectively use the Tekton CLI:

 * `tkn help:` Displays a list of the commands with helpful information.
 * [`tkn bundle:`](docs/cmd/tkn_bundle.md) Manage Tekton [bundles](https://github.com/tektoncd/pipeline/blob/main/docs/tekton-bundle-contracts.md)
 * [`tkn clustertask:`](docs/cmd/tkn_clustertask.md) Parent command of the ClusterTask command group.
 * [`tkn clustertriggerbinding:`](docs/cmd/tkn_clustertriggerbinding.md) Parent command of the ClusterTriggerBinding command group.
 * [`tkn completion:`](docs/cmd/tkn_completion.md) Outputs a BASH, ZSH, Fish or PowerShell completion script for `tkn` to allow command completion with Tab.
 * [`tkn eventlistener:`](docs/cmd/tkn_eventlistener.md) Parent command of the Eventlistener command group.
 * [`tkn hub:`](docs/cmd/tkn_hub.md) Search and install Tekton Resources from [Hub](https://hub.tekton.dev)
 * [`tkn pipeline:`](docs/cmd/tkn_pipeline.md) Parent command of the Pipeline command group.
 * [`tkn pipelinerun:`](docs/cmd/tkn_pipelinerun.md) Parent command of the Pipelinerun command group.
 * [`tkn resource:`](docs/cmd/tkn_resource.md) Parent command of the Resource command group.
 * [`tkn task:`](docs/cmd/tkn_task.md) Parent command of the Task command group.
 * [`tkn taskrun:`](docs/cmd/tkn_taskrun.md) Parent command of the Taskrun command group.
 * [`tkn triggerbinding:`](docs/cmd/tkn_triggerbinding.md) Parent command of the Triggerbinding command group.
 * [`tkn triggertemplate:`](docs/cmd/tkn_triggertemplate.md) Parent command of the Triggertemplate command group.
 * [`tkn version:`](docs/cmd/tkn_version.md) Outputs the cli version.

For every `tkn` command, you can use `-h` or `--help` flags to display specific help for that command.

## Disable Color and Emojis in Output

For many `tkn` commands, color and emojis by default will appear in command
output.

It will only shows if you are in interactive shell with a [standard
input](https://en.wikipedia.org/wiki/Standard_streams#Standard_input_(stdin))
attached. If you pipe the tkn command or run it in a non interactive way (ie:
from tekton itself in a Task) the coloring and emojis will *always* be disabled.

`tkn` offers two approaches for disabling color and emojis from command output.

To remove the color and emojis from all `tkn` command output, set the environment variable `NO_COLOR`, such as shown below:

```shell
export NO_COLOR=""
```

More information on `NO_COLOR` can be found in the [`NO_COLOR` documentation](https://no-color.org/).

To remove color and emojis from the output of a single command execution, the `--no-color` option can be used with any command,
such as in the example below:

```bash
tkn taskrun describe --no-color
```


## Want to contribute

We are so excited to have you!

- See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
- See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
- See [ROADMAP.md](ROADMAP.md) for the current roadmap
- See [releases.md][releases.md] for our release cadence and processes
- Look at our
  [good first issues](https://github.com/tektoncd/cli/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our
  [help wanted issues](https://github.com/tektoncd/cli/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
