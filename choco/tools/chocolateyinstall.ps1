$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.32.1/tkn_0.32.1_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '14dd923656ea9426b71e50d4f0856b03c341ee3f7453082061129bec1d6b7161'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
