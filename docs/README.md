---
title: "CLI"
linkTitle: "CLI"
weight: 5
description: >
  Command-Line Interface
---

{{% pageinfo %}}
This document is a work in progress.
{{% /pageinfo %}}

## Overview

Tekton provides a CLI, `tkn`, for easier interaction with Tekton components.
It is available as a binary executable on major platforms; you may also build
it from the source, or set it up as a [`kubectl` plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/).

## Installation

{{% tabs %}}

{{% tab "macOS" %}}
`tkn` is available on macOS via [`brew`](https://brew.sh/):

```bash
brew tap tektoncd/tools
brew install tektoncd/tools/tektoncd-cli
```

You can also download it as a tarball from the [`tkn` Releases page](https://github.com/tektoncd/cli/releases).
After downloading the file, extract it to your `PATH`:

```bash
# Replace YOUR-DOWNLOADED-FILE with the file path of your own.
sudo tar xvzf YOUR-DOWNLOADED-FILE -C /usr/local/bin/ tkn
```

{{% /tab %}}

{{% tab "Windows" %}}
`tkn` is available on Windows via [Chocolatey](https://chocolatey.org/):

```cmd
choco install tektoncd-cli --confirm
```

You can also download it as a `.zip` file from the [`tkn` Releases page](https://github.com/tektoncd/cli/releases).
After downloading the file, add it to your `Path`:

* Uncompress the `.zip` file.
* Open **Control Panel** > **System and Security** > **System** > **Advanced System Settings**.
* Click **Environment Variables**, select the `Path` variable and click **Edit**.
* Click **New** and add the path to your uncompressed file.
* Click **OK**.
{{% /tab %}}

{{% tab "Linux" %}}
`tkn` is available on Linux as a `.deb` package (for Debian, Ubuntu and
other deb-based distros) and `.rpm` package (for Fedora, CentOS, and other
rpm-based distros).

* Debian, Ubuntu, and other deb-based distros

    Find the `.deb` package of the `tkn` release you would like to install on
    the [`tkn` Releases page](https://github.com/tektoncd/cli/releases) and
    install it with

    ```bash
    # Replace LINK-TO-THE-PACKAGE with the package URL you would like to use.
    rpm -Uvh LINK-TO-THE-PACKAGE
    ```

    If you are using the latest releases of Ubuntu or Debian, you may use the
    TektonCD CLI PPA instead:

    ```bash
    sudo apt update;sudo apt install -y gnupg
    sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 3EFE0E0A2F2F60AA
    echo "deb http://ppa.launchpad.net/tektoncd/cli/ubuntu eoan main"|sudo tee /etc/apt/sources.list.d/tektoncd-ubuntu-cli.list
    sudo apt update && sudo apt install -y tektoncd-cli
    ```

* Fedora, CentOS, and other rpm-based distros

    Find the `.rpm` package of the `tkn` release you would like to install on
    the [`tkn` Releases page](https://github.com/tektoncd/cli/releases) and
    install it with

    ```bash
    # Replace LINK-TO-THE-PACKAGE with the package URL you would like to use.
    rpm -Uvh LINK-TO-THE-PACKAGE
    ```

    If you are using Fedora 30/31, CentOS 7/8, EPEL, or RHEL 8, @chmousel
    provides an unofficial `copr` package repository for installing the
    package:

    ```bash
    dnf copr enable chmouel/tektoncd-cli
    dnf install tektoncd-cli
    ```

Alternatively, you may download `tkn` as a tarball:

Find the tarball of the `tkn` release for your platform (`ARM` or `X86-64`)
you would like to install on the [`tkn` Releases page](https://github.com/tektoncd/cli/releases)
and install it with

```bash
# Replace LINK-TO-TARBALL with the package URL you would like to use.
curl -LO LINK-TO-TARBALL
# Replace YOUR-DOWNLOADED-FILE with the file path of your own.
sudo tar xvzf YOUR-DOWNLOADED-FILE -C /usr/local/bin/ tkn
```
{{% /tab %}}

{{% /tabs %}}

### Build from source

If you would like to build `tkn` from the source, set up your [`Go`](https://golang.org/) development
environment, clone the [GitHub repository for `tkn`](https://github.com/tektoncd/cli),
and run the following commands in the cloned directory:

```bash
export GO111MODULE=on
make bin/tkn
```

The `tkn` executable will be available at `/bin`.

### Add as a `kubectl` plugin

To add `tkn` as a `kubectl` plugin, run the following commands:

```bash
# The following commands assumes that your `tkn` executable is available
# at /usr/local/bin/tkn. You may need to use a different value on your
# system.
ln -s /usr/local/bin/tkn /usr/local/bin/kubectl-tkn
kubectl plugin list
```

If configured correctly, you should see `tkn` listed in the output.

## Usage

Run `tkn help` to see the list of available commands.

## What's next

[A manual for using Tekton CLI is available on GitHub](https://github.com/tektoncd/cli/tree/master/docs).
