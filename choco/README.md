# tkn Chocolatey Package

[![badge](https://img.shields.io/chocolatey/v/tektoncd-cli)](https://chocolatey.org/packages/tektoncd-cli/)

[Chocolatey package](https://chocolatey.org/packages/tektoncd-cli/) 🍫 for the [tektoncd-cli](https://github.com/tektoncd/cli) (`tkn`).

This package can be installed by running the following:

```
choco install tektoncd-cli --confirm
```

## Development and Testing

### Install

1. Clone this repository and make updates to the package

2. In the `choco` folder of this repository, run `choco pack`

3. Run `choco install tektoncd-cli -s .` to run the package locally

After running the commands above, `tkn` should be installed. You can run `tkn` to verify a successful installation.

### Uninstall

Run `choco uninstall tektoncd-cli` to uninstall `tkn`. 

### Updating the Package to a New Version

To update this package to a newer version of `tkn`, the following updates should be made:

* Edit the version property in [tektoncd-cli.nuspec](./tektoncd-cli.nuspec#L5) to the latest available version of `tkn`. 
* Change the [version in the download url for the Windows `tkn` zip](./tools/chocolateyinstall.ps1#L4) to the latest available version of `tkn`. 
* Update the [checksum for the package](./tools/chocolateyinstall.ps1#L11) by getting the sha256 of the zip file. Example: `Get-FileHash '.\tkn_0.38.1_Windows_x86_64.zip'`. The zip can be downloaded using the release download url from GitHub; just remember to change the version numbers in the url: https://github.com/tektoncd/cli/releases/download/v0.38.1/tkn_0.38.1_Windows_x86_64.zip.
* Run `choco pack` in the `choco` directory to build the package
* Run `choco push` to push the built package and make it available for download (NOTE: You will need permissions to push the package/to be authenticated. Reach out to [@danielhelfand](https://github.com/danielhelfand) for more details.)
