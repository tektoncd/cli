$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.32.2/tkn_0.32.2_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'b179199a2a2c7a551e2726f3c65be8df928b945701bc1d7d2b837a718bbdbdd6'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
