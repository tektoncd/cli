$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.23.0/tkn_0.23.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '377c228c3c3266dbfec0081fda2216760f6059eb7c40469ce7ac873d4795fab0'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
