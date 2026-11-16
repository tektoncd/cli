$ErrorActionPreference = 'Stop';
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.47.0/tkn_0.47.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*'
  checksum64     = 'a96ee17daf3f203adaf39ce33ceac9a93553b7557a86b5281f8e53368452341b'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
