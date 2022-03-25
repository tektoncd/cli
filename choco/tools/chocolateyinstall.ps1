$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.23.1/tkn_0.23.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '7f8ffdf830e52b856e6b7e389bd62bad9e30e1c4e6e909bd80034a58a021ca13'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
