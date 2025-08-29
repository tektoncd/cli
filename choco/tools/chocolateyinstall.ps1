$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.42.0/tkn_0.42.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'd49db0ce35f4b6c38d83048e35d9eb8f286a2d251a6bd365a8124e730069ebc7'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
