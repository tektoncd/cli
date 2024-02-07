$ErrorActionPreference = 'Stop'; 
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'tektoncd-cli'
$url64       = 'https://github.com/tektoncd/cli/releases/download/v0.35.0/tkn_0.35.0_Windows_x86_64.zip'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  softwareName   = 'tektoncd-cli*' 
  checksum64     = 'dc8936b285715bbdd8eb9335ce674d852cf1b2cd959335ca6438f6faa8dfd539'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
