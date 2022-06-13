$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.24.0/tkn_0.24.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '618683DC0E226373814560ACB260FA31A806648825AA184F623AF086E74FD3E9'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
