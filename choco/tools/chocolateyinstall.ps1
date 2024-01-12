$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.34.0/tkn_0.34.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'b87ceebeea935522faab40d420d5893d52de01437fb386eeab9556167e6b20fe'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
