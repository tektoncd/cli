$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.41.0/tkn_0.41.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = '3894e99917fb0962989bd8001a24031bfcf47e6f7fabec655b8b895ce2e1e6f3'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
