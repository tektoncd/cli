$ErrorActionPreference = 'Stop';
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.46.0/tkn_0.46.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*'
  checksum64     = 'aea8a3d5bdf3bec966501cf0d7655c7c9a13cb4e1ad53f5235db5a1375960d0c '
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
